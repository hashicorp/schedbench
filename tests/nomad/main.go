package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/nomad/jobspec"
	"github.com/hashicorp/nomad/nomad/structs"
)

const (
	interval = 5 * time.Minute
)

var numJobs, totalProcs int
var jobFile string

func main() {
	// Check the args
	if len(os.Args) != 2 {
		log.Fatalln(usage)
	}

	// Get the number of jobs to submit
	var err error
	v := os.Getenv("JOBS")
	if numJobs, err = strconv.Atoi(v); err != nil {
		log.Fatalln("JOBS must be numeric")
	}

	// Get the location of the job file
	if jobFile = os.Getenv("JOBSPEC"); jobFile == "" {
		log.Fatalln("JOBSPEC must be provided")
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
		log.Fatalf("failed parsing job file: %v", err)
	}

	// Convert to an API struct for submission
	apiJob, err := convertStructJob(job)
	if err != nil {
		log.Fatalf("failed converting job: %v", err)
	}
	jobID := apiJob.ID

	// Get the API client
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		log.Fatalf("failed creating nomad client: %v", err)
	}
	jobs := client.Jobs()

	// Submit the job the requested number of times
	for i := 0; i < numJobs; i++ {
		// Increment the job ID
		apiJob.ID = fmt.Sprintf("%s-%d", jobID, i)
		if _, _, err := jobs.Register(apiJob, nil); err != nil {
			log.Fatalf("failed registering jobs: %v", err)
		}
	}

	return 0
}

func handleStatus() int {
	// Parse the job file to get the total expected allocs
	job, err := jobspec.ParseFile(jobFile)
	if err != nil {
		log.Fatalf("failed parsing job file: %v", err)
	}
	var totalAllocs int
	for _, group := range job.TaskGroups {
		totalAllocs += (group.Count * len(group.Tasks))
	}
	totalAllocs *= numJobs
	minEvals := numJobs

	// Get the API client
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		log.Fatalf("failed creating nomad client: %v", err)
	}
	evalEndpoint := client.Evaluations()

	// Set up the args
	args := &api.QueryOptions{
		AllowStale: true,
		WaitIndex:  1,
	}

	// Wait for all the allocations to be complete.
	evals := make(map[string]*api.Evaluation, minEvals)
	failedEvals := make(map[string]struct{})
EVAL_POLL:
	for {
		time.Sleep(interval)

		// Start the query
		resp, qm, err := evalEndpoint.List(args)
		if err != nil {
			// Only log and continue to skip minor errors
			log.Printf("failed querying evals: %v", err)
			continue
		}

		// Check the index
		if qm.LastIndex == args.WaitIndex {
			continue
		}
		args.WaitIndex = qm.LastIndex

		// Wait til all evals have gone through the scheduler.
		if len(resp) < minEvals {
			continue
		}

		// Ensure that all the evals are terminal, otherwise new allocations
		// could be made.
		for _, eval := range resp {
			switch eval.Status {
			case "failed":
				failedEvals[eval.ID] = struct{}{}
				continue EVAL_POLL
			case "complete":
				evals[eval.ID] = eval
			case "canceled":
				// Do nothing since it was a redundant eval.
			default:
				continue EVAL_POLL
			}
		}

		break
	}

	// We now have all the evals, gather the allocations times.
	startTimes := make(map[string]int64, totalAllocs)
	failedAllocs := make(map[string]struct{})
ALLOC_POLL:
	for {
		time.Sleep(interval)

		for evalID := range evals {
			// Start the query
			resp, _, err := evalEndpoint.Allocations(evalID, args)
			if err != nil {
				// Only log and continue to skip minor errors
				log.Printf("failed querying allocations: %v", err)
				continue
			}

			for _, alloc := range resp {
				switch alloc.ClientStatus {
				case "failed":
					failedAllocs[alloc.ID] = struct{}{}
					continue
				case "pending":
					continue ALLOC_POLL
				}

				// Detect the start time.
				for task, state := range alloc.TaskStates {
					if state.State == "pending" || len(state.Events) == 0 {
						continue ALLOC_POLL
					}

					// Get the first event.
					startEvent := state.Events[0]
					time := startEvent.Time
					startTimes[fmt.Sprintf("%v-%v", alloc.ID, task)] = time
				}
			}
		}

		break
	}

	// Print the results.
	if l := len(failedEvals); l != 0 {
		fmt.Fprintf(os.Stdout, "failed_evals|%f\n", float64(l))
	}
	if l := len(failedAllocs); l != 0 {
		fmt.Fprintf(os.Stdout, "failed_allocs|%f\n", float64(l))
	}
	for _, time := range startTimes {
		fmt.Fprintf(os.Stdout, "running|%f|%d\n", float64(1), time)
	}

	return 0
}

func handleTeardown() int {
	// Get the API client
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		log.Fatalf("failed creating nomad client: %v", err)
	}

	// Iterate all of the jobs and stop them
	jobs, _, err := client.Jobs().List(nil)
	if err != nil {
		log.Fatalf("failed listing jobs: %v", err)
	}
	for _, job := range jobs {
		if _, _, err := client.Jobs().Deregister(job.ID, nil); err != nil {
			log.Fatalf("failed deregistering job: %v", err)
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
