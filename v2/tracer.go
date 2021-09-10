package errnie

import (
	"runtime"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/viper"
)

type Tracer struct {
	on bool
}

func NewTracer(on bool) *Tracer {
	return &Tracer{on: on}
}

func (tracer Tracer) Caller(prefix string, suffix ...interface{}) {
	if !tracer.on {
		return
	}

	pc := make([]uintptr, 15)
	n := runtime.Callers(3, pc)
	frame, _ := runtime.CallersFrames(pc[:n]).Next()

	fnpart := strings.Split(frame.Function, ".")
	fp := strings.Join(fnpart[len(fnpart)-1:], "")

	fnfile := strings.Split(frame.File, "/")
	fl := strings.Join(fnfile[len(fnfile)-1:], "")

	pterm.Debug.Prefix = pterm.Prefix{Text: "ERRNIE", Style: pterm.NewStyle(pterm.BgBlack, pterm.FgDarkGray)}

	switch fl + fp {
	case "context.goHandle":
		pterm.Debug.Println(prefix, "-- handling potential errors", suffix)
	case "context.goLog":
		pterm.Debug.Println(prefix, "-- logging potential errors", suffix)
	case "context.goAdd":
		pterm.Debug.Println(prefix, "-- adding potential errors", suffix)
	case "collector.goAdd":
		pterm.Debug.Println(prefix, "-- finding real errors", suffix)
	default:
		pterm.Debug.Prefix = pterm.Prefix{Text: "TRACER", Style: pterm.NewStyle(pterm.BgBlack, pterm.FgDarkGray)}
		pterm.Debug.Println(prefix, fl, frame.Line, fp, suffix)
	}

	pterm.Debug.Prefix = pterm.Prefix{Text: "DEBUG", Style: pterm.NewStyle(pterm.BgDarkGray, pterm.FgBlack)}
}

func (ambient AmbientContext) Trace(suffix ...interface{}) {
	if !viper.GetViper().GetBool("trace") {
		return
	}

	//ambient.trace.Caller("\xF0\x9F\x98\x9B <>", suffix...)
}

func (ambient AmbientContext) TraceIn(suffix ...interface{}) {
	if !viper.GetViper().GetBool("trace") {
		return
	}

	//ambient.trace.Caller("\xF0\x9F\x94\x8D <-", suffix...)
}

func (ambient AmbientContext) TraceOut(suffix ...interface{}) {
	if !viper.GetViper().GetBool("trace") {
		return
	}

	//ambient.trace.Caller("\xF0\x9F\x98\x8E ->", suffix...)
}
