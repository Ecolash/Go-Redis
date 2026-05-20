package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/codecrafters-io/redis-starter-go/internal/server"
)

func main() {
	port := flag.Int("port", 6379, "port to listen on")
	repl := flag.String("replicaof", "", `make this server a replica of "<host> <port>"`)
	flag.Parse()

	role := "master"
	if *repl != "" {
		role = "slave"
	}		
	addr := fmt.Sprintf("0.0.0.0:%d", *port)
	redisServer, err := server.New(addr, role)
	if err != nil {
		log.Fatalf("failed to bind to port %d: %v", *port, err)
	}
	log.Printf("Redis server listening on :%d (role := %s)", *port, role)
	redisServer.Run()
}
