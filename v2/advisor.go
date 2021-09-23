package errnie

import (
	"bytes"
	"container/ring"

	"github.com/spf13/viper"
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

type Sampler struct {
	frequency int
	cycles    *ring.Ring
}

/*
Advisor describes an interface that contains two methods of analyzing a
buffer of error values. It provides an abstraction layer over a complicated
error state in a system.
*/
type Advisor interface {
	initialize(Sampler) Advisor

	Static(*ring.Ring) bool
	Dynamic(<-chan bytes.Buffer) State
}

/*
DefaultAdvisor is the built in and most basic implementation of the concept.
*/
type DefaultAdvisor struct {
	sampler struct {
		frequency int
		cycles    *ring.Ring
	}
}

/*
NewAdvisor... I'd hate to call it a factory, but it sure seems like one.
*/
func NewAdvisor(advisorType Advisor) Advisor {
	return advisorType.initialize(Sampler{
		frequency: viper.GetInt("errnie.advisors.default.frequency"),
		cycles:    ring.New(viper.GetInt("errnie.advisors.default.cycles")),
	})
}

func (advisor DefaultAdvisor) initialize(sampler Sampler) Advisor {
	advisor.sampler = sampler
	return advisor
}

/*
Static is the entry point for the fast and loose method of determining state.
*/
func (advisor DefaultAdvisor) Static(ringBuffer *ring.Ring) bool {
	yc := 0
	nc := 0

	ringBuffer.Do(func(p interface{}) {
		if advisor.isInTypeList(
			[]ErrType{PANIC, FATAL, CRITICAL}, p.(Error).ErrType,
		) {
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
value scope.
*/
func (advisor DefaultAdvisor) Dynamic(<-chan bytes.Buffer) State {
	return OK
}

func (advisor DefaultAdvisor) isInTypeList(list []ErrType, item ErrType) bool {
	for k := range list {
		if list[k] == item {
			return true
		}
	}

	return false
}
