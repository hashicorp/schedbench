package main

import (
	"log"
	"os"
	"os/exec"
)

func main() {
	// Check the args
	if len(os.Args) != 2 {
		log.Fatalln("usage: bench-runner <path>")
	}
	path := os.Args[1]

	// Make sure the script exists and is executable
	fi, err := os.Stat(path)
	if err != nil {
		log.Fatalf("failed to stat %q: %v", path, err)
	}
	if fi.Mode().Perm()|0111 == 0 {
		log.Fatalf("file %q is not executable", path)
	}

	// Perform setup
	setupCmd := exec.Command(path, "setup")
	if out, err := setupCmd.CombinedOutput(); err != nil {
		log.Fatalf("failed running setup: %v\nOutput: %s", err, string(out))
	}

	// Always run the teardown
	defer func() {
		teardownCmd := exec.Command(path, "teardown")
		if out, err := teardownCmd.CombinedOutput(); err != nil {
			log.Fatalf("failed teardown: %v\nOutput: %s", err, string(out))
		}
	}()

	// Create the status collector cmd
	statusCmd := exec.Command(path, "status")
	outBuf, err := statusCmd.StdoutPipe()
	if err != nil {
		log.Fatalf("failed attaching stdout: %v", err)
	}

	// Create the server and start running
	srv := newStatusServer(outBuf)
	if err := statusCmd.Start(); err != nil {
		log.Fatalf("failed to run status submitter: %v", err)
	}

	// Start listening for updates
	go srv.run()

	// Start running the benchmark
	runCmd := exec.Command(path, "run")
	if out, err := runCmd.CombinedOutput(); err != nil {
		log.Fatalf("failed running benchmark: %v\nOutput: %s", err, string(out))
	}

	// Wait for the status command to return
	if err := statusCmd.Wait(); err != nil {
		log.Fatalf("status command got error: %v", err)
	}
}
