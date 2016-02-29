job "bench-docker-classlogger" {
  datacenters = ["us-central1"]

  group "classlogger_1" {
    count = 20

    constraint {
      attribute = "${node.class}"
      value   = "class_1"
    }

    restart {
      mode   = "fail"
      attempts = 3
    }

    task "classlogger_1" {
      driver = "docker"

      config {
        image    = "hashicorp/nomad-c1m:0.1"
        network_mode = "host"
      }

      resources {
        cpu    = 20
        memory = 15
        disk   = 10
      }

      logs {
        max_files   = 1
        max_file_size = 5
      }

      env {
        REDIS_ADDR = "redis.service.consul:6379"
        NODE_CLASS = "${node.class}"
      }

      service {
        name = "${JOB}-${TASKGROUP}-classlogger"
      }
    }
  }

  group "classlogger_2" {
    count = 20

    constraint {
      attribute = "${node.class}"
      value   = "class_2"
    }

    restart {
      mode   = "fail"
      attempts = 3
    }

    task "classlogger_2" {
      driver = "docker"

      config {
        image    = "hashicorp/nomad-c1m:0.1"
        network_mode = "host"
      }

      resources {
        cpu    = 20
        memory = 15
        disk   = 10
      }

      logs {
        max_files   = 1
        max_file_size = 5
      }

      env {
        REDIS_ADDR = "redis.service.consul:6379"
        NODE_CLASS = "${node.class}"
      }

      service {
        name = "${JOB}-${TASKGROUP}-classlogger"
      }
    }
  }

  group "classlogger_3" {
    count = 20

    constraint {
      attribute = "${node.class}"
      value   = "class_3"
    }

    restart {
      mode   = "fail"
      attempts = 3
    }

    task "classlogger_3" {
      driver = "docker"

      config {
        image    = "hashicorp/nomad-c1m:0.1"
        network_mode = "host"
      }

      resources {
        cpu    = 20
        memory = 15
        disk   = 10
      }

      logs {
        max_files   = 1
        max_file_size = 5
      }

      env {
        REDIS_ADDR = "redis.service.consul:6379"
        NODE_CLASS = "${node.class}"
      }

      service {
        name = "${JOB}-${TASKGROUP}-classlogger"
      }
    }
  }

  group "classlogger_4" {
    count = 20

    constraint {
      attribute = "${node.class}"
      value   = "class_4"
    }

    restart {
      mode   = "fail"
      attempts = 3
    }

    task "classlogger_4" {
      driver = "docker"

      config {
        image    = "hashicorp/nomad-c1m:0.1"
        network_mode = "host"
      }

      resources {
        cpu    = 20
        memory = 15
        disk   = 10
      }

      logs {
        max_files   = 1
        max_file_size = 5
      }

      env {
        REDIS_ADDR = "redis.service.consul:6379"
        NODE_CLASS = "${node.class}"
      }

      service {
        name = "${JOB}-${TASKGROUP}-classlogger"
      }
    }
  }

  group "classlogger_5" {
    count = 20

    constraint {
      attribute = "${node.class}"
      value   = "class_5"
    }

    restart {
      mode   = "fail"
      attempts = 3
    }

    task "classlogger_5" {
      driver = "docker"

      config {
        image    = "hashicorp/nomad-c1m:0.1"
        network_mode = "host"
      }

      resources {
        cpu    = 20
        memory = 15
        disk   = 10
      }

      logs {
        max_files   = 1
        max_file_size = 5
      }

      env {
        REDIS_ADDR = "redis.service.consul:6379"
        NODE_CLASS = "${node.class}"
      }

      service {
        name = "${JOB}-${TASKGROUP}-classlogger"
      }
    }
  }
}
