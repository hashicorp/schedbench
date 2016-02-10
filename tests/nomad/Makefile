default:
	GOOS=linux GOARCH=amd64 go build -o ../bin/bench-nomad

classlogger:
	GOOS=linux GOARCH=amd64 go build -o classlogger/classlogger ./classlogger

docker: classlogger
	docker build -t hashicorp/nomad-c1m:latest .

.PHONY: default classlogger docker
