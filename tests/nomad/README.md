Nomad Benchmark
===============

This is the Nomad-specific benchmark implementation. It is laid out like so:

* `jobs/` - This directory holds the job files for running various Nomad
  scenarios through the benchmark utility.

* `Dockerfile` - Build instructions for creating the container image used
  to benchmark Docker-specific tasks in Nomad.

* `classlogger/` - A simple CLI utility which logs the Nomad node class to a
  Redis server.

## Building

To build the specific components, use the Makefile:

* `make` - Builds the Nomad benchmark implementation
* `make classlogger` - Builds only the classlogger utility
* `make docker` - Builds the classlogger, and then the Docker container which
  wraps it.
