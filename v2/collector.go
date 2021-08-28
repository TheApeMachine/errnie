package errnie

import (
	"container/ring"
)

/*
Collector saves Errors in a ring buffer.
*/
type Collector struct {
	ringBuffer *ring.Ring
}

/*
NewCollector sets up the ring buffer we will collect our errors in.
*/
func NewCollector(ringSize int) *Collector {
	return &Collector{
		ringBuffer: ring.New(ringSize),
	}
}

/*
Add an error to the Collector's ring buffer and report OK if no errors.
*/
func (collector *Collector) Add(errs []interface{}, errType ErrType) bool {
	real := getRealErrors(errs)

	for _, err := range real {
		collector.ringBuffer.Next().Value = Error{
			err:     err,
			errType: errType,
		}
	}

	return len(real) != 0
}

/*
Dump returns all the errors currently present in the ring buffer.
*/
func (collector *Collector) Dump() []Error {
	var errs []Error

	collector.ringBuffer.Do(func(err interface{}) {
		if err != nil {
			errs = append(errs, err.(Error))
		}
	})

	return errs
}
