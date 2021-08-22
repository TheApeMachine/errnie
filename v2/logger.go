package errnie

import "github.com/pterm/pterm"

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

func (logger *Logger) Send(logLevel ErrType, msgs ...interface{}) bool {
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
		}
	}

	return false
}
