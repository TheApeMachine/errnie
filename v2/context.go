package errnie

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/viper"
)

type ContextType uint

const (
	DEFAULT ContextType = iota
)

type CancelType uint

const (
	CANCEL CancelType = iota
	TIMEOUT
)

type Context interface {
	initialize(string) Context
	Handle(ErrType, ...error)
	Add(ErrType, ...error)
	Log(ErrType, ...interface{})
	Context() context.Context
	Cancel()
	Timeout()
}

var ambient Context

func init() {
	ambient = NewContext(AmbientContext{}, "default")
}

func NewContext(contextType Context, namespace string) Context {
	return contextType.initialize(namespace)
}

type AmbientContext struct {
	ID   uuid.UUID
	nmsp string
	ctxs []context.Context
	cnls map[CancelType]context.CancelFunc
	errs *Collector
	logs *Logger
}

func Ambient() Context {
	return ambient
}

func (ambient AmbientContext) initialize(namespace string) Context {
	ambient.ID = uuid.New()
	ambient.cnls = make(map[CancelType]context.CancelFunc)

	timeout := viper.GetDuration("errnie.contexts.default.timeout")

	ctx := context.Background()
	ctx, cf := context.WithCancel(ctx)
	ctx, to := context.WithTimeout(ctx, timeout*time.Second)

	ambient.nmsp = namespace
	ambient.ctxs = append(ambient.ctxs, ctx)
	ambient.cnls[CANCEL] = cf
	ambient.cnls[TIMEOUT] = to
	ambient.errs = NewCollector(viper.GetInt("errnie.collectors.default.buffer"))
	ambient.logs = NewLogger(&ConsoleLogger{})

	return ambient
}

func (ambient AmbientContext) Handle(errType ErrType, errs ...error) {
	ambient.Add(errType, errs...)

	for _, err := range errs {
		ambient.Log(errType, err.Error())
	}
}

func (ambient AmbientContext) Add(errType ErrType, errs ...error) {
	for _, err := range errs {
		ambient.errs.Add(err, errType)
	}
}

func (ambient AmbientContext) Log(errType ErrType, msgs ...interface{}) {
	ambient.logs.Send(errType, msgs)
}

func (ambient AmbientContext) Context() context.Context {
	return ambient.ctxs[DEFAULT]
}

func (ambient AmbientContext) Cancel() {
	ambient.cnls[CANCEL]()
}

func (ambient AmbientContext) Timeout() {
	ambient.cnls[TIMEOUT]()
}
