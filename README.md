Scheduler Benchmarks
====================

This is a benchmark framework used to generically test scheduler performance.
It was created to run Nomad's [Million Container Challenge](#).

There is a single "runner", which is the main program to run the benchmarks,
and multiple "tests", which are individual measurement implementations.

## Recorded Metrics

The measurements recorded are simple:

* **Number of placements made**<br>
  The number of placement decisions successfully completed by the scheduler.
  A placement is defined as an association between a task and an available
  node. A task is a single logical unit of work.
* **Number of tasks running**<br>
  This is the number of tasks which are actually running, as seen by the
  scheduler. A running task means that execution has begun (fork/exec has
  started, container or VM booting, etc.).

With the above measurements recorded as the numbers change, we can glean some
interesting performance data from a scheduler such as:

* Time to first placement
* Time to complete all placements
* Time to P95/P99 placements
* Time to first task running
* Time to all tasks running
* Time to P95/P99 tasks running

## Test Implementations

A major goal of this framework is to allow the suite to be run against
different schedulers. Because schedulers can vary greatly in core concepts as
well as client implementations, we provide a simple fork/exec interface which
can be implemented in various ways to allow the test to be run against any
scheduler.

A single implementation of a test will be invoked as an executable command.
It will be called numerous times with varying options to implement different
pieces of the benchmarking interface.

At a minimum, a test must implement the following subcommands:

```
setup
```

This is used to perform any pre-test setup. Any arbitrary code can execute
here, and the time elapsed in this step is not counted on the result. This is
intended to help with testing connections, setting up temporary files, or
anything else required prior to starting the test.

If a non-zero exit code is returned, the test is aborted.

```
run
```

This sub-command begins running the test. This is where the scheduler should
be instructed to begin work, i.e. submitting jobs to the system. This time is
reflected in the result.

This sub-command should exit when job submission is complete. If a non-zero
exit code is returned, the test is aborted.

```
status
```

This is used to monitor status of the scheduler. This sub-command's main
function should block until the test is complete.

Status is provided to the benchmark utility over STDOUT. The main status run
loop should print status information as soon as it is available in a simple
ASCII format. This format is `<metric>|<value>\n`. This data will be
automatically consumed by the benchmark utility and recorded in the results.
The metric name can be any string, and the value can be any float value.

The status command should exit when all work has completed. If a non-zero exit
code is returned, the benchmark is considered failed.

```
teardown
```

This sub-command is invoked to allow cleaning up/terminating any running
tasks, if required. This is intended to help prepare the system for future
tests to be run. It is called after the `status` sub-command completes.

## Metrics

Currently this benchmark framework only requires two metrics to be measured
by the `status` command (above) when running. These metric names are:

* `placed` - The number of completed, successful placements
* `running` - The number of tasks which are currently running

## Results

The results of the test are written to a file named `result.csv` in the current
working directory. The output is a simple 3-column format containing the
following fields:

* `elapsed_ms` - The total time elapsed in the test, in milliseconds
* `placed` - Number of placements which have been made
* `running` - Number of tasks in running state
