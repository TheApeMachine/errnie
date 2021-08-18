package errnie

import (
	"bytes"
	"container/ring"
)

/*
State defines an enumerable type of 4 arbitrary values of degradation.
Use them however suits you.
*/
type State uint

const (
	OK State = iota
	NO
	BAD
	DEAD
)

/*
Advisor describes an interface that contains two methods of analyzing a
buffer of error values.
Static analysis is a fast and simple way to make a determination of what in
most cases could be fairly described as a "count".
Dynamic analysis is a method that takes in a channel where it feeds on
additional metadata to get a more refined view of program state.
It takes a bytes.Buffer to keep things "generic" so just be sure what you are
sending it and hard cast to it, or be a barbarian and use reflection.
Remember the calling function right before the advisor though, and that you are
basically always looking for a boolean answer to "Are we OK()?".
*/
type Advisor interface {
	Static(ring.Ring) bool
	Dynamic(<-chan bytes.Buffer) State
}

/*
RookieAdvisor is the built in and most basic implementation of the concept.
*/
type RookieAdvisor struct {
}

/*
NewAdvisor... I'd hate to call it a factory, but it sure seems like one.
*/
func NewAdvisor(advisorType *Advisor) *Advisor {
	return advisorType
}

/*
Static is the entry point for the fast and loose method of determining state.
*/
func (advisor RookieAdvisor) Static(ringBuffer ring.Ring) bool {
	yc := 0
	nc := 0

	ringBuffer.Do(func(p interface{}) {
		if advisor.isInTypeList([]ErrType{PANIC, FATAL, CRITICAL}, p.(Error).errType) {
			nc++
		} else {
			yc++
		}
	})

	return yc > nc
}

/*
Dynamic takes a bytes.Buffer channel so we can send it metadata. The call stack
would be an idea for instance. Dynamic advice will also be twice as broad in
value scope, which allows for additional complexity.
*/
func (advisor RookieAdvisor) Dynamic(<-chan bytes.Buffer) State {
	return OK
}

func (advisor RookieAdvisor) isInTypeList(list []ErrType, item ErrType) bool {
	for k := range list {
		if list[k] == item {
			return true
		}
	}

	return false
}
