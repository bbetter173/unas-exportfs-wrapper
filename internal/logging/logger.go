package logging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type Logger struct {
	writer io.WriteCloser
}

func New(logPath string) (*Logger, error) {
	dir := filepath.Dir(logPath)
	if _, err := os.Stat(dir); err != nil {
		return nil, fmt.Errorf("log directory does not exist: %w", err)
	}

	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return &Logger{writer: file}, nil
}

func NewNoop() *Logger {
	return &Logger{writer: &noopWriter{}}
}

type noopWriter struct{}

func (nw *noopWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func (nw *noopWriter) Close() error {
	return nil
}

func (l *Logger) Info(format string, args ...interface{}) {
	l.log("INFO", format, args...)
}

func (l *Logger) Warn(format string, args ...interface{}) {
	l.log("WARN", format, args...)
}

func (l *Logger) Error(format string, args ...interface{}) {
	l.log("ERROR", format, args...)
}

func (l *Logger) log(level, format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	message := fmt.Sprintf(format, args...)
	logLine := fmt.Sprintf("%s [%s] %s\n", timestamp, level, message)
	l.writer.Write([]byte(logLine))
}

func (l *Logger) Close() error {
	return l.writer.Close()
}
