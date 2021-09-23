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
	Err     error
	ErrType ErrType
}

/*
ToString outputs the error message as it sits in the wrapperd Go error.
*/
func (wrapper Error) ToString() string {
	ambctx.trace.Caller("\xF0\x9F\x90\x9E")
	return wrapper.Err.Error()
}

func getRealErrors(errs []interface{}) []error {
	ambctx.trace.Caller("\xF0\x9F\x90\x9E", errs)
	var real []error

	if len(errs) == 0 {
		return real
	}

	for _, err := range errs {
		if err == nil {
			continue
		}

		switch err := err.(type) {
		case error:
			ambctx.trace.Caller("\xF0\x9F\x94\xA5", err)
			real = append(real, err)
		}
	}

	return real
}
