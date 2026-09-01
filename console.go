package errnie

import (
	"io"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/phuslu/log"
)

/*
ANSI escape sequences used by the pretty console renderer. Kept as untyped
string constants so the compiler can fold them into the emitted output.
*/
const (
	ansiReset     = "\x1b[0m"
	ansiBold      = "\x1b[1m"
	ansiDim       = "\x1b[2m"
	ansiRed       = "\x1b[31m"
	ansiGreen     = "\x1b[32m"
	ansiYellow    = "\x1b[33m"
	ansiBlue      = "\x1b[34m"
	ansiMagenta   = "\x1b[35m"
	ansiCyan      = "\x1b[36m"
	ansiGray      = "\x1b[90m"
	ansiWhiteBold = "\x1b[97m\x1b[1m"

	bgRed     = "\x1b[41m\x1b[97m\x1b[1m"
	bgYellow  = "\x1b[43m\x1b[30m\x1b[1m"
	bgGreen   = "\x1b[42m\x1b[30m\x1b[1m"
	bgBlue    = "\x1b[44m\x1b[97m\x1b[1m"
	bgMagenta = "\x1b[45m\x1b[97m\x1b[1m"
	bgGray    = "\x1b[100m\x1b[97m\x1b[1m"
)

/*
causeSeparator is the delimiter ErrnieError.Error uses when flattening a wrapped
cause into a single string. The console renderer splits on it to rebuild the
chain as indented lines.
*/
const causeSeparator = " | "

/*
levelStyle describes how one log level is presented: a badge emoji, a fixed
width label, and the colors used for the badge and the message body.
*/
type levelStyle struct {
	emoji   string
	label   string
	badge   string
	message string
}

/*
levelStyles maps phuslu/log level names to their console presentation. Unknown
levels fall back to unknownLevelStyle.
*/
var levelStyles = map[string]levelStyle{
	"trace": {emoji: "🔍", label: "TRACE", badge: bgMagenta, message: ansiGray},
	"debug": {emoji: "🐛", label: "DEBUG", badge: bgBlue, message: ansiGray},
	"info":  {emoji: "💡", label: "INFO ", badge: bgGreen, message: ansiReset},
	"warn":  {emoji: "⚠️ ", label: "WARN ", badge: bgYellow, message: ansiYellow},
	"error": {emoji: "⛔", label: "ERROR", badge: bgRed, message: ansiRed},
	"fatal": {emoji: "💀", label: "FATAL", badge: bgRed, message: ansiRed + ansiBold},
	"panic": {emoji: "🔥", label: "PANIC", badge: bgRed, message: ansiRed + ansiBold},
}

var unknownLevelStyle = levelStyle{emoji: "❔", label: "?????", badge: bgGray, message: ansiGray}

/*
kindEmoji maps an ErrnieError kind name, as emitted in the kind field, to a
glyph that makes the failure class recognizable without reading the label.
*/
var kindEmoji = map[string]string{
	"validation":                "📋",
	"io":                        "💾",
	"EOF":                       "🏁",
	"context canceled":          "🚫",
	"context deadline exceeded": "⏰",
	"bad_request":               "📛",
	"unauthorized":              "🔒",
	"forbidden":                 "⛔",
	"not_found":                 "🔎",
	"method_not_allowed":        "🚧",
	"not_acceptable":            "🙅",
	"timeout":                   "⏰",
	"conflict":                  "💥",
	"precondition_failed":       "🧩",
	"unsupported_media_type":    "📦",
	"expectation_failed":        "🎭",
	"unprocessable_content":     "🧾",
	"too_many_requests":         "🌊",
	"internal":                  "🧨",
	"not_implemented":           "🚜",
	"bad_gateway":               "🛰️",
	"service_unavailable":       "📴",
	"unknown":                   "❔",
}

/*
noisyFields are keys the pretty renderer folds into the header rather than
printing again in the trailing key/value list.
*/
var noisyFields = map[string]bool{
	"error": true,
	"kind":  true,
}

/*
ConsoleEnabled reports whether errnie should render human-friendly console
output. It is true when stdout is a terminal, unless NO_COLOR is set or
ERRNIE_CONSOLE explicitly overrides the detection.
*/
func ConsoleEnabled() bool {
	if override, ok := os.LookupEnv("ERRNIE_CONSOLE"); ok {
		return isTruthy(override)
	}

	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}

	return log.IsTerminal(os.Stdout.Fd())
}

/*
isTruthy interprets an environment override as a boolean, treating unparseable
values as false.
*/
func isTruthy(value string) bool {
	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
	if err != nil {
		return false
	}

	return parsed
}

/*
newConsoleWriter builds the stdout writer used when pretty rendering is active.
The formatter receives entries already parsed out of errnie's JSON encoding.
*/
func newConsoleWriter(out io.Writer) log.Writer {
	return &log.ConsoleWriter{
		ColorOutput: true,
		Writer:      out,
		Formatter:   formatConsoleEntry,
	}
}

var consoleBuffers = sync.Pool{
	New: func() any {
		buffer := make([]byte, 0, 512)
		return &buffer
	},
}

/*
formatConsoleEntry renders one parsed log entry as a colored header line
followed, when the entry carries a wrapped error chain, by indented cause
lines. It satisfies log.ConsoleWriter.Formatter.
*/
func formatConsoleEntry(out io.Writer, args *log.FormatterArgs) (int, error) {
	pointer := consoleBuffers.Get().(*[]byte)
	buffer := (*pointer)[:0]

	defer func() {
		*pointer = buffer
		consoleBuffers.Put(pointer)
	}()

	style, ok := levelStyles[args.Level]

	if !ok {
		style = unknownLevelStyle
	}

	buffer = appendHeader(buffer, args, style)
	buffer = appendBody(buffer, args, style)
	buffer = appendFields(buffer, args)
	buffer = append(buffer, '\n')
	buffer = appendCauses(buffer, args, style)

	return out.Write(buffer)
}

/*
appendHeader writes the timestamp, level badge, and call site prefix shared by
every console line.
*/
func appendHeader(buffer []byte, args *log.FormatterArgs, style levelStyle) []byte {
	buffer = append(buffer, ansiGray...)
	buffer = append(buffer, clockTime(args.Time)...)
	buffer = append(buffer, ansiReset...)
	buffer = append(buffer, ' ')

	buffer = append(buffer, style.badge...)
	buffer = append(buffer, ' ')
	buffer = append(buffer, style.emoji...)
	buffer = append(buffer, ' ')
	buffer = append(buffer, style.label...)
	buffer = append(buffer, ' ')
	buffer = append(buffer, ansiReset...)
	buffer = append(buffer, ' ')

	if origin := shortOrigin(args); origin != "" {
		buffer = append(buffer, ansiCyan...)
		buffer = append(buffer, origin...)
		buffer = append(buffer, ansiReset...)
		buffer = append(buffer, ' ')
	}

	return buffer
}

/*
appendBody writes the message, or for error entries the head of the error
chain, prefixed by a kind badge when the entry carries one.
*/
func appendBody(buffer []byte, args *log.FormatterArgs, style levelStyle) []byte {
	if kind := args.Get("kind"); kind != "" {
		buffer = append(buffer, ansiMagenta...)
		buffer = append(buffer, kindBadge(kind)...)
		buffer = append(buffer, ansiReset...)
		buffer = append(buffer, ' ')
	}

	message := args.Message

	if message == "" {
		message = causeHead(args.Get("error"))
	}

	buffer = append(buffer, style.message...)
	buffer = append(buffer, message...)
	buffer = append(buffer, ansiReset...)

	return buffer
}

/*
appendFields writes the trailing key=value pairs, skipping the keys already
rendered into the header.
*/
func appendFields(buffer []byte, args *log.FormatterArgs) []byte {
	for _, pair := range args.KeyValues {
		if noisyFields[pair.Key] || pair.Value == "null" {
			continue
		}

		buffer = append(buffer, ' ')
		buffer = append(buffer, ansiDim...)
		buffer = append(buffer, ansiCyan...)
		buffer = append(buffer, pair.Key...)
		buffer = append(buffer, '=')
		buffer = append(buffer, ansiReset...)
		buffer = append(buffer, ansiWhiteBold...)
		buffer = append(buffer, pair.Value...)
		buffer = append(buffer, ansiReset...)
	}

	return buffer
}

/*
appendCauses writes the wrapped cause chain as indented tree lines. The first
segment already appears in the header, so only the remainder is rendered.
*/
func appendCauses(buffer []byte, args *log.FormatterArgs, style levelStyle) []byte {
	causes := causeTail(args.Get("error"))

	for index, cause := range causes {
		buffer = append(buffer, ansiGray...)
		buffer = appendIndent(buffer, index)
		buffer = append(buffer, "└─ "...)
		buffer = append(buffer, ansiReset...)
		buffer = append(buffer, style.message...)
		buffer = append(buffer, cause...)
		buffer = append(buffer, ansiReset...)
		buffer = append(buffer, '\n')
	}

	return buffer
}

/*
appendIndent writes the leading whitespace for a cause line at the given depth,
capped so deep chains stay inside a normal terminal width.
*/
func appendIndent(buffer []byte, depth int) []byte {
	const (
		base   = 2
		step   = 3
		maxPad = 32
	)

	padding := base + depth*step

	if padding > maxPad {
		padding = maxPad
	}

	for count := 0; count < padding; count++ {
		buffer = append(buffer, ' ')
	}

	return buffer
}

/*
clockTime reduces a "2006-01-02 15:04:05" timestamp to its clock portion, which
is the part that varies between adjacent lines. Values in another shape are
returned unchanged.
*/
func clockTime(timestamp string) string {
	if index := strings.IndexByte(timestamp, ' '); index >= 0 {
		return timestamp[index+1:]
	}

	return timestamp
}

/*
shortOrigin renders the call site as "func @ file:line", trimming package paths
so the line stays readable. Returns an empty string when caller capture is off.
*/
func shortOrigin(args *log.FormatterArgs) string {
	caller := trimPath(args.Caller)
	function := trimPath(args.CallerFunc)

	switch {
	case caller == "" && function == "":
		return ""
	case function == "":
		return caller
	case caller == "":
		return function
	}

	return function + " @ " + caller
}

/*
trimPath keeps only the final two path segments of a caller or function name.
*/
func trimPath(value string) string {
	if value == "" {
		return ""
	}

	last := strings.LastIndexByte(value, '/')

	if last < 0 {
		return value
	}

	return value[last+1:]
}

/*
kindBadge renders an ErrnieError kind as an emoji plus bracketed name.
*/
func kindBadge(kind string) string {
	emoji, ok := kindEmoji[kind]

	if !ok {
		emoji = "•"
	}

	return emoji + " [" + kind + "]"
}

/*
causeHead returns the first segment of a flattened error chain.
*/
func causeHead(rendered string) string {
	if index := strings.Index(rendered, causeSeparator); index >= 0 {
		return trimInlineFields(rendered[:index])
	}

	return trimInlineFields(rendered)
}

/*
trimInlineFields removes the " key=value" tail ErrnieError.Error appends for
attached metadata. Those pairs are also emitted as structured fields, so the
console renders them once, in the field list, instead of twice.
*/
func trimInlineFields(segment string) string {
	for {
		space := strings.LastIndexByte(segment, ' ')

		if space < 0 {
			return segment
		}

		token := segment[space+1:]
		equals := strings.IndexByte(token, '=')

		if equals <= 0 || strings.IndexByte(token[:equals], ' ') >= 0 {
			return segment
		}

		segment = segment[:space]
	}
}

/*
causeTail returns every segment of a flattened error chain after the first, in
wrap order. Returns nil when the error has no wrapped cause.
*/
func causeTail(rendered string) []string {
	index := strings.Index(rendered, causeSeparator)

	if index < 0 {
		return nil
	}

	segments := strings.Split(rendered[index+len(causeSeparator):], causeSeparator)

	for position, segment := range segments {
		segments[position] = trimInlineFields(segment)
	}

	return segments
}
