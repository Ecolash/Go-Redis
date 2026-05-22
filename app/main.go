package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/internal/server"
)

func main() {
	port := flag.Int("port", 6379, "port to listen on")
	repl := flag.String("replicaof", "", `make this server a replica of "<host> <port>"`)
	dir := flag.String("dir", "", "directory where RDB file is stored")
	dbfilename := flag.String("dbfilename", "", "name of the RDB file")
	flag.Parse()

	role := "master"
	masterAddr := ""
	if *repl != "" {
		role = "slave"
		host, mport, ok := strings.Cut(*repl, " ")
		if !ok {
			log.Fatalf("invalid --replicaof value %q: expected \"<host> <port>\"", *repl)
		}
		masterAddr = host + ":" + mport
	}
	addr := fmt.Sprintf("0.0.0.0:%d", *port)
	redisServer, err := server.New(addr, role, masterAddr, server.WithDir(*dir), server.WithDBFilename(*dbfilename))
	if err != nil {
		log.Fatalf("failed to bind to port %d: %v", *port, err)
	}
	log.Printf("Redis server listening on :%d (role := %s)", *port, role)
	redisServer.Run()
}
