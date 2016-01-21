package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/nomad/jobspec"
	"github.com/hashicorp/nomad/nomad/structs"
)

// Exec interface:
//   setup(numJobs, numContainers int) string
//   run(dir string, numJobs, numContainers int)
//   status(addr string)
//   teardown(dir string)

func main() {
	// Check the args
	if len(os.Args) < 2 {
		log.Fatalln("usage: nomad-bench <command> [args]")
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
	// Check the args
	if len(os.Args) != 4 {
		log.Fatalln("usage: nomad-bench setup <num jobs> <num containers>")
	}

	// Parse the inputs
	var err error
	var numContainers int
	if numContainers, err = strconv.Atoi(os.Args[3]); err != nil {
		log.Fatalln("number of containers must be numeric")
	}

	// Create the temp dir
	dir, err := ioutil.TempDir("", "nomad-bench")
	if err != nil {
		log.Fatalf("failed creating temp dir: %v", err)
	}

	// Write the job file to the temp dir
	fh, err := os.Create(filepath.Join(dir, "job.nomad"))
	if err != nil {
		log.Fatalf("failed creating job file: %v", err)
	}
	defer fh.Close()

	jobContent := fmt.Sprintf(jobTemplate, numContainers)
	if _, err := fh.WriteString(jobContent); err != nil {
		log.Fatalf("failed writing to job file: %v", err)
	}

	// Return the temp dir path
	fmt.Fprintln(os.Stdout, dir)
	return 0
}

func handleRun() int {
	// Check the args
	if len(os.Args) != 5 {
		log.Fatalln("usage: nomad-bench run <dir> <num jobs> <num containers>")
	}

	// Parse the inputs
	var err error
	var numJobs int
	if numJobs, err = strconv.Atoi(os.Args[3]); err != nil {
		log.Fatalln("number of jobs must be numeric")
	}

	// Get the job file path
	jobFile := filepath.Join(os.Args[2], "job.nomad")

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

	// Get the API client
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		log.Fatalf("failed creating nomad client: %v", err)
	}
	jobs := client.Jobs()

	// Submit the job the requested number of times
	for i := 0; i < numJobs; i++ {
		// Increment the job ID
		apiJob.ID = fmt.Sprintf("job-%d", i)
		if _, _, err := jobs.Register(apiJob, nil); err != nil {
			log.Fatalf("failed registering jobs: %v", err)
		}
	}

	return 0
}

func handleStatus() int {
	// Check the args
	if len(os.Args) != 3 {
		log.Fatalln("usage: nomad-bench status <addr>")
	}

	// Get the API client
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		log.Fatalf("failed creating nomad client: %v", err)
	}
	allocs := client.Allocations()

	// Connect to the status server
	conn, err := net.Dial("tcp", os.Args[2])
	if err != nil {
		log.Fatalf("failed contacting status server: %v", err)
	}
	defer conn.Close()

	// Wait loop for allocation statuses
	var last int64
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
		var allocsRunning int64
		for _, alloc := range resp {
			if alloc.ClientStatus == structs.AllocClientStatusRunning {
				allocsRunning++
			}
		}

		// Skip if there was no change
		if allocsRunning == last {
			continue
		}
		last = allocsRunning

		// Send the current count
		payload := strconv.FormatInt(allocsRunning, 10) + "\n"
		if _, err := conn.Write([]byte(payload)); err != nil {
			log.Printf("failed writing status update: %v", err)
			continue
		}
	}

	return 0
}

func handleTeardown() int {
	// Check the args
	if len(os.Args) != 3 {
		log.Fatalln("usage: nomad-bench teardown <dir>")
	}

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

	// Nuke the dir
	if err := os.RemoveAll(os.Args[2]); err != nil {
		log.Fatalf("failed cleaning up temp dir: %v", err)
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

const jobTemplate = `
job "bench" {
	datacenters = ["dc1"]

	group "cache" {
		count = %d

		restart {
			attempts = 0
		}

		task "bench" {
			driver = "docker"

			config {
				image = "redis:latest"
			}

			resources {
				cpu = 100
				memory = 100
			}
		}
	}
}
`
