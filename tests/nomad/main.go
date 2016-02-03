package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/nomad/jobspec"
	"github.com/hashicorp/nomad/nomad/structs"
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

	// Get the API client
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		log.Fatalf("failed creating nomad client: %v", err)
	}
	allocs := client.Allocations()

	// Wait loop for allocation statuses
	var lastRunning, lastTotal int64
	var index uint64 = 1
	for {
		// Set up the args
		args := &api.QueryOptions{
			AllowStale: true,
			WaitIndex:  index,
		}

		// Start the query
		resp, qm, err := allocs.List(args)
		if err != nil {
			// Only log and continue to skip minor errors
			log.Printf("failed querying allocations: %v", err)
			continue
		}

		// Check the index
		if qm.LastIndex == index {
			continue
		}
		index = qm.LastIndex

		// Check the response
		var allocsTotal, allocsRunning int64
		for _, alloc := range resp {
			if alloc.DesiredStatus == structs.AllocDesiredStatusRun {
				allocsTotal++
			}
			if alloc.ClientStatus == structs.AllocClientStatusRunning {
				allocsRunning++
			}
		}

		// Write the metrics, if there were changes.
		if allocsTotal != lastTotal {
			lastTotal = allocsTotal
			fmt.Fprintf(os.Stdout, "placed|%f\n", float64(allocsTotal))
		}
		if allocsRunning != lastRunning {
			lastRunning = allocsRunning
			fmt.Fprintf(os.Stdout, "running|%f\n", float64(allocsRunning))
		}

		// Break out if all of our allocs are running
		if allocsRunning == int64(totalAllocs) {
			break
		}
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
