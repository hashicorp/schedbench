package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// statusServer is responsible for consuming status information which is
// output by a test implementation.
type statusServer struct {
	// The output stream. This is attached to the output from the test
	// implementation and is used to scan line-by-line over it.
	outStream *bufio.Scanner

	// Simple metrics about the status collector. Used to print some basic
	// debugging information to stdout.
	lastUpdate        time.Time
	totalUpdates      int
	updateMetricsLock sync.Mutex

	// The updateCh is used to pass status data from the scanner to the
	// result collector.
	updateCh chan *statusUpdate
}

// newStatusServer makes a new statusServer and initializes the fields.
func newStatusServer(outStream io.Reader) *statusServer {
	return &statusServer{
		outStream: bufio.NewScanner(outStream),
		updateCh:  make(chan *statusUpdate, 512),
	}
}

// run is the main loop of the status server which is responsible for
// scanning lines of output from a test, parsing it into a status update,
// and sending it down to the update handler.
func (s *statusServer) run() {
	// Start the update parser
	doneCh := make(chan struct{})
	defer close(doneCh)
	go s.handleUpdates(doneCh)
	go s.logUpdateTimes(doneCh)

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
			ts = time.Now().UnixNano()
		case 3:
			// Timestamp present, parse and use
			ts, err = strconv.ParseInt(parts[2], 10, 64)
			if err != nil {
				log.Printf("[ERR] runner: failed parsing metric timestamp in %q: %v", payload, err)
				continue
			}
		default:
			log.Printf("[ERR] runner: invalid metric payload: %q", payload)
			continue
		}

		// Parse the metric
		val, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			log.Printf("[ERR] runner: failed parsing metric value in %q: %v", payload, err)
			continue
		}

		// Send the update
		update := &statusUpdate{
			key:       parts[0],
			val:       val,
			timestamp: ts,
		}
		s.updateCh <- update
	}

	// Check if we broke out due to an error
	if err := s.outStream.Err(); err != nil {
		log.Fatalf("[ERR] runner: failed reading payload: %v", err)
	}
}

// handleUpdates is used to read updates off of the updateCh and populate them
// into a time-indexed map. Blocks until the doneCh is closed.
func (s *statusServer) handleUpdates(doneCh <-chan struct{}) {
	// Used to store events and times
	metrics := make(map[int64]map[string]float64)

	// Record the start time and start the clock with 0 running
	start := time.Now().UnixNano()
	metrics[0] = map[string]float64{
		"running": 0,
	}

	for {
		select {
		case update := <-s.updateCh:
			// Compute elapsed time and log the value away. We reduce to
			// milliseconds here so that our map keys are guaranteed unique
			// per millisecond. We otherwise could have a rounding problem.
			elapsed := (update.timestamp - start) / int64(time.Millisecond)
			if _, ok := metrics[elapsed]; !ok {
				metrics[elapsed] = make(map[string]float64)
			}
			metrics[elapsed][update.key] = update.val

			// Refresh the last update time
			s.updateMetricsLock.Lock()
			s.lastUpdate = time.Now()
			s.totalUpdates++
			s.updateMetricsLock.Unlock()

		case <-doneCh:
			// Format and write the metrics to the result file.
			if err := writeResult(metrics); err != nil {
				log.Fatalf("[ERR] runner: failed writing result: %v", err)
			}
			return
		}
	}
}

// logUpdateTimes periodically logs the last time we saw an update from the
// status collector. This is helpful when debugging so that we know if the
// sub-command has halted for some reason and is no longer fetching status.
func (s *statusServer) logUpdateTimes(doneCh chan struct{}) {
	for {
		select {
		case <-time.After(10 * time.Second):
			s.updateMetricsLock.Lock()
			last := s.lastUpdate
			total := s.totalUpdates
			s.updateMetricsLock.Unlock()
			if total == 0 {
				log.Printf("[DEBUG] runner: no status updates yet...")
			} else {
				log.Printf("[DEBUG] runner: last status update %s ago (%d total)",
					time.Now().Sub(last), total)
			}

		case <-doneCh:
			return
		}
	}
}

// writeResult takes a time-indexed map of metrics and formats them into
// a CSV format. The data is then flushed to a result.csv file in the
// current directory.
func writeResult(metrics map[int64]map[string]float64) error {
	// Create the output buffer and CSV writer.
	buf := new(bytes.Buffer)
	csvWriter := csv.NewWriter(buf)

	// Get the unique names of the data fields. These will be the names
	// of the columns in the CSV output.
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

	// Write the field names as a header row.
	csvWriter.Write(append([]string{"elapsed_ms"}, fields...))

	// Used to record the last values as we iterate, making it possible to fill
	// in all known data fields at each known timestamp.
	last := make(map[string]float64, len(fields))

	// Sort events by timestamp
	var times []int64
	for time, _ := range metrics {
		times = append(times, time)
	}
	sort.Sort(Int64Sort(times))

	for _, ts := range times {
		records := make([]string, len(fields)+1)

		// Log the elapsed time
		records[0] = strconv.FormatInt(ts, 10)

		// Go over the events for the given time, using the field
		// header mappings to ensure we correctly order the columns.
		events := metrics[ts]
		for i, field := range fields {
			if value, ok := events[field]; ok {
				last[field] = value
			}
			records[i+1] = strconv.FormatFloat(last[field], 'f', -1, 64)
		}

		// Flush the line to the CSV encoder
		csvWriter.Write(records)
	}

	// Flush the lines to the buffer and check for write errors
	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		return fmt.Errorf("failed writing CSV data: %v", err)
	}

	// Create the output file
	fh, err := os.Create("result.csv")
	if err != nil {
		return fmt.Errorf("failed creating result file: %v", err)
	}
	defer fh.Close()

	// Copy the buffer onto the file handle
	if _, err := io.Copy(fh, buf); err != nil {
		return fmt.Errorf("failed writing result file: %v", err)
	}

	log.Printf("[INFO] runner: results written to result.csv")
	return nil
}

// statusUpdate is used to hold a 3-tuple of key/value/timestamp. This is
// used to ship a single measurement between the status reader and result
// writer.
type statusUpdate struct {
	key       string  // The name of the metric.
	val       float64 // The value of the measurement.
	timestamp int64   // The (optional) timestamp.
}

// Int64Sort is used to sort slices of int64 numbers
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
