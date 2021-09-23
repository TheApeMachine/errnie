package errnie

import (
	"runtime"
	"strings"

	"github.com/pterm/pterm"
)

type Tracer struct{}

func NewTracer(on bool) *Tracer {
	return &Tracer{}
}

func (tracer Tracer) Caller(prefix string, suffix ...interface{}) {
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
		pterm.Debug.Prefix = pterm.Prefix{
			Text: "TRACER", Style: pterm.NewStyle(pterm.BgBlack, pterm.FgDarkGray),
		}
		pterm.Debug.Println(prefix, fl, frame.Line, fp, suffix)
	}

	pterm.Debug.Prefix = pterm.Prefix{Text: "DEBUG", Style: pterm.NewStyle(
		pterm.BgDarkGray, pterm.FgBlack),
	}
}

func Trace(suffix ...interface{})    { ambctx.Trace(suffix...) }
func TraceIn(suffix ...interface{})  { ambctx.TraceIn(suffix...) }
func TraceOut(suffix ...interface{}) { ambctx.TraceOut(suffix...) }

func (ambctx *AmbientContext) Trace(suffix ...interface{}) {
	ambctx.trace.Caller("\xF0\x9F\x98\x9B <>", suffix...)
}

func (ambctx *AmbientContext) TraceIn(suffix ...interface{}) {
	ambctx.trace.Caller("\xF0\x9F\x94\x8D <-", suffix...)
}

func (ambctx *AmbientContext) TraceOut(suffix ...interface{}) {
	ambctx.trace.Caller("\xF0\x9F\x98\x8E ->", suffix...)
}
