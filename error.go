package errnie

// ErrType defines an enumerable type.
type ErrType uint

// Define the enumeration for ErrType.
const (
	DEBUG ErrType = iota
	INFO
	WARNING
	ERROR
	CRITICAL
	FATAL
	PANIC
)

/*
Error wraps Go's built in error type to extend its functionality with a
severity level. This starts to blur the lines between errors and logs,
which is somewhat on purpose.
*/
type Error struct {
	err     error
	errType ErrType
}

/*
Error returns the canonical error value as given to us by Go's built in
error type. It is mainly used to be able to have a return value that is
compatible with idiomatic Go.
*/
func (wrapper Error) Error() error {
	return wrapper.err
}
