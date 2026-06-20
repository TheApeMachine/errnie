package errnie

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/phuslu/log"
	. "github.com/smartystreets/goconvey/convey"
)

/*
configureTestLogger redirects the global errnie logger to a buffer for the
duration of a test and restores the previous logger on cleanup.
*/
func configureTestLogger(t *testing.T, level log.Level) *bytes.Buffer {
	t.Helper()

	var buffer bytes.Buffer
	previous := log.DefaultLogger

	log.DefaultLogger = log.Logger{
		Level:      level,
		Caller:     callerSkip,
		TimeField:  "date",
		TimeFormat: "2006-01-02 15:04:05",
		Writer:     log.IOWriter{Writer: &buffer},
	}
	logger = NewLogger()

	t.Cleanup(func() {
		log.DefaultLogger = previous
		logger = NewLogger()
	})

	return &buffer
}

/*
TestApply verifies that Apply reconfigures the global logger from Config.
*/
func TestApply(t *testing.T) {
	Convey("Given a debug-level config", t, func() {
		cfg := &Config{Level: "debug"}

		Convey("When Apply is called", func() {
			Apply(cfg)

			Convey("Then the default logger level should be updated", func() {
				So(log.DefaultLogger.Level, ShouldEqual, log.DebugLevel)
			})
		})
	})
}

/*
TestBuildWriter verifies stdout-only and multi-sink writer assembly.
*/
func TestBuildWriter(t *testing.T) {
	Convey("Given a config with only default fields", t, func() {
		cfg := &Config{}

		Convey("When buildWriter is called", func() {
			writer := buildWriter(cfg)

			Convey("Then it should return a single stdout writer", func() {
				So(writer, ShouldHaveSameTypeAs, log.IOWriter{})
			})
		})
	})

	Convey("Given a config with file logging enabled", t, func() {
		tempDir := t.TempDir()
		cfg := &Config{}
		cfg.File.Active = true
		cfg.File.Path = filepath.Join(tempDir, "nested", "app.log")

		Convey("When buildWriter is called", func() {
			writer := buildWriter(cfg)

			Convey("Then it should return a multi-entry writer", func() {
				So(writer, ShouldHaveSameTypeAs, &log.MultiEntryWriter{})
			})
		})
	})

	Convey("Given a config with invalid Elasticsearch settings", t, func() {
		cfg := &Config{}
		cfg.Elasticsearch.Active = true
		cfg.Elasticsearch.URL = ""
		cfg.Elasticsearch.Index = "logs"

		Convey("When buildWriter is called", func() {
			writer := buildWriter(cfg)

			Convey("Then it should fall back to stdout only", func() {
				So(writer, ShouldHaveSameTypeAs, log.IOWriter{})
			})
		})
	})
}

/*
TestParseLogLevel verifies level string parsing and defaults.
*/
func TestParseLogLevel(t *testing.T) {
	Convey("Given supported log level strings", t, func() {
		cases := map[string]log.Level{
			"trace":    log.TraceLevel,
			"debug":    log.DebugLevel,
			"info":     log.InfoLevel,
			"":         log.InfoLevel,
			"warn":     log.WarnLevel,
			"warning":  log.WarnLevel,
			"error":    log.ErrorLevel,
			"fatal":    log.FatalLevel,
			"panic":    log.PanicLevel,
			"unknown":  log.InfoLevel,
			"  DEBUG ": log.DebugLevel,
		}

		Convey("When parseLogLevel is called for each", func() {
			Convey("Then it should map to the expected level", func() {
				for level, expected := range cases {
					So(parseLogLevel(level), ShouldEqual, expected)
				}
			})
		})
	})
}

/*
TestNewLogger verifies Logger construction against the current default logger.
*/
func TestNewLogger(t *testing.T) {
	Convey("Given the default phuslu logger", t, func() {
		Convey("When NewLogger is called", func() {
			instance := NewLogger()

			Convey("Then it should wrap the default logger handle", func() {
				So(instance, ShouldNotBeNil)
				So(instance.handle, ShouldEqual, &log.DefaultLogger)
			})
		})
	})
}

/*
TestLoggerCaller verifies caller attribution skips errnie wrappers.
*/
func TestLoggerCaller(t *testing.T) {
	Convey("Given logging with caller capture enabled", t, func() {
		buffer := configureTestLogger(t, log.InfoLevel)

		Convey("When Info is called from this test", func() {
			Info("caller probe")

			Convey("Then the caller should not point at errnie/logger.go", func() {
				output := buffer.String()
				So(output, ShouldContainSubstring, `"caller":`)
				So(output, ShouldNotContainSubstring, "errnie/logger.go")
				So(output, ShouldContainSubstring, "logger_test.go")
			})
		})
	})
}

/*
TestError verifies log-and-return behaviour for Error.
*/
func TestError(t *testing.T) {
	Convey("Given a nil error", t, func() {
		buffer := configureTestLogger(t, log.ErrorLevel)

		Convey("When Error is called", func() {
			result := Error(nil, "key", "value")

			Convey("Then it should return nil without logging", func() {
				So(result, ShouldBeNil)
				So(buffer.Len(), ShouldEqual, 0)
			})
		})
	})

	Convey("Given a non-nil error and logging enabled", t, func() {
		buffer := configureTestLogger(t, log.ErrorLevel)
		expected := errors.New("boom")

		Convey("When Error is called", func() {
			result := Error(expected, "key", "value")

			Convey("Then it should log and return the same error", func() {
				So(result, ShouldEqual, expected)
				So(buffer.String(), ShouldContainSubstring, "boom")
			})
		})
	})

	Convey("Given a non-nil error and logging suppressed", t, func() {
		buffer := configureTestLogger(t, log.ErrorLevel)
		expected := errors.New("quiet boom")
		restore := SuppressLogging()
		defer restore()

		Convey("When Error is called", func() {
			result := Error(expected)

			Convey("Then it should return the error without logging", func() {
				So(result, ShouldEqual, expected)
				So(buffer.Len(), ShouldEqual, 0)
			})
		})
	})

	Convey("Given an ErrnieError with attached fields", t, func() {
		buffer := configureTestLogger(t, log.ErrorLevel)
		expected := Err(Validation, "payload is empty", nil).With(
			"origin", "kraken:public",
			"role", "trade",
			"scope", "update",
		)

		Convey("When Error is called", func() {
			result := Error(expected)

			Convey("Then it should log the attached fields", func() {
				So(result, ShouldEqual, expected)

				logLine := buffer.String()

				So(logLine, ShouldContainSubstring, "payload is empty")
				So(logLine, ShouldContainSubstring, "origin")
				So(logLine, ShouldContainSubstring, "kraken:public")
				So(logLine, ShouldContainSubstring, "role")
				So(logLine, ShouldContainSubstring, "trade")
				So(logLine, ShouldContainSubstring, "scope")
				So(logLine, ShouldContainSubstring, "update")
			})
		})
	})
}

/*
TestWarn verifies warn-level logging.
*/
func TestWarn(t *testing.T) {
	Convey("Given logging is enabled", t, func() {
		buffer := configureTestLogger(t, log.WarnLevel)

		Convey("When Warn is called", func() {
			Warn("warn message", "key", "value")

			Convey("Then it should write a warn log entry", func() {
				So(buffer.String(), ShouldContainSubstring, "warn message")
			})
		})
	})

	Convey("Given logging is suppressed", t, func() {
		buffer := configureTestLogger(t, log.WarnLevel)
		restore := SuppressLogging()
		defer restore()

		Convey("When Warn is called", func() {
			Warn("hidden warn")

			Convey("Then it should not write a log entry", func() {
				So(buffer.Len(), ShouldEqual, 0)
			})
		})
	})
}

/*
TestInfo verifies info-level logging.
*/
func TestInfo(t *testing.T) {
	Convey("Given logging is enabled", t, func() {
		buffer := configureTestLogger(t, log.InfoLevel)

		Convey("When Info is called", func() {
			Info("info message", "key", "value")

			Convey("Then it should write an info log entry", func() {
				So(buffer.String(), ShouldContainSubstring, "info message")
			})
		})
	})

	Convey("Given logging is suppressed", t, func() {
		buffer := configureTestLogger(t, log.InfoLevel)
		restore := SuppressLogging()
		defer restore()

		Convey("When Info is called", func() {
			Info("hidden info")

			Convey("Then it should not write a log entry", func() {
				So(buffer.Len(), ShouldEqual, 0)
			})
		})
	})
}

/*
TestDebug verifies debug-level logging.
*/
func TestDebug(t *testing.T) {
	Convey("Given logging is enabled at debug level", t, func() {
		buffer := configureTestLogger(t, log.DebugLevel)

		Convey("When Debug is called", func() {
			Debug("debug message", "key", "value")

			Convey("Then it should write a debug log entry", func() {
				So(buffer.String(), ShouldContainSubstring, "debug message")
			})
		})
	})

	Convey("Given logging is suppressed", t, func() {
		buffer := configureTestLogger(t, log.DebugLevel)
		restore := SuppressLogging()
		defer restore()

		Convey("When Debug is called", func() {
			Debug("hidden debug")

			Convey("Then it should not write a log entry", func() {
				So(buffer.Len(), ShouldEqual, 0)
			})
		})
	})
}

/*
TestTrace verifies trace-level logging.
*/
func TestTrace(t *testing.T) {
	Convey("Given logging is enabled at trace level", t, func() {
		buffer := configureTestLogger(t, log.TraceLevel)

		Convey("When Trace is called", func() {
			Trace("trace message", "key", "value")

			Convey("Then it should write a trace log entry", func() {
				So(buffer.String(), ShouldContainSubstring, "trace message")
			})
		})
	})

	Convey("Given logging is suppressed", t, func() {
		buffer := configureTestLogger(t, log.TraceLevel)
		restore := SuppressLogging()
		defer restore()

		Convey("When Trace is called", func() {
			Trace("hidden trace")

			Convey("Then it should not write a log entry", func() {
				So(buffer.Len(), ShouldEqual, 0)
			})
		})
	})
}

var (
	benchmarkLoggerBuffer bytes.Buffer
	benchmarkLoggerErr    error
)

func configureBenchmarkLogger(b *testing.B, level log.Level) {
	b.Helper()

	benchmarkLoggerBuffer.Reset()
	log.DefaultLogger = log.Logger{
		Level:      level,
		TimeField:  "date",
		TimeFormat: "2006-01-02 15:04:05",
		Writer:     log.IOWriter{Writer: io.Discard},
	}
	logger = NewLogger()
}

/*
BenchmarkApply measures Apply with a minimal config.
*/
func BenchmarkApply(b *testing.B) {
	cfg := &Config{Level: "info"}

	b.ResetTimer()
	for range b.N {
		Apply(cfg)
	}
}

/*
BenchmarkBuildWriter measures writer assembly for stdout-only and file sinks.
*/
func BenchmarkBuildWriter(b *testing.B) {
	stdoutOnly := &Config{}
	withFile := &Config{}
	withFile.File.Active = true
	withFile.File.Path = os.TempDir() + string(os.PathSeparator) + "errnie-bench.log"

	b.Run("stdout only", func(b *testing.B) {
		for range b.N {
			benchmarkLoggerWriterSink = buildWriter(stdoutOnly)
		}
	})

	b.Run("stdout and file", func(b *testing.B) {
		for range b.N {
			benchmarkLoggerWriterSink = buildWriter(withFile)
		}
	})
}

/*
BenchmarkParseLogLevel measures level parsing for known and default values.
*/
func BenchmarkParseLogLevel(b *testing.B) {
	b.Run("debug", func(b *testing.B) {
		for range b.N {
			benchmarkLoggerLevelSink = parseLogLevel("debug")
		}
	})

	b.Run("default", func(b *testing.B) {
		for range b.N {
			benchmarkLoggerLevelSink = parseLogLevel("not-a-level")
		}
	})
}

/*
BenchmarkNewLogger measures Logger construction.
*/
func BenchmarkNewLogger(b *testing.B) {
	for range b.N {
		benchmarkLoggerInstanceSink = NewLogger()
	}
}

/*
BenchmarkError measures Error for nil and non-nil errors.
*/
func BenchmarkError(b *testing.B) {
	configureBenchmarkLogger(b, log.ErrorLevel)
	err := errors.New("benchmark error")

	b.Run("nil error", func(b *testing.B) {
		for range b.N {
			benchmarkLoggerErr = Error(nil)
		}
	})

	b.Run("non-nil error", func(b *testing.B) {
		for range b.N {
			benchmarkLoggerErr = Error(err, "key", "value")
		}
	})

	b.Run("suppressed non-nil error", func(b *testing.B) {
		restore := SuppressLogging()
		defer restore()

		b.ResetTimer()
		for range b.N {
			benchmarkLoggerErr = Error(err)
		}
	})
}

/*
BenchmarkWarn measures Warn with logging enabled and suppressed.
*/
func BenchmarkWarn(b *testing.B) {
	configureBenchmarkLogger(b, log.WarnLevel)

	b.Run("enabled", func(b *testing.B) {
		for range b.N {
			Warn("benchmark warn", "key", "value")
		}
	})

	b.Run("suppressed", func(b *testing.B) {
		restore := SuppressLogging()
		defer restore()

		b.ResetTimer()
		for range b.N {
			Warn("benchmark warn")
		}
	})
}

/*
BenchmarkInfo measures Info with logging enabled and suppressed.
*/
func BenchmarkInfo(b *testing.B) {
	configureBenchmarkLogger(b, log.InfoLevel)

	b.Run("enabled", func(b *testing.B) {
		for range b.N {
			Info("benchmark info", "key", "value")
		}
	})

	b.Run("suppressed", func(b *testing.B) {
		restore := SuppressLogging()
		defer restore()

		b.ResetTimer()
		for range b.N {
			Info("benchmark info")
		}
	})
}

/*
BenchmarkDebug measures Debug with logging enabled and suppressed.
*/
func BenchmarkDebug(b *testing.B) {
	configureBenchmarkLogger(b, log.DebugLevel)

	b.Run("enabled", func(b *testing.B) {
		for range b.N {
			Debug("benchmark debug", "key", "value")
		}
	})

	b.Run("suppressed", func(b *testing.B) {
		restore := SuppressLogging()
		defer restore()

		b.ResetTimer()
		for range b.N {
			Debug("benchmark debug")
		}
	})
}

/*
BenchmarkTrace measures Trace with logging enabled and suppressed.
*/
func BenchmarkTrace(b *testing.B) {
	configureBenchmarkLogger(b, log.TraceLevel)

	b.Run("enabled", func(b *testing.B) {
		for range b.N {
			Trace("benchmark trace", "key", "value")
		}
	})

	b.Run("suppressed", func(b *testing.B) {
		restore := SuppressLogging()
		defer restore()

		b.ResetTimer()
		for range b.N {
			Trace("benchmark trace")
		}
	})
}

var (
	benchmarkLoggerWriterSink   log.Writer
	benchmarkLoggerLevelSink    log.Level
	benchmarkLoggerInstanceSink *Logger
)
