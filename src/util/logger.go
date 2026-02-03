package util

import (
	"fmt"
	"io"
	"os"
	"time"

	"quality-bot/src/config"
)

// LogLevel represents logging level
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

// Logger provides structured logging
type Logger struct {
	level            LogLevel
	output           io.Writer
	includeTimestamp bool
	includeCaller    bool
}

// NewLogger creates a new logger from config
func NewLogger(cfg config.LoggingConfig) *Logger {
	level := LogLevelInfo
	switch cfg.Level {
	case "debug":
		level = LogLevelDebug
	case "info":
		level = LogLevelInfo
	case "warn":
		level = LogLevelWarn
	case "error":
		level = LogLevelError
	}

	output := io.Writer(os.Stderr)
	if cfg.File != "" {
		if f, err := os.OpenFile(cfg.File, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
			output = f
		}
	}

	return &Logger{
		level:            level,
		output:           output,
		includeTimestamp: cfg.IncludeTimestamp,
		includeCaller:    cfg.IncludeCaller,
	}
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, args ...any) {
	if l.level <= LogLevelDebug {
		l.log("DEBUG", msg, args...)
	}
}

// Info logs an info message
func (l *Logger) Info(msg string, args ...any) {
	if l.level <= LogLevelInfo {
		l.log("INFO", msg, args...)
	}
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, args ...any) {
	if l.level <= LogLevelWarn {
		l.log("WARN", msg, args...)
	}
}

// Error logs an error message
func (l *Logger) Error(msg string, args ...any) {
	if l.level <= LogLevelError {
		l.log("ERROR", msg, args...)
	}
}

func (l *Logger) log(level, msg string, args ...any) {
	var prefix string
	if l.includeTimestamp {
		prefix = time.Now().Format("2006-01-02 15:04:05") + " "
	}
	prefix += "[" + level + "] "

	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}

	fmt.Fprintln(l.output, prefix+msg)
}

// DefaultLogger is the package-level default logger
var DefaultLogger = NewLogger(config.LoggingConfig{
	Level:            "info",
	IncludeTimestamp: true,
})

// SetDefaultLogger updates the default logger with new configuration
func SetDefaultLogger(cfg config.LoggingConfig) {
	DefaultLogger = NewLogger(cfg)
}

// GetLevel returns the current log level as a string
func (l *Logger) GetLevel() string {
	switch l.level {
	case LogLevelDebug:
		return "debug"
	case LogLevelInfo:
		return "info"
	case LogLevelWarn:
		return "warn"
	case LogLevelError:
		return "error"
	default:
		return "info"
	}
}

// Debug logs using the default logger
func Debug(msg string, args ...any) {
	DefaultLogger.Debug(msg, args...)
}

// Info logs using the default logger
func Info(msg string, args ...any) {
	DefaultLogger.Info(msg, args...)
}

// Warn logs using the default logger
func Warn(msg string, args ...any) {
	DefaultLogger.Warn(msg, args...)
}

// Error logs using the default logger
func Error(msg string, args ...any) {
	DefaultLogger.Error(msg, args...)
}
