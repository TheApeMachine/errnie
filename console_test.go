package errnie

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/phuslu/log"
	. "github.com/smartystreets/goconvey/convey"
)

/*
renderConsoleArgs formats already-parsed entry args through the console
formatter and returns the rendered output.
*/
func renderConsoleArgs(args *log.FormatterArgs) string {
	var buffer bytes.Buffer

	_, _ = formatConsoleEntry(&buffer, args)

	return buffer.String()
}

/*
renderConsole formats one entry through the full console writer, exercising the
same JSON parse and format path used at runtime. Fields are supplied as
alternating key/value pairs.
*/
func renderConsole(entry consoleEntry) string {
	var buffer bytes.Buffer

	logger := log.Logger{
		Level:      log.TraceLevel,
		TimeField:  "date",
		TimeFormat: "2006-01-02 15:04:05",
		Writer:     newConsoleWriter(&buffer),
	}

	event := logger.Log()

	if entry.level != "" {
		event = logger.WithLevel(log.ParseLevel(entry.level))
	}

	if entry.caller != "" {
		event = event.Str("caller", entry.caller).Str("callerfunc", entry.callerFunc)
	}

	if entry.err != "" {
		event = event.Str("error", entry.err)
	}

	if entry.kind != "" {
		event = event.Str("kind", entry.kind)
	}

	event.KeysAndValues(entry.fields...).Msg(entry.message)

	return buffer.String()
}

/*
consoleEntry describes a log entry for renderConsole.
*/
type consoleEntry struct {
	level      string
	caller     string
	callerFunc string
	message    string
	err        string
	kind       string
	fields     []any
}

/*
stripANSI removes escape sequences so assertions can target the text content.
*/
func stripANSI(value string) string {
	var builder strings.Builder

	for index := 0; index < len(value); index++ {
		if value[index] != 0x1b {
			builder.WriteByte(value[index])
			continue
		}

		for index < len(value) && value[index] != 'm' {
			index++
		}
	}

	return builder.String()
}

/*
TestFormatConsoleEntryRendersHeader verifies the level badge, call site, and
message all reach the rendered line.
*/
func TestFormatConsoleEntryRendersHeader(t *testing.T) {
	Convey("Given an info entry", t, func() {
		output := renderConsole(consoleEntry{
			level:      "info",
			caller:     "websocket/live.go:502",
			callerFunc: "websocket.NewWithClient.func2",
			message:    "connected",
		})

		Convey("Then the rendered line carries badge, origin, and message", func() {
			plain := stripANSI(output)

			So(plain, ShouldContainSubstring, "INFO")
			So(plain, ShouldContainSubstring, "live.go:502")
			So(plain, ShouldContainSubstring, "connected")
			So(strings.Count(output, "\n"), ShouldEqual, 1)
		})

		Convey("Then the output is colorized", func() {
			So(output, ShouldContainSubstring, "\x1b[")
		})
	})
}

/*
TestFormatConsoleEntryUnfoldsCauseChain verifies a flattened cause chain is
rendered as indented tree lines under the header.
*/
func TestFormatConsoleEntryUnfoldsCauseChain(t *testing.T) {
	Convey("Given an error entry wrapping two causes", t, func() {
		output := renderConsole(consoleEntry{
			level:      "error",
			caller:     "strategy/strategy.go:66",
			callerFunc: "strategy.(*Strategy).Step",
			err:        "strategy: observe envelope | strategy: measurement failed | calculus: quotient denominator must be non-zero",
			kind:       "validation",
		})
		plain := stripANSI(output)

		Convey("Then the head appears on the first line", func() {
			So(strings.Split(plain, "\n")[0], ShouldContainSubstring, "strategy: observe envelope")
		})

		Convey("Then each cause gets its own indented line", func() {
			So(plain, ShouldContainSubstring, "  └─ strategy: measurement failed")
			So(plain, ShouldContainSubstring, "     └─ calculus: quotient denominator must be non-zero")
			So(strings.Count(output, "\n"), ShouldEqual, 3)
		})

		Convey("Then the kind is badged and not repeated as a field", func() {
			So(plain, ShouldContainSubstring, "[validation]")
			So(plain, ShouldNotContainSubstring, "kind=validation")
		})
	})
}

/*
TestFormatConsoleEntryRendersFields verifies structured fields print once and
that header-folded keys are not duplicated in the field list.
*/
func TestFormatConsoleEntryRendersFields(t *testing.T) {
	Convey("Given an entry carrying fields", t, func() {
		plain := stripANSI(renderConsole(consoleEntry{
			level:   "warn",
			message: "reconnect",
			fields:  []any{"attempt", 3, "pair", "FIL/USD"},
		}))

		Convey("Then every field is rendered as key=value", func() {
			So(plain, ShouldContainSubstring, "attempt=3")
			So(plain, ShouldContainSubstring, "pair=FIL/USD")
		})
	})

	Convey("Given an error whose message already inlines its fields", t, func() {
		plain := stripANSI(renderConsole(consoleEntry{
			level:  "error",
			err:    "strategy: measurement failed pair=FIL/USD",
			fields: []any{"pair", "FIL/USD"},
		}))

		Convey("Then the inline copy is trimmed from the message", func() {
			So(strings.Count(plain, "FIL/USD"), ShouldEqual, 1)
			So(plain, ShouldContainSubstring, "strategy: measurement failed")
		})
	})
}

/*
TestFormatConsoleEntryFallsBackToErrorText verifies entries with no message use
the head of the error chain as the body.
*/
func TestFormatConsoleEntryFallsBackToErrorText(t *testing.T) {
	Convey("Given an error entry with an empty message", t, func() {
		plain := stripANSI(renderConsole(consoleEntry{
			level: "error",
			err:   "connection reset by peer",
		}))

		Convey("Then the error text becomes the body", func() {
			So(plain, ShouldContainSubstring, "connection reset by peer")
		})
	})
}

/*
TestTrimInlineFields verifies the inline key=value tail stripper keeps message
text that merely contains spaces or equals signs.
*/
func TestTrimInlineFields(t *testing.T) {
	Convey("Given rendered error segments", t, func() {
		Convey("Then trailing key=value pairs are removed", func() {
			So(trimInlineFields("op: failed pair=FIL/USD"), ShouldEqual, "op: failed")
			So(trimInlineFields("op: failed a=1 b=2"), ShouldEqual, "op: failed")
		})

		Convey("Then plain messages are left intact", func() {
			So(trimInlineFields("op: failed"), ShouldEqual, "op: failed")
			So(trimInlineFields("connection reset by peer"), ShouldEqual, "connection reset by peer")
			So(trimInlineFields(""), ShouldEqual, "")
		})
	})
}

/*
TestConsoleActiveHonoursConfig verifies the Config override takes precedence
over terminal detection in both directions.
*/
func TestConsoleActiveHonoursConfig(t *testing.T) {
	Convey("Given explicit console settings", t, func() {
		Convey("Then on-values force the renderer on", func() {
			So(consoleActive(&Config{Console: "on"}), ShouldBeTrue)
			So(consoleActive(&Config{Console: "TRUE"}), ShouldBeTrue)
		})

		Convey("Then off-values force the renderer off", func() {
			So(consoleActive(&Config{Console: "off"}), ShouldBeFalse)
			So(consoleActive(&Config{Console: "never"}), ShouldBeFalse)
		})
	})

	Convey("Given a non-terminal stdout and no override", t, func() {
		t.Setenv("ERRNIE_CONSOLE", "false")

		Convey("Then the renderer stays off", func() {
			So(consoleActive(&Config{}), ShouldBeFalse)
			So(consoleActive(nil), ShouldBeFalse)
		})
	})
}

/*
TestConsoleEnabledEnvOverrides verifies ERRNIE_CONSOLE and NO_COLOR handling.
*/
func TestConsoleEnabledEnvOverrides(t *testing.T) {
	Convey("Given ERRNIE_CONSOLE is set", t, func() {
		t.Setenv("ERRNIE_CONSOLE", "true")

		Convey("Then detection is bypassed", func() {
			So(ConsoleEnabled(), ShouldBeTrue)
		})
	})

	Convey("Given NO_COLOR is set and no explicit override", t, func() {
		t.Setenv("ERRNIE_CONSOLE", "")
		os.Unsetenv("ERRNIE_CONSOLE")
		t.Setenv("NO_COLOR", "1")

		Convey("Then the renderer stays off", func() {
			So(ConsoleEnabled(), ShouldBeFalse)
		})
	})
}
