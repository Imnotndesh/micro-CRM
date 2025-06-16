// logger/logger.go
package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

// Logger defines the interface for our custom logger.
// This allows for flexible implementations (console, file, third-party service).
type Logger interface {
	Debug(format string, v ...interface{})
	Info(format string, v ...interface{})
	Warn(format string, v ...interface{})
	Error(format string, v ...interface{})
	Fatal(format string, v ...interface{}) // Calls os.Exit(1) after logging
	SetOutput(w io.Writer)
	SetPrefix(prefix string)
	SetFlags(flag int)
}

// LogLevel represents the severity of a log message.
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)

// String returns the string representation of a LogLevel.
func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO "
	case LogLevelWarn:
		return "WARN "
	case LogLevelError:
		return "ERROR"
	case LogLevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

type ConsoleLogger struct {
	*log.Logger
	mu       sync.Mutex
	minLevel LogLevel
}

func NewConsoleLogger(output io.Writer, prefix string, flag int, minLevel LogLevel) *ConsoleLogger {
	return &ConsoleLogger{
		Logger:   log.New(output, prefix, flag),
		minLevel: minLevel,
	}
}

// logf formats and prints a log message if its level is at or above minLevel.
func (cl *ConsoleLogger) logf(level LogLevel, format string, v ...interface{}) {
	if level < cl.minLevel {
		return // Do not log if severity is below minimum level
	}
	cl.mu.Lock()
	defer cl.mu.Unlock()

	// Add timestamp and level prefix
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	message := fmt.Sprintf(format, v...)
	cl.Logger.Printf("%s [%s] %s", timestamp, level.String(), message)

	if level == LogLevelFatal {
		os.Exit(1) // Exit the application on Fatal errors
	}
}

// Debug logs a debug message.
func (cl *ConsoleLogger) Debug(format string, v ...interface{}) {
	cl.logf(LogLevelDebug, format, v...)
}

// Info logs an info message.
func (cl *ConsoleLogger) Info(format string, v ...interface{}) {
	cl.logf(LogLevelInfo, format, v...)
}

// Warn logs a warning message.
func (cl *ConsoleLogger) Warn(format string, v ...interface{}) {
	cl.logf(LogLevelWarn, format, v...)
}

// Error logs an error message.
func (cl *ConsoleLogger) Error(format string, v ...interface{}) {
	cl.logf(LogLevelError, format, v...)
}

// Fatal logs a fatal message and exits the application.
func (cl *ConsoleLogger) Fatal(format string, v ...interface{}) {
	cl.logf(LogLevelFatal, format, v...)
}

// SetOutput sets the output destination for the logger.
func (cl *ConsoleLogger) SetOutput(w io.Writer) {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	cl.Logger.SetOutput(w)
}

// SetPrefix sets the output prefix for the logger.
func (cl *ConsoleLogger) SetPrefix(prefix string) {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	cl.Logger.SetPrefix(prefix)
}

// SetFlags sets the output flags for the logger.
func (cl *ConsoleLogger) SetFlags(flag int) {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	cl.Logger.SetFlags(flag)
}
