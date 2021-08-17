package errnie

import (
	"container/ring"
	"fmt"
	"os"

	"github.com/pterm/pterm"
)

/*
ErrCollector is the canonical implementation of the underlying
interface. Use this as a field in your object to handle errors.
*/
type Collector struct {
	advisor Advisor
	stack   *ring.Ring
	printer pterm.PrefixPrinter
}

/*
New acts as a constructor and returns an instance of Collector.
*/
func New(advisor *Advisor) *Collector {
	pterm.PrintDebugMessages = true

	// Explain later...
	finalAdvisor := NewAdvisor(RookieAdvisor{})

	if advisor != nil {
		finalAdvisor = *advisor
	}

	return &Collector{
		advisor: finalAdvisor,
		stack:   ring.New(20),
		printer: pterm.PrefixPrinter{},
	}
}

/*
Stack is a circular buffer that holds a certain amount of recent errors that
were collected. The basic concept is to have a real-time sample we can analyze
at any time to determine the health of our application. It is therefor recommended
to send one errnie down an entire call stack, and the longer you keep him alive,
the more useful he is across multiple domains in your code.
*/
func (collector *Collector) Stack(err error, errType ErrType) *Collector {
	if err == nil {
		return collector
	}

	// Add a new Error object to the circular buffer.
	collector.stack.Value = Error{
		err:     err,
		errType: errType,
	}

	// Proceed one unit down the buffer so we are ready
	// on the next call coming for us.
	collector.stack.Next()

	// Return a reference to ourselves so we get chainable methods.
	return collector
}

func (collector *Collector) StackOut(err error, errType ErrType) *Collector {
	if err == nil {
		return collector
	}

	collector.print(err, errType)
	return collector.Stack(err, errType)
}

func (collector *Collector) StackDump(err error, errType ErrType) *Collector {
	if err == nil {
		return collector
	}

	collector.print(err, errType)
	collector.Dump()
	return collector.Stack(err, errType)
}

func (collector Collector) ToSlice() []string {
	var out []string

	collector.stack.Do(func(p interface{}) {
		if p != nil {
			out = append(out, (p.(Error).err.Error()))
		}
	})

	return out
}

/*
Dump the error stack. This prints our the raw Error object data and is
not a method you want to use in any code that is not in debug mode.
*/
func (collector Collector) Dump() {
	// Iterate through the buffer and dumps each unit's contents.
	collector.stack.Do(func(p interface{}) {
		if p != nil {
			fmt.Println(p.(Error))
		}
	})

	os.Exit(1)
}

/*
OK takes an advisor interface and uses it to determine the calculated state of
the runtime environment. At any time call this to get an advice on whether or
not something drastic like a restart, fatal, or panic should be performed.
Errnie comes with a default (naive) implementation of an advisor, however the
most benefit can be obtained by providing something more custom to your situation.
*/
func (collector Collector) OK() bool {
	// We don't support dynamic yet.
	return collector.advisor.Static(*collector.stack)
}
