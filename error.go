package errnie

type Error struct {
	Message string
	Detail  interface{}
}

func (e Error) Error() string {
	return e.Message
}
