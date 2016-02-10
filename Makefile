default:
	GOOS=linux GOARCH=amd64 go build -o bin/bench-runner ./runner

nomad:
	cd tests/nomad; $(MAKE)

.PHONY: default nomad
