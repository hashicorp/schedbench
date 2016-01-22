build:
	GOOS=linux GOARCH=amd64 go build -o bench-runner ./runner
	GOOS=linux GORACH=amd64 go build -o nomad-bench
