package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func newTestChecker(t *testing.T) (*StatusChecker, string) {
	dir := t.TempDir()
	return &StatusChecker{
		ExportfsPath:       filepath.Join(dir, "exportfs"),
		ExportfsOrigPath:   filepath.Join(dir, "exportfs.orig"),
		SmbcontrolPath:     filepath.Join(dir, "smbcontrol"),
		SmbcontrolOrigPath: filepath.Join(dir, "smbcontrol.orig"),
		DropInPath:         filepath.Join(dir, "unas-custom.conf"),
		SmbConfPath:        filepath.Join(dir, "smb.conf"),
		IncludeLine:        "include = /persistent/unas-custom/smb-overrides.conf",
		OverridesPath:      filepath.Join(dir, "smb-overrides.conf"),
		ConfigPath:         filepath.Join(dir, "config.yaml"),
		Out:                io.Discard,
	}, dir
}

func TestStatus_AllInstalled(t *testing.T) {
	checker, dir := newTestChecker(t)

	// Create all required files
	if err := os.WriteFile(filepath.Join(dir, "exportfs"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "exportfs.orig"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "smbcontrol"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "smbcontrol.orig"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "unas-custom.conf"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "smb-overrides.conf"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	// Create smb.conf with include line
	smbConfContent := `[global]
	workgroup = WORKGROUP
	include = /persistent/unas-custom/smb-overrides.conf
`
	if err := os.WriteFile(filepath.Join(dir, "smb.conf"), []byte(smbConfContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create valid config.yaml
	configContent := `nfs:
  rules: []
smb:
  overrides: []
`
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Capture output
	var buf bytes.Buffer
	checker.Out = &buf

	// Run check
	exitCode := checker.Check()

	// Verify exit code
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
		t.Logf("output:\n%s", buf.String())
	}

	// Verify all checks passed
	output := buf.String()
	if !bytes.Contains(buf.Bytes(), []byte("NFS wrapper:      installed ✓")) {
		t.Errorf("expected NFS wrapper installed check to pass")
	}
	if !bytes.Contains(buf.Bytes(), []byte("SMB wrapper:      installed ✓")) {
		t.Errorf("expected SMB wrapper installed check to pass")
	}
	if !bytes.Contains(buf.Bytes(), []byte("systemd drop-in:  installed ✓")) {
		t.Errorf("expected systemd drop-in check to pass")
	}
	if !bytes.Contains(buf.Bytes(), []byte("SMB include:      present ✓")) {
		t.Errorf("expected SMB include check to pass")
	}
	if !bytes.Contains(buf.Bytes(), []byte("SMB overrides:    generated ✓")) {
		t.Errorf("expected SMB overrides check to pass")
	}
	if !bytes.Contains(buf.Bytes(), []byte("Config:           valid ✓")) {
		t.Errorf("expected Config check to pass")
	}

	t.Logf("output:\n%s", output)
}

func TestStatus_NothingInstalled(t *testing.T) {
	checker, dir := newTestChecker(t)
	_ = dir // temp dir is empty

	// Capture output
	var buf bytes.Buffer
	checker.Out = &buf

	// Run check
	exitCode := checker.Check()

	// Verify exit code is 1 (failure)
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}

	// Verify checks failed
	output := buf.String()
	if !bytes.Contains(buf.Bytes(), []byte("✗")) {
		t.Errorf("expected at least one failed check (✗)")
	}

	t.Logf("output:\n%s", output)
}

func TestStatus_PartialInstall_NFSOnly(t *testing.T) {
	checker, dir := newTestChecker(t)

	// Create both exportfs and exportfs.orig
	if err := os.WriteFile(filepath.Join(dir, "exportfs"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "exportfs.orig"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	// Capture output
	var buf bytes.Buffer
	checker.Out = &buf

	// Run check
	exitCode := checker.Check()

	// Verify exit code is 1 (failure)
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}

	// Verify NFS check passed but others failed
	output := buf.String()
	if !bytes.Contains(buf.Bytes(), []byte("NFS wrapper:      installed ✓")) {
		t.Errorf("expected NFS wrapper check to pass")
	}
	if !bytes.Contains(buf.Bytes(), []byte("✗")) {
		t.Errorf("expected at least one failed check (✗)")
	}

	t.Logf("output:\n%s", output)
}

func TestStatus_PartialInstall_NoDropIn(t *testing.T) {
	checker, dir := newTestChecker(t)

	// Create wrappers but no drop-in
	if err := os.WriteFile(filepath.Join(dir, "exportfs"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "exportfs.orig"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "smbcontrol"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "smbcontrol.orig"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "smb-overrides.conf"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	// Create smb.conf with include line
	smbConfContent := `[global]
	include = /persistent/unas-custom/smb-overrides.conf
`
	if err := os.WriteFile(filepath.Join(dir, "smb.conf"), []byte(smbConfContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create valid config
	configContent := `nfs:
  rules: []
smb:
  overrides: []
`
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Capture output
	var buf bytes.Buffer
	checker.Out = &buf

	// Run check
	exitCode := checker.Check()

	// Verify exit code is 1 (failure)
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}

	// Verify drop-in check failed
	output := buf.String()
	if !bytes.Contains(buf.Bytes(), []byte("systemd drop-in:  installed ✗")) {
		t.Errorf("expected systemd drop-in check to fail")
	}

	t.Logf("output:\n%s", output)
}

func TestStatus_BrokenConfig(t *testing.T) {
	checker, dir := newTestChecker(t)

	// Create all files except config
	if err := os.WriteFile(filepath.Join(dir, "exportfs"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "exportfs.orig"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "smbcontrol"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "smbcontrol.orig"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "unas-custom.conf"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "smb-overrides.conf"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	// Create smb.conf with include line
	smbConfContent := `[global]
	include = /persistent/unas-custom/smb-overrides.conf
`
	if err := os.WriteFile(filepath.Join(dir, "smb.conf"), []byte(smbConfContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create invalid config (bad YAML)
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("invalid: yaml: content:"), 0644); err != nil {
		t.Fatal(err)
	}

	// Capture output
	var buf bytes.Buffer
	checker.Out = &buf

	// Run check
	exitCode := checker.Check()

	// Verify exit code is 1 (failure)
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}

	// Verify config check failed
	output := buf.String()
	if !bytes.Contains(buf.Bytes(), []byte("Config:           valid ✗")) {
		t.Errorf("expected Config check to fail")
	}

	t.Logf("output:\n%s", output)
}

func TestStatus_ExitCodes(t *testing.T) {
	// Test that exit code is 0 for all OK, 1 for any issue
	tests := []struct {
		name     string
		setup    func(string) error
		wantCode int
	}{
		{
			name: "all_ok",
			setup: func(dir string) error {
				files := []string{"exportfs", "exportfs.orig", "smbcontrol", "smbcontrol.orig", "unas-custom.conf", "smb-overrides.conf"}
				for _, f := range files {
					if err := os.WriteFile(filepath.Join(dir, f), []byte(""), 0644); err != nil {
						return err
					}
				}
				if err := os.WriteFile(filepath.Join(dir, "smb.conf"), []byte("include = /persistent/unas-custom/smb-overrides.conf\n"), 0644); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("nfs:\n  rules: []\nsmb:\n  overrides: []\n"), 0644)
			},
			wantCode: 0,
		},
		{
			name: "missing_exportfs_orig",
			setup: func(dir string) error {
				files := []string{"smbcontrol", "smbcontrol.orig", "unas-custom.conf", "smb-overrides.conf"}
				for _, f := range files {
					if err := os.WriteFile(filepath.Join(dir, f), []byte(""), 0644); err != nil {
						return err
					}
				}
				if err := os.WriteFile(filepath.Join(dir, "smb.conf"), []byte("include = /persistent/unas-custom/smb-overrides.conf\n"), 0644); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("nfs:\n  rules: []\nsmb:\n  overrides: []\n"), 0644)
			},
			wantCode: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker, dir := newTestChecker(t)
			if err := tt.setup(dir); err != nil {
				t.Fatal(err)
			}

			var buf bytes.Buffer
			checker.Out = &buf

			exitCode := checker.Check()
			if exitCode != tt.wantCode {
				t.Errorf("expected exit code %d, got %d", tt.wantCode, exitCode)
				t.Logf("output:\n%s", buf.String())
			}
		})
	}
}

func TestStatus_NfsWrapper_OrigExistsButTargetMissing(t *testing.T) {
	checker, dir := newTestChecker(t)
	if err := os.WriteFile(filepath.Join(dir, "exportfs.orig"), []byte("orig"), 0755); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	checker.Out = &buf
	result := checker.Check()

	if result == 0 {
		t.Error("expected non-zero exit when target missing but orig exists")
	}
	if !bytes.Contains(buf.Bytes(), []byte("NFS wrapper:      not installed")) {
		t.Errorf("expected 'not installed' in output, got: %s", buf.String())
	}
}

func TestStatus_SmbWrapper_OrigExistsButTargetMissing(t *testing.T) {
	checker, dir := newTestChecker(t)
	if err := os.WriteFile(filepath.Join(dir, "smbcontrol.orig"), []byte("orig"), 0755); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	checker.Out = &buf
	result := checker.Check()

	if result == 0 {
		t.Error("expected non-zero exit when target missing but orig exists")
	}
	if !bytes.Contains(buf.Bytes(), []byte("SMB wrapper:      not installed")) {
		t.Errorf("expected 'not installed' in output, got: %s", buf.String())
	}
}

func TestRunStatusImpl(t *testing.T) {
	// Test that runStatusImpl creates a checker and calls Check()
	// This is a simple integration test
	exitCode := runStatusImpl([]string{})
	// We can't easily test the actual output without mocking, but we can verify it doesn't panic
	if exitCode < 0 || exitCode > 1 {
		t.Errorf("expected exit code 0 or 1, got %d", exitCode)
	}
}
