package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type statusServer struct {
	outStream  *bufio.Reader
	updateCh   chan *statusUpdate
	shutdownCh chan struct{}
}

func newStatusServer(outStream io.Reader) *statusServer {
	return &statusServer{
		outStream:  bufio.NewReader(outStream),
		updateCh:   make(chan *statusUpdate, 128),
		shutdownCh: make(chan struct{}),
	}
}

func (s *statusServer) shutdown() {
	close(s.shutdownCh)
}

func (s *statusServer) run() {
	// Mark start of the test
	s.updateCh <- &statusUpdate{
		key: "started",
	}

	// Start the update parser
	doneCh := make(chan struct{})
	defer close(doneCh)
	go s.handleUpdates(doneCh)

	for {
		// Read the next payload
		payload, err := s.outStream.ReadString('\n')
		if err != nil {
			log.Printf("failed reading payload: %v", err)
			return
		}

		// Strip the newline
		payload = payload[:len(payload)-1]

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
			key: parts[1],
			val: val,
		}
		select {
		case s.updateCh <- update:
		default:
			log.Printf("update channel full! dropping update: %v", update)
		}
	}
}

func (s *statusServer) handleUpdates(doneCh <-chan struct{}) {
	var placed, booting, running float64
	var start time.Time

	// Open the status file
	fh, err := os.Create("result.csv")
	if err != nil {
		log.Fatalf("failed creating result file: %v", err)
	}
	defer fh.Close()

	for {
		select {
		case update := <-s.updateCh:
			now := time.Now()

			switch update.key {
			case "started":
				if start.IsZero() {
					start = now
				}
			case "placed":
				placed = update.val
			case "booting":
				booting = update.val
			case "running":
				running = update.val
			}
			elapsed := now.Sub(start).Nanoseconds()
			fmt.Fprintf(fh, "%d,%f,%f,%f\n", elapsed, placed, booting, running)

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
	file := os.Args[1]

	// Make sure the script exists and is executable
	fi, err := os.Stat(file)
	if err != nil {
		log.Fatalf("failed to stat %q: %v", file, err)
	}
	if fi.Mode().Perm()|0111 == 0 {
		log.Fatalf("file %q is not executable", file)
	}

	// Create the temp dir
	dir, err := ioutil.TempDir("", "bench")
	if err != nil {
		log.Fatalf("failed creating temp dir: %v", err)
	}
	defer os.RemoveAll(dir)
	log.Printf("using temp dir: %s", dir)

	// Perform setup
	cmd := exec.Command(file, "setup")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Fatalf("failed running setup: %v\nOutput: %s", err, string(out))
	}

	// Create the server
	outBuf := new(bytes.Buffer)
	srv := newStatusServer(outBuf)

	// Call the status submitter
	cmd = exec.Command(file, "status")
	cmd.Dir = dir
	cmd.Stdout = outBuf
	if err := cmd.Start(); err != nil {
		log.Fatalf("failed to run status submitter: %v", err)
	}

	// Start listening for updates
	go srv.run()

	// Start running the benchmark
	cmd = exec.Command(file, "run")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Fatalf("failed running benchmark: %v\nOutput: %s", err, string(out))
	}

	// TODO: this just sleeps forever...
	for {
	}
}
