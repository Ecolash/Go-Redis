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
)
