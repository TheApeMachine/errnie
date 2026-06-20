package errnie

import (
	"context"
	"errors"
	"fmt"
	"io"
)

/*
Kind classifies an ErrnieError using domain semantics that translate cleanly
across REST, gRPC, databases, queues, and filesystems. Prefer NotFound over
HTTP-specific codes; map transport status at the boundary layer.
*/
type Kind error

var (
	Unknown              Kind = errors.New("unknown")
	Validation           Kind = errors.New("validation")
	IO                   Kind = errors.New("io")
	EOF                  Kind = io.EOF
	Canceled             Kind = context.Canceled
	DeadlineExceeded     Kind = context.DeadlineExceeded
	BadRequest           Kind = errors.New("bad_request")
	Unauthorized         Kind = errors.New("unauthorized")
	Forbidden            Kind = errors.New("forbidden")
	NotFound             Kind = errors.New("not_found")
	MethodNotAllowed     Kind = errors.New("method_not_allowed")
	NotAcceptable        Kind = errors.New("not_acceptable")
	Timeout              Kind = errors.New("timeout")
	Conflict             Kind = errors.New("conflict")
	PreconditionFailed   Kind = errors.New("precondition_failed")
	UnsupportedMedia     Kind = errors.New("unsupported_media_type")
	ExpectationFailed    Kind = errors.New("expectation_failed")
	UnprocessableContent Kind = errors.New("unprocessable_content")
	TooManyRequests      Kind = errors.New("too_many_requests")
	Internal             Kind = errors.New("internal")
	NotImplemented       Kind = errors.New("not_implemented")
	BadGateway           Kind = errors.New("bad_gateway")
	ServiceUnavailable   Kind = errors.New("service_unavailable")
)

/*
ErrnieError is the canonical typed error for errnie-aware projects. Use Kind
for semantic classification, Message for human-readable detail, Cause for
wrapping, and With for structured metadata. ErrnieError supports errors.Is
and errors.As through Unwrap; it does not capture stack traces.

Construct and enrich an ErrnieError (E, Operation, With) before sharing it
across goroutines. Mutation after concurrent use is not safe.
*/
type ErrnieError struct {
	Kind      Kind
	Op        string
	Message   string
	Cause     error
	Timestamp int64
	fields    []any
	rendered  string
}

/*
E constructs an ErrnieError with the given kind, message, and optional wrapped
cause. Cause is preserved for errors.Is and errors.As, including
context.Canceled and context.DeadlineExceeded when passed as cause.
*/
func Err(kind Kind, message string, cause error) *ErrnieError {
	return &ErrnieError{
		Kind:    kind,
		Message: message,
		Cause:   cause,
	}
}

/*
Guard returns nil if the cause is nil, otherwise it returns
an ErrnieError with the given kind, message, and cause.
*/
func Guard(kind Kind, message string, cause error) error {
	if cause == nil {
		return nil
	}

	return Err(kind, message, cause)
}

func (err *ErrnieError) WithTimestamp(timestamp int64) *ErrnieError {
	if err == nil {
		return nil
	}

	err.Timestamp = timestamp
	return err
}

/*
Operation records the logical operation name (for example "user.load") and
returns the same error for chaining.
*/
func (err *ErrnieError) Operation(name string) *ErrnieError {
	if err == nil {
		return nil
	}

	err.Op = name
	err.rendered = ""

	return err
}

/*
With attaches structured key/value metadata to the error. Keys must be strings;
values are stored in a flat alternating slice to avoid map allocation and hashing.
*/
func (err *ErrnieError) With(keysAndValues ...any) *ErrnieError {
	if err == nil || len(keysAndValues) == 0 {
		return err
	}

	for index := 0; index+1 < len(keysAndValues); index += 2 {
		key, ok := keysAndValues[index].(string)
		if !ok {
			continue
		}

		value := keysAndValues[index+1]
		replaced := false

		for fieldIndex := 0; fieldIndex+1 < len(err.fields); fieldIndex += 2 {
			existingKey, ok := err.fields[fieldIndex].(string)
			if ok && existingKey == key {
				err.fields[fieldIndex+1] = value
				replaced = true

				break
			}
		}

		if !replaced {
			err.fields = append(err.fields, key, value)
		}
	}

	return err
}

/*
Fields returns the metadata slice attached via With as alternating key/value
pairs, or nil when none was added. The returned slice must be treated as
read-only.
*/
func (err *ErrnieError) Fields() []any {
	if err == nil {
		return nil
	}

	return err.fields
}

/*
Error implements the error interface. When Op is set it prefixes the message.
*/
func (err *ErrnieError) Error() string {
	if err == nil {
		return ""
	}

	if err.rendered != "" {
		return err.rendered
	}

	message := err.Message

	if message == "" && err.Cause != nil {
		message = err.Cause.Error()
	}

	if message == "" {
		message = err.Kind.Error()
	}

	if err.Op != "" {
		err.rendered = err.Op + ": " + message
	} else {
		err.rendered = message
	}

	fields := err.Fields()

	for index := 0; index+1 < len(fields); index += 2 {
		err.rendered += fmt.Sprintf(" %s=%v", fields[index], fields[index+1])
	}

	return err.rendered
}

/*
Unwrap returns the wrapped cause for errors.Is and errors.As traversal.
*/
func (err *ErrnieError) Unwrap() error {
	if err == nil {
		return nil
	}

	return err.Cause
}

/*
joinedPair joins exactly two errors without errors.Join slice allocation.
It implements Unwrap() []error for errors.Is and errors.As traversal.
*/
type joinedPair struct {
	unwrapped [2]error
}

func joinPair(first, second error) joinedPair {
	return joinedPair{unwrapped: [2]error{first, second}}
}

func (joined joinedPair) Error() string {
	return joined.unwrapped[0].Error() + "\n" + joined.unwrapped[1].Error()
}

func (joined joinedPair) Unwrap() []error {
	return joined.unwrapped[:]
}

/*
Combine joins non-nil errors. Returns nil when every error is nil. The common
two-error cleanup path uses a specialized joinPair; three or more fall back to
errors.Join.
*/
func Combine(errs ...error) error {
	var first, second error
	count := 0
	var extra []error

	for _, err := range errs {
		if err == nil {
			continue
		}

		switch count {
		case 0:
			first = err
		case 1:
			second = err
		default:
			if extra == nil {
				extra = make([]error, 0, len(errs))
				extra = append(extra, first, second)
			}

			extra = append(extra, err)
		}

		count++
	}

	switch count {
	case 0:
		return nil
	case 1:
		return first
	case 2:
		return joinPair(first, second)
	default:
		return errors.Join(extra...)
	}
}

/*
AsErrnie reports whether err matches or wraps an ErrnieError.
*/
func AsErrnie(err error) (*ErrnieError, bool) {
	return asErrnieInChain(err)
}

/*
IsKind reports whether err matches or wraps an ErrnieError with the given kind.
*/
func IsKind(err error, kind Kind) bool {
	for err != nil {
		if target, ok := err.(*ErrnieError); ok {
			return target.Kind == kind
		}

		if joined, ok := err.(interface{ Unwrap() []error }); ok {
			for _, child := range joined.Unwrap() {
				if IsKind(child, kind) {
					return true
				}
			}

			return false
		}

		err = errors.Unwrap(err)
	}

	return false
}

/*
asErrnieInChain walks an error chain without errors.As reflection.
*/
func asErrnieInChain(err error) (*ErrnieError, bool) {
	for err != nil {
		if target, ok := err.(*ErrnieError); ok {
			return target, true
		}

		if joined, ok := err.(interface{ Unwrap() []error }); ok {
			for _, child := range joined.Unwrap() {
				if target, ok := asErrnieInChain(child); ok {
					return target, true
				}
			}

			return nil, false
		}

		err = errors.Unwrap(err)
	}

	return nil, false
}

/* IsUnknown reports whether err is an unknown-class ErrnieError. */
func IsUnknown(err error) bool { return IsKind(err, Unknown) }

/* IsValidation reports whether err is a validation-class ErrnieError. */
func IsValidation(err error) bool { return IsKind(err, Validation) }

/* IsIO reports whether err is an IO-class ErrnieError. */
func IsIO(err error) bool { return IsKind(err, IO) }

/* IsEOF reports whether err is an EOF-class ErrnieError. */
func IsEOF(err error) bool { return IsKind(err, EOF) }

/* IsCanceled reports whether err is a canceled-class ErrnieError. */
func IsCanceled(err error) bool { return IsKind(err, Canceled) }

/* IsDeadlineExceeded reports whether err is a deadline-exceeded-class ErrnieError. */
func IsDeadlineExceeded(err error) bool { return IsKind(err, DeadlineExceeded) }

/* IsBadRequest reports whether err is a bad-request-class ErrnieError. */
func IsBadRequest(err error) bool { return IsKind(err, BadRequest) }

/* IsUnauthorized reports whether err is an unauthorized-class ErrnieError. */
func IsUnauthorized(err error) bool { return IsKind(err, Unauthorized) }

/* IsForbidden reports whether err is a forbidden-class ErrnieError. */
func IsForbidden(err error) bool { return IsKind(err, Forbidden) }

/* IsNotFound reports whether err is a not-found-class ErrnieError. */
func IsNotFound(err error) bool { return IsKind(err, NotFound) }

/* IsMethodNotAllowed reports whether err is a method-not-allowed-class ErrnieError. */
func IsMethodNotAllowed(err error) bool { return IsKind(err, MethodNotAllowed) }

/* IsNotAcceptable reports whether err is a not-acceptable-class ErrnieError. */
func IsNotAcceptable(err error) bool { return IsKind(err, NotAcceptable) }

/* IsTimeout reports whether err is a timeout-class ErrnieError. */
func IsTimeout(err error) bool { return IsKind(err, Timeout) }

/* IsConflict reports whether err is a conflict-class ErrnieError. */
func IsConflict(err error) bool { return IsKind(err, Conflict) }

/* IsPreconditionFailed reports whether err is a precondition-failed-class ErrnieError. */
func IsPreconditionFailed(err error) bool { return IsKind(err, PreconditionFailed) }

/* IsUnsupportedMedia reports whether err is an unsupported-media-class ErrnieError. */
func IsUnsupportedMedia(err error) bool { return IsKind(err, UnsupportedMedia) }

/* IsExpectationFailed reports whether err is an expectation-failed-class ErrnieError. */
func IsExpectationFailed(err error) bool { return IsKind(err, ExpectationFailed) }

/* IsUnprocessableContent reports whether err is an unprocessable-content-class ErrnieError. */
func IsUnprocessableContent(err error) bool { return IsKind(err, UnprocessableContent) }

/* IsTooManyRequests reports whether err is a too-many-requests-class ErrnieError. */
func IsTooManyRequests(err error) bool { return IsKind(err, TooManyRequests) }

/* IsInternal reports whether err is an internal-class ErrnieError. */
func IsInternal(err error) bool { return IsKind(err, Internal) }

/* IsNotImplemented reports whether err is a not-implemented-class ErrnieError. */
func IsNotImplemented(err error) bool { return IsKind(err, NotImplemented) }

/* IsBadGateway reports whether err is a bad-gateway-class ErrnieError. */
func IsBadGateway(err error) bool { return IsKind(err, BadGateway) }

/* IsServiceUnavailable reports whether err is a service-unavailable-class ErrnieError. */
func IsServiceUnavailable(err error) bool { return IsKind(err, ServiceUnavailable) }

/*
IsContext reports whether err is context.Canceled or context.DeadlineExceeded,
including when wrapped inside an ErrnieError or errors.Join result.
*/
func IsContext(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
