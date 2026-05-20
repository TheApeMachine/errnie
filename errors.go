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
type Kind uint8

const (
	Unknown Kind = iota
	Validation
	IO
	Network
	HTTP
	Database
	Unauthorized
	Forbidden
	NotFound
	Conflict
	Timeout
)

/*
String returns the stable name of a Kind for logging and display.
*/
func (kind Kind) String() string {
	switch kind {
	case Validation:
		return "validation"
	case IO:
		return "io"
	case Network:
		return "network"
	case HTTP:
		return "http"
	case Database:
		return "database"
	case Unauthorized:
		return "unauthorized"
	case Forbidden:
		return "forbidden"
	case NotFound:
		return "not_found"
	case Conflict:
		return "conflict"
	case Timeout:
		return "timeout"
	default:
		return "unknown"
	}
}

/*
ErrnieError is the canonical typed error for errnie-aware projects. Use Kind
for semantic classification, Message for human-readable detail, Cause for
wrapping, and With for structured metadata. ErrnieError supports errors.Is
and errors.As through Unwrap; it does not capture stack traces.
*/
type ErrnieError struct {
	Kind    Kind
	Op      string
	Message string
	Cause   error
	fields  map[string]any
}

/*
E constructs an ErrnieError with the given kind, message, and optional wrapped
cause. Cause is preserved for errors.Is and errors.As, including
context.Canceled and context.DeadlineExceeded when passed as cause.
*/
func E(kind Kind, message string, cause error) *ErrnieError {
	return &ErrnieError{
		Kind:    kind,
		Message: message,
		Cause:   cause,
	}
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

	return err
}

/*
With attaches structured key/value metadata to the error. Keys must be strings;
values are stored in a map allocated on first use.
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

		if err.fields == nil {
			err.fields = make(map[string]any, len(keysAndValues)/2)
		}

		err.fields[key] = keysAndValues[index+1]
	}

	return err
}

/*
Fields returns the metadata map attached via With, or nil when none was added.
The returned map must be treated as read-only.
*/
func (err *ErrnieError) Fields() map[string]any {
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

	message := err.Message
	if message == "" && err.Cause != nil {
		message = err.Cause.Error()
	}

	if message == "" {
		message = err.Kind.String()
	}

	if err.Op != "" {
		return err.Op + ": " + message
	}

	return message
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
Combine joins non-nil errors using errors.Join. Returns nil when every error
is nil. Use for cleanup, shutdown, and concurrent work that may fail in
multiple places at once.
*/
func Combine(errs ...error) error {
	return errors.Join(errs...)
}

/*
AsErrnie reports whether err matches or wraps an ErrnieError.
*/
func AsErrnie(err error) (*ErrnieError, bool) {
	var target *ErrnieError

	if errors.As(err, &target) {
		return target, true
	}

	return nil, false
}

/*
IsKind reports whether err matches or wraps an ErrnieError with the given kind.
*/
func IsKind(err error, kind Kind) bool {
	target, ok := AsErrnie(err)

	return ok && target.Kind == kind
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
