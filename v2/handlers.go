package errnie

import (
	"os"
)

type OpCode uint

const (
	NOP OpCode = iota
	KIL
	REC
	RET
	CTX
)

var opcodes = map[OpCode]func(){
	NOP: nop, KIL: kil, REC: nop, RET: nop, CTX: ctx,
}

func nop() {}
func kil() { os.Exit(1) }
func ctx() {}
