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
New acts as a constructor and returns an reference pointer.
*/
func NewCollector(ringSize int) *Collector {
	return &Collector{
		ringBuffer: ring.New(ringSize),
	}
}

/*
Add an error to the Collector's ring buffer.
*/
func (collector *Collector) Add(err error, errType ErrType) *Collector {
	// Ignore nil errors.
	if err == nil {
		return
	}

	collector.ringBuffer.Value = Error{
		err:     err,
		errType: errType,
	}

	// Turn the ring buffer one unit, ready for the next error.
	collector.ringBuffer.Next()

	return collector
}

/*
Dump returns all the errors currently present in the ring buffer.
*/
func (collector *Collector) Dump() []Error {
	var errors []Error

	collector.ringBuffer.Do(func(err interface{}) {
		if err != nil {
			errors = append(errors, err.(Error))
		}
	})

	return errors
}
