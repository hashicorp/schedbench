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
		log.Fatalf("[ERR] Failed to stat %q: %v", path, err)
	}
	if fi.Mode().Perm()|0111 == 0 {
		log.Fatalf("[ERR] File %q is not executable", path)
	}

	// Perform setup
	log.Println("[DEBUG] Executing step 'setup'")
	setupCmd := exec.Command(path, "setup")
	if out, err := setupCmd.CombinedOutput(); err != nil {
		log.Fatalf("[ERR] Failed running setup: %v\nOutput: %s", err, string(out))
	}

	// Always run the teardown
	defer func() {
		log.Println("[DEBUG] Executing step 'teardown'")
		teardownCmd := exec.Command(path, "teardown")
		if out, err := teardownCmd.CombinedOutput(); err != nil {
			log.Fatalf("[ERR] Failed teardown: %v\nOutput: %s", err, string(out))
		}
	}()

	// Start running the status collector
	log.Println("[DEBUG] Executing step 'status'")
	statusCmd := exec.Command(path, "status")
	statusCmd.Stderr = os.Stderr
	outBuf, err := statusCmd.StdoutPipe()
	if err != nil {
		log.Fatalf("[ERR] Failed attaching stdout: %v", err)
	}
	if err := statusCmd.Start(); err != nil {
		log.Fatalf("[ERR] Failed to run status submitter: %v", err)
	}

	// Start listening for updates
	srv := newStatusServer(outBuf)
	go srv.run()

	// Start running the benchmark
	log.Println("[DEBUG] Executing step 'run'")
	runCmd := exec.Command(path, "run")
	if out, err := runCmd.CombinedOutput(); err != nil {
		log.Fatalf("[ERR] Failed running benchmark: %v\nOutput: %s", err, string(out))
	}

	// Wait for the status command to return
	log.Println("[DEBUG] Waiting for step 'status' to complete...")
	if err := statusCmd.Wait(); err != nil {
		log.Fatalf("[ERR] Status command got error: %v", err)
	}
}
