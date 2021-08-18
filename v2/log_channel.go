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
	pterm.PrintDebugMessages = true
	return ConsoleLogger{}
}

func (logChannel ConsoleLogger) Panic(msgs ...interface{})    { pterm.Fatal.Println(msgs) }
func (logChannel ConsoleLogger) Fatal(msgs ...interface{})    { pterm.Fatal.Println(msgs) }
func (logChannel ConsoleLogger) Critical(msgs ...interface{}) { pterm.Error.Println(msgs) }
func (logChannel ConsoleLogger) Error(msgs ...interface{})    { pterm.Error.Println(msgs) }
func (logChannel ConsoleLogger) Info(msgs ...interface{})     { pterm.Info.Println(msgs) }
func (logChannel ConsoleLogger) Warning(msgs ...interface{})  { pterm.Warning.Println(msgs) }
func (logChannel ConsoleLogger) Debug(msgs ...interface{})    { pterm.Debug.Println(msgs) }
