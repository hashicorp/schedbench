package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
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

		// Parse the payload parts
		parts := strings.Split(payload, "|")
		if len(parts) != 2 {
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
			key: parts[0],
			val: val,
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
	var placed, running float64

	// Record the start time
	start := time.Now()

	// Open the status file
	fh, err := os.Create("result.csv")
	if err != nil {
		log.Fatalf("failed creating result file: %v", err)
	}
	defer fh.Close()
	log.Printf("results will be streamed to result.csv")

	// Write the headers
	if _, err := fmt.Fprintln(fh, "elapsed_ms,placed,running"); err != nil {
		log.Fatalf("failed writing headers: %v", err)
	}
	s.updateCh <- &statusUpdate{key: "start"}

	for {
		select {
		case update := <-s.updateCh:
			now := time.Now()

			switch update.key {
			case "start":
				// Observe the start of the test
				now, placed, running = start, 0, 0
			case "placed":
				placed = update.val
			case "running":
				running = update.val
			}
			elapsed := now.Sub(start).Nanoseconds() / int64(time.Millisecond)
			fmt.Fprintf(fh, "%d,%f,%f\n", elapsed, placed, running)

		case <-doneCh:
			return
		}
	}
}

type statusUpdate struct {
	key  string
	val  float64
	time int64
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
