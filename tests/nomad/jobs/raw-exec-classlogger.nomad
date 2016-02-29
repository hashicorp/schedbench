job "bench-raw-exec-classlogger" {
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
      interval = "5m"
      delay = "2s"
    }

    task "classlogger_1" {
      driver = "raw_exec"

      config {
        command = "classlogger"
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
      interval = "5m"
      delay = "2s"
    }

    task "classlogger_2" {
      driver = "raw_exec"

      config {
        command = "classlogger"
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
      interval = "5m"
      delay = "2s"
    }

    task "classlogger_3" {
      driver = "raw_exec"

      config {
        command = "classlogger"
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
      interval = "5m"
      delay = "2s"
    }

    task "classlogger_4" {
      driver = "raw_exec"

      config {
        command = "classlogger"
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
      interval = "5m"
      delay = "2s"
    }

    task "classlogger_5" {
      driver = "raw_exec"

      config {
        command = "classlogger"
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
    }
  }
}
