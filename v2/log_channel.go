package errnie

import "github.com/pterm/pterm"

type LogChannel interface {
	Panic(msgs ...interface{})
	Fatal(msgs ...interface{})
	Critical(msgs ...interface{})
	Error(msgs ...interface{})
	Info(msgs ...interface{})
	Warning(msgs ...interface{})
	Debug(msgs ...interface{})
}

/*
ConsoleLogger is a canonical implementation of the LogChannel interface,
and provides the basic local terminal output for log messages.
*/
type ConsoleLogger struct{}

func NewConsoleLogger() LogChannel {
	return ConsoleLogger{}
}

func (logChannel ConsoleLogger) Panic(msgs ...interface{}) {
	if len(msgs) == 0 {
		return
	}

	pterm.Fatal.Println(msgs...)
}

func (logChannel ConsoleLogger) Fatal(msgs ...interface{}) {
	if len(msgs) == 0 {
		return
	}

	pterm.Fatal.Println(msgs...)
}

func (logChannel ConsoleLogger) Critical(msgs ...interface{}) {
	if len(msgs) == 0 {
		return
	}

	pterm.Error.Println(msgs...)
}

func (logChannel ConsoleLogger) Error(msgs ...interface{}) {
	if len(msgs) == 0 {
		return
	}

	pterm.Error.Println(msgs...)
}

func (logChannel ConsoleLogger) Info(msgs ...interface{}) {
	if len(msgs) == 0 {
		return
	}

	pterm.Info.Println(msgs...)
}
func (logChannel ConsoleLogger) Warning(msgs ...interface{}) {
	if len(msgs) == 0 {
		return
	}

	pterm.Warning.Println(msgs...)
}

func (logChannel ConsoleLogger) Debug(msgs ...interface{}) {
	if len(msgs) == 0 {
		return
	}

	pterm.Debug.Println(msgs...)
}
