package errnie

import (
	"errors"
	"io"
	"testing"

	"github.com/phuslu/log"
)

var hotpathSink any

/*
BenchmarkHotpathErrorReturn measures the common log-and-return error path.
*/
func BenchmarkHotpathErrorReturn(b *testing.B) {
	configureBenchmarkLogger(b, log.ErrorLevel)
	err := E(Validation, "invalid input", nil)

	b.Run("nil error", func(b *testing.B) {
		for range b.N {
			hotpathSink = Error(nil)
		}
	})

	b.Run("typed error", func(b *testing.B) {
		for range b.N {
			hotpathSink = Error(err)
		}
	})

	b.Run("suppressed typed error", func(b *testing.B) {
		restore := SuppressLogging()
		defer restore()

		b.ResetTimer()
		for range b.N {
			hotpathSink = Error(err)
		}
	})
}

/*
BenchmarkHotpathDoesReturn measures Does followed by Err and Value inspection.
*/
func BenchmarkHotpathDoesReturn(b *testing.B) {
	b.Run("success", func(b *testing.B) {
		for range b.N {
			result := Does(benchmarkDoesSuccessFn)
			hotpathSink = result.Err()
			if result.Err() == nil {
				hotpathSink = result.Value()
			}
		}
	})

	b.Run("failure", func(b *testing.B) {
		for range b.N {
			result := Does(benchmarkDoesErrorFn)
			if err := result.Err(); err != nil {
				hotpathSink = err
			}
		}
	})
}

/*
BenchmarkHotpathLoggingDisabledCaller measures logging with caller capture off.
*/
func BenchmarkHotpathLoggingDisabledCaller(b *testing.B) {
	previous := log.DefaultLogger
	log.DefaultLogger = log.Logger{
		Level:      log.InfoLevel,
		Caller:     0,
		TimeField:  "date",
		TimeFormat: "2006-01-02 15:04:05",
		Writer:     log.IOWriter{Writer: io.Discard},
	}
	logger = NewLogger()
	b.Cleanup(func() {
		log.DefaultLogger = previous
		logger = NewLogger()
	})

	b.Run("info", func(b *testing.B) {
		for range b.N {
			Info("benchmark", "key", "value")
		}
	})
}

/*
BenchmarkHotpathLoggingSuppressedCheck measures the atomic suppression read in
isolation from phuslu/log encoding.
*/
func BenchmarkHotpathLoggingSuppressedCheck(b *testing.B) {
	b.Run("not suppressed", func(b *testing.B) {
		for range b.N {
			benchmarkLoggingSuppressedSink = loggingSuppressed()
		}
	})

	b.Run("suppressed", func(b *testing.B) {
		restore := SuppressLogging()
		defer restore()

		b.ResetTimer()
		for range b.N {
			benchmarkLoggingSuppressedSink = loggingSuppressed()
		}
	})
}

/*
BenchmarkHotpathCombineCleanup measures shutdown-style error joining.
*/
func BenchmarkHotpathCombineCleanup(b *testing.B) {
	first := errors.New("close failed")
	second := E(IO, "flush failed", errors.New("io"))

	for range b.N {
		hotpathSink = Combine(first, second)
	}
}
