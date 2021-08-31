package errnie

import (
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
	ambient = NewContext(AmbientContext{})
}

func NewContext(contextType AmbientContext) AmbientContext {
	return contextType.initialize()
}

type AmbientContext struct {
	ID    uuid.UUID
	TS    int64
	errs  *Collector
	logs  *Logger
	trace *Tracer
}

func Ambient() AmbientContext {
	return ambient
}

func (ambient AmbientContext) initialize() AmbientContext {
	ringSize := 10

	if newSize := viper.GetInt("errnie.collectors.default.buffer"); newSize != 0 {
		ringSize = newSize
	}

	ambient.ID = uuid.New()
	ambient.TS = time.Now().UnixNano()
	ambient.errs = NewCollector(ringSize)
	ambient.logs = NewLogger(&ConsoleLogger{})
	ambient.trace = NewTracer(true)

	return ambient
}

/*
Handle takes an error type, an arbitrary but basic handler functor, and a splat of errors.
The functor may be nil, it will simply do nothing but add the errors to the collector and logging.
*/
func (ambient AmbientContext) Handle(errType ErrType, op OpCode, errs ...interface{}) bool {
	ambient.trace.Caller("\xF0\x9F\x90\x9E", op, errs)
	var ok bool = true

	if ok = ambient.Add(errType, errs...) && ambient.Log(errType, errs...); !ok {
		opcodes[op]()
		ambient.trace.Caller("\xF0\x9F\x91\x8E", "BAD")
		return ok
	}

	ambient.trace.Caller("\xF0\x9F\x91\x8D", "OK")
	return ok
}

func (ambient AmbientContext) Add(errType ErrType, errs ...interface{}) bool {
	ambient.trace.Caller("\xF0\x9F\x90\x9E", errs)
	return ambient.errs.Add(errs, errType)
}

func (ambient AmbientContext) Log(errType ErrType, msgs ...interface{}) bool {
	ambient.trace.Caller("\xF0\x9F\x90\x9E", errType, msgs)
	return ambient.logs.Send(errType, msgs...)
}

func (ambient AmbientContext) Dump() chan Error {
	return ambient.errs.Dump()
}
