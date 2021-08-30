package errnie

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/spf13/viper"
)

type Tracer struct {
	on bool
}

func NewTracer(on bool) *Tracer {
	go func() {
		for {
			fmt.Println(">", string(debug.Stack()))
		}
	}()

	return &Tracer{on: on}
}

func (tracer Tracer) Caller(prefix string, suffix []interface{}) {
	if !tracer.on {
		return
	}

	pc := make([]uintptr, 15)
	n := runtime.Callers(3, pc)
	frames := runtime.CallersFrames(pc[:n])
	frame, _ := frames.Next()

	fnpart := strings.Split(frame.Function, ".")
	fp := "-> " + strings.Join(fnpart[len(fnpart)-1:], "")

	fnfile := strings.Split(frame.File, "/")
	fl := "-> " + strings.Join(fnfile[len(fnfile)-1:], "")

	ambient.Log(DEBUG, prefix, fl, frame.Line, fp, suffix)
}

func (ambient AmbientContext) Trace(suffix ...interface{}) {
	if !viper.GetBool("trace") {
		return
	}

	ambient.trace.Caller("\xF0\x9F\x98\x9B", suffix)
}

func (ambient AmbientContext) TraceIn(suffix ...interface{}) {
	if !viper.GetBool("trace") {
		return
	}

	ambient.trace.Caller("\xF0\x9F\x98\xAC", suffix)
}

func (ambient AmbientContext) TraceOut(suffix ...interface{}) {
	if !viper.GetBool("trace") {
		return
	}

	ambient.trace.Caller("\xF0\x9F\x98\x8E", suffix)
}
