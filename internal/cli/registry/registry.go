package registry

import "strings"

// ArgDef describes one argument of a command.
type ArgDef struct {
	Name     string
	Required bool
	Hint     string // e.g. "key", "integer", "seconds"
}

// CommandDef is the metadata for one Redis command.
type CommandDef struct {
	Name     string
	Category string
	Aliases  []string
	Synopsis string
	Example  string
	Args     []ArgDef
}

var all = []CommandDef{
	// Basic
	{Name: "PING", Category: "basic", Synopsis: "PING [message]", Example: "PING", Args: []ArgDef{{Name: "message", Hint: "string"}}},
	{Name: "ECHO", Category: "basic", Synopsis: "ECHO message", Example: "ECHO hello", Args: []ArgDef{{Name: "message", Required: true, Hint: "string"}}},
	{Name: "TYPE", Category: "basic", Synopsis: "TYPE key", Example: "TYPE mykey", Args: []ArgDef{{Name: "key", Required: true, Hint: "key"}}},
	{Name: "INFO", Category: "basic", Synopsis: "INFO [section]", Example: "INFO server", Args: []ArgDef{{Name: "section", Hint: "string"}}},
	{Name: "QUIT", Category: "basic", Aliases: []string{"EXIT", "BYE"}, Synopsis: "QUIT", Example: "QUIT"},

	// String
	{Name: "SET", Category: "string", Synopsis: "SET key value [EX seconds] [PX ms] [NX|XX]", Example: "SET user:1 alice EX 3600", Args: []ArgDef{{Name: "key", Required: true, Hint: "key"}, {Name: "value", Required: true, Hint: "string"}}},
	{Name: "GET", Category: "string", Synopsis: "GET key", Example: "GET user:1", Args: []ArgDef{{Name: "key", Required: true, Hint: "key"}}},
	{Name: "INCR", Category: "string", Synopsis: "INCR key", Example: "INCR counter", Args: []ArgDef{{Name: "key", Required: true, Hint: "key"}}},
	{Name: "DECR", Category: "string", Synopsis: "DECR key", Example: "DECR counter", Args: []ArgDef{{Name: "key", Required: true, Hint: "key"}}},

	// Transaction
	{Name: "MULTI", Category: "tx", Synopsis: "MULTI", Example: "MULTI"},
	{Name: "EXEC", Category: "tx", Synopsis: "EXEC", Example: "EXEC"},
	{Name: "DISCARD", Category: "tx", Synopsis: "DISCARD", Example: "DISCARD"},
	{Name: "WATCH", Category: "tx", Synopsis: "WATCH key [key ...]", Example: "WATCH mykey", Args: []ArgDef{{Name: "key", Required: true, Hint: "key"}}},
	{Name: "UNWATCH", Category: "tx", Synopsis: "UNWATCH", Example: "UNWATCH"},

	// Stream
	{Name: "XADD", Category: "stream", Synopsis: "XADD key id field value [field value ...]", Example: "XADD mystream * name alice age 30"},
	{Name: "XRANGE", Category: "stream", Synopsis: "XRANGE key start end [COUNT count]", Example: "XRANGE mystream - +"},
	{Name: "XREAD", Category: "stream", Synopsis: "XREAD [COUNT count] [BLOCK ms] STREAMS key [key ...] id [id ...]", Example: "XREAD COUNT 10 STREAMS mystream 0"},

	// List
	{Name: "LPUSH", Category: "list", Synopsis: "LPUSH key value [value ...]", Example: "LPUSH mylist a b c"},
	{Name: "RPUSH", Category: "list", Synopsis: "RPUSH key value [value ...]", Example: "RPUSH mylist a b c"},
	{Name: "LPOP", Category: "list", Synopsis: "LPOP key [count]", Example: "LPOP mylist"},
	{Name: "RPOP", Category: "list", Synopsis: "RPOP key [count]", Example: "RPOP mylist"},
	{Name: "LLEN", Category: "list", Synopsis: "LLEN key", Example: "LLEN mylist"},
	{Name: "LRANGE", Category: "list", Synopsis: "LRANGE key start stop", Example: "LRANGE mylist 0 -1"},
	{Name: "BLPOP", Category: "list", Synopsis: "BLPOP key [key ...] timeout", Example: "BLPOP mylist 0"},

	// Replication
	{Name: "REPLCONF", Category: "replication", Synopsis: "REPLCONF <option> <value>", Example: "REPLCONF listening-port 6380"},
	{Name: "PSYNC", Category: "replication", Synopsis: "PSYNC replicationid offset", Example: "PSYNC ? -1"},
	{Name: "WAIT", Category: "replication", Synopsis: "WAIT numreplicas timeout", Example: "WAIT 1 0"},

	// Keys / Config
	{Name: "KEYS", Category: "server", Synopsis: "KEYS pattern", Example: "KEYS user:*"},
	{Name: "CONFIG", Category: "server", Synopsis: "CONFIG GET|SET|REWRITE parameter [value]", Example: "CONFIG GET maxmemory"},

	// Pub/Sub
	{Name: "SUBSCRIBE", Category: "pubsub", Synopsis: "SUBSCRIBE channel [channel ...]", Example: "SUBSCRIBE news alerts"},
	{Name: "PSUBSCRIBE", Category: "pubsub", Synopsis: "PSUBSCRIBE pattern [pattern ...]", Example: "PSUBSCRIBE news.*"},
	{Name: "UNSUBSCRIBE", Category: "pubsub", Synopsis: "UNSUBSCRIBE [channel ...]", Example: "UNSUBSCRIBE news"},
	{Name: "PUNSUBSCRIBE", Category: "pubsub", Synopsis: "PUNSUBSCRIBE [pattern ...]", Example: "PUNSUBSCRIBE news.*"},
	{Name: "PUBLISH", Category: "pubsub", Synopsis: "PUBLISH channel message", Example: "PUBLISH news 'hello world'"},

	// Sorted Set
	{Name: "ZADD", Category: "sorted-set", Synopsis: "ZADD key [NX|XX] [GT|LT] [CH] score member [score member ...]", Example: "ZADD leaderboard 100 alice 200 bob"},
	{Name: "ZRANGE", Category: "sorted-set", Synopsis: "ZRANGE key min max [BYSCORE|BYLEX] [REV] [LIMIT offset count] [WITHSCORES]", Example: "ZRANGE leaderboard 0 -1 WITHSCORES"},
	{Name: "ZRANK", Category: "sorted-set", Synopsis: "ZRANK key member [WITHSCORE]", Example: "ZRANK leaderboard alice"},
	{Name: "ZSCORE", Category: "sorted-set", Synopsis: "ZSCORE key member", Example: "ZSCORE leaderboard alice"},
	{Name: "ZCARD", Category: "sorted-set", Synopsis: "ZCARD key", Example: "ZCARD leaderboard"},
	{Name: "ZREM", Category: "sorted-set", Synopsis: "ZREM key member [member ...]", Example: "ZREM leaderboard alice"},

	// Geo
	{Name: "GEOADD", Category: "geo", Synopsis: "GEOADD key [NX|XX] [CH] longitude latitude member [...]", Example: "GEOADD mygeo 13.361389 38.115556 Palermo"},
	{Name: "GEOPOS", Category: "geo", Synopsis: "GEOPOS key member [member ...]", Example: "GEOPOS mygeo Palermo"},
	{Name: "GEODIST", Category: "geo", Synopsis: "GEODIST key member1 member2 [m|km|ft|mi]", Example: "GEODIST mygeo Palermo Catania km"},
	{Name: "GEOSEARCH", Category: "geo", Synopsis: "GEOSEARCH key FROMMEMBER member BYRADIUS radius m|km|ft|mi ASC|DESC", Example: "GEOSEARCH mygeo FROMMEMBER Palermo BYRADIUS 200 km ASC"},

	// ACL / Auth
	{Name: "AUTH", Category: "server", Synopsis: "AUTH [username] password", Example: "AUTH mypassword"},
	{Name: "ACL", Category: "server", Synopsis: "ACL WHOAMI|LIST|SETUSER|DELUSER|GETUSER|CAT|LOG|RESET|SAVE|LOAD", Example: "ACL WHOAMI"},

	// Meta
	{Name: "HELP", Category: "meta", Synopsis: "HELP [command]", Example: "HELP SET"},
}

// All returns every registered CommandDef.
func All() []CommandDef { return all }

// Find returns the CommandDef for name (case-insensitive), or nil if not found.
func Find(name string) *CommandDef {
	upper := strings.ToUpper(name)
	for i := range all {
		if all[i].Name == upper {
			return &all[i]
		}
		for _, alias := range all[i].Aliases {
			if alias == upper {
				return &all[i]
			}
		}
	}
	return nil
}
