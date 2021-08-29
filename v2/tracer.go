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

func (ambient AmbientContext) Trace() {
	ambient.trace.Caller("\xF0\x9F\x99\x88", "")
}

func (ambient AmbientContext) TraceIn() {
	ambient.trace.Caller("\xF0\x9F\x99\x89", "")
}

func (ambient AmbientContext) TraceOut() {
	ambient.trace.Caller("\xF0\x9F\x99\x8A", "")
}
