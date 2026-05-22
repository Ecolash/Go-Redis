package main

import (
	"fmt"
	"log"
	"os"

	"github.com/codecrafters-io/redis-starter-go/internal/server"
)

func main() {
	cfg, err := parseConfig(os.Args[1:])
	if err != nil {
		log.Fatalf("failed to parse flags: %v", err)
	}

	addr := fmt.Sprintf("0.0.0.0:%d", cfg.port)
	redisServer, err := server.New(
		addr,
		cfg.role,
		cfg.masterAddr,
		server.WithDir(cfg.dir),
		server.WithDBFilename(cfg.dbfilename),
		server.WithConfigOverrides(cfg.aofOverrides),
	)
	if err != nil {
		log.Fatalf("failed to bind to port %d: %v", cfg.port, err)
	}
	log.Printf("Redis server listening on :%d (role := %s)", cfg.port, cfg.role)
	redisServer.Run()
}
