package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
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
	case "teardown":
		os.Exit(handleTeardown())
	default:
		log.Fatalln("unknown command: %q", os.Args[1])
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
	datacenters = ["us-central1-a"]

	group "cache" {
		count = %d

		task "redis" {
			driver = "docker"

			config {
				image = "redis"
			}

			resources {
				cpu = 100
				memory = 100
			}
		}
	}
}
`
