package command

type Command string

const (
	PING  Command = "PING"
	ECHO  Command = "ECHO"
	SET   Command = "SET"
	GET   Command = "GET"
	INCR  Command = "INCR"
	DECR  Command = "DECR"

	TYPE Command = "TYPE"
	INFO Command = "INFO"

	// Transaction commands
	MULTI   Command = "MULTI"
	EXEC    Command = "EXEC"
	DISCARD Command = "DISCARD"
	WATCH   Command = "WATCH"
	UNWATCH Command = "UNWATCH"

	// STREAM Commands
	XADD   Command = "XADD"
	XRANGE Command = "XRANGE"
	XREAD  Command = "XREAD"

	// LIST Commands
	BLPOP  Command = "BLPOP"
	LPUSH  Command = "LPUSH"
	RPUSH  Command = "RPUSH"
	LPOP   Command = "LPOP"
	RPOP   Command = "RPOP"
	LLEN   Command = "LLEN"
	LRANGE Command = "LRANGE"

	// Replication Commands
	REPLCONF Command = "REPLCONF"
	PSYNC    Command = "PSYNC"
	WAIT     Command = "WAIT"

	// RDB Commands
	CONFIG Command = "CONFIG"
	KEYS   Command = "KEYS"

	// Pub/Sub Commands
	SUBSCRIBE    Command = "SUBSCRIBE"
	PSUBSCRIBE   Command = "PSUBSCRIBE"
	UNSUBSCRIBE  Command = "UNSUBSCRIBE"
	PUNSUBSCRIBE Command = "PUNSUBSCRIBE"
	PUBLISH      Command = "PUBLISH"

	// Sorted Set Commands
	ZADD   Command = "ZADD"
	ZRANGE Command = "ZRANGE"
	ZRANK  Command = "ZRANK"
	ZSCORE Command = "ZSCORE"
	ZCARD  Command = "ZCARD"
	ZREM   Command = "ZREM"

	// Geo Commands
	GEOADD    Command = "GEOADD"
	GEOPOS    Command = "GEOPOS"
	GEODIST   Command = "GEODIST"
	GEOSEARCH Command = "GEOSEARCH"
)
