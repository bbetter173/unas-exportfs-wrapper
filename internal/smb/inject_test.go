package smb

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestInject_AddsIncludeLine tests that Inject adds our include line after share.conf include.
func TestInject_AddsIncludeLine(t *testing.T) {
	tmpDir := t.TempDir()
	smbConfPath := filepath.Join(tmpDir, "smb.conf")

	// Create smb.conf with share.conf include
	content := `[global]
	workgroup = WORKGROUP
	include = /etc/samba/share.conf
	security = user
`
	if err := os.WriteFile(smbConfPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Inject our include line
	if err := Inject(smbConfPath, DefaultIncludeLine); err != nil {
		t.Fatalf("Inject failed: %v", err)
	}

	// Verify file contains our include line after share.conf
	result, err := os.ReadFile(smbConfPath)
	if err != nil {
		t.Fatalf("failed to read result: %v", err)
	}

	resultStr := string(result)
	if !strings.Contains(resultStr, DefaultIncludeLine) {
		t.Errorf("include line not found in result:\n%s", resultStr)
	}

	// Verify it comes after share.conf include
	shareIdx := strings.Index(resultStr, ShareConfInclude)
	injectIdx := strings.Index(resultStr, DefaultIncludeLine)
	if shareIdx >= injectIdx {
		t.Errorf("include line not placed after share.conf include")
	}
}

// TestInject_AlreadyPresent tests that Inject is idempotent (no-op if already present).
func TestInject_AlreadyPresent(t *testing.T) {
	tmpDir := t.TempDir()
	smbConfPath := filepath.Join(tmpDir, "smb.conf")

	// Create smb.conf with our include already present
	content := `[global]
	workgroup = WORKGROUP
	include = /etc/samba/share.conf
	include = /persistent/unas-custom/smb-overrides.conf
	security = user
`
	if err := os.WriteFile(smbConfPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	originalContent := content

	// Inject again (should be no-op)
	if err := Inject(smbConfPath, DefaultIncludeLine); err != nil {
		t.Fatalf("Inject failed: %v", err)
	}

	// Verify file is unchanged
	result, err := os.ReadFile(smbConfPath)
	if err != nil {
		t.Fatalf("failed to read result: %v", err)
	}

	if string(result) != originalContent {
		t.Errorf("file was modified when it should be idempotent")
	}
}

// TestInject_NoShareConfLine tests that Inject appends to end if share.conf include not found.
func TestInject_NoShareConfLine(t *testing.T) {
	tmpDir := t.TempDir()
	smbConfPath := filepath.Join(tmpDir, "smb.conf")

	// Create smb.conf without share.conf include
	content := `[global]
	workgroup = WORKGROUP
	security = user
`
	if err := os.WriteFile(smbConfPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Inject our include line
	if err := Inject(smbConfPath, DefaultIncludeLine); err != nil {
		t.Fatalf("Inject failed: %v", err)
	}

	// Verify file contains our include line
	result, err := os.ReadFile(smbConfPath)
	if err != nil {
		t.Fatalf("failed to read result: %v", err)
	}

	resultStr := string(result)
	if !strings.Contains(resultStr, DefaultIncludeLine) {
		t.Errorf("include line not found in result:\n%s", resultStr)
	}
}

// TestInject_PreservesContent tests that all other content is preserved byte-for-byte.
func TestInject_PreservesContent(t *testing.T) {
	tmpDir := t.TempDir()
	smbConfPath := filepath.Join(tmpDir, "smb.conf")

	// Create smb.conf with specific content
	content := `[global]
	workgroup = WORKGROUP
	include = /etc/samba/share.conf
	security = user
	map to guest = bad user

[homes]
	comment = Home Directories
	browseable = no
	read only = no
`
	if err := os.WriteFile(smbConfPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Inject our include line
	if err := Inject(smbConfPath, DefaultIncludeLine); err != nil {
		t.Fatalf("Inject failed: %v", err)
	}

	// Verify all original content is still present
	result, err := os.ReadFile(smbConfPath)
	if err != nil {
		t.Fatalf("failed to read result: %v", err)
	}

	resultStr := string(result)
	for _, line := range strings.Split(content, "\n") {
		if line != "" && !strings.Contains(resultStr, line) {
			t.Errorf("original content lost: %q", line)
		}
	}
}

// TestInject_MissingSmbConf tests that Inject returns error if smb.conf doesn't exist.
func TestInject_MissingSmbConf(t *testing.T) {
	tmpDir := t.TempDir()
	smbConfPath := filepath.Join(tmpDir, "nonexistent.conf")

	// Try to inject into non-existent file
	err := Inject(smbConfPath, DefaultIncludeLine)
	if err == nil {
		t.Errorf("expected error for missing file, got nil")
	}
}

// TestInject_EmptyFile tests that Inject appends include line to empty file.
func TestInject_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	smbConfPath := filepath.Join(tmpDir, "smb.conf")

	// Create empty smb.conf
	if err := os.WriteFile(smbConfPath, []byte(""), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Inject our include line
	if err := Inject(smbConfPath, DefaultIncludeLine); err != nil {
		t.Fatalf("Inject failed: %v", err)
	}

	// Verify file contains our include line
	result, err := os.ReadFile(smbConfPath)
	if err != nil {
		t.Fatalf("failed to read result: %v", err)
	}

	resultStr := string(result)
	if !strings.Contains(resultStr, DefaultIncludeLine) {
		t.Errorf("include line not found in result:\n%s", resultStr)
	}
}

// TestRemove_RemovesIncludeLine tests that Remove removes our include line.
func TestRemove_RemovesIncludeLine(t *testing.T) {
	tmpDir := t.TempDir()
	smbConfPath := filepath.Join(tmpDir, "smb.conf")

	// Create smb.conf with our include line
	content := `[global]
	workgroup = WORKGROUP
	include = /etc/samba/share.conf
	include = /persistent/unas-custom/smb-overrides.conf
	security = user
`
	if err := os.WriteFile(smbConfPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Remove our include line
	if err := Remove(smbConfPath, DefaultIncludeLine); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	// Verify file no longer contains our include line
	result, err := os.ReadFile(smbConfPath)
	if err != nil {
		t.Fatalf("failed to read result: %v", err)
	}

	resultStr := string(result)
	if strings.Contains(resultStr, DefaultIncludeLine) {
		t.Errorf("include line still present after removal:\n%s", resultStr)
	}

	// Verify share.conf include is still there
	if !strings.Contains(resultStr, ShareConfInclude) {
		t.Errorf("share.conf include was removed (should be preserved)")
	}
}

// TestRemove_NoIncludeLine tests that Remove is idempotent (no-op if not present).
func TestRemove_NoIncludeLine(t *testing.T) {
	tmpDir := t.TempDir()
	smbConfPath := filepath.Join(tmpDir, "smb.conf")

	// Create smb.conf without our include line
	content := `[global]
	workgroup = WORKGROUP
	include = /etc/samba/share.conf
	security = user
`
	if err := os.WriteFile(smbConfPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	originalContent := content

	// Remove (should be no-op)
	if err := Remove(smbConfPath, DefaultIncludeLine); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	// Verify file is unchanged
	result, err := os.ReadFile(smbConfPath)
	if err != nil {
		t.Fatalf("failed to read result: %v", err)
	}

	if string(result) != originalContent {
		t.Errorf("file was modified when it should be idempotent")
	}
}

// TestRemove_PreservesContent tests that all other content is preserved after removal.
func TestRemove_PreservesContent(t *testing.T) {
	tmpDir := t.TempDir()
	smbConfPath := filepath.Join(tmpDir, "smb.conf")

	// Create smb.conf with our include line and other content
	content := `[global]
	workgroup = WORKGROUP
	include = /etc/samba/share.conf
	include = /persistent/unas-custom/smb-overrides.conf
	security = user
	map to guest = bad user

[homes]
	comment = Home Directories
	browseable = no
	read only = no
`
	if err := os.WriteFile(smbConfPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Remove our include line
	if err := Remove(smbConfPath, DefaultIncludeLine); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	// Verify all other content is still present
	result, err := os.ReadFile(smbConfPath)
	if err != nil {
		t.Fatalf("failed to read result: %v", err)
	}

	resultStr := string(result)
	for _, line := range strings.Split(content, "\n") {
		if line != "" && !strings.Contains(line, DefaultIncludeLine) && !strings.Contains(resultStr, line) {
			t.Errorf("content was lost: %q", line)
		}
	}
}
