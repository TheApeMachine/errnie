package errnie

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

type Error struct {
	err     error
	errType ErrType
}

func (wrapper Error) Error() error {
	return wrapper.err
}
