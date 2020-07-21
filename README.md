# errnie
(Second) Go (at an) error handling package.

One day none of our functions or methods will ever crash again
and we will all be drinking half-pints on the beach.
Because we're not barbarics.

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

	// Call Check() to evaluate the error and determine whether
	// or not to short-circuit the function.
	guard.Check()

	// Or simply write some code that panics.
	panic("don't")
}

/*
All code above leads to here.
*/
func errorHandler() {
	// Do some...
}
```

## Usage in a type

```go
package myaveragepackage

/*
JustMy type, really really.
*/
type JustMy struct {
	guard *errnie.Guard
}

/*
NewJustMy returns a pointer to an instance of JustMy.
*/
func NewJustMy() *JustMy {
	jm := JustMy{}
	jm.guard = errnie.NewGuard(jm.handleError)
}

/*
MethodMan the most typical use-case, where you catch your errors inside the guard,
and call a check instead of the standard if statement pattern.
*/
func (jm *JustMy) MethodMan() {
	defer jm.Rescue()()
	// Optionally you can start with calling jm.guard.Check() here if you want to
	// implement the early short-circuit.

	// Make sure we do not have an error and call Check.
	// This will not short-circuit the method.
	jm.guard.Err := nil
	jm.guard.Check()

	// Where the magic happens... This simulates an unhandled
	// panic, and will call the error handler.
	val := 42 / 0
}

/*
handleError, because that's kinda what we're doing here. It's the law.
*/
func (jm *JustMy) handleError() {
	fmt.Println(jm.guard.Err)
}
```

## WIP: Usage with custom errors
```go
package robocop

type NoBulletsLeftError errnie.Error

/*
Weapon wraps the Auto 9 in a type.
I am relatively sure this is how they implemented those infinite magazines
in the 80s/90s action movies.
*/
type Weapon struct {
	bullets uint
	guard errnie.Guard
}

/*
NewWeapon returns a pointer to an instance of Weapon.
*/
func NewWeapon() *Weapon {
	w := Weapon{bullets: 50}
	w.guard = errnie.NewGuard(w.handleError)
}

/*
SprayAndPray. Look at something that moves, vaguely point in that direction
and call in infinite for loop.
*/
func (weapon *Weapon) SprayAndPray() {
	if weapon.bullets == 0 {
		weapon.guard.Err = NoBulletsLeftError
		weapon.guard.Check()
	}

	weapon.bullets--
}

/*
Reload simulates some sort of fault-tolerant method that recovers the state
of the type such that it can resume operation.
*/
func (weapon *Weapon) reload() {
	// Source: https://robocop.fandom.com/wiki/Auto_9
	weapon.bullets = 50
}

/*
handleError should recover the state of the type.
*/
func (weapon *Weapon) handleError {
	if weapon.guard.Err == NoBulletsLeftError {
		weapon.reload()
	}
}
```
