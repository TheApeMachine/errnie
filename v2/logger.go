package errnie

import (
	"github.com/pterm/pterm"
)

/*
Logger is a custom logger which allows a consistent interface to
multiple output channels.
*/
type Logger struct {
	channels []LogChannel
}

func NewLogger(channels ...LogChannel) *Logger {
	pterm.PrintDebugMessages = true

	return &Logger{
		channels: channels,
	}
}

func (logger *Logger) Debug(msgs ...interface{}) bool {
	// We always return the answer to the question: are we ok?
	// Thus we beginning in a happy state here.
	state := true

	for _, channel := range logger.channels {
		state = channel.Debug(msgs...)
	}

	// TODO: This is far from a flawless answer.
	return state
}

func (logger *Logger) Send(msgs ...interface{}) bool {
	if logger == nil {
		return true
	}

	logLevel := DEBUG

	for _, channel := range logger.channels {
		switch logLevel {
		case PANIC:
			return channel.Panic(msgs...)
		case FATAL:
			return channel.Fatal(msgs...)
		case CRITICAL:
			return channel.Critical(msgs...)
		case ERROR:
			return channel.Error(msgs...)
		case WARNING:
			return channel.Warning(msgs...)
		case INFO:
			return channel.Info(msgs...)
		case DEBUG:
			return channel.Debug(msgs...)
		default:
			return channel.Debug(msgs...)
		}
	}

	return true
}
