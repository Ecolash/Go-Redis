package command

type Command string

const (
	PING  Command = "PING"
	ECHO  Command = "ECHO"
	SET   Command = "SET"
	GET   Command = "GET"
	LPUSH Command = "LPUSH"
	RPUSH Command = "RPUSH"
	LPOP   Command = "LPOP"
	RPOP   Command = "RPOP"
	LLEN   Command = "LLEN"
	LRANGE Command = "LRANGE"
)
