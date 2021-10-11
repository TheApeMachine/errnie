package errnie

import (
	"os"
)

/*
OpCode is a type that can be used to identify an intent when using
errnie as the handler of one or multiple errors. To use it you need
to use the `errnie.Handles()` and wrap it around the error. To use
errnie's built-in handlers you can use it `in the middle` like so:

```go
errnie.Handles(errnie.KILL(err))
```

in which case you have chosen to exit the program if the error was
not nil.
*/
type OpCode uint

const (
	// NOOP is the empty value for this enumerable type and does nothing.
	NOOP OpCode = iota
	// KILL wraps the error and if triggered will exit the program.
	KILL
	// RECV attempts to recover the program from any kind of error.
	RECV
	// RETR will attempt a retry of the previous operation.
	RETR
	// RTRN will return out of the current method.
	RTRN
)

/*
opcodes is a lookup map used to retrieve a handler function definition
from an OpCode which can then be called at any time that makes sense.
*/
var opcodes = map[OpCode]func() *AmbientContext{
	NOOP: ambctx.noop,
	KILL: ambctx.kill,
	RECV: ambctx.recv,
	RETR: ambctx.noop,
	RTRN: ambctx.noop,
}

/*
With is a method to chain onto a previous method (most likely `Handles`) and designed to
specify the type of action to take when an error has been detected.
*/
func (ambctx *AmbientContext) With(opcode OpCode) *AmbientContext {
	if ambctx.ERR != nil || !ambctx.OK {
		return opcodes[opcode]()
	}

	return ambctx
}

func (ambctx *AmbientContext) kill() *AmbientContext {
	if ambctx.ERR != nil {
		os.Exit(1)
	}

	return ambctx
}

func (ambctx *AmbientContext) recv() *AmbientContext {
	ctxguard.Rescue()()
	return ambctx
}

func (ambctx *AmbientContext) noop() *AmbientContext {
	return ambctx
}
