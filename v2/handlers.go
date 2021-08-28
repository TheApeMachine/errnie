package errnie

import (
	"os"

	"github.com/google/uuid"
)

type HandlerType uint

const (
	NOOP HandlerType = iota
	EXIT
	CTX
)

func Kill(handlerType HandlerType) func(interface{}) {
	switch handlerType {
	case EXIT:
		return exit
	case CTX:
		return ctx
	}

	return noop
}

func noop(null interface{}) {}
func exit(code interface{}) { os.Exit(code.(int)) }
func ctx(id interface{})    { ambient.Cancel(id.(uuid.UUID)) }
