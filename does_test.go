package errnie

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

/*
TestDoes verifies that Does wraps function outcomes in a Result for success,
failure, generic types, and concurrent use.
*/
func TestDoes(t *testing.T) {
	Convey("Given a function that returns a value and a nil error", t, func() {
		fn := func() (int, error) {
			return 42, nil
		}

		Convey("When Does is called", func() {
			result := Does(fn)

			Convey("Then it should capture the value with a nil error", func() {
				So(result.Value(), ShouldEqual, 42)
				So(result.Err(), ShouldBeNil)
			})
		})
	})

	Convey("Given a function that returns a value and a non-nil error", t, func() {
		expected := errors.New("some error")
		fn := func() (int, error) {
			return 42, expected
		}

		Convey("When Does is called", func() {
			result := Does(fn)

			Convey("Then it should capture both the value and the error without panicking", func() {
				So(result.Value(), ShouldEqual, 42)
				So(result.Err(), ShouldEqual, expected)
			})
		})
	})

	Convey("Given a function that returns a string and a nil error", t, func() {
		fn := func() (string, error) {
			return "test", nil
		}

		Convey("When Does is called", func() {
			result := Does(fn)

			Convey("Then it should work with non-int result types", func() {
				So(result.Value(), ShouldEqual, "test")
				So(result.Err(), ShouldBeNil)
			})
		})
	})

	Convey("Given a function that performs complex logic and returns a nil error", t, func() {
		fn := func() (string, error) {
			return "complex result", nil
		}

		Convey("When Does is called", func() {
			result := Does(fn)

			Convey("Then it should return the computed result", func() {
				So(result.Value(), ShouldEqual, "complex result")
				So(result.Err(), ShouldBeNil)
			})
		})
	})

	Convey("Given multiple goroutines calling Does concurrently", t, func() {
		var wg sync.WaitGroup
		results := make(chan int, 10)

		Convey("When Does is called concurrently", func() {
			for range 10 {
				wg.Add(1)
				go func() {
					defer wg.Done()
					result := Does(func() (int, error) {
						return 42, nil
					})
					results <- result.Value()
				}()
			}
			wg.Wait()
			close(results)

			Convey("Then every result should contain the expected value", func() {
				for result := range results {
					So(result, ShouldEqual, 42)
				}
			})
		})
	})
}

/*
TestResultOr verifies optional error handling on Result, including chaining,
custom error types, and concurrent use.
*/
func TestResultOr(t *testing.T) {
	Convey("Given a successful Does result", t, func() {
		result := Does(func() (int, error) {
			return 1, nil
		})

		Convey("When Or is called", func() {
			called := false
			after := result.Or(func(error) {
				called = true
			})

			Convey("Then the handler should not run and the result should be unchanged", func() {
				So(called, ShouldBeFalse)
				So(after.Value(), ShouldEqual, 1)
				So(after.Err(), ShouldBeNil)
			})
		})
	})

	Convey("Given a failed Does result", t, func() {
		expected := errors.New("handler error")
		result := Does(func() (int, error) {
			return 0, expected
		})

		Convey("When Or is called", func() {
			var handled error
			after := result.Or(func(err error) {
				handled = err
			})

			Convey("Then the handler should receive the error and the result should be unchanged", func() {
				So(handled, ShouldEqual, expected)
				So(after.Value(), ShouldEqual, 0)
				So(after.Err(), ShouldEqual, expected)
			})
		})
	})

	Convey("Given a failed Does result that will be chained", t, func() {
		expected := errors.New("chain error")
		result := Does(func() (string, error) {
			return "", expected
		})

		Convey("When Or is chained twice", func() {
			var first, second int
			after := result.
				Or(func(error) { first++ }).
				Or(func(error) { second++ })

			Convey("Then both handlers should run once", func() {
				So(first, ShouldEqual, 1)
				So(second, ShouldEqual, 1)
				So(after.Err(), ShouldEqual, expected)
			})
		})
	})

	Convey("Given a Does result with a custom error type", t, func() {
		expected := &doesCustomError{msg: "custom error"}
		result := Does(func() (int, error) {
			return 0, expected
		})

		Convey("When Or is called", func() {
			var handled error
			result.Or(func(err error) {
				handled = err
			})

			Convey("Then the handler should receive the same error value", func() {
				So(handled, ShouldEqual, expected)
			})
		})
	})

	Convey("Given multiple goroutines calling Or on failed results concurrently", t, func() {
		var wg sync.WaitGroup
		var handled atomic.Int32
		expected := errors.New("concurrent error")

		Convey("When Or is called concurrently", func() {
			for range 10 {
				wg.Add(1)
				go func() {
					defer wg.Done()
					Does(func() (int, error) {
						return 0, expected
					}).Or(func(error) {
						handled.Add(1)
					})
				}()
			}
			wg.Wait()

			Convey("Then every handler should run exactly once", func() {
				So(handled.Load(), ShouldEqual, 10)
			})
		})
	})
}

/*
TestResultValue verifies that Value returns the wrapped value on success and
failure paths.
*/
func TestResultValue(t *testing.T) {
	Convey("Given a successful Does result", t, func() {
		result := Does(func() (int, error) {
			return 42, nil
		})

		Convey("When Value is called", func() {
			value := result.Value()

			Convey("Then it should return the wrapped value", func() {
				So(value, ShouldEqual, 42)
			})
		})
	})

	Convey("Given a failed Does result whose function still returned a value", t, func() {
		result := Does(func() (int, error) {
			return 42, errors.New("some error")
		})

		Convey("When Value is called", func() {
			value := result.Value()

			Convey("Then it should still return the value from the function", func() {
				So(value, ShouldEqual, 42)
			})
		})
	})

	Convey("Given a successful Does result with a string value", t, func() {
		result := Does(func() (string, error) {
			return "test", nil
		})

		Convey("When Value is called", func() {
			value := result.Value()

			Convey("Then it should return the string value", func() {
				So(value, ShouldEqual, "test")
			})
		})
	})
}

/*
TestResultErr verifies that Err exposes nil on success and the stored error on
failure.
*/
func TestResultErr(t *testing.T) {
	Convey("Given a successful Does result", t, func() {
		result := Does(func() (int, error) {
			return 42, nil
		})

		Convey("When Err is called", func() {
			err := result.Err()

			Convey("Then it should return nil", func() {
				So(err, ShouldBeNil)
			})
		})
	})

	Convey("Given a failed Does result", t, func() {
		expected := errors.New("some error")
		result := Does(func() (int, error) {
			return 0, expected
		})

		Convey("When Err is called", func() {
			err := result.Err()

			Convey("Then it should return the stored error", func() {
				So(err, ShouldEqual, expected)
			})
		})
	})

	Convey("Given a failed Does result with a custom error type", t, func() {
		expected := &doesCustomError{msg: "custom error"}
		result := Does(func() (int, error) {
			return 0, expected
		})

		Convey("When Err is called", func() {
			err := result.Err()

			Convey("Then it should preserve error identity", func() {
				So(err, ShouldEqual, expected)
			})
		})
	})
}

/*
doesCustomError is a custom error type used to assert error identity in Or
and Err tests.
*/
type doesCustomError struct {
	msg string
}

/*
Error implements the error interface for doesCustomError.
*/
func (err *doesCustomError) Error() string {
	return err.msg
}

var (
	benchmarkResultInt    Result[int]
	benchmarkResultString Result[string]
	benchmarkIntSink      int
	benchmarkStringSink   string
	benchmarkErrSink      error
	benchmarkStaticErr    = errors.New("benchmark error")
	benchmarkHandlerErr   = errors.New("handler error")
	benchmarkChainErr     = errors.New("chain error")
	benchmarkSomeErr      = errors.New("some error")
	benchmarkCustomErr    = &doesCustomError{msg: "custom error"}
)

func benchmarkDoesSuccessFn() (int, error) {
	return 42, nil
}

func benchmarkDoesErrorFn() (int, error) {
	return 0, benchmarkStaticErr
}

func benchmarkDoesStringFn() (string, error) {
	return "test", nil
}

func benchmarkDoesValueSuccessFn() (int, error) {
	return 42, nil
}

func benchmarkDoesValueErrorFn() (int, error) {
	return 42, benchmarkSomeErr
}

func benchmarkDoesOrSuccessFn() (int, error) {
	return 1, nil
}

func benchmarkDoesOrErrorFn() (int, error) {
	return 0, benchmarkHandlerErr
}

func benchmarkDoesOrChainFn() (int, error) {
	return 0, benchmarkChainErr
}

func benchmarkDoesErrSuccessFn() (int, error) {
	return 42, nil
}

func benchmarkDoesErrFailureFn() (int, error) {
	return 0, benchmarkSomeErr
}

func benchmarkDoesErrCustomFn() (int, error) {
	return 0, benchmarkCustomErr
}

func benchmarkNoOpHandler(error) {}

/*
BenchmarkDoes measures Does for successful, failed, and alternate value types.
Uses named functions and typed sinks so results reflect Does rather than closure
or interface boxing overhead.
*/
func BenchmarkDoes(b *testing.B) {
	b.Run("success", func(b *testing.B) {
		for range b.N {
			benchmarkResultInt = Does(benchmarkDoesSuccessFn)
		}
	})

	b.Run("error", func(b *testing.B) {
		for range b.N {
			benchmarkResultInt = Does(benchmarkDoesErrorFn)
		}
	})

	b.Run("string success", func(b *testing.B) {
		for range b.N {
			benchmarkResultString = Does(benchmarkDoesStringFn)
		}
	})
}

/*
BenchmarkResultOr measures Or on success, failure, and chained failure paths.
*/
func BenchmarkResultOr(b *testing.B) {
	b.Run("success", func(b *testing.B) {
		result := Does(benchmarkDoesOrSuccessFn)

		b.ResetTimer()
		for range b.N {
			benchmarkResultInt = result.Or(benchmarkNoOpHandler)
		}
	})

	b.Run("error", func(b *testing.B) {
		result := Does(benchmarkDoesOrErrorFn)

		b.ResetTimer()
		for range b.N {
			benchmarkResultInt = result.Or(benchmarkNoOpHandler)
		}
	})

	b.Run("chained error", func(b *testing.B) {
		result := Does(benchmarkDoesOrChainFn)

		b.ResetTimer()
		for range b.N {
			benchmarkResultInt = result.Or(benchmarkNoOpHandler).Or(benchmarkNoOpHandler)
		}
	})
}

/*
BenchmarkResultValue measures Value on success and failure results.
*/
func BenchmarkResultValue(b *testing.B) {
	b.Run("success", func(b *testing.B) {
		result := Does(benchmarkDoesValueSuccessFn)

		b.ResetTimer()
		for range b.N {
			benchmarkIntSink = result.Value()
		}
	})

	b.Run("error", func(b *testing.B) {
		result := Does(benchmarkDoesValueErrorFn)

		b.ResetTimer()
		for range b.N {
			benchmarkIntSink = result.Value()
		}
	})

	b.Run("string success", func(b *testing.B) {
		result := Does(benchmarkDoesStringFn)

		b.ResetTimer()
		for range b.N {
			benchmarkStringSink = result.Value()
		}
	})
}

/*
BenchmarkResultErr measures Err on success and failure results.
*/
func BenchmarkResultErr(b *testing.B) {
	b.Run("success", func(b *testing.B) {
		result := Does(benchmarkDoesErrSuccessFn)

		b.ResetTimer()
		for range b.N {
			benchmarkErrSink = result.Err()
		}
	})

	b.Run("error", func(b *testing.B) {
		result := Does(benchmarkDoesErrFailureFn)

		b.ResetTimer()
		for range b.N {
			benchmarkErrSink = result.Err()
		}
	})

	b.Run("custom error", func(b *testing.B) {
		result := Does(benchmarkDoesErrCustomFn)

		b.ResetTimer()
		for range b.N {
			benchmarkErrSink = result.Err()
		}
	})
}
