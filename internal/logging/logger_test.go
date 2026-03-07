package logging

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// TestLoggerWritesToFile verifies that Info logs are written with correct format
func TestLoggerWritesToFile(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := New(logPath)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer logger.Close()

	logger.Info("test message")

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile() failed: %v", err)
	}

	logLine := string(content)
	// Format: 2006-01-02 15:04:05 [INFO] message
	pattern := `^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2} \[INFO\] test message\n?$`
	if !regexp.MustCompile(pattern).MatchString(logLine) {
		t.Errorf("log format mismatch.\nGot: %q\nExpected pattern: %s", logLine, pattern)
	}
}

// TestLoggerAppendsNotTruncates verifies that multiple writes append, not truncate
func TestLoggerAppendsNotTruncates(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := New(logPath)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer logger.Close()

	logger.Info("first message")
	logger.Info("second message")

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile() failed: %v", err)
	}

	logContent := string(content)
	if !regexp.MustCompile(`\[INFO\] first message`).MatchString(logContent) {
		t.Errorf("first message not found in log")
	}
	if !regexp.MustCompile(`\[INFO\] second message`).MatchString(logContent) {
		t.Errorf("second message not found in log")
	}
}

// TestNoopLogger verifies that Noop logger silently discards all output
func TestNoopLogger(t *testing.T) {
	logger := NewNoop()

	// Should not panic or error
	logger.Info("test info")
	logger.Warn("test warn")
	logger.Error("test error")

	// Close should not error
	err := logger.Close()
	if err != nil {
		t.Errorf("Close() on Noop logger failed: %v", err)
	}
}

// TestLoggerMissingDirectory verifies that missing parent directory returns error
func TestLoggerMissingDirectory(t *testing.T) {
	logPath := "/nonexistent/path/that/does/not/exist/test.log"

	_, err := New(logPath)
	if err == nil {
		t.Errorf("New() should return error for non-existent directory, got nil")
	}
}

// TestLogLevels verifies that Warn and Error use correct level markers
func TestLogLevels(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := New(logPath)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer logger.Close()

	logger.Warn("warning message")
	logger.Error("error message")

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile() failed: %v", err)
	}

	logContent := string(content)
	if !regexp.MustCompile(`\[WARN\] warning message`).MatchString(logContent) {
		t.Errorf("WARN level marker not found in log")
	}
	if !regexp.MustCompile(`\[ERROR\] error message`).MatchString(logContent) {
		t.Errorf("ERROR level marker not found in log")
	}
}

// TestLoggerWithFormatting verifies that format strings work correctly
func TestLoggerWithFormatting(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := New(logPath)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer logger.Close()

	logger.Info("value: %d, name: %s", 42, "test")

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile() failed: %v", err)
	}

	logContent := string(content)
	if !regexp.MustCompile(`value: 42, name: test`).MatchString(logContent) {
		t.Errorf("formatted message not found in log")
	}
}
