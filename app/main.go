package main

import (
	"log"

	"github.com/codecrafters-io/redis-starter-go/internal/server"
)

func main() {
	redisServer, err := server.New("0.0.0.0:6379")
	if err != nil {
		log.Fatalf("failed to bind to port 6379: %v", err)
	}
	log.Println("Redis server listening on :6379")
	redisServer.Run()
}
