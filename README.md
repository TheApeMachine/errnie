# errnie
(Second) Go (at an) error handling package.

## Installation

```bash
go get -u github.com/TheApeMachine/errnie
```

## Basic usage
```go
package main

func main() {
	// Instantiate a new guard object and defer a Recue call.
	guard := NewGuard(errorHandler)
	defer guard.Rescue()()

	// Load the guard with an error.
	guard.Err := errors.New("errors rule, call-stacks drool!")

	// Or simply write some code that panics.
	panic("don't")
}

// All code above leads to here.
func errorHandler() {
	// Do some...
}
```

## Usage in a type

```go
package myaveragepackage

type JustMy struct {
	guard *errnie.Guard
}

func NewJustMy() *JustMy {
	jm := JustMy{}
	jm.guard = errnie.NewGuard(jm.handleError)
}

func (jm *JustMy) MethodMan() {
	defer jm.Rescue()()

	// Make sure we do not have an error and call Check.
	// This will not short-circuit the method.
	jm.guard.Err := nil
	jm.guard.Check()

	// Where the magic happens... This simulates an unhandled
	// panic, and will call the error handler.
	val := 42 / 0
}

func (jm *JustMy) handleError() {
	fmt.Println(jm.guard.Err)
}
```
