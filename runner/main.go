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
		log.Fatalf("[ERR] runner: failed to stat %q: %v", path, err)
	}
	if fi.Mode().Perm()|0111 == 0 {
		log.Fatalf("[ERR] runner: file %q is not executable", path)
	}

	// Perform setup
	log.Println("[DEBUG] runner: executing step 'setup'")
	setupCmd := exec.Command(path, "setup")
	setupCmd.Stderr = os.Stdout
	if out, err := setupCmd.Output(); err != nil {
		log.Fatalf("[ERR] runner: failed running setup: %v\nStdout: %s", err, string(out))
	}

	// Always run the teardown
	defer func() {
		log.Println("[DEBUG] runner: executing step 'teardown'")
		teardownCmd := exec.Command(path, "teardown")
		teardownCmd.Stderr = os.Stdout
		if out, err := teardownCmd.Output(); err != nil {
			log.Fatalf("[ERR] runner: failed teardown: %v\nStdout: %s", err, string(out))
		}
	}()

	// Start running the status collector
	log.Println("[DEBUG] runner: executing step 'status'")
	statusCmd := exec.Command(path, "status")
	statusCmd.Stderr = os.Stdout
	outBuf, err := statusCmd.StdoutPipe()
	if err != nil {
		log.Fatalf("[ERR] runner: failed attaching stdout: %v", err)
	}
	if err := statusCmd.Start(); err != nil {
		log.Fatalf("[ERR] runner: failed to run status submitter: %v", err)
	}

	// Start listening for updates
	srv := newStatusServer(outBuf)
	go srv.run()

	// Start running the benchmark
	log.Println("[DEBUG] runner: executing step 'run'")
	runCmd := exec.Command(path, "run")
	runCmd.Stderr = os.Stdout
	if out, err := runCmd.Output(); err != nil {
		log.Fatalf("[ERR] runner: failed running benchmark: %v\nStdout: %s", err, string(out))
	}

	// Wait for the status command to return
	log.Println("[DEBUG] runner: waiting for step 'status' to complete...")
	if err := statusCmd.Wait(); err != nil {
		log.Fatalf("[ERR] runner: status command got error: %v", err)
	}
}
