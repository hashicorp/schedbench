package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
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
	return list.Addr().String(), server, nil
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
		log.Fatalln("port number required")
	}
	port, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Fatalln("failed parsing port number: %v", err)
	}

	// Create the server
	addr, srv, err := newStatusServer(int(port))
	if err != nil {
		log.Fatalf("failed starting server: %v", err)
	}
	log.Printf("status server started on %q", addr)

	// Start running
	updateCh := make(chan *statusUpdate, 128)
	go srv.run(updateCh)

	// Wait for updates
	for {
		select {
		case update := <-srv.updateCh:
			log.Printf("%#v", update)
		}
	}
}
