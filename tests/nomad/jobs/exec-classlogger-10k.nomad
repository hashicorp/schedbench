job "bench-exec-classlogger-10k" {
    datacenters = ["us-central1", "us-east1", "europe-west1", "asia-east1"]

    group "classlogger_0" {
        count = 2000

        constraint {
            attribute = "$node.class"
            value     = "0"
        }

        restart {
            mode = "fail"
            attempts = 0
        }

        task "classlogger_0" {
            driver = "exec"

            config {
                command = "classlogger"
            }

            resources {
                cpu = 50
                memory = 50
            }

            env {
                REDIS_ADDR = "redis.service.consul:6379"
                NODE_CLASS = "$node.class"
            }
        }
    }

    group "classlogger_1" {
        count = 2000

        constraint {
            attribute = "$node.class"
            value     = "1"
        }

        restart {
            mode = "fail"
            attempts = 0
        }

        task "classlogger_1" {
            driver = "exec"

            config {
                command = "classlogger"
            }

            resources {
                cpu = 50
                memory = 50
            }

            env {
                REDIS_ADDR = "redis.service.consul:6379"
                NODE_CLASS = "$node.class"
            }
        }
    }

    group "classlogger_2" {
        count = 2000

        constraint {
            attribute = "$node.class"
            value     = "2"
        }

        restart {
            mode = "fail"
            attempts = 0
        }

        task "classlogger_2" {
            driver = "exec"

            config {
                command = "classlogger"
            }

            resources {
                cpu = 50
                memory = 50
            }

            env {
                REDIS_ADDR = "redis.service.consul:6379"
                NODE_CLASS = "$node.class"
            }
        }
    }

    group "classlogger_3" {
        count = 2000

        constraint {
            attribute = "$node.class"
            value     = "3"
        }

        restart {
            mode = "fail"
            attempts = 0
        }

        task "classlogger_3" {
            driver = "exec"

            config {
                command = "classlogger"
            }

            resources {
                cpu = 50
                memory = 50
            }

            env {
                REDIS_ADDR = "redis.service.consul:6379"
                NODE_CLASS = "$node.class"
            }
        }
    }

    group "classlogger_4" {
        count = 2000

        constraint {
            attribute = "$node.class"
            value     = "4"
        }

        restart {
            mode = "fail"
            attempts = 0
        }

        task "classlogger_4" {
            driver = "exec"

            config {
                command = "classlogger"
            }

            resources {
                cpu = 50
                memory = 50
            }

            env {
                REDIS_ADDR = "redis.service.consul:6379"
                NODE_CLASS = "$node.class"
            }
        }
    }
}
