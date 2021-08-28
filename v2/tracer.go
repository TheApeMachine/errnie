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

func (tracer Tracer) Caller() {
	if !tracer.on {
		return
	}

	pc := make([]uintptr, 15)
	n := runtime.Callers(2, pc)
	frames := runtime.CallersFrames(pc[:n])
	frame, _ := frames.Next()

	ambient.Log(DEBUG, frame.File, frame.Line, frame.Function)
}

func (ambient AmbientContext) Trace() {
	ambient.trace.Caller()
}
