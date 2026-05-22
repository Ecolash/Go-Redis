package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
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
	Port    = "port"
	Replica = "replicaof"
	Dbfile  = "dbfilename"
	Dir     = "dir"

	Aof      = "appendonly"
	AofDir   = "appenddirname"
	AofFile  = "appendfilename"
	AofFsync = "appendfsync"
)

func parseConfig(args []string) (serverConfig, error) {
	fs := flag.NewFlagSet("redis-server", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	port := fs.Int(Port, 6379, "port to listen on")
	repl := fs.String(Replica, "", "master host and port")
	dir := fs.String(Dir, "", "directory for persistence")
	file := fs.String(Dbfile, "", "RDB filename")

	aonly := fs.String(Aof, "", "enable AOF persistence")
	adir := fs.String(AofDir, "", "AOF subdirectory under dir")
	afile := fs.String(AofFile, "", "AOF filename")
	afsync := fs.String(AofFsync, "", "AOF fsync policy")

	if err := fs.Parse(args); err != nil {
		return serverConfig{}, err
	}

	cfg := serverConfig{
		port:         *port,
		role:         "master",
		dir:          *dir,
		dbfilename:   *file,
		aofOverrides: map[string]string{},
	}
	for key, val := range map[string]string{
		Aof:      *aonly,
		AofDir:   *adir,
		AofFile:  *afile,
		AofFsync: *afsync,
	} {
		if val != "" {
			cfg.aofOverrides[key] = val
		}
	}

	if *repl != "" {
		host, port, ok := strings.Cut(*repl, " ")
		if !ok {
			return serverConfig{},
				fmt.Errorf("invalid --replicaof value %q: expected \"<host> <port>\"", *repl)
		}
		cfg.role = "slave"
		cfg.masterAddr = host + ":" + port
	}
	return cfg, nil
}
