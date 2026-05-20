package errnie

import (
	"context"
	"errors"
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

/*
TestE verifies ErrnieError construction through E.
*/
func TestE(t *testing.T) {
	Convey("Given a kind and message without a cause", t, func() {
		Convey("When E is called", func() {
			err := E(Validation, "email is invalid", nil)

			Convey("Then it should return a typed error with the expected fields", func() {
				So(err, ShouldNotBeNil)
				So(err.Kind, ShouldEqual, Validation)
				So(err.Message, ShouldEqual, "email is invalid")
				So(err.Cause, ShouldBeNil)
			})
		})
	})

	Convey("Given a wrapped standard library error", t, func() {
		cause := errors.New("read failed")

		Convey("When E is called", func() {
			err := E(IO, "load config", cause)

			Convey("Then the cause should be preserved for Unwrap", func() {
				So(err.Cause, ShouldEqual, cause)
				So(errors.Is(err, cause), ShouldBeTrue)
			})
		})
	})

	Convey("Given a context cancellation cause", t, func() {
		Convey("When E wraps context.Canceled", func() {
			err := E(Timeout, "request cancelled", context.Canceled)

			Convey("Then context matching should still work through the chain", func() {
				So(errors.Is(err, context.Canceled), ShouldBeTrue)
				So(IsContext(err), ShouldBeTrue)
			})
		})
	})
}

/*
TestErrnieErrorOperation verifies operation metadata chaining.
*/
func TestErrnieErrorOperation(t *testing.T) {
	Convey("Given an ErrnieError from E", t, func() {
		err := E(NotFound, "user missing", nil)

		Convey("When Operation is called", func() {
			same := err.Operation("user.load")

			Convey("Then it should set Op and return the same error", func() {
				So(same, ShouldEqual, err)
				So(err.Op, ShouldEqual, "user.load")
				So(err.Error(), ShouldEqual, "user.load: user missing")
			})
		})
	})

	Convey("Given a nil ErrnieError", t, func() {
		var err *ErrnieError

		Convey("When Operation is called", func() {
			result := err.Operation("noop")

			Convey("Then it should return nil", func() {
				So(result, ShouldBeNil)
			})
		})
	})
}

/*
TestErrnieErrorWith verifies structured metadata attachment.
*/
func TestErrnieErrorWith(t *testing.T) {
	Convey("Given an ErrnieError from E", t, func() {
		err := E(HTTP, "request failed", nil)

		Convey("When With is called with key/value pairs", func() {
			same := err.With("status", 500, "url", "https://example.com")

			Convey("Then it should attach fields and return the same error", func() {
				So(same, ShouldEqual, err)
				So(err.Fields()["status"], ShouldEqual, 500)
				So(err.Fields()["url"], ShouldEqual, "https://example.com")
			})
		})
	})

	Convey("Given a nil ErrnieError", t, func() {
		var err *ErrnieError

		Convey("When With is called", func() {
			result := err.With("key", "value")

			Convey("Then it should return nil", func() {
				So(result, ShouldBeNil)
			})
		})
	})
}

/*
TestErrnieErrorFields verifies read access to metadata maps.
*/
func TestErrnieErrorFields(t *testing.T) {
	Convey("Given an ErrnieError without metadata", t, func() {
		err := E(Unknown, "plain", nil)

		Convey("When Fields is called", func() {
			fields := err.Fields()

			Convey("Then it should return nil", func() {
				So(fields, ShouldBeNil)
			})
		})
	})

	Convey("Given a nil ErrnieError", t, func() {
		var err *ErrnieError

		Convey("When Fields is called", func() {
			fields := err.Fields()

			Convey("Then it should return nil", func() {
				So(fields, ShouldBeNil)
			})
		})
	})
}

/*
TestErrnieErrorError verifies the error string format.
*/
func TestErrnieErrorError(t *testing.T) {
	Convey("Given an ErrnieError with message only", t, func() {
		err := E(Validation, "invalid email", nil)

		Convey("When Error is called", func() {
			text := err.Error()

			Convey("Then it should return the message", func() {
				So(text, ShouldEqual, "invalid email")
			})
		})
	})

	Convey("Given an ErrnieError with operation and empty message", t, func() {
		err := E(Unknown, "", errors.New("underlying"))

		Convey("When Error is called after Operation", func() {
			err.Operation("db.query")
			text := err.Error()

			Convey("Then it should fall back to the cause message", func() {
				So(text, ShouldEqual, "db.query: underlying")
			})
		})
	})

	Convey("Given a nil ErrnieError", t, func() {
		var err *ErrnieError

		Convey("When Error is called", func() {
			text := err.Error()

			Convey("Then it should return an empty string", func() {
				So(text, ShouldEqual, "")
			})
		})
	})
}

/*
TestErrnieErrorUnwrap verifies wrapping support for errors.Is and errors.As.
*/
func TestErrnieErrorUnwrap(t *testing.T) {
	Convey("Given a wrapped ErrnieError chain", t, func() {
		root := E(Validation, "invalid", errors.New("root"))
		wrapped := fmt.Errorf("outer: %w", root)

		Convey("When errors.As is used", func() {
			target, ok := AsErrnie(wrapped)

			Convey("Then it should find the typed error", func() {
				So(ok, ShouldBeTrue)
				So(target.Kind, ShouldEqual, Validation)
			})
		})
	})

	Convey("Given a nil ErrnieError", t, func() {
		var err *ErrnieError

		Convey("When Unwrap is called", func() {
			cause := err.Unwrap()

			Convey("Then it should return nil", func() {
				So(cause, ShouldBeNil)
			})
		})
	})
}

/*
TestCombine verifies nil-safe error joining via errors.Join.
*/
func TestCombine(t *testing.T) {
	Convey("Given only nil errors", t, func() {
		Convey("When Combine is called", func() {
			err := Combine(nil, nil)

			Convey("Then it should return nil", func() {
				So(err, ShouldBeNil)
			})
		})
	})

	Convey("Given a single non-nil error", t, func() {
		expected := errors.New("one")

		Convey("When Combine is called", func() {
			err := Combine(nil, expected, nil)

			Convey("Then it should return that error", func() {
				So(errors.Is(err, expected), ShouldBeTrue)
			})
		})
	})

	Convey("Given multiple non-nil errors", t, func() {
		first := errors.New("first")
		second := E(IO, "close failed", errors.New("second"))

		Convey("When Combine is called", func() {
			err := Combine(first, second)

			Convey("Then both errors should be matchable", func() {
				So(errors.Is(err, first), ShouldBeTrue)
				So(IsIO(err), ShouldBeTrue)
			})
		})
	})
}

/*
TestAsErrnie verifies typed extraction from wrapped errors.
*/
func TestAsErrnie(t *testing.T) {
	Convey("Given a wrapped ErrnieError", t, func() {
		inner := E(NotFound, "missing", nil)
		outer := fmt.Errorf("wrap: %w", inner)

		Convey("When AsErrnie is called", func() {
			target, ok := AsErrnie(outer)

			Convey("Then it should extract the typed error", func() {
				So(ok, ShouldBeTrue)
				So(target, ShouldEqual, inner)
			})
		})
	})

	Convey("Given a standard error", t, func() {
		Convey("When AsErrnie is called", func() {
			target, ok := AsErrnie(errors.New("plain"))

			Convey("Then it should not match", func() {
				So(ok, ShouldBeFalse)
				So(target, ShouldBeNil)
			})
		})
	})
}

/*
TestIsKind verifies kind-based classification through wrapping.
*/
func TestIsKind(t *testing.T) {
	Convey("Given a wrapped ErrnieError", t, func() {
		err := fmt.Errorf("wrap: %w", E(Conflict, "duplicate", nil))

		Convey("When IsKind is called", func() {
			Convey("Then it should match the expected kind", func() {
				So(IsKind(err, Conflict), ShouldBeTrue)
				So(IsKind(err, NotFound), ShouldBeFalse)
			})
		})
	})
}

/*
TestIsValidation verifies validation error classification.
*/
func TestIsValidation(t *testing.T) {
	Convey("Given a validation ErrnieError", t, func() {
		err := E(Validation, "bad input", nil)

		Convey("When IsValidation is called", func() {
			Convey("Then it should match", func() {
				So(IsValidation(err), ShouldBeTrue)
				So(IsNotFound(err), ShouldBeFalse)
			})
		})
	})
}

/*
TestIsIO verifies IO error classification.
*/
func TestIsIO(t *testing.T) {
	Convey("Given an IO ErrnieError", t, func() {
		err := E(IO, "read failed", nil)

		Convey("When IsIO is called", func() {
			So(IsIO(err), ShouldBeTrue)
		})
	})
}

/*
TestIsNetwork verifies network error classification.
*/
func TestIsNetwork(t *testing.T) {
	Convey("Given a network ErrnieError", t, func() {
		err := E(Network, "dial failed", nil)

		Convey("When IsNetwork is called", func() {
			So(IsNetwork(err), ShouldBeTrue)
		})
	})
}

/*
TestIsHTTP verifies HTTP error classification.
*/
func TestIsHTTP(t *testing.T) {
	Convey("Given an HTTP ErrnieError", t, func() {
		err := E(HTTP, "bad response", nil)

		Convey("When IsHTTP is called", func() {
			So(IsHTTP(err), ShouldBeTrue)
		})
	})
}

/*
TestIsDatabase verifies database error classification.
*/
func TestIsDatabase(t *testing.T) {
	Convey("Given a database ErrnieError", t, func() {
		err := E(Database, "query failed", nil)

		Convey("When IsDatabase is called", func() {
			So(IsDatabase(err), ShouldBeTrue)
		})
	})
}

/*
TestIsUnauthorized verifies unauthorized error classification.
*/
func TestIsUnauthorized(t *testing.T) {
	Convey("Given an unauthorized ErrnieError", t, func() {
		err := E(Unauthorized, "login required", nil)

		Convey("When IsUnauthorized is called", func() {
			So(IsUnauthorized(err), ShouldBeTrue)
		})
	})
}

/*
TestIsForbidden verifies forbidden error classification.
*/
func TestIsForbidden(t *testing.T) {
	Convey("Given a forbidden ErrnieError", t, func() {
		err := E(Forbidden, "access denied", nil)

		Convey("When IsForbidden is called", func() {
			So(IsForbidden(err), ShouldBeTrue)
		})
	})
}

/*
TestIsNotFound verifies not-found error classification.
*/
func TestIsNotFound(t *testing.T) {
	Convey("Given a not-found ErrnieError", t, func() {
		err := E(NotFound, "missing", nil)

		Convey("When IsNotFound is called", func() {
			So(IsNotFound(err), ShouldBeTrue)
		})
	})
}

/*
TestIsConflict verifies conflict error classification.
*/
func TestIsConflict(t *testing.T) {
	Convey("Given a conflict ErrnieError", t, func() {
		err := E(Conflict, "duplicate", nil)

		Convey("When IsConflict is called", func() {
			So(IsConflict(err), ShouldBeTrue)
		})
	})
}

/*
TestIsTimeout verifies timeout error classification.
*/
func TestIsTimeout(t *testing.T) {
	Convey("Given a timeout ErrnieError", t, func() {
		err := E(Timeout, "deadline exceeded", context.DeadlineExceeded)

		Convey("When IsTimeout is called", func() {
			So(IsTimeout(err), ShouldBeTrue)
			So(IsContext(err), ShouldBeTrue)
		})
	})
}

/*
TestIsContext verifies context cancellation and deadline detection.
*/
func TestIsContext(t *testing.T) {
	Convey("Given a wrapped context.Canceled error", t, func() {
		err := E(Timeout, "cancelled", context.Canceled)

		Convey("When IsContext is called", func() {
			So(IsContext(err), ShouldBeTrue)
		})
	})

	Convey("Given a non-context error", t, func() {
		err := E(Validation, "bad", nil)

		Convey("When IsContext is called", func() {
			So(IsContext(err), ShouldBeFalse)
		})
	})
}

/*
TestKindString verifies stable kind names for logging.
*/
func TestKindString(t *testing.T) {
	Convey("Given each Kind constant", t, func() {
		cases := map[Kind]string{
			Unknown:      "unknown",
			Validation:   "validation",
			IO:           "io",
			Network:      "network",
			HTTP:         "http",
			Database:     "database",
			Unauthorized: "unauthorized",
			Forbidden:    "forbidden",
			NotFound:     "not_found",
			Conflict:     "conflict",
			Timeout:      "timeout",
		}

		Convey("When String is called", func() {
			Convey("Then it should return the expected name", func() {
				for kind, name := range cases {
					So(kind.String(), ShouldEqual, name)
				}
			})
		})
	})
}

/*
TestErrnieErrorWithDoes verifies ErrnieError integration with Does and Or.
*/
func TestErrnieErrorWithDoes(t *testing.T) {
	Convey("Given a function that returns an ErrnieError", t, func() {
		expected := E(NotFound, "user missing", nil)

		Convey("When Does and Or are used", func() {
			var handled *ErrnieError
			result := Does(func() (string, error) {
				return "", expected
			}).Or(func(err error) {
				handled, _ = AsErrnie(err)
			})

			Convey("Then Or should receive the typed error", func() {
				So(handled, ShouldEqual, expected)
				So(result.Err(), ShouldEqual, expected)
				So(IsNotFound(result.Err()), ShouldBeTrue)
			})
		})
	})
}

var (
	benchmarkErrnieSink      *ErrnieError
	benchmarkErrnieErrorSink error
	benchmarkErrnieBoolSink  bool
	benchmarkErrnieKindSink  Kind
	benchmarkStaticCause     = errors.New("cause")
)

/*
BenchmarkE measures ErrnieError construction with and without a cause.
*/
func BenchmarkE(b *testing.B) {
	b.Run("without cause", func(b *testing.B) {
		for range b.N {
			benchmarkErrnieSink = E(Validation, "invalid", nil)
		}
	})

	b.Run("with cause", func(b *testing.B) {
		for range b.N {
			benchmarkErrnieSink = E(IO, "read failed", benchmarkStaticCause)
		}
	})
}

/*
BenchmarkErrnieErrorOperation measures Operation chaining.
*/
func BenchmarkErrnieErrorOperation(b *testing.B) {
	err := E(NotFound, "missing", nil)

	b.ResetTimer()
	for range b.N {
		benchmarkErrnieSink = err.Operation("user.load")
	}
}

/*
BenchmarkErrnieErrorWith measures metadata attachment.
*/
func BenchmarkErrnieErrorWith(b *testing.B) {
	err := E(HTTP, "request failed", nil)

	b.Run("two fields", func(b *testing.B) {
		for range b.N {
			benchmarkErrnieSink = err.With("status", 500, "url", "https://example.com")
		}
	})
}

/*
BenchmarkErrnieErrorError measures Error string formatting.
*/
func BenchmarkErrnieErrorError(b *testing.B) {
	err := E(Validation, "invalid email", nil).Operation("user.create")

	b.ResetTimer()
	for range b.N {
		benchmarkErrnieMessageSink = err.Error()
	}
}

/*
BenchmarkErrnieErrorUnwrap measures Unwrap on wrapped errors.
*/
func BenchmarkErrnieErrorUnwrap(b *testing.B) {
	err := E(IO, "read", benchmarkStaticCause)

	b.ResetTimer()
	for range b.N {
		benchmarkErrnieErrorSink = err.Unwrap()
	}
}

/*
BenchmarkCombine measures nil-safe error joining.
*/
func BenchmarkCombine(b *testing.B) {
	first := errors.New("first")
	second := errors.New("second")

	b.Run("all nil", func(b *testing.B) {
		for range b.N {
			benchmarkErrnieErrorSink = Combine(nil, nil)
		}
	})

	b.Run("single error", func(b *testing.B) {
		for range b.N {
			benchmarkErrnieErrorSink = Combine(nil, first, nil)
		}
	})

	b.Run("multiple errors", func(b *testing.B) {
		for range b.N {
			benchmarkErrnieErrorSink = Combine(first, second)
		}
	})
}

/*
BenchmarkAsErrnie measures typed extraction from wrapped errors.
*/
func BenchmarkAsErrnie(b *testing.B) {
	inner := E(NotFound, "missing", nil)
	outer := fmt.Errorf("wrap: %w", inner)

	b.ResetTimer()
	for range b.N {
		benchmarkErrnieSink, benchmarkErrnieBoolSink = AsErrnie(outer)
	}
}

/*
BenchmarkIsKind measures kind classification through wrapping.
*/
func BenchmarkIsKind(b *testing.B) {
	err := fmt.Errorf("wrap: %w", E(Conflict, "duplicate", nil))

	b.ResetTimer()
	for range b.N {
		benchmarkErrnieBoolSink = IsKind(err, Conflict)
	}
}

/*
BenchmarkIsNotFound measures the NotFound classification helper.
*/
func BenchmarkIsNotFound(b *testing.B) {
	err := fmt.Errorf("wrap: %w", E(NotFound, "missing", nil))

	b.ResetTimer()
	for range b.N {
		benchmarkErrnieBoolSink = IsNotFound(err)
	}
}

/*
BenchmarkIsContext measures context cancellation detection.
*/
func BenchmarkIsContext(b *testing.B) {
	err := E(Timeout, "cancelled", context.Canceled)

	b.ResetTimer()
	for range b.N {
		benchmarkErrnieBoolSink = IsContext(err)
	}
}

/*
BenchmarkKindString measures Kind name formatting.
*/
func BenchmarkKindString(b *testing.B) {
	for range b.N {
		benchmarkErrnieKindSink = NotFound
		benchmarkErrnieMessageSink = benchmarkErrnieKindSink.String()
	}
}

var benchmarkErrnieMessageSink string
