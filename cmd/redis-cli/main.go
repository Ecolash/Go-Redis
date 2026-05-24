package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/internal/cli/banner"
	"github.com/codecrafters-io/redis-starter-go/internal/cli/client"
	"github.com/codecrafters-io/redis-starter-go/internal/cli/demo"
	"github.com/codecrafters-io/redis-starter-go/internal/cli/repl"
	"github.com/codecrafters-io/redis-starter-go/internal/cli/state"
	"github.com/spf13/cobra"
)

func main() {
	var (
		host     string
		port     int
		password string
		demoMode bool
	)

	root := &cobra.Command{
		Use:   "redis-cli",
		Short: "A rich interactive Redis client CLI",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.New(host, port, password)
			if err != nil {
				return fmt.Errorf("✗ %w", err)
			}

			// Fetch server version for banner (best-effort)
			versionResp, _ := c.Do("INFO", "server")
			version := parseVersion(versionResp)
			latMs := c.Latency().Milliseconds()

			banner.Print(host, port, latMs, version)

			sess := state.New(host, port)
			sess.UpdateLatency(c.Latency())

			if demoMode {
				demo.Run(c, sess)
				return nil
			}
			repl.Run(c, sess)
			return nil
		},
	}

	root.Flags().StringVarP(&host, "host", "H", "127.0.0.1", "Redis server host")
	root.Flags().IntVarP(&port, "port", "p", 6379, "Redis server port")
	root.Flags().StringVarP(&password, "password", "a", "", "Redis password (AUTH)")
	root.Flags().BoolVar(&demoMode, "demo", false, "Run scripted demo and exit")

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func parseVersion(v *client.RESPValue) string {
	if v == nil || v.IsNull || v.Type != '$' {
		return "unknown"
	}
	for _, line := range strings.Split(v.Value, "\r\n") {
		if strings.HasPrefix(line, "redis_version:") {
			return strings.TrimPrefix(line, "redis_version:")
		}
	}
	return "unknown"
}
