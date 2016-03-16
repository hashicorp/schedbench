Scheduler Benchmarks
====================

This is a benchmark framework used to generically test scheduler performance.
It was created to run Nomad's [Million Container Challenge](https://hashicorp.com/c1m.html).

There is a single "runner", which is the main program to run the benchmarks,
and multiple "tests", which are individual measurement implementations.

## Metrics

The primary metric gathered by this framework is the number of "tasks" which
are in a "running" state. A running task means that execution has begun
(fork/exec has started, container or VM booting, etc.). With just this
measurement captured over time, we can glean some interesting performance data
from a scheduler, such as:

* Time to first task running
* Time to all tasks running
* Time to P95/P99 tasks running

It is also possible to capture any arbitrary performance data available. This
may vary between schedulers, and there may not always be a 1:1 comparison, but
it is useful to highlight the performance of specific features.

## Test Implementations

This framework is intended to allow multiple schedulers to implement the test.
Because schedulers can vary greatly in core concepts, features, and client
implementations, the framework provides a simple fork/exec interface which can
be implemented in various ways.

A single implementation of a test will be invoked as an executable command.
It will be called numerous times with varying options to implement different
pieces of the benchmarking interface.

To log debug messages from within a test implementation, simply write messages
to stderr. They will be piped through and displayed on the terminal to allow
for useful troubleshooting information.

At a minimum, a test must implement the following sub-commands:

---

### `setup`

This is used to perform any pre-test setup. Any arbitrary code can execute
here, and the time elapsed in this step is not counted on the result. This is
intended to help with testing connections, setting up temporary files, or
anything else required prior to starting the test.

If a non-zero exit code is returned, the test is aborted.

---

### `run`

This sub-command begins running the test. This is where the scheduler should
be instructed to begin work, i.e. submitting jobs to the system. This time is
reflected in the result.

This sub-command should exit when job submission is complete. If a non-zero
exit code is returned, the test is aborted.

---

### `status`

This is used to monitor status of the scheduler. This sub-command's main
function should block until the test is complete.

Status is provided to the benchmark utility over STDOUT. The main status run
loop should print status information as soon as it is available in a simple
ASCII format. This format is `<metric>|<value>\n`. This data will be
automatically consumed by the benchmark utility and recorded in the results.
The metric name can be any string, and the value can be any float value.
An optional timestamp may be provided using a third field. This timestamp is
given using the current Unix time in nanosecond precision. If the timestamp is
not given, the current timestamp will be used at the time the metric is
recorded, resulting in a less accurate value. The payload will look like
`<metric>|<value>|<timestamp>\n`, if a timestamp is provided.

Although metric names are arbitrary, there are reserved metric names which have
special meanings to this framework. At a minimum, the following metric names
are expected to be emitted by each test implementation:

* `running` - The number of tasks which are in the running state.

The status command should exit when all work has completed. If a non-zero exit
code is returned, the benchmark is considered failed.

---

### `teardown`

This sub-command is invoked to allow cleaning up/terminating any running
tasks, if required. This is intended to help prepare the system for future
tests to be run. It is called after the `status` sub-command completes.

---

## Results

The results of the test are written to a file named `result.csv` in the current
working directory.
