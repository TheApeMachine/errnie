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
func (collector *Collector) Add(errs []interface{}) bool {
	real := getRealErrors(errs)

	for _, err := range real {
		collector.ringBuffer.Value = Error{
			Err:     err,
			ErrType: DEBUG,
		}

		collector.ringBuffer.Next()
	}

	return len(real) == 0
}

/*
Dump returns all the errors currently present in the ring buffer.
*/
func (collector *Collector) Dump() chan Error {
	out := make(chan Error)

	go func() {
		defer close(out)
		ambctx.Log(DEBUG, "dumping errors...")
		collector.ringBuffer.Do(func(err interface{}) {
			if err != nil {
				ambctx.Log(DEBUG, err)
				out <- err.(Error)
			}
		})
	}()

	return out
}
