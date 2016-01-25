package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type statusServer struct {
	listener   net.Listener
	updateCh   chan *statusUpdate
	shutdownCh chan struct{}
}

func newStatusServer(port int) (string, *statusServer, error) {
	// Try to set up the listener
	list, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return "", nil, fmt.Errorf("failed to listen: %v", err)
	}

	// Create and return the server
	server := &statusServer{
		listener:   list,
		updateCh:   make(chan *statusUpdate, 128),
		shutdownCh: make(chan struct{}),
	}
	//return list.Addr().String(), server, nil
	return "127.0.0.1:9000", server, nil
}

func (s *statusServer) shutdown() {
	close(s.shutdownCh)
}

func (s *statusServer) run(updateCh chan<- *statusUpdate) {
	for {
		// Wait for a connection
		conn, err := s.listener.Accept()
		if err != nil {
			log.Fatalf("failed to accept connection: %v", err)
		}
		go s.handleConn(conn)
	}
}

func (s *statusServer) handleConn(conn net.Conn) {
	reader := bufio.NewReader(conn)
	for {
		// Read the next payload
		payload, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("failed reading payload: %v", err)
			return
		}

		// Strip the newline
		payload = payload[:len(payload)-1]

		// Parse the payload parts
		parts := strings.Split(payload, "|")
		if len(parts) != 3 {
			log.Printf("invalid metric payload: %v", payload)
			continue
		}

		// Parse the timestamp
		ts, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			log.Printf("failed parsing timestamp in %q: %v", payload, err)
			continue
		}

		// Parse the metric
		val, err := strconv.ParseFloat(parts[2], 64)
		if err != nil {
			log.Printf("failed parsing metric value in %q: %v", payload, err)
			continue
		}

		// Send the update
		update := &statusUpdate{
			key:  parts[1],
			val:  val,
			time: ts,
		}
		select {
		case s.updateCh <- update:
		default:
			log.Printf("update channel full! dropping update: %v", update)
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
		log.Fatalln("usage: c1m <path>")
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

	// Perform setup
	cmd := exec.Command(file, "setup")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		log.Fatalf("failed running setup: %v", err)
	}

	// Create the server
	addr, srv, err := newStatusServer(9000)
	if err != nil {
		log.Fatalf("failed starting server: %v", err)
	}
	log.Printf("status server started on %q", addr)

	// Call the status submitter
	cmd = exec.Command(file, "status", addr)
	cmd.Dir = dir
	if err := cmd.Start(); err != nil {
		log.Fatalf("failed to run status submitter: %v", err)
	}

	// Start listening for updates
	updateCh := make(chan *statusUpdate, 128)
	go srv.run(updateCh)

	// Start the update handler
	go handleUpdates(updateCh)

	// Mark start of the test
	updateCh <- &statusUpdate{
		key:  "started",
		time: time.Now().UnixNano(),
	}

	// Start running the benchmark
	cmd = exec.Command(file, "run")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		log.Fatalf("failed running benchmark: %v", err)
	}

	for {
	}
}

func handleUpdates(updateCh <-chan *statusUpdate) {
	var placed, booting, running float64
	var start int64
	for {
		select {
		case update := <-updateCh:
			switch update.key {
			case "started":
				start = update.time
			case "placed":
				placed = update.val
			case "booting":
				booting = update.val
			case "running":
				running = update.val
			}
			elapsed := update.time - start
			fmt.Printf("%d,%f,%f,%f\n", elapsed, placed, booting, running)
		}
	}
}
