job "bench-exec-classlogger-1" {
    datacenters = ["us-central1"]

    group "classlogger_1" {
        count = 1

        constraint {
            attribute = "${node.class}"
            value     = "class_1"
        }

        restart {
            mode     = "fail"
            attempts = 0
        }

        task "classlogger_1" {
            driver = "exec"

            config {
                command = "classlogger"
            }

            resources {
                cpu    = 100
                memory = 15
                disk   = 10
            }

            logs {
                max_files     = 1
                max_file_size = 5
            }

            env {
                REDIS_ADDR = "redis.service.consul:6379"
                NODE_CLASS = "${node.class}"
            }
        }
    }

    group "classlogger_2" {
        count = 1

        constraint {
            attribute = "${node.class}"
            value     = "class_2"
        }

        restart {
            mode     = "fail"
            attempts = 0
        }

        task "classlogger_2" {
            driver = "exec"

            config {
                command = "classlogger"
            }

            resources {
                cpu    = 100
                memory = 15
                disk   = 10
            }

            logs {
                max_files     = 1
                max_file_size = 5
            }

            env {
                REDIS_ADDR = "redis.service.consul:6379"
                NODE_CLASS = "${node.class}"
            }
        }
    }

    group "classlogger_3" {
        count = 1

        constraint {
            attribute = "${node.class}"
            value     = "class_3"
        }

        restart {
            mode     = "fail"
            attempts = 0
        }

        task "classlogger_3" {
            driver = "exec"

            config {
                command = "classlogger"
            }

            resources {
                cpu    = 100
                memory = 15
                disk   = 10
            }

            logs {
                max_files     = 1
                max_file_size = 5
            }

            env {
                REDIS_ADDR = "redis.service.consul:6379"
                NODE_CLASS = "${node.class}"
            }
        }
    }

    group "classlogger_4" {
        count = 1

        constraint {
            attribute = "${node.class}"
            value     = "class_4"
        }

        restart {
            mode     = "fail"
            attempts = 0
        }

        task "classlogger_4" {
            driver = "exec"

            config {
                command = "classlogger"
            }

            resources {
                cpu    = 100
                memory = 15
                disk   = 10
            }

            logs {
                max_files     = 1
                max_file_size = 5
            }

            env {
                REDIS_ADDR = "redis.service.consul:6379"
                NODE_CLASS = "${node.class}"
            }
        }
    }

    group "classlogger_5" {
        count = 1

        constraint {
            attribute = "${node.class}"
            value     = "class_5"
        }

        restart {
            mode     = "fail"
            attempts = 0
        }

        task "classlogger_5" {
            driver = "exec"

            config {
                command = "classlogger"
            }

            resources {
                cpu    = 100
                memory = 15
                disk   = 10
            }

            logs {
                max_files     = 1
                max_file_size = 5
            }

            env {
                REDIS_ADDR = "redis.service.consul:6379"
                NODE_CLASS = "${node.class}"
            }
        }
    }
}
