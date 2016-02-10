job "bench-docker-classlogger-100k" {
    datacenters = ["us-central1", "us-east1", "europe-west1", "asia-east1"]

    group "classlogger_0" {
        count = 20000

        constraint {
            attribute = "$node.class"
            value     = "0"
        }

        restart {
            mode = "fail"
            attempts = 0
        }

        task "classlogger_0" {
            driver = "docker"

            config {
                image = "hashicorp/nomad-c1m:latest"
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
        count = 20000

        constraint {
            attribute = "$node.class"
            value     = "1"
        }

        restart {
            mode = "fail"
            attempts = 0
        }

        task "classlogger_1" {
            driver = "docker"

            config {
                image = "hashicorp/nomad-c1m:latest"
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
        count = 20000

        constraint {
            attribute = "$node.class"
            value     = "2"
        }

        restart {
            mode = "fail"
            attempts = 0
        }

        task "classlogger_2" {
            driver = "docker"

            config {
                image = "hashicorp/nomad-c1m:latest"
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
        count = 20000

        constraint {
            attribute = "$node.class"
            value     = "3"
        }

        restart {
            mode = "fail"
            attempts = 0
        }

        task "classlogger_3" {
            driver = "docker"

            config {
                image = "hashicorp/nomad-c1m:latest"
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
        count = 20000

        constraint {
            attribute = "$node.class"
            value     = "4"
        }

        restart {
            mode = "fail"
            attempts = 0
        }

        task "classlogger_4" {
            driver = "docker"

            config {
                image = "hashicorp/nomad-c1m:latest"
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
