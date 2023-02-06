# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

job "bench-raw-exec-redis" {
  datacenters = ["dc1"]
  type = "batch"
  group "cache" {
    count = 5

    restart {
      mode = "fail"
      attempts = 0
    }

    task "redis" {
      driver = "raw_exec"

      config {
        command = "redis-server"
        args = ["--port", "$NOMAD_PORT_redis"]
      }

      resources {
        cpu = 100
        memory = 100
        network {
          port "redis" {}
          mbits = 1
        }
      }
    }
  }
}
