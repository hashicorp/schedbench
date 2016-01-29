build:
	GOOS=linux GOARCH=amd64 go build -o bin/bench-runner ./runner
	GOOS=linux GORACH=amd64 go build -o bin/test-nomad-docker ./tests/nomad-docker
	GOOS=linux GORACH=amd64 go build -o bin/test-nomad-exec ./tests/nomad-exec
	GOOS=linux GORACH=amd64 go build -o bin/test-nomad-raw-exec ./tests/nomad-raw-exec
