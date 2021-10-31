package errnie

import (
	"runtime"

	"github.com/pterm/pterm"
)

type LogChannel interface {
	Panic(msgs ...interface{}) bool
	Fatal(msgs ...interface{}) bool
	Critical(msgs ...interface{}) bool
	Error(msgs ...interface{}) bool
	Info(msgs ...interface{}) bool
	Warning(msgs ...interface{}) bool
	Debug(msgs ...interface{}) bool
}

/*
ConsoleLogger is a canonical implementation of the LogChannel interface,
and provides the basic local terminal output for log messages.
*/
type ConsoleLogger struct{}

func NewConsoleLogger() LogChannel {
	return ConsoleLogger{}
}

func (logChannel ConsoleLogger) Panic(msgs ...interface{}) bool {
	if len(msgs) == 0 || msgs[0] == nil {
		return true
	}

	pterm.Fatal.Println(msgs...)
	return false
}

func (logChannel ConsoleLogger) Fatal(msgs ...interface{}) bool {
	if len(msgs) == 0 || msgs[0] == nil {
		return true
	}

	pterm.Fatal.Println(msgs...)
	return false
}

func (logChannel ConsoleLogger) Critical(msgs ...interface{}) bool {
	if len(msgs) == 0 || msgs[0] == nil {
		return true
	}

	pterm.Error.Println(msgs...)
	return false
}

func (logChannel ConsoleLogger) Error(msgs ...interface{}) bool {
	if len(msgs) == 0 || msgs[0] == nil {
		return true
	}

	pc, fn, line, _ := runtime.Caller(1)

	pterm.Error.Println(runtime.FuncForPC(pc).Name(), fn, line, msgs)
	return false
}

func (logChannel ConsoleLogger) Info(msgs ...interface{}) bool {
	if len(msgs) == 0 || msgs[0] == nil {
		return true
	}

	pterm.Info.Println(msgs...)
	return false
}

func (logChannel ConsoleLogger) Warning(msgs ...interface{}) bool {
	if len(msgs) == 0 || msgs[0] == nil {
		return true
	}

	pterm.Warning.Println(msgs...)
	return false
}

/*
Debug shows a muted colored message in the terminal when `debug: true`
in the host program's config file.
*/
func (logChannel ConsoleLogger) Debug(msgs ...interface{}) bool {
	if len(msgs) == 0 || msgs[0] == nil {
		// No messages, no errors, so `we are ok`.
		return true
	}

	// We went passed the guard above, that means we have messages that
	// potentially could be errors. It is up to the user to decide which
	// values to check for, but to be safe we say `we are not ok`.
	pterm.Debug.Println(msgs...)
	return false
}
