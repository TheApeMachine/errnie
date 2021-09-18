package errnie

import (
	"bufio"
	"os"
	"time"
)

type FileLogger struct{}

func NewFileLogger() LogChannel {
	return FileLogger{}
}

func (logChannel FileLogger) Panic(msgs ...interface{}) bool {
	if len(msgs) == 0 || msgs[0] == nil {
		return true
	}

	logChannel.writeLines("PANIC", msgs)
	return false
}

func (logChannel FileLogger) Fatal(msgs ...interface{}) bool {
	if len(msgs) == 0 || msgs[0] == nil {
		return true
	}

	logChannel.writeLines("FATAL", msgs)
	return false
}

func (logChannel FileLogger) Critical(msgs ...interface{}) bool {
	if len(msgs) == 0 || msgs[0] == nil {
		return true
	}

	logChannel.writeLines("CRITICAL", msgs)
	return false
}

func (logChannel FileLogger) Error(msgs ...interface{}) bool {
	if len(msgs) == 0 || msgs[0] == nil {
		return true
	}

	logChannel.writeLines("ERROR", msgs)
	return false
}

func (logChannel FileLogger) Warning(msgs ...interface{}) bool {
	if len(msgs) == 0 || msgs[0] == nil {
		return true
	}

	logChannel.writeLines("WARNING", msgs)
	return false
}

func (logChannel FileLogger) Info(msgs ...interface{}) bool {
	if len(msgs) == 0 || msgs[0] == nil {
		return true
	}

	logChannel.writeLines("INFO", msgs)
	return false
}

func (logChannel FileLogger) Debug(msgs ...interface{}) bool {
	if len(msgs) == 0 || msgs[0] == nil {
		return true
	}

	logChannel.writeLines("DEBUG", msgs)
	return false
}

func (logChannel FileLogger) writeLines(level string, msgs ...interface{}) {
	file, _ := os.OpenFile("errnie.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	datawriter := bufio.NewWriter(file)

	for _, msg := range msgs {
		_, _ = datawriter.WriteString(
			"[" + time.Now().String() + "] ::" + level + "::" + msg.(string) + "\n",
		)
	}
}
