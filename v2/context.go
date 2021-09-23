package errnie

var ambctx *AmbientContext

func init() {
	ambctx = New()
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
	log     *Logger
	trace   *Tracer
	ERR     error
	OK      bool
}

/*
New gives us back a reference to the instance, so we should be able to call
the package anywhere we want in our host code.
*/
func New() *AmbientContext {
	ambient := new(AmbientContext)
	ambient.collect = NewCollector(20)
	ambient.trace = NewTracer(true)
	ambient.log = NewLogger(&ConsoleLogger{})
	ambient.ERR = nil
	ambient.OK = true
	return ambctx
}

func Handles(errs ...interface{}) { ambctx.Handles(errs...) }
func Add(errs ...interface{})     { ambctx.Add(errs...) }
func Log(errs ...interface{})     { ambctx.Log(errs...) }

/*
Handles the error in some semi-significant want so we don't have to think too
much about it and sets the err and ok values so we can do some nifty syntactical
sugar tricks upstream.
*/
func (ambient *AmbientContext) Handles(errs ...interface{}) {
	ambient.trace.Caller("\xF0\x9F\x90\x9E", errs)

	ambient.ERR = nil
	ambient.OK = true

	ambient.Add(errs...)
	ambient.Log(errs...)

	if !ambient.OK {
		ambient.trace.Caller("\xF0\x9F\x91\x8E", "BAD")
		return
	}

	ambient.trace.Caller("\xF0\x9F\x91\x8D", "OK")
}

func (ambient AmbientContext) Add(errs ...interface{}) bool {
	ambient.trace.Caller("\xF0\x9F\x90\x9E", errs)
	return ambient.collect.Add(errs)
}

func (ambient AmbientContext) Log(msgs ...interface{}) bool {
	ambient.trace.Caller("\xF0\x9F\x90\x9E", msgs)
	return ambient.log.Send(msgs...)
}

func (ambient AmbientContext) Dump() chan Error {
	return ambient.collect.Dump()
}
