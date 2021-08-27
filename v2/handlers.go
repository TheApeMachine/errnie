package errnie

import "os"

type HandlerType uint

const (
	NOOP HandlerType = iota
	EXIT
)

func Kill(handlerType HandlerType) func() {
	switch handlerType {
	case EXIT:
		return exit
	}

	return noop
}

func noop() {}
func exit() { os.Exit(1) }
