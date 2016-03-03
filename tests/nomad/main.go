package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/nomad/jobspec"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/mitchellh/copystructure"
)

const (
	// pollInterval is how often the status command will poll for results.
	pollInterval = 300 * time.Second

	maxWait = 10 * time.Minute

	// blockedEvalTries is how many times we will wait for a blocked eval to
	// complete before moving on.
	blockedEvalTries = 3

	// pendingAllocTries is how many times we will wait for a pending alloc to
	// complete before moving on.
	pendingAllocTries = 3
)

var numJobs, totalProcs int
var jobFile string

func main() {
	// Log everything to stderr so the runner can pipe it through
	log.SetOutput(os.Stderr)

	// Check the args
	if len(os.Args) != 2 {
		log.Fatalln(usage)
	}

	// Get the number of jobs to submit
	var err error
	v := os.Getenv("JOBS")
	if numJobs, err = strconv.Atoi(v); err != nil {
		log.Fatalln("[ERR] nomad: JOBS must be numeric")
	}

	// Get the location of the job file
	if jobFile = os.Getenv("JOBSPEC"); jobFile == "" {
		log.Fatalln("[ERR] nomad: JOBSPEC must be provided")
	}

	// Switch on the command
	switch os.Args[1] {
	case "setup":
		os.Exit(handleSetup())
	case "run":
		os.Exit(handleRun())
	case "status":
		os.Exit(handleStatus())
	case "teardown":
		os.Exit(handleTeardown())
	default:
		log.Fatalf("unknown command: %q", os.Args[1])
	}
}

func handleSetup() int {
	// No setup required
	return 0
}

func handleRun() int {
	// Parse the job file
	job, err := jobspec.ParseFile(jobFile)
	if err != nil {
		log.Fatalf("[ERR] nomad: failed parsing job file: %v", err)
	}

	// Convert to an API struct for submission
	apiJob, err := convertStructJob(job)
	if err != nil {
		log.Fatalf("[ERR] nomad: failed converting job: %v", err)
	}
	jobID := apiJob.ID

	// Get the API client
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		log.Fatalf("[ERR] nomad: failed creating nomad client: %v", err)
	}
	jobs := client.Jobs()

	jobSubmitters := 64
	if numJobs < jobSubmitters {
		jobSubmitters = numJobs
	}
	log.Printf("[DEBUG] nomad: using %d parallel job submitters", jobSubmitters)

	// Submit the job the requested number of times
	errCh := make(chan error, numJobs)
	stopCh := make(chan struct{})
	jobsCh := make(chan *api.Job, jobSubmitters)
	defer close(stopCh)
	for i := 0; i < jobSubmitters; i++ {
		go submitJobs(jobs, jobsCh, stopCh, errCh)
	}

	log.Printf("[DEBUG] nomad: submitting %d jobs", numJobs)
	submitting := make(map[string]*api.Job, numJobs)
	for i := 0; i < numJobs; i++ {
		copy, err := copystructure.Copy(apiJob)
		if err != nil {
			log.Fatalf("[ERR] nomad: failed to copy api job: %v", err)
		}

		// Increment the job ID
		jobCopy := copy.(*api.Job)
		jobCopy.ID = fmt.Sprintf("%s-%d", jobID, i)
		submitting[jobCopy.ID] = jobCopy
		jobsCh <- jobCopy
	}

	// Collect errors if any
	for i := 0; i < numJobs; i++ {
		select {
		case err := <-errCh:
			if err != nil {
				log.Fatalf("[ERR] nomad: failed submitting job: %v", err)
			}
		case <-stopCh:
			return 0
		}
	}

	// Get the jobs were submitted.
	submitted, _, err := jobs.List(nil)
	if err != nil {
		log.Fatalf("[ERR] nomad: failed listing jobs: %v", err)
	}

	// See if anything didn't get registered
	for _, job := range submitted {
		delete(submitting, job.ID)
	}

	// Resubmitting anything missed
	for id, missed := range submitting {
		log.Printf("[DEBUG] nomad: failed submitting job %q; retrying", id)
		_, _, err := jobs.Register(missed, nil)
		if err != nil {
			log.Printf("[ERR] nomad: failed submitting job: %v", err)
		}
	}

	return 0
}

func submitJobs(client *api.Jobs, jobs <-chan *api.Job, stopCh chan struct{}, errCh chan<- error) {
	for {
		select {
		case job := <-jobs:
			_, _, err := client.Register(job, nil)
			errCh <- err
		case <-stopCh:
			return
		}
	}
}

func handleStatus() int {
	// Parse the job file to get the total expected allocs
	job, err := jobspec.ParseFile(jobFile)
	if err != nil {
		log.Fatalf("[ERR] nomad: failed parsing job file: %v", err)
	}
	var totalAllocs int
	for _, group := range job.TaskGroups {
		totalAllocs += (group.Count * len(group.Tasks))
	}
	totalAllocs *= numJobs
	minEvals := numJobs
	log.Printf("[DEBUG] nomad: expecting %d allocs (%d evals minimum)", totalAllocs, minEvals)

	// Determine the set of jobs we should track.
	jobs := make(map[string]struct{})
	for i := 0; i < numJobs; i++ {
		// Increment the job ID
		jobs[fmt.Sprintf("%s-%d", job.ID, i)] = struct{}{}
	}

	// Get the API client
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		log.Fatalf("[ERR] nomad: failed creating nomad client: %v", err)
	}
	evalEndpoint := client.Evaluations()

	// Set up the args
	args := &api.QueryOptions{
		AllowStale: true,
	}

	// Wait for all the evals to be complete.
	cutoff := time.Now().Add(maxWait)
	evals := make(map[string]*api.Evaluation, minEvals)
	failedEvals := make(map[string]struct{})
	blockedEvals := make(map[string]int)
EVAL_POLL:
	for {
		waitTime, exceeded := getSleepTime(cutoff)
		if !exceeded {
			log.Printf("[DEBUG] nomad: next eval poll in %s", waitTime)
			time.Sleep(waitTime)
		}

		// Start the query
		resp, _, err := evalEndpoint.List(args)
		if err != nil {
			// Only log and continue to skip minor errors
			log.Printf("[ERR] nomad: failed querying evals: %v", err)
			continue
		}

		// Filter out evaluations that aren't for the jobs we are tracking.
		var filter []*api.Evaluation
		for _, eval := range resp {
			if _, ok := jobs[eval.JobID]; ok {
				filter = append(filter, eval)
			}
		}

		// Wait til all evals have gone through the scheduler.
		if n := len(filter); n < minEvals {
			log.Printf("[DEBUG] nomad: expect %d evals, have %d, polling again",
				minEvals, n)
			continue
		}

		// Ensure that all the evals are terminal, otherwise new allocations
		// could be made.
		needPoll := false
		for _, eval := range filter {
			switch eval.Status {
			case "failed":
				failedEvals[eval.ID] = struct{}{}
			case "complete":
				evals[eval.ID] = eval
			case "canceled":
				// Do nothing since it was a redundant eval.
			case "blocked":
				blockedEvals[eval.ID]++
				tries := blockedEvals[eval.ID]
				if tries < blockedEvalTries {
					needPoll = true
				} else if tries == blockedEvalTries {
					log.Printf("[DEBUG] nomad: abandoning blocked eval %q", eval.ID)
				}
			case "pending":
				needPoll = true
			}
		}

		if needPoll && !exceeded {
			continue EVAL_POLL
		}

		break
	}

	// We now have all the evals, gather the allocations and placement times.

	// scheduleTime is a map of alloc ID to map of desired status and time.
	scheduleTimes := make(map[string]map[string]int64, totalAllocs)
	startTimes := make(map[string]int64, totalAllocs)    // When a task was started
	receivedTimes := make(map[string]int64, totalAllocs) // When a task was received by the client
	failedAllocs := make(map[string]int64)               // Time an alloc failed
	failedReason := make(map[string]string)              // Reason an alloc failed
	pendingAllocs := make(map[string]int)                // Counts how many time the alloc was in pending state
	first := true
ALLOC_POLL:
	for {
		waitTime, exceeded := getSleepTime(cutoff)
		if !exceeded && !first {
			log.Printf("[DEBUG] nomad: next eval poll in %s", waitTime)
			time.Sleep(waitTime)
		}
		first = false

		needPoll := false
		for evalID := range evals {
			// Start the query
			resp, _, err := evalEndpoint.Allocations(evalID, args)
			if err != nil {
				// Only log and continue to skip minor errors
				log.Printf("[ERR] nomad: failed querying allocations: %v", err)
				continue
			}

			for _, alloc := range resp {
				// Capture the schedule time.
				allocTimes, ok := scheduleTimes[alloc.ID]
				if !ok {
					allocTimes = make(map[string]int64, 3)
					scheduleTimes[alloc.ID] = allocTimes
				}
				allocTimes[alloc.DesiredStatus] = alloc.CreateTime

				// Ensure that they have started or have failed.
				switch alloc.ClientStatus {
				case "failed":
					failedAllocs[alloc.ID] = alloc.CreateTime
					var failures []string
					for _, state := range alloc.TaskStates {
						if state.State == "failed" {
							failures = append(failures, state.Events[0].DriverError)
						}
					}
					failedReason[alloc.ID] = strings.Join(failures, ",")
					continue
				case "pending":
					pendingAllocs[alloc.ID]++
					tries := pendingAllocs[alloc.ID]
					if tries < pendingAllocTries {
						needPoll = true
					} else if tries == pendingAllocTries {
						log.Printf("[DEBUG] nomad: abandoning alloc %q", alloc.ID)
					}
					continue
				}

				// Detect the start time.
				for _, state := range alloc.TaskStates {
					if len(state.Events) == 0 {
						needPoll = true
					}

					for _, event := range state.Events {
						time := event.Time
						switch event.Type {
						case "Started":
							startTimes[alloc.ID] = time
						case "Received":
							receivedTimes[alloc.ID] = time
						}
					}
				}
			}
		}

		if needPoll && !exceeded {
			continue ALLOC_POLL
		}

		break
	}

	// Print the failure reasons for client allocs.
	for id, reason := range failedReason {
		log.Printf("[DEBUG] nomad: alloc id %q failed on client: %v", id, reason)
	}

	// Print the results.
	if l := len(failedEvals); l != 0 {
		fmt.Fprintf(os.Stdout, "failed_evals|%f\n", float64(l))
	}
	for time, count := range accumTimes(failedAllocs) {
		fmt.Fprintf(os.Stdout, "failed_allocs|%f|%d\n", float64(count), time)
	}
	for time, count := range accumTimes(startTimes) {
		fmt.Fprintf(os.Stdout, "running|%f|%d\n", float64(count), time)
	}
	for time, count := range accumTimes(receivedTimes) {
		fmt.Fprintf(os.Stdout, "received|%f|%d\n", float64(count), time)
	}
	for time, count := range accumTimesOn("run", scheduleTimes) {
		fmt.Fprintf(os.Stdout, "placed_run|%f|%d\n", float64(count), time)
	}
	for time, count := range accumTimesOn("failed", scheduleTimes) {
		fmt.Fprintf(os.Stdout, "placed_failed|%f|%d\n", float64(count), time)
	}
	for time, count := range accumTimesOn("stop", scheduleTimes) {
		fmt.Fprintf(os.Stdout, "placed_stop|%f|%d\n", float64(count), time)
	}

	// Aggregate eval triggerbys.
	triggers := make(map[string]int, len(evals))
	for _, eval := range evals {
		triggers[eval.TriggeredBy]++
	}
	for trigger, count := range triggers {
		fmt.Fprintf(os.Stdout, "trigger:%s|%f\n", trigger, float64(count))
	}

	// Print if the scheduler changed scheduling decisions
	flips := make(map[string]map[string]int64) // alloc id -> map[flipType]time
	flipTypes := make(map[string]struct{})
	for id, decisions := range scheduleTimes {
		if len(decisions) < 2 {
			continue
		}
		// Have decision -> time
		// 1) time -> decision
		// 2) sort times
		// 3) print transitions
		flips[id] = make(map[string]int64)
		inverted := make(map[int64]string, len(decisions))
		times := make([]int, 0, len(decisions))
		for k, v := range decisions {
			inverted[v] = k
			times = append(times, int(v))
		}
		sort.Ints(times)
		for i := 1; i < len(times); i++ {
			from := decisions[inverted[int64(times[i-1])]]
			to := decisions[inverted[int64(times[i])]]
			flipType := fmt.Sprintf("%s-to-%s", from, to)
			flips[id][flipType] = int64(times[i])
			flipTypes[flipType] = struct{}{}
		}
	}

	for flipType, _ := range flips {
		for time, count := range accumTimesOn(flipType, flips) {
			fmt.Fprintf(os.Stdout, "%v|%f|%d\n", flipType, float64(count), time)
		}
	}

	return 0
}

// getSleepTime takes a cutoff time and returns how long you should sleep
// between polls and whether you have exceeded the cutoff.
func getSleepTime(cutoff time.Time) (time.Duration, bool) {
	now := time.Now()
	if now.After(cutoff) {
		return time.Duration(0), true
	}

	desiredEnd := now.Add(pollInterval)
	if desiredEnd.After(cutoff) {
		return cutoff.Sub(now), false
	}

	return pollInterval, false
}

// accumTimes returns a mapping of time to cumulative counts. Takes a map
// of ID's to timestamps (ID is unimportant), and returns a mapping of
// timestamps to the cumulative count of events from that time.
// Ex: {foo: 10, bar: 10, baz: 20} -> {10: 2, 20: 3}
func accumTimes(in map[string]int64) map[int64]int64 {
	// Initialize the result.
	out := make(map[int64]int64)

	// Hot path if we have no times.
	if len(in) == 0 {
		return out
	}

	// Convert to intermediate format to handle counting multiple events
	// from the same timestamp.
	intermediate := make(map[int64]int64)
	for _, v := range in {
		intermediate[v] += 1
	}

	// Create a slice of times so we can sort it.
	var times []int64
	for time := range intermediate {
		times = append(times, time)
	}
	sort.Sort(Int64Sort(times))

	// Go over the times and populate the counts for each in the result.
	out[times[0]] = intermediate[times[0]]
	for i := 1; i < len(times); i++ {
		out[times[i]] = out[times[i-1]] + intermediate[times[i]]
	}

	return out
}

func accumTimesOn(innerKey string, in map[string]map[string]int64) map[int64]int64 {
	converted := make(map[string]int64)
	for outerKey, data := range in {
		for k, v := range data {
			if k == innerKey {
				converted[outerKey] = v
			}
		}
	}
	return accumTimes(converted)
}

func handleTeardown() int {
	// Get the API client
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		log.Fatalf("[ERR] nomad: failed creating nomad client: %v", err)
	}

	// Iterate all of the jobs and stop them
	log.Printf("[DEBUG] nomad: deregistering benchmark jobs")
	jobs, _, err := client.Jobs().List(nil)
	if err != nil {
		log.Fatalf("[ERR] nomad: failed listing jobs: %v", err)
	}
	for _, job := range jobs {
		if _, _, err := client.Jobs().Deregister(job.ID, nil); err != nil {
			log.Fatalf("[ERR] nomad: failed deregistering job: %v", err)
		}
	}
	return 0
}

func convertStructJob(in *structs.Job) (*api.Job, error) {
	gob.Register([]map[string]interface{}{})
	gob.Register([]interface{}{})
	var apiJob *api.Job
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(in); err != nil {
		return nil, err
	}
	if err := gob.NewDecoder(buf).Decode(&apiJob); err != nil {
		return nil, err
	}
	return apiJob, nil
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

const usage = `
NOTICE: This is a benchmark implementation binary and is not intended to be
run directly. The full path to this binary should be passed to bench-runner.

This benchmark measures the time taken to schedule and begin running tasks
using Nomad by HashiCorp.

To run this benchmark:

  1. Create a job file for Nomad to execute.
  2. Run the benchmark runner utility, passing this executable as the first
     argument. The following environment variables must be set:

       JOBSPEC - The path to a valid job definition file
       JOBS    - The number of times to submit the job

     An example use would look like this:

       $ JOBSPEC=./job1.nomad JOBS=100 bench-runner bench-nomad
`
