package command

type Command string

const (
	PING  Command = "PING"
	ECHO  Command = "ECHO"
	SET   Command = "SET"
	GET   Command = "GET"
	RPUSH Command = "RPUSH"
	LRANGE Command = "LRANGE"
)
