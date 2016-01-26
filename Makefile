build:
	GOOS=linux GOARCH=amd64 go build -o bench-runner ./runner
	GOOS=linux GORACH=amd64 go build -o tests/nomad-docker ./bench/nomad-docker
	GOOS=linux GORACH=amd64 go build -o tests/nomad-exec ./bench/nomad-exec
