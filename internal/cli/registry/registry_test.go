package registry_test

import (
	"testing"

	"github.com/codecrafters-io/redis-starter-go/internal/cli/registry"
)

var requiredCommands = []string{
	"PING", "ECHO", "TYPE", "INFO",
	"SET", "GET", "INCR", "DECR",
	"MULTI", "EXEC", "DISCARD", "WATCH", "UNWATCH",
	"XADD", "XRANGE", "XREAD",
	"BLPOP", "LPUSH", "RPUSH", "LPOP", "RPOP", "LLEN", "LRANGE",
	"REPLCONF", "PSYNC", "WAIT",
	"CONFIG", "KEYS",
	"SUBSCRIBE", "PSUBSCRIBE", "UNSUBSCRIBE", "PUNSUBSCRIBE", "PUBLISH",
	"ZADD", "ZRANGE", "ZRANK", "ZSCORE", "ZCARD", "ZREM",
	"GEOADD", "GEOPOS", "GEODIST", "GEOSEARCH",
	"ACL", "AUTH",
}

func TestAllCommandsRegistered(t *testing.T) {
	reg := registry.All()
	index := make(map[string]bool, len(reg))
	for _, def := range reg {
		index[def.Name] = true
		for _, alias := range def.Aliases {
			index[alias] = true
		}
	}
	for _, cmd := range requiredCommands {
		if !index[cmd] {
			t.Errorf("command %q not found in registry", cmd)
		}
	}
}

func TestCommandHasRequiredFields(t *testing.T) {
	for _, def := range registry.All() {
		if def.Name == "" {
			t.Error("CommandDef with empty Name")
		}
		if def.Category == "" {
			t.Errorf("%s: missing Category", def.Name)
		}
		if def.Synopsis == "" {
			t.Errorf("%s: missing Synopsis", def.Name)
		}
		if def.Example == "" {
			t.Errorf("%s: missing Example", def.Name)
		}
	}
}
