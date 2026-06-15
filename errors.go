package errnie

import (
	"context"
	"errors"
)

/*
Kind classifies an ErrnieError using domain semantics that translate cleanly
across REST, gRPC, databases, queues, and filesystems. Prefer NotFound over
HTTP-specific codes; map transport status at the boundary layer.
*/
type Kind error

var (
	Unknown      Kind = errors.New("unknown")
	Validation   Kind = errors.New("validation")
	IO           Kind = errors.New("io")
	Network      Kind = errors.New("network")
	HTTP         Kind = errors.New("http")
	Database     Kind = errors.New("database")
	Unauthorized Kind = errors.New("unauthorized")
	Forbidden    Kind = errors.New("forbidden")
	NotFound     Kind = errors.New("not_found")
	Conflict     Kind = errors.New("conflict")
	Timeout      Kind = errors.New("timeout")
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

/*
IsValidation reports whether err is a validation-class ErrnieError.
*/
func IsValidation(err error) bool {
	return IsKind(err, Validation)
}

/*
IsIO reports whether err is an IO-class ErrnieError.
*/
func IsIO(err error) bool {
	return IsKind(err, IO)
}

/*
IsNetwork reports whether err is a network-class ErrnieError.
*/
func IsNetwork(err error) bool {
	return IsKind(err, Network)
}

/*
IsHTTP reports whether err is an HTTP-class ErrnieError.
*/
func IsHTTP(err error) bool {
	return IsKind(err, HTTP)
}

/*
IsDatabase reports whether err is a database-class ErrnieError.
*/
func IsDatabase(err error) bool {
	return IsKind(err, Database)
}

/*
IsUnauthorized reports whether err is an unauthorized-class ErrnieError.
*/
func IsUnauthorized(err error) bool {
	return IsKind(err, Unauthorized)
}

/*
IsForbidden reports whether err is a forbidden-class ErrnieError.
*/
func IsForbidden(err error) bool {
	return IsKind(err, Forbidden)
}

/*
IsNotFound reports whether err is a not-found-class ErrnieError.
*/
func IsNotFound(err error) bool {
	return IsKind(err, NotFound)
}

/*
IsConflict reports whether err is a conflict-class ErrnieError.
*/
func IsConflict(err error) bool {
	return IsKind(err, Conflict)
}

/*
IsTimeout reports whether err is a timeout-class ErrnieError.
*/
func IsTimeout(err error) bool {
	return IsKind(err, Timeout)
}

/*
IsContext reports whether err is context.Canceled or context.DeadlineExceeded,
including when wrapped inside an ErrnieError or errors.Join result.
*/
func IsContext(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
