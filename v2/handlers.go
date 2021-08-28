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

func Kill(handlerType HandlerType) (func(interface{}), interface{}) {
	switch handlerType {
	case EXIT:
		return exit, 1
	}

	return noop, nil
}

func Cancel(handlerType HandlerType, id uuid.UUID) (func(interface{}), interface{}) {
	switch handlerType {
	case CTX:
		return ctx, id
	}

	return noop, nil
}

func noop(null interface{}) {}
func exit(code interface{}) { os.Exit(code.(int)) }
func ctx(id interface{})    { ambient.Cancel(id.(uuid.UUID)) }
