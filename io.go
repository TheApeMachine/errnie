package errnie

/*
Write implements the io.Writer interface.

It is a wrapper around the underlying file MultiWriter Write method.
*/
func (ctx *Context) Write(p []byte) (n int, err error) {
	return ctx.output.Write(p)
}

/*
Close implements the io.Closer interface.

It is a wrapper around the underlying file handle Close method.
*/
func (ctx *Context) Close() error {
	return Handles(ctx.fh.Close())
}
