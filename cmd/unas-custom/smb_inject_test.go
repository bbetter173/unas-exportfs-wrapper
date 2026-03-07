package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunSmbInjectImplWithPath_Success(t *testing.T) {
	dir := t.TempDir()
	smbConfPath := filepath.Join(dir, "smb.conf")
	if err := os.WriteFile(smbConfPath, []byte("[global]\n"), 0644); err != nil {
		t.Fatalf("failed to write smb.conf: %v", err)
	}

	code := runSmbInjectImplWithPath(nil, smbConfPath)
	if code != 0 {
		t.Errorf("runSmbInjectImplWithPath() = %d, want 0", code)
	}

	content, err := os.ReadFile(smbConfPath)
	if err != nil {
		t.Fatalf("failed to read smb.conf: %v", err)
	}
	if !strings.Contains(string(content), "include") {
		t.Errorf("expected include line in smb.conf, got: %q", string(content))
	}
}

func TestRunSmbInjectImplWithPath_Error(t *testing.T) {
	dir := t.TempDir()
	nonExistentPath := filepath.Join(dir, "nonexistent.conf")

	code := runSmbInjectImplWithPath(nil, nonExistentPath)
	if code != 1 {
		t.Errorf("runSmbInjectImplWithPath() with missing conf = %d, want 1", code)
	}
}
