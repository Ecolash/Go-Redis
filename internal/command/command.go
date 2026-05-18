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
)
