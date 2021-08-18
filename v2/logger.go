package errnie

/*
Logger is a custom logger which allows a consistent interface to
multiple output channels.
*/
type Logger struct {
	channels []LogChannel
}

func NewLogger(channels ...LogChannel) *Logger {
	return &Logger{
		channels: channels,
	}
}

func (logger *Logger) Send(logLevel ErrType, msgs ...interface{}) {
	for _, channel := range logger.channels {
		switch logLevel {
		case PANIC:
			channel.Panic(msgs)
		case FATAL:
			channel.Fatal(msgs)
		case CRITICAL:
			channel.Critical(msgs)
		case ERROR:
			channel.Error(msgs)
		case WARNING:
			channel.Warning(msgs)
		case INFO:
			channel.Info(msgs)
		case DEBUG:
			channel.Debug(msgs)
		}
	}
}
