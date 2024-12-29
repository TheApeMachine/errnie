package errnie

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/acarl005/stripansi"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/davecgh/go-spew/spew"
	"github.com/spf13/viper"
)

var (
	logFile   *os.File
	logFileMu sync.Mutex

	logger = log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    true,
		CallerOffset:    1,
		ReportTimestamp: true,
		TimeFormat:      time.TimeOnly,
		Level:           log.DebugLevel,
	})
)

/*
Initialize logging system by configuring log styles, setting log levels,
and initializing log files if applicable.
*/
func InitLogger() {
	fmt.Printf("LOGFILE=%s\n", os.Getenv("LOGFILE"))
	fmt.Printf("NOCONSOLE=%s\n", os.Getenv("NOCONSOLE"))

	if os.Getenv("LOGFILE") == "true" {
		// Initialize the log file
		initLogFile()

		if logFile == nil {
			fmt.Println("WARNING: Log file initialization failed!")
		}
	}

	// Set log level based on configuration
	setLogLevel()

	if os.Getenv("LOGGOROUTINES") == "true" {
		// Periodic routine to print the number of active goroutines
		go func() {
			for range time.Tick(time.Second * 5) {
				logger.Debug("active goroutines", "count", runtime.NumGoroutine())
			}
		}()
	}
}

/*
Set the appropriate logging level from Viper configuration.
*/
func setLogLevel() {
	switch viper.GetString("loglevel") {
	case "trace", "debug":
		logger.SetLevel(log.DebugLevel)
	case "info":
		logger.SetLevel(log.InfoLevel)
	case "warn":
		logger.SetLevel(log.WarnLevel)
	case "error":
		logger.SetLevel(log.ErrorLevel)
	default:
		logger.SetLevel(log.DebugLevel)
	}
}

/*
Initialize the log file by creating or overwriting the log file.
Handles any errors during initialization gracefully.
*/
func initLogFile() {
	wd, err := os.Getwd()
	if err != nil {
		logger.Warn("Failed to get working directory", "error", err)
		return
	}

	logDir := filepath.Join(wd, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		logger.Warn("Failed to create log directory", "error", err)
		return
	}

	logFilePath := filepath.Join(logDir, "amsh.log")
	logFile, err = os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		logger.Warn("Failed to open log file", "error", err)
		return
	}
	logger.Debug("Log file successfully initialized", "path", logFilePath)
}

/*
Log a formatted message to the standard logger as well as to the log file.
*/
func Log(format string, v ...interface{}) {
	message := fmt.Sprintf(format, v...)
	if message == "" {
		return
	}
	writeToLog(message)
}

/*
Raw is a full decomposition of the object passed in.
It will print the object into the console using spew, and it will write the object to the logfile.
*/
func Raw(v ...interface{}) {
	formatted := spew.Sdump(v...)
	if os.Getenv("NOCONSOLE") != "true" {
		spew.Dump(v...)
	}

	writeToLog(formatted)
}

/*
Trace logs a trace message to the logger.
*/
func Trace(v ...interface{}) {
	if os.Getenv("NOCONSOLE") != "true" {
		logger.Debug(v[0], v[1:]...)
	}

	writeToLog(fmt.Sprintf("%v", v))
}

/*
Debug logs a debug message to the logger.
*/
func Debug(format string, v ...interface{}) {
	if os.Getenv("NOCONSOLE") != "true" {
		logger.Debug(fmt.Sprintf(format, v...))
	}

	writeToLog(fmt.Sprintf(format, v...))
}

/*
Info logs an info message to the logger.
*/
func Info(format string, v ...interface{}) {
	if os.Getenv("NOCONSOLE") != "true" {
		logger.Info(fmt.Sprintf(format, v...))
	}

	writeToLog(fmt.Sprintf(format, v...))
}

/*
Warn logs a warn message to the logger.
*/
func Warn(format string, v ...interface{}) {
	if os.Getenv("NOCONSOLE") != "true" {
		logger.Warn(fmt.Sprintf(format, v...))
	}

	writeToLog(fmt.Sprintf(format, v...))
}

/*
ErrorSafe logs a simple version of the error, used in SafeMust.
*/
func ErrorSafe(err error, v ...interface{}) {
	if err == nil {
		return
	}

	if os.Getenv("NOCONSOLE") != "true" {
		logger.Error(err.Error(), v...)
	}

	writeToLog(err.Error())

}

/*
Error logs the error and returns it, useful for inline error logging and returning.

Example usage:

	err := someFunction()
	if err != nil {
		return Error(err, "additional context")
	}
*/
func Error(err error, v ...interface{}) error {
	if err == nil {
		return nil
	}

	errMsg := err.Error()
	var parts []string
	parts = append(parts, errMsg)

	if os.Getenv("NOSTACK") != "true" {
		trace := getStackTrace()
		parts = append(parts, trace)

		// Get the first non-errnie stack frame for the code snippet
		if os.Getenv("NOSNIPPET") != "true" {
			const maxFrames = 10 // Limit how far we look back
			pc := make([]uintptr, maxFrames)
			n := runtime.Callers(2, pc)
			frames := runtime.CallersFrames(pc[:n])

			var frame runtime.Frame
			more := true
			// Skip frames until we find one outside the errnie package
			for more {
				frame, more = frames.Next()
				if !strings.Contains(frame.Function, "errnie.") {
					break
				}
			}

			if frame.File != "" {
				snippet := getCodeSnippet(frame.File, frame.Line, 5)
				if snippet != "" {
					parts = append(parts, "\n===[CODE SNIPPET]===\n"+snippet+"===[/CODE SNIPPET]===\n")
				}
			}
		}
	}

	message := strings.Join(parts, "\n")

	if os.Getenv("NOCONSOLE") != "true" {
		logger.Error(message, v...)
	}

	writeToLog(message)

	if os.Getenv("NOSTACK") == "true" {
		return fmt.Errorf("%s", errMsg)
	}

	return fmt.Errorf("%s", message)
}

/*
Write a log message to the log file, ensuring thread safety.
*/
func writeToLog(message string) {
	if os.Getenv("LOGFILE") != "true" || message == "" || logFile == nil {
		return
	}

	logFileMu.Lock()
	defer logFileMu.Unlock()

	// Strip ANSI escape codes and add a timestamp
	formattedMessage := fmt.Sprintf("[%s] %s\n", time.Now().Format("15:04:05"), stripansi.Strip(strings.TrimSpace(message)))

	_, err := logFile.WriteString(formattedMessage)
	if err != nil {
		logger.Warn("Failed to write to log file", "error", err)
	}

	// Ensure the write is flushed to disk
	if err := logFile.Sync(); err != nil {
		logger.Warn("Failed to sync log file", "error", err)
	}
}

/*
Retrieve and format a stack trace from the current execution point.
*/
func getStackTrace() string {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(3, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])

	var trace strings.Builder
	for {
		frame, more := frames.Next()
		if !more {
			break
		}

		funcName := frame.Function
		if lastSlash := strings.LastIndexByte(funcName, '/'); lastSlash >= 0 {
			funcName = funcName[lastSlash+1:]
		}
		funcName = strings.Replace(funcName, ".", ":", 1)

		line := fmt.Sprintf("%s at %s(line %d)\n",
			lipgloss.NewStyle().Foreground(lipgloss.Color("#6E95F7")).Render(funcName),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#06C26F")).Render(filepath.Base(frame.File)),
			frame.Line,
		)
		trace.WriteString(line)
	}

	return "\n===[STACK TRACE]===\n" + trace.String() + "===[/STACK TRACE]===\n"
}

/*
Retrieve and return a code snippet surrounding the given line in the provided file.
*/
func getCodeSnippet(file string, line, radius int) string {
	if file == "" {
		return ""
	}

	fileHandle, err := os.Open(file)
	if err != nil {
		logger.Warn("Failed to open file for code snippet", "file", file, "error", err)
		return ""
	}
	defer fileHandle.Close()

	scanner := bufio.NewScanner(fileHandle)
	currentLine := 1
	var snippet strings.Builder

	for scanner.Scan() {
		if currentLine >= line-radius && currentLine <= line+radius {
			prefix := "  "
			if currentLine == line {
				prefix = "> "
			}
			snippet.WriteString(fmt.Sprintf("%s%d: %s\n", prefix, currentLine, scanner.Text()))
		}
		currentLine++
	}

	if err := scanner.Err(); err != nil {
		logger.Warn("Failed to read from code snippet file", "file", file, "error", err)
		return ""
	}

	return snippet.String()
}
