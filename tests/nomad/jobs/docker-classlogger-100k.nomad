job "bench-docker-classlogger-100k" {
    datacenters = ["us-central1", "us-east1", "europe-west1", "asia-east1"]

    group "classlogger" {
        count = 100000

        restart {
            mode = "fail"
            attempts = 0
        }

        task "classlogger" {
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
