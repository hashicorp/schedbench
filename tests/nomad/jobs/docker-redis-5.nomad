job "bench-docker" {
    datacenters = ["dc1"]

    group "cache" {
        count = 5

        restart {
            mode = "fail"
            attempts = 0
        }

        task "redis" {
            driver = "docker"

            config {
                image = "redis:latest"
                port_map {
                    redis = 6379
                }
            }

            resources {
                cpu = 100
                memory = 100
                network {
                    port "redis" {}
                }
            }
        }
    }
}
