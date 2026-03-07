package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestRunNfsWrapper_Success(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	exportsDir := filepath.Join(dir, "exports.d")
	if err := os.MkdirAll(exportsDir, 0755); err != nil {
		t.Fatalf("failed to create exports dir: %v", err)
	}

	configContent := `nfs:
  rules:
    - path: "/volume1/media"
      options: "rw,sync,no_subtree_check"`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	exportsPath := filepath.Join(exportsDir, "media.exports")
	originalExports := "/volume1/media *(ro,sync)\n"
	if err := os.WriteFile(exportsPath, []byte(originalExports), 0644); err != nil {
		t.Fatalf("failed to write exports file: %v", err)
	}

	argsPath := filepath.Join(dir, "nfs-args.txt")
	originalPath := createMockExecutable(t, filepath.Join(dir, "exportfs.orig"), "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$MOCK_ARGS_PATH\"\nexit 0\n")
	t.Setenv("MOCK_ARGS_PATH", argsPath)

	exitCode := runNfsWrapperWithPaths([]string{"-ra", "--foo"}, configPath, exportsDir, originalPath)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}

	updatedExports, err := os.ReadFile(exportsPath)
	if err != nil {
		t.Fatalf("failed to read exports file: %v", err)
	}
	if !strings.Contains(string(updatedExports), "rw,sync,no_subtree_check") {
		t.Fatalf("expected modified options in exports file, got %q", string(updatedExports))
	}

	gotArgs := readLines(t, argsPath)
	wantArgs := []string{"-ra", "--foo"}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("expected args %v, got %v", wantArgs, gotArgs)
	}
}

func TestRunNfsWrapper_MissingConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "missing.yaml")
	exportsDir := filepath.Join(dir, "exports.d")
	if err := os.MkdirAll(exportsDir, 0755); err != nil {
		t.Fatalf("failed to create exports dir: %v", err)
	}

	exportsPath := filepath.Join(exportsDir, "media.exports")
	originalExports := "/volume1/media *(ro,sync)\n"
	if err := os.WriteFile(exportsPath, []byte(originalExports), 0644); err != nil {
		t.Fatalf("failed to write exports file: %v", err)
	}

	originalPath := createMockExecutable(t, filepath.Join(dir, "exportfs.orig"), "#!/bin/sh\nexit 0\n")

	exitCode := runNfsWrapperWithPaths([]string{"-ra"}, configPath, exportsDir, originalPath)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}

	currentExports, err := os.ReadFile(exportsPath)
	if err != nil {
		t.Fatalf("failed to read exports file: %v", err)
	}
	if string(currentExports) != originalExports {
		t.Fatalf("expected exports unchanged, got %q", string(currentExports))
	}
}

func TestRunNfsWrapper_InvalidConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	exportsDir := filepath.Join(dir, "exports.d")
	if err := os.MkdirAll(exportsDir, 0755); err != nil {
		t.Fatalf("failed to create exports dir: %v", err)
	}

	if err := os.WriteFile(configPath, []byte("nfs: ["), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	exportsPath := filepath.Join(exportsDir, "media.exports")
	originalExports := "/volume1/media *(ro,sync)\n"
	if err := os.WriteFile(exportsPath, []byte(originalExports), 0644); err != nil {
		t.Fatalf("failed to write exports file: %v", err)
	}

	originalPath := createMockExecutable(t, filepath.Join(dir, "exportfs.orig"), "#!/bin/sh\nexit 0\n")

	exitCode := runNfsWrapperWithPaths([]string{"-ra"}, configPath, exportsDir, originalPath)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}

	currentExports, err := os.ReadFile(exportsPath)
	if err != nil {
		t.Fatalf("failed to read exports file: %v", err)
	}
	if string(currentExports) != originalExports {
		t.Fatalf("expected exports unchanged, got %q", string(currentExports))
	}
}

func TestRunNfsWrapper_MissingOriginal(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("nfs:\n  rules: []\n"), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	exportsDir := filepath.Join(dir, "exports.d")
	if err := os.MkdirAll(exportsDir, 0755); err != nil {
		t.Fatalf("failed to create exports dir: %v", err)
	}

	exitCode := runNfsWrapperWithPaths([]string{"-ra"}, configPath, exportsDir, filepath.Join(dir, "missing-exportfs.orig"))
	if exitCode == 0 {
		t.Fatalf("expected non-zero exit code when original binary is missing")
	}
}

func TestRunNfsWrapper_NoMatchingRules(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	configContent := `nfs:
  rules:
    - path: "/volume1/other"
      options: "rw,sync"`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	exportsDir := filepath.Join(dir, "exports.d")
	if err := os.MkdirAll(exportsDir, 0755); err != nil {
		t.Fatalf("failed to create exports dir: %v", err)
	}

	exportsPath := filepath.Join(exportsDir, "media.exports")
	originalExports := "/volume1/media *(ro,sync)\n"
	if err := os.WriteFile(exportsPath, []byte(originalExports), 0644); err != nil {
		t.Fatalf("failed to write exports file: %v", err)
	}

	originalPath := createMockExecutable(t, filepath.Join(dir, "exportfs.orig"), "#!/bin/sh\nexit 0\n")

	exitCode := runNfsWrapperWithPaths([]string{"-ra"}, configPath, exportsDir, originalPath)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}

	currentExports, err := os.ReadFile(exportsPath)
	if err != nil {
		t.Fatalf("failed to read exports file: %v", err)
	}
	if string(currentExports) != originalExports {
		t.Fatalf("expected exports unchanged, got %q", string(currentExports))
	}
}

func TestRunSmbcontrolWrapper_Success(t *testing.T) {
	dir := t.TempDir()
	smbConfPath := filepath.Join(dir, "smb.conf")
	if err := os.WriteFile(smbConfPath, []byte("include = /etc/samba/share.conf\n"), 0644); err != nil {
		t.Fatalf("failed to write smb.conf: %v", err)
	}

	includeLine := "include = /tmp/test-overrides.conf"
	argsPath := filepath.Join(dir, "smb-args.txt")
	originalPath := createMockExecutable(t, filepath.Join(dir, "smbcontrol.orig"), "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$MOCK_ARGS_PATH\"\nexit 0\n")
	t.Setenv("MOCK_ARGS_PATH", argsPath)

	exitCode := runSmbcontrolWrapperWithPaths([]string{"all", "reload-config"}, smbConfPath, includeLine, originalPath)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}

	smbConfContent, err := os.ReadFile(smbConfPath)
	if err != nil {
		t.Fatalf("failed to read smb.conf: %v", err)
	}
	if !strings.Contains(string(smbConfContent), includeLine) {
		t.Fatalf("expected include line to be injected, got %q", string(smbConfContent))
	}

	gotArgs := readLines(t, argsPath)
	wantArgs := []string{"all", "reload-config"}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("expected args %v, got %v", wantArgs, gotArgs)
	}
}

func TestRunSmbcontrolWrapper_InjectFails(t *testing.T) {
	dir := t.TempDir()
	missingSmbConfPath := filepath.Join(dir, "missing-smb.conf")
	originalPath := createMockExecutable(t, filepath.Join(dir, "smbcontrol.orig"), "#!/bin/sh\nexit 0\n")

	exitCode := runSmbcontrolWrapperWithPaths([]string{"all", "reload-config"}, missingSmbConfPath, "include = /tmp/test-overrides.conf", originalPath)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0 on inject failure fail-open, got %d", exitCode)
	}
}

func TestRunSmbcontrolWrapper_MissingOriginal(t *testing.T) {
	dir := t.TempDir()
	smbConfPath := filepath.Join(dir, "smb.conf")
	if err := os.WriteFile(smbConfPath, []byte("[global]\n"), 0644); err != nil {
		t.Fatalf("failed to write smb.conf: %v", err)
	}

	exitCode := runSmbcontrolWrapperWithPaths([]string{"all", "reload-config"}, smbConfPath, "include = /tmp/test-overrides.conf", filepath.Join(dir, "missing-smbcontrol.orig"))
	if exitCode == 0 {
		t.Fatalf("expected non-zero exit code when original binary is missing")
	}
}

func TestRunSmbcontrolWrapper_ArgsPassthrough(t *testing.T) {
	dir := t.TempDir()
	smbConfPath := filepath.Join(dir, "smb.conf")
	if err := os.WriteFile(smbConfPath, []byte("include = /etc/samba/share.conf\n"), 0644); err != nil {
		t.Fatalf("failed to write smb.conf: %v", err)
	}

	argsPath := filepath.Join(dir, "args.txt")
	originalPath := createMockExecutable(t, filepath.Join(dir, "smbcontrol.orig"), "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$MOCK_ARGS_PATH\"\nexit 0\n")
	t.Setenv("MOCK_ARGS_PATH", argsPath)

	args := []string{"-n", "123", "all", "close-share", "share-name"}
	exitCode := runSmbcontrolWrapperWithPaths(args, smbConfPath, "include = /tmp/test-overrides.conf", originalPath)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}

	gotArgs := readLines(t, argsPath)
	if !reflect.DeepEqual(gotArgs, args) {
		t.Fatalf("expected args %v, got %v", args, gotArgs)
	}
}

func TestRunSmbcontrolWrapper_ExitCodePropagation(t *testing.T) {
	dir := t.TempDir()
	smbConfPath := filepath.Join(dir, "smb.conf")
	if err := os.WriteFile(smbConfPath, []byte("include = /etc/samba/share.conf\n"), 0644); err != nil {
		t.Fatalf("failed to write smb.conf: %v", err)
	}

	originalPath := createMockExecutable(t, filepath.Join(dir, "smbcontrol.orig"), "#!/bin/sh\nexit 42\n")

	exitCode := runSmbcontrolWrapperWithPaths([]string{"all", "reload-config"}, smbConfPath, "include = /tmp/test-overrides.conf", originalPath)
	if exitCode != 42 {
		t.Fatalf("expected exit code 42, got %d", exitCode)
	}
}

func createMockExecutable(t *testing.T, path, content string) string {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		t.Fatalf("failed to write mock executable %s: %v", path, err)
	}
	return path
}

func readLines(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}
