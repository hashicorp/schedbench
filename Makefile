build:
	GOOS=linux GOARCH=amd64 go build -o bin/bench-runner ./runner
	GOOS=linux GOARCH=amd64 go build -o bin/nomad ./tests/nomad
