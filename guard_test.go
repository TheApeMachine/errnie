package errnie

import (
	"errors"
	"testing"
)

type Check struct {
	handled bool
}

func (check *Check) Validate(t *testing.T) {
	if !check.handled {
		t.Error("expected errors to be handled")
	}
}

func (check *Check) mockHandler() {
	check.handled = true
}

func setup() (*Check, *Guard) {
	check := &Check{handled: false}
	return check, NewGuard(check.mockHandler)
}

func TestCheck(t *testing.T) {
	check, guard := setup()
	defer check.Validate(t)
	defer guard.Rescue()()

	guard.err = errors.New("errors rule, call stacks drool")
	guard.Check()
}

func TestPanic(t *testing.T) {
	check, guard := setup()
	defer check.Validate(t)
	defer guard.Rescue()()

	guard.err = errors.New("errors rule, call stacks drool")
	panic(guard.err)
}

func TestCheckPanic(t *testing.T) {
	check, guard := setup()
	defer check.Validate(t)
	defer guard.Rescue()()

	guard.err = errors.New("errors rule, call stacks drool")
	guard.Check()
	panic(guard.err)
}
