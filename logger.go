package errnie

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/phuslu/log"
)

var logger *Logger

var elasticsearchWriterWarn sync.Once

// callerSkip is the phuslu/log runtime.Caller depth. Two frames sit above the
// real call site: phuslu/log's level method and errnie's public wrapper.
const callerSkip = 2

func init() {
	log.DefaultLogger = log.Logger{
		Level:      log.InfoLevel,
		Caller:     callerSkip,
		TimeField:  "date",
		TimeFormat: "2006-01-02 15:04:05",
		Writer:     log.IOWriter{Writer: os.Stdout},
	}

	logger = NewLogger()
}

/*
Apply reconfigures the global errnie logger from Config. Call after Viper or
another loader has populated cfg. Configures level, stdout, and optional file
and Elasticsearch sinks via buildWriter.
*/
func Apply(cfg *Config) {
	log.DefaultLogger = log.Logger{
		Level:      parseLogLevel(cfg.Level),
		Caller:     loggerCaller(cfg),
		TimeField:  "date",
		TimeFormat: "2006-01-02 15:04:05",
		Writer:     buildWriter(cfg),
	}
}

/*
loggerCaller returns the phuslu/log caller skip depth. Set disable_caller in
Config to skip runtime.Caller on hot logging paths.
*/
func loggerCaller(cfg *Config) int {
	if cfg != nil && cfg.DisableCaller {
		return 0
	}

	return callerSkip
}

/*
buildWriter assembles the log.Writer used by Apply. Always includes stdout;
optionally adds a file writer and an async Elasticsearch indexer when enabled
in cfg.
*/
func buildWriter(cfg *Config) log.Writer {
	writers := make([]log.Writer, 0, 3)
	writers = append(writers, log.IOWriter{Writer: os.Stdout})

	if cfg.File.Active && strings.TrimSpace(cfg.File.Path) != "" {
		writers = append(writers, &log.FileWriter{
			Filename:     cfg.File.Path,
			EnsureFolder: true,
		})
	}

	if cfg.Elasticsearch.Active {
		elasticSink, err := newElasticPostWriter(
			cfg.Elasticsearch.URL,
			cfg.Elasticsearch.Index,
			cfg.Elasticsearch.Username,
			cfg.Elasticsearch.Password,
		)

		if err != nil {
			elasticsearchWriterWarn.Do(func() {
				fmt.Fprintf(os.Stderr, "errnie: %v\n", err)
			})
		} else {
			writers = append(writers, &log.AsyncWriter{
				Writer:        log.IOWriter{Writer: elasticSink},
				ChannelSize:   256,
				DiscardOnFull: true,
			})
		}
	}

	if len(writers) == 1 {
		return writers[0]
	}

	multi := log.MultiEntryWriter(writers)

	return &multi
}

/*
parseLogLevel maps a configuration string to a phuslu/log level. Empty or
unknown values default to info.
*/
func parseLogLevel(level string) log.Level {
	level = strings.TrimSpace(level)

	switch {
	case level == "":
		return log.InfoLevel
	case strings.EqualFold(level, "trace"):
		return log.TraceLevel
	case strings.EqualFold(level, "debug"):
		return log.DebugLevel
	case strings.EqualFold(level, "info"):
		return log.InfoLevel
	case strings.EqualFold(level, "warn"), strings.EqualFold(level, "warning"):
		return log.WarnLevel
	case strings.EqualFold(level, "error"):
		return log.ErrorLevel
	case strings.EqualFold(level, "fatal"):
		return log.FatalLevel
	case strings.EqualFold(level, "panic"):
		return log.PanicLevel
	default:
		return log.InfoLevel
	}
}

/*
Logger is the main logger for the errnie package.
*/
type Logger struct {
	handle *log.Logger
}

/*
NewLogger creates a new Logger with the default logger.
*/
func NewLogger() *Logger {
	return &Logger{handle: &log.DefaultLogger}
}

/*
Error logs error at error level with optional alternating key/value fields.
It explicitly returns the error, which allows it to wrap and log the error
directly, preventing yet more repetitive error handling code.

Examples:

```

	func DoSomething() error {
		return fmt.Errorf("something went wrong: %w", errors.New("something else went wrong"))
	}

	func DoAndReturnSomething() (string, error) {
		return "something", errnie.Error(DoSomething())
	}

	func main() {
		errnie.Error(DoSomething())

		something, err := DoAndReturnSomething()

		if err != nil {
			fmt.Println("something went wrong:", err)
		}

		fmt.Println("something:", something)
	}

```
*/
func Error(err error, fields ...any) error {
	if err == io.EOF {
		return err
	}

	if err != nil && !loggingSuppressed() {
		logFields := fields

		if errnieError, ok := AsErrnie(err); ok {
			if attached := errnieError.Fields(); len(attached) > 0 {
				logFields = append(append([]any(nil), attached...), fields...)
			}

			logger.handle.Error().Err(errnieError).KeysAndValues(logFields...).Msg("")

			return err
		}

		logger.handle.Error().Err(err).KeysAndValues(fields...).Msg("")
	}

	return err
}

/*
Warn logs message at warn level with optional alternating key/value fields.
*/
func Warn(message string, fields ...any) {
	if loggingSuppressed() {
		return
	}

	logger.handle.Warn().KeysAndValues(fields).Msg(message)
}

/*
Info logs message at info level with optional alternating key/value fields.
*/
func Info(message string, fields ...any) {
	if loggingSuppressed() {
		return
	}

	logger.handle.Info().KeysAndValues(fields).Msg(message)
}

/*
Debug logs message at debug level with optional alternating key/value fields.
*/
func Debug(message string, fields ...any) {
	if loggingSuppressed() {
		return
	}

	logger.handle.Debug().KeysAndValues(fields).Msg(message)
}

/*
Trace logs message at trace level with optional alternating key/value fields.
*/
func Trace(message string, fields ...any) {
	if loggingSuppressed() {
		return
	}

	logger.handle.Trace().KeysAndValues(fields).Msg(message)
}
