package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

type statusServer struct {
	outStream  *bufio.Scanner
	updateCh   chan *statusUpdate
	shutdownCh chan struct{}
}

func newStatusServer(outStream io.Reader) *statusServer {
	return &statusServer{
		outStream:  bufio.NewScanner(outStream),
		updateCh:   make(chan *statusUpdate, 128),
		shutdownCh: make(chan struct{}),
	}
}

func (s *statusServer) shutdown() {
	close(s.shutdownCh)
}

func (s *statusServer) run() {
	// Start the update parser
	doneCh := make(chan struct{})
	defer close(doneCh)
	go s.handleUpdates(doneCh)

	for s.outStream.Scan() {
		payload := s.outStream.Text()

		// Used to parse and store a timestamp if given
		var ts int64
		var err error

		// Parse the payload parts
		parts := strings.Split(payload, "|")
		switch len(parts) {
		case 2:
			// Missing timestamp (will auto-generate)
		case 3:
			// Timestamp present, parse and use
			ts, err = strconv.ParseInt(parts[2], 10, 64)
			if err != nil {
				log.Printf("failed parsing timestamp: %v", err)
				continue
			}
		default:
			log.Printf("invalid metric payload: %v", payload)
			continue
		}

		// Parse the metric
		val, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			log.Printf("failed parsing metric value in %q: %v", payload, err)
			continue
		}

		// Send the update
		update := &statusUpdate{
			key:       parts[0],
			val:       val,
			timestamp: ts,
		}
		select {
		case s.updateCh <- update:
		default:
			log.Printf("update channel full! dropping update: %v", update)
		}
	}

	if err := s.outStream.Err(); err != nil {
		log.Printf("failed reading payload: %v", err)
		return
	}
}

func (s *statusServer) handleUpdates(doneCh <-chan struct{}) {
	// Used to store events and times
	metrics := make(map[int64]map[string]float64)

	// Record the start time
	start := time.Now().UnixNano()
	metrics[0] = map[string]float64{
		"running": 0,
	}

	for {
		select {
		case update := <-s.updateCh:
			// Check if timestamp is present
			ts := update.timestamp
			if ts == 0 {
				ts = time.Now().UnixNano()
			}

			// Compute elapsed time and log the value away
			elapsed := ts - start
			if _, ok := metrics[elapsed]; !ok {
				metrics[elapsed] = make(map[string]float64)
			}
			metrics[elapsed][update.key] = update.val

		case <-doneCh:
			if err := writeResult(metrics); err != nil {
				log.Fatalf("failed writing result: %v", err)
			}
			return
		}
	}
}

type Int64Sort []int64

func (s Int64Sort) Len() int {
	return len(s)
}

func (s Int64Sort) Less(a, b int) bool {
	return s[a] < s[b]
}

func (s Int64Sort) Swap(a, b int) {
	s[a], s[b] = s[b], s[a]
}

func writeResult(metrics map[int64]map[string]float64) error {
	// Create the output buffer
	buf := new(bytes.Buffer)

	// Write the field headers
	fieldsMap := make(map[string]struct{})
	for _, events := range metrics {
		for name, _ := range events {
			fieldsMap[name] = struct{}{}
		}
	}
	fields := make([]string, 0, len(fieldsMap))
	for field, _ := range fieldsMap {
		fields = append(fields, field)
	}
	sort.Strings(fields)
	fmt.Fprint(buf, "elapsed_ms,")
	fmt.Fprintln(buf, strings.Join(fields, ","))

	// Used to record the last values as we iterate
	last := make(map[string]float64, len(fields))

	// Write the values
	var times []int64
	for time, _ := range metrics {
		times = append(times, time)
	}
	sort.Sort(Int64Sort(times))

	for _, ts := range times {
		// Log the elapsed time in milliseconds
		fmt.Fprintf(buf, "%d,", ts/int64(time.Millisecond))

		// Go over the events for the given time, using the field
		// header mappings to ensure we correctly order the columns.
		events := metrics[ts]
		for _, field := range fields {
			if value, ok := events[field]; ok {
				last[field] = value
			}
			// Flush the float values to the buffer in order
			fmt.Fprintf(buf, "%f,", last[field])
		}
		buf.WriteString("\n")
	}

	// Write the result to the file
	fh, err := os.Create("result.csv")
	if err != nil {
		return fmt.Errorf("failed creating result file: %v", err)
	}
	defer fh.Close()
	if _, err := io.Copy(fh, buf); err != nil {
		log.Fatalf("failed writing result file: %v", err)
	}

	log.Printf("Results written to result.csv")
	return nil
}

type statusUpdate struct {
	key       string
	val       float64
	timestamp int64
}

func main() {
	// Check the args
	if len(os.Args) != 2 {
		log.Fatalln("usage: bench-runner <path>")
	}
	path := os.Args[1]

	// Make sure the script exists and is executable
	fi, err := os.Stat(path)
	if err != nil {
		log.Fatalf("failed to stat %q: %v", path, err)
	}
	if fi.Mode().Perm()|0111 == 0 {
		log.Fatalf("file %q is not executable", path)
	}

	// Perform setup
	setupCmd := exec.Command(path, "setup")
	if out, err := setupCmd.CombinedOutput(); err != nil {
		log.Fatalf("failed running setup: %v\nOutput: %s", err, string(out))
	}

	// Always run the teardown
	defer func() {
		teardownCmd := exec.Command(path, "teardown")
		if out, err := teardownCmd.CombinedOutput(); err != nil {
			log.Fatalf("failed teardown: %v\nOutput: %s", err, string(out))
		}
	}()

	// Create the status collector cmd
	statusCmd := exec.Command(path, "status")
	outBuf, err := statusCmd.StdoutPipe()
	if err != nil {
		log.Fatalf("failed attaching stdout: %v", err)
	}

	// Create the server and start running
	srv := newStatusServer(outBuf)
	if err := statusCmd.Start(); err != nil {
		log.Fatalf("failed to run status submitter: %v", err)
	}

	// Start listening for updates
	go srv.run()

	// Start running the benchmark
	runCmd := exec.Command(path, "run")
	if out, err := runCmd.CombinedOutput(); err != nil {
		log.Fatalf("failed running benchmark: %v\nOutput: %s", err, string(out))
	}

	// Wait for the status command to return
	if err := statusCmd.Wait(); err != nil {
		log.Fatalf("status command got error: %v", err)
	}
}
