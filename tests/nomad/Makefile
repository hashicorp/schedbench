default:
	go get -d -v -u -f ./...
	GOOS=linux GOARCH=amd64 go build -o ../../bin/bench-nomad

classlogger:
	cd classlogger; go get -d -v -u -f ./...
	GOOS=linux GOARCH=amd64 go build -o classlogger/classlogger ./classlogger

docker: classlogger
	docker build -t hashicorp/nomad-c1m:0.1 .

.PHONY: default classlogger docker
