// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

// This program is intended to be run as a scheduled task. The NODE_CLASS
// and REDIS_ADDR must both be provided. The program then simply
// increments a counter in Redis for the given class type. This is used
// to validate the node types which were scheduled to during a test.

import (
	"log"
	"os"

	"github.com/garyburd/redigo/redis"
)

func main() {
	class := os.Getenv("NODE_CLASS")
	if class == "" {
		log.Fatalln("NODE_CLASS must be non-empty")
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		log.Fatalln("REDIS_ADDR must be non-empty")
	}

	conn, err := redis.Dial("tcp", redisAddr)
	if err != nil {
		log.Fatalf("error connecting to redis: %v", err)
	}

	_, err = conn.Do("INCR", class)
	conn.Close()
	if err != nil {
		log.Fatalf("error incrementing class count: %v", err)
	}

	// Wait forever
	wait := make(chan struct{})
	<-wait
}
