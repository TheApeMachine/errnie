package errnie

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/viper"
)

type ContextType uint

const (
	TODO ContextType = iota
	BACKGROUND
	CANCEL
	TIMEOUT
	DEADLINE
)

var ambient AmbientContext

func init() {
	ambient = NewContext(AmbientContext{}, "")
	ambient.Log(DEBUG, "errnie running as ambient context")
}

func NewContext(contextType AmbientContext, namespace string) AmbientContext {
	return contextType.initialize(namespace)
}

type AmbientContext struct {
	ID   uuid.UUID
	TS   int64
	nmsp string
	ctxs map[uuid.UUID][]context.Context
	cnls map[uuid.UUID][]context.CancelFunc
	errs *Collector
	logs *Logger
}

func Ambient() AmbientContext {
	return ambient
}

func (ambient AmbientContext) initialize(namespace string) AmbientContext {
	ambient.ID = uuid.New()
	ambient.TS = time.Now().UnixNano()
	ambient.nmsp = namespace
	ambient.errs = NewCollector(viper.GetInt("errnie.collectors.default.buffer"))
	ambient.logs = NewLogger(&ConsoleLogger{})

	return ambient
}

/*
Handle takes an error type, an arbitrary but basic handler functor, and a splat of errors.
The functor may be nil, it will simply do nothing but add the errors to the collector and logging.
*/
func (ambient AmbientContext) Handle(
	errType ErrType, handler func(interface{}), arg interface{}, errs ...interface{},
) bool {
	if handler != nil {
		defer handler(arg)
	}

	ambient.Add(errType, errs...)
	return ambient.Log(errType, errs...)
}

func (ambient AmbientContext) Add(errType ErrType, errs ...interface{}) bool {
	return ambient.errs.Add(errs, errType)
}

func (ambient AmbientContext) Log(errType ErrType, msgs ...interface{}) bool {
	return ambient.logs.Send(errType, msgs...)
}
