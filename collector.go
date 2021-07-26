package errnie

import (
	"os"

	"github.com/pterm/pterm"
	"github.com/spf13/viper"
)

/*
ErrCollector is the canonical implementation of the underlying
interface. Use this as a field in your object to handle errors.
*/
type Collector struct {
	stack   []Error
	printer pterm.PrefixPrinter
}

/*
New acts as a constructor and returns an instance of Collector.
*/
func New() *Collector {
	pterm.PrintDebugMessages = true

	return &Collector{
		printer: pterm.PrefixPrinter{},
	}
}

/*
Stack appends exactly one error to the in-memory stack of
errors we have collected. We also have to allocate on the first
call. So as long as we don't use the object yet, it will not
be allocated. Good for memory.
*/
func (collector *Collector) Stack(err error, errType ErrType) *Collector {
	if err != nil && collector.stack == nil {
		collector.stack = make([]Error, 0)
	}

	if err != nil {
		collector.stack = append(collector.stack, Error{
			err:     err,
			errType: errType,
		})

		collector.handle(err, errType)
	}

	return collector
}

/*
Dump the error stack.
*/
func (collector Collector) Dump() {
	for _, err := range collector.stack {
		collector.Debug(err.err.Error())
	}
}

/*
Info is a pretty printed info line.
*/
func (collector Collector) Info(msg string) {
	pterm.Info.Println(msg)
}

/*
Debug is a pretty printed info line.
*/
func (collector Collector) Debug(msg string) {
	// Don't print noise if we are not in debug mode.
	if !viper.GetBool("debug") {
		return
	}

	pterm.Debug.Println(msg)
}

/*
Bad indicates the current state of the system as calculated
over the entries on the stack.
*/
func (collector Collector) Bad() bool {
	for _, item := range collector.stack {
		if item.errType == PANIC || item.errType == FATAL {
			return true
		}
	}

	return false
}

/*
handle takes the last error added to the stack and rund a basic
validation on it to see if there is anything more we should do.
*/
func (collector Collector) handle(err error, errType ErrType) *Collector {
	switch errType {
	case PANIC:
		pterm.Fatal.Println(err)
		panic(err)
	case FATAL:
		pterm.Fatal.Println(err)
		os.Exit(1)
	case CRITICAL:
		pterm.Error.Println(err)
		os.Exit(1)
	case ERROR:
		pterm.Error.Println(err)
	case WARNING:
		pterm.Warning.Println(err)
	case INFO:
		pterm.Info.Println(err)
	case DEBUG:
		pterm.Debug.Println(err)
	}

	return &collector
}
