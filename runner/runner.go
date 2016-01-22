package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

type statusServer struct {
	listener   net.Listener
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
		shutdownCh: make(chan struct{}),
	}
	return list.Addr().String(), server, nil
}

func (s *statusServer) shutdown() {
	close(s.shutdownCh)
}

func (s *statusServer) run(updateCh chan<- *statusUpdate) {
	// Wait for a connection
	conn, err := s.listener.Accept()
	if err != nil {
		log.Fatalf("failed to accept connection: %v", err)
	}

	for {
		// Read the next packet
		var payload []byte
		if _, err := conn.Read(payload); err != nil {
			log.Fatalf("failed reading payload: %v", err)
		}

		// Parse the payload parts
		parts := strings.Split(string(payload), ":")
		if len(parts) != 2 {
			log.Printf("invalid metric payload: %v", payload)
			continue
		}

		// Parse the metric
		val, err := strconv.ParseFloat(parts[1], 10)
		if err != nil {
			log.Printf("failed parsing metric value: %v", payload)
			continue
		}

		// Send the update
		updateCh <- &statusUpdate{
			key: parts[0],
			val: val,
		}
	}
}

type statusUpdate struct {
	key string
	val float64
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
		case update := <-updateCh:
			log.Printf("%#v", update)
		}
	}
}
