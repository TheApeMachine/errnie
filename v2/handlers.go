package errnie

import (
	"os"
)

type OpCode uint

const (
	NOOP OpCode = iota
	KILL
	RECV
	RETR
	RTRN
)

var opcodes = map[OpCode]func(){
	NOOP: noop, KILL: kill, RECV: noop, RETR: noop, RTRN: noop,
}

func noop() {}
func kill() { os.Exit(1) }
