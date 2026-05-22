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
	port         int
	role         string
	masterAddr   string
	dir          string
	dbfilename   string
	aofOverrides map[string]string
}

// FLAGS
const (
	PORT = "port"
	REPLICA = "replicaof"
	DBFILE = "dbfilename"
	DIR = "dir"

	AOF = "appendonly"
	AOF_DIR = "appenddirname"
	AOF_FILE = "appendfilename"
	AOF_FSYNC = "appendfsync"
)

func parseConfig(args []string) (serverConfig, error) {
	fs := flag.NewFlagSet("redis-server", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	port := fs.Int(PORT, 6379, "port to listen on")
	repl := fs.String(REPLICA, "", "master host and port")
	dir := fs.String(DIR, "", "directory for persistence")
	file := fs.String(DBFILE, "", "RDB filename")

	aonly  := fs.String(AOF, "", "enable AOF persistence")
	adir   := fs.String(AOF_DIR, "", "AOF subdirectory under dir")
	afile   := fs.String(AOF_FILE, "", "AOF filename")
	afsync := fs.String(AOF_FSYNC, "", "AOF fsync policy")

	if err := fs.Parse(args); err != nil {
		return serverConfig{}, err
	}

	cfg := serverConfig{
		port: *port,
		role: "master",
		dir: *dir,
		dbfilename: *file,
		aofOverrides: map[string]string{},
	}
	for key, val := range map[string]string{
		AOF: *aonly,
		AOF_DIR: *adir,
		AOF_FILE: *afile,
		AOF_FSYNC: *afsync,
	} {
		if val != "" {
			cfg.aofOverrides[key] = val
		}
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
		server.WithConfigOverrides(cfg.aofOverrides),
	)
	if err != nil {
		log.Fatalf("failed to bind to port %d: %v", cfg.port, err)
	}
	log.Printf("Redis server listening on :%d (role := %s)", cfg.port, cfg.role)
	redisServer.Run()
}
