job "bench-exec-redis" {
    datacenters = ["dc1"]

    group "cache" {
        count = 5

        restart {
            mode = "fail"
            attempts = 0
        }

        task "redis" {
            driver = "exec"

            config {
                command = "redis-server"
                args = ["--port", "$NOMAD_PORT_redis"]
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
