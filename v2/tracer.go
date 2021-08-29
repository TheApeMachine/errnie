package errnie

import (
	"runtime"
)

type Tracer struct {
	on bool
}

func NewTracer(on bool) *Tracer {
	return &Tracer{on: on}
}

func (tracer Tracer) Caller(prefix, suffix string) {
	if !tracer.on {
		return
	}

	pc := make([]uintptr, 15)
	n := runtime.Callers(3, pc)
	frames := runtime.CallersFrames(pc[:n])
	frame, _ := frames.Next()

	ambient.Log(DEBUG, prefix, frame.File, frame.Line, frame.Function, suffix)
}

func (ambient AmbientContext) Trace(prefix, suffix string) {
	ambient.trace.Caller(prefix, suffix)
}
