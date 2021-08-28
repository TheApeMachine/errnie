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
	ambient.Log(INFO, "errnie running as ambient context")
}

func NewContext(contextType AmbientContext, namespace string) AmbientContext {
	return contextType.initialize(namespace)
}

type AmbientContext struct {
	ID    uuid.UUID
	TS    int64
	nmsp  string
	ctxs  map[uuid.UUID][]context.Context
	cnls  map[uuid.UUID][]context.CancelFunc
	errs  *Collector
	logs  *Logger
	trace *Tracer
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
	ambient.trace = NewTracer(true)

	return ambient
}

/*
Handle takes an error type, an arbitrary but basic handler functor, and a splat of errors.
The functor may be nil, it will simply do nothing but add the errors to the collector and logging.
*/
func (ambient AmbientContext) Handle(
	errType ErrType, handler func(interface{}), arg interface{}, errs ...interface{},
) bool {
	ambient.Log(DEBUG, "errnie.AmbientContext.Handle <-", errType, handler, arg, errs)

	var bad bool

	if bad = ambient.Add(errType, errs...) && ambient.Log(errType, errs...); bad {
		if handler != nil {
			handler(arg)
		}

		return bad
	}

	return bad
}

func (ambient AmbientContext) Add(errType ErrType, errs ...interface{}) bool {
	return ambient.errs.Add(errs, errType)
}

func (ambient AmbientContext) Log(errType ErrType, msgs ...interface{}) bool {
	return ambient.logs.Send(errType, msgs...)
}
