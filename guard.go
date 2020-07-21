package errnie

import (
	"fmt"
	"log"
)

type Guard struct {
	Err     error
	handler func()
}

/*
NewGuard constructs a reference to an error handler.
*/
func NewGuard(handler func()) *Guard {
	return &Guard{
		handler: handler,
	}
}

func (guard *Guard) Check() {
	if guard.Err != nil {
		panic(guard.Err)
	}
}

/*
Rescue a method from errors and panics.
This method should be called at the top of another method as a deferred call.
*/
func (guard *Guard) Rescue() func() {
	return func() {
		if r := recover(); r != nil || guard.Err != nil {
			if guard.handler == nil {
				log.Println(fmt.Sprintf("%v:%v", r, guard.Err))
				return
			}

			guard.handler()
		}
	}
}
