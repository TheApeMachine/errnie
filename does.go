package errnie

/*
Result holds the outcome of a function executed through Does. The value and
error are kept together so callers can inspect or handle failures without
panicking, and chain optional side effects with Or.
*/
type Result[T any] struct {
	value T
	err   error
}

/*
Does runs fn immediately and wraps its return value and error in a Result.
Use Value and Err to read the outcome, or Or to run a handler when fn failed.

Example:

	result := Does(func() (string, error) {
		return fetchName(id)
	}).Or(func(err error) {
		Warn("fetch failed", "err", err)
	})

	if result.Err() != nil {
		return result.Err()
	}

	return use(result.Value())
*/
func Does[T any](fn func() (T, error)) Result[T] {
	value, err := fn()

	return Result[T]{
		value: value,
		err:   err,
	}
}

/*
Or invokes fn with the stored error when the result represents a failure.
When err is nil, fn is not called. The same Result is returned so Or can be
chained or followed by Value and Err without reassignment.
*/
func (result Result[T]) Or(fn func(error)) Result[T] {
	if result.err != nil {
		fn(result.err)
	}

	return result
}

/*
Value returns the value produced by the function passed to Does. When fn
returned a non-nil error, this is still the zero value fn returned alongside
that error; callers should check Err before using the value.
*/
func (result Result[T]) Value() T {
	return result.value
}

/*
Err returns the error produced by the function passed to Does, or nil when
fn succeeded.
*/
func (result Result[T]) Err() error {
	return result.err
}
