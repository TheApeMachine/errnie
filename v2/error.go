package errnie

// ErrType defines an enumerable type.
type ErrType uint

const (
	PANIC ErrType = iota
	FATAL
	CRITICAL
	ERROR
	WARNING
	INFO
	DEBUG
)

/*
Error wraps Go's built in error type to extend its functionality with a
severity level.
*/
type Error struct {
	err     error
	errType ErrType
}

/*
ToString outputs the error message as it sits in the wrapperd Go error.
*/
func (wrapper Error) ToString() string {
	return wrapper.err.Error()
}

func getRealErrors(errs []interface{}) []error {
	var real []error

	if len(errs) == 0 {
		return real
	}

	for _, err := range errs {
		if err == nil {
			continue
		}

		real = append(real, err.(error))
	}

	return real
}
