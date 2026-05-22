package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/internal/server"
)

type serverConfig struct {
	port       int
	role       string
	masterAddr string
	dir        string
	dbfilename  string
}

func parseConfig(args []string) (serverConfig, error) {
	fs := flag.NewFlagSet("redis-server", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	port := fs.Int("port", 6379, "port to listen on")
	repl := fs.String("replicaof", "", "master host and port")
	dir  := fs.String("dir", "", "directory for persistence")
	file  := fs.String("dbfilename", "", "RDB filename")

	if err := fs.Parse(args); err != nil {
		return serverConfig{}, err
	}

	cfg := serverConfig{
		port: *port,
		role: "master",
		dir: *dir,
		dbfilename: *file,
	}

	if *repl != "" {
		host, mport, ok := strings.Cut(*repl, " ")
		if !ok {
			return serverConfig{}, 
			fmt.Errorf("invalid --replicaof value %q: expected \"<host> <port>\"", *repl)
		}
		cfg.role = "slave"
		cfg.masterAddr = host + ":" + mport
	}
	return cfg, nil
}


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
	)
	if err != nil {
		log.Fatalf("failed to bind to port %d: %v", cfg.port, err)
	}
	log.Printf("Redis server listening on :%d (role := %s)", cfg.port, cfg.role)
	redisServer.Run()
}
