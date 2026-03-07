package smb

import (
	"fmt"
	"os"
	"strings"
)

const (
	DefaultSmbConfPath = "/etc/samba/smb.conf"
	DefaultIncludeLine = "include = /persistent/unas-custom/smb-overrides.conf"
	ShareConfInclude   = "include = /etc/samba/share.conf"
)

// Inject idempotently adds includeLine to smbConfPath.
// Places it after ShareConfInclude if found; appends to end otherwise.
// Uses atomic write (temp file + rename) to prevent partial reads.
// Returns nil if includeLine is already present (idempotent).
// Returns error if smbConfPath cannot be read.
func Inject(smbConfPath string, includeLine string) error {
	content, err := os.ReadFile(smbConfPath)
	if err != nil {
		return fmt.Errorf("reading smb.conf: %w", err)
	}

	lines := strings.Split(string(content), "\n")

	// Idempotency check: already present?
	for _, line := range lines {
		if strings.TrimSpace(line) == strings.TrimSpace(includeLine) {
			return nil
		}
	}

	// Find insertion point: after ShareConfInclude
	insertAt := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == strings.TrimSpace(ShareConfInclude) {
			insertAt = i + 1
			break
		}
	}

	// Build new content
	var newLines []string
	if insertAt >= 0 {
		newLines = append(newLines, lines[:insertAt]...)
		newLines = append(newLines, includeLine)
		newLines = append(newLines, lines[insertAt:]...)
	} else {
		newLines = append(lines, includeLine)
	}

	newContent := strings.Join(newLines, "\n")
	return atomicWrite(smbConfPath, newContent)
}

// Remove idempotently removes all instances of includeLine from smbConfPath.
// Returns nil if includeLine is not present (idempotent).
// Returns error if smbConfPath cannot be read (except if file doesn't exist).
func Remove(smbConfPath string, includeLine string) error {
	content, err := os.ReadFile(smbConfPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading smb.conf: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	for _, line := range lines {
		if strings.TrimSpace(line) != strings.TrimSpace(includeLine) {
			newLines = append(newLines, line)
		}
	}

	// If nothing removed, no-op
	if len(newLines) == len(lines) {
		return nil
	}

	return atomicWrite(smbConfPath, strings.Join(newLines, "\n"))
}

// atomicWrite writes content to path using atomic write pattern:
// 1. Write to temporary file (path + ".tmp")
// 2. Rename temp file to target path
// This prevents partial reads if process crashes mid-write.
func atomicWrite(path string, content string) error {
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("renaming temp file: %w", err)
	}
	return nil
}
