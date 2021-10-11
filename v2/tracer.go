package errnie

import (
	"runtime"
	"strings"

	"github.com/pterm/pterm"
)

/*
Tracer provides some basic stack tracing features at the moment which expose the
method chains that are in use for any given operation. This is planned to be
extended quite a bit in the near future.
*/
type Tracer struct{}

/*
NewTracer constructs the Tracer object and hands back a pointer to it.
*/
func NewTracer(on bool) *Tracer {
	return &Tracer{}
}

/*
Caller extracts useful information from the call stack and presents it in a
structured form to the user to aid in debugging processes.
*/
func (tracer Tracer) Caller(prefix string, suffix ...interface{}) {
	pc := make([]uintptr, 15)
	n := runtime.Callers(4, pc)
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
		pterm.Debug.Prefix = pterm.Prefix{
			Text: "TRACER", Style: pterm.NewStyle(pterm.BgBlack, pterm.FgDarkGray),
		}
		pterm.Debug.Println(prefix, fl, frame.Line, fp, suffix)
	}

	pterm.Debug.Prefix = pterm.Prefix{Text: "DEBUG", Style: pterm.NewStyle(
		pterm.BgDarkGray, pterm.FgBlack),
	}
}

/*
Trace is the generic tracing method one can call, designed to go at any
desired place in the code.
*/
func Trace(suffix ...interface{}) { ambctx.Trace(suffix...) }

/*
TraceIn was designed to go at the beginning of a method, but this is not
required. Following that pattern does have the benefit of the information
being structured in a way that makes sense.
*/
func TraceIn(suffix ...interface{}) { ambctx.TraceIn(suffix...) }

/*
TraceOut can be used at the end of a method, before the return. One good
use-case is to see which data the method is returning with.
*/
func TraceOut(suffix ...interface{}) { ambctx.TraceOut(suffix...) }

/*
Trace is the proxied method described above with a similar name.
*/
func (ambctx *AmbientContext) Trace(suffix ...interface{}) {
	ambctx.trace.Caller("\xF0\x9F\x98\x9B <>", suffix...)
}

/*
TraceIn is the proxied method described above with a similar name.
*/
func (ambctx *AmbientContext) TraceIn(suffix ...interface{}) {
	ambctx.trace.Caller("\xF0\x9F\x94\x8D <-", suffix...)
}

/*
TraceOut is the proxied method described above with a similar name.
*/
func (ambctx *AmbientContext) TraceOut(suffix ...interface{}) {
	ambctx.trace.Caller("\xF0\x9F\x98\x8E ->", suffix...)
}
