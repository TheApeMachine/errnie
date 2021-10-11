package errnie

import (
	"fmt"
)

/*
ambctx is the internal `global` object for errnie that allows it to prvide itself
as an AmbientContext to any package that includes errnie. No initialization needed or
having to inject it over and over again as a dependency, however you will need to
provide it with a config it can use, which is easiest using viper.
*/
var ambctx *AmbientContext
var ctxguard *Guard
var Logs *Logger

/*
Since the init method runs before anything it ensures that we always have access to
our ambient error handler with a logging complex. Or was it the other way around?
*/
func init() {
	ambctx = New()

	// Experimental at the moment. This is a test to see whether setting up a guard at this
	// point would make the RECV (recover) OpCode work globally.
	ctxguard = NewGuard(nil)
}

/*
AmbientContext is the main wrapper for errnie and tries to keep the amount of
having to deal with this tool as minimal as possible. I wanted something that
I did not have to pass around all the time, cluttering up method signatures
and basically being a repetative hassle. I feel in Go's case, having your own
favorite way of error handling always at the ready can only be a good thing.
And yes, I got inspired (and shamelessly stole) this setup from Viper.
*/
type AmbientContext struct {
	collect *Collector
	Logs    *Logger
	trace   *Tracer
	ERR     error
	OK      bool
}

/*
New gives us back a reference to the instance, so we should be able to call
the package anywhere we want in our host code.
*/
func New() *AmbientContext {
	ambctx := new(AmbientContext)
	ambctx.collect = NewCollector(20)
	ambctx.trace = NewTracer(true)
	ambctx.Logs = NewLogger(&ConsoleLogger{})
	ambctx.ERR = nil
	ambctx.OK = true
	Logs = ambctx.Logs
	return ambctx
}

/*
Handles is a proxy method to facilitate shorter style syntax.
*/
func Handles(errs ...interface{}) *AmbientContext { return ambctx.Handles(errs...) }

/*
Add is a proxy method to facilitate shorter style syntax.
*/
func Add(errs ...interface{}) *AmbientContext { return ambctx.Add(errs...) }

/*
Debug is a proxy method to facilitate shorter style syntax.
<Deprecated: See proxy referenced implementation>
*/
func Log(errs ...interface{}) *AmbientContext { return ambctx.Log(errs...) }

/*
Handles the error in some semi-significant want so we don't have to think too
much about it and sets the err and ok values so we can do some nifty syntactical
sugar tricks upstream.
*/
func (ambctx *AmbientContext) Handles(errs ...interface{}) *AmbientContext {
	// We start with a positive outcome. As we are at this point not sure yet Whether
	// or not we actually have a real error, this would be the default outcome.
	ambctx.ERR = nil
	ambctx.OK = true

	// We defer the passed in values to the add and log methods described below,
	// which will also set the ERR and OK values when needed.
	ambctx.Add(errs...)
	ambctx.Log(errs...)

	// Return ourselves so we open up a chainable call setup.
	return ambctx
}

/*
Add the errors to the internal error collector, a ring buffer that will
keep them in memory so the can be used in combination with an Advisor,
together providing a mechanism to judge state (`health`) of a chain
od methods.
*/
func (ambctx *AmbientContext) Add(errs ...interface{}) *AmbientContext {
	ambctx.collect.Add(errs)
	return ambctx
}

/*
Log a list of errors or any other object you want to inspect. Good use-cases
for this method are info or debug lines. Maybe errors you don't care about
handling, but I would suggest using a no-op handler for that.
<Deprecated, use the new format: `errnie.Logs.Error(x)`> (Error(x) can be
replaced with any of errnie's known log levels.)
*/
func (ambctx *AmbientContext) Log(msgs ...interface{}) *AmbientContext {
	ambctx.OK = ambctx.Logs.Send(msgs...)
	ambctx.ERR = fmt.Errorf("%v", msgs)
	return ambctx
}

/*
Dump errnie's internal error collector out to the log receiver, which could be
anything from the local console to a log data store. This actually returns a channel
of errnie's internal Error type, which is a razor thin wrapper around Go's error
type, value, no, type... Errors are Weird (not values).
*/
func (ambctx *AmbientContext) Dump() chan Error {
	return ambctx.collect.Dump()
}
