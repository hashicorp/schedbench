job "bench-exec-classlogger" {
    datacenters = ["dc1"]

    group "cache" {
        count = 5

        restart {
            mode = "fail"
            attempts = 0
        }

        task "classlogger" {
            driver = "exec"

            config {
                command = "classlogger"
            }

            env {
                REDIS_ADDR = "127.0.0.1:6379"
                NODE_CLASS = "$node.class"
            }

            resources {
                cpu = 100
                memory = 100
            }
        }
    }
}
