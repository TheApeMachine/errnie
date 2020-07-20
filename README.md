# errnie
(Second) Go (at an) error handling package.

## Basic usage
```go
package main

func main() {
	// Instantiate a new guard object and defer a Recue call.
	guard := NewGuard(error_handler)
	defer guard.Rescue()

  // Load the guard with an error.
	guard.err := errors.New("errors rule, call-stacks drool!")

	// Or simply write some code that panics.
	panic("don't")
}

// All code above leads to here.
func error_handler() {
	// Do some...
}
```
