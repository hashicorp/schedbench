build:
	GOOS=linux GOARCH=amd64 go build -o bin/bench-runner ./runner
	GOOS=linux GOARCH=amd64 go build -o bin/test-nomad-docker ./tests/nomad-docker
	GOOS=linux GOARCH=amd64 go build -o bin/test-nomad-exec ./tests/nomad-exec
	GOOS=linux GOARCH=amd64 go build -o bin/test-nomad-raw-exec ./tests/nomad-raw-exec
	GOOS=linux GOARCH=amd64 go build -o bin/test-nomad-classlogger-exec ./tests/nomad-classlogger-exec
	GOOS=linux GOARCH=amd64 go build -o bin/classlogger ./tests/nomad-classlogger-exec/classlogger
