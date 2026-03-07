package installer

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestInstaller(t *testing.T) (*Installer, string) {
	t.Helper()
	dir := t.TempDir()
	inst := &Installer{
		BinaryPath:         filepath.Join(dir, "unas-custom"),
		ExportfsPath:       filepath.Join(dir, "exportfs"),
		ExportfsOrigPath:   filepath.Join(dir, "exportfs.orig"),
		SmbcontrolPath:     filepath.Join(dir, "smbcontrol"),
		SmbcontrolOrigPath: filepath.Join(dir, "smbcontrol.orig"),
		SmbConfPath:        filepath.Join(dir, "smb.conf"),
		IncludeLine:        "include = /persistent/unas-custom/smb-overrides.conf",
		ConfigDir:          filepath.Join(dir, "config"),
		OverridesPath:      filepath.Join(dir, "smb-overrides.conf"),
		DropInDir:          filepath.Join(dir, "smbd.service.d"),
		DropInPath:         filepath.Join(dir, "smbd.service.d", "unas-custom.conf"),
		OldInstallDir:      filepath.Join(dir, "nfs-intercept"),
		DaemonReloadFn:     func() error { return nil },
		Stdout:             io.Discard,
	}
	os.WriteFile(inst.BinaryPath, []byte("fake binary"), 0755)
	os.WriteFile(inst.ExportfsPath, []byte("original exportfs"), 0755)
	os.WriteFile(inst.SmbcontrolPath, []byte("original smbcontrol"), 0755)
	os.WriteFile(inst.SmbConfPath, []byte("[global]\n\tworkgroup = WORKGROUP\n"), 0644)
	return inst, dir
}

func TestInstall_NFS_BackupsExportfs(t *testing.T) {
	inst, _ := newTestInstaller(t)
	if err := inst.Install(inst.BinaryPath); err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	data, err := os.ReadFile(inst.ExportfsOrigPath)
	if err != nil {
		t.Fatalf("exportfs.orig not created: %v", err)
	}
	if string(data) != "original exportfs" {
		t.Errorf("exportfs.orig = %q, want %q", string(data), "original exportfs")
	}
	target, err := os.Readlink(inst.ExportfsPath)
	if err != nil {
		t.Fatalf("exportfs is not a symlink: %v", err)
	}
	if target != inst.BinaryPath {
		t.Errorf("exportfs symlink = %q, want %q", target, inst.BinaryPath)
	}
}

func TestInstall_NFS_AlreadyInstalled(t *testing.T) {
	inst, _ := newTestInstaller(t)
	os.WriteFile(inst.ExportfsOrigPath, []byte("sentinel original"), 0755)
	if err := inst.Install(inst.BinaryPath); err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	data, _ := os.ReadFile(inst.ExportfsOrigPath)
	if string(data) != "sentinel original" {
		t.Errorf("exportfs.orig was overwritten; got %q, want %q", string(data), "sentinel original")
	}
	target, err := os.Readlink(inst.ExportfsPath)
	if err != nil {
		t.Fatalf("exportfs is not a symlink: %v", err)
	}
	if target != inst.BinaryPath {
		t.Errorf("exportfs symlink = %q, want %q", target, inst.BinaryPath)
	}
}

func TestInstall_SMB_BackupsSmbcontrol(t *testing.T) {
	inst, _ := newTestInstaller(t)
	if err := inst.Install(inst.BinaryPath); err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	data, err := os.ReadFile(inst.SmbcontrolOrigPath)
	if err != nil {
		t.Fatalf("smbcontrol.orig not created: %v", err)
	}
	if string(data) != "original smbcontrol" {
		t.Errorf("smbcontrol.orig = %q, want %q", string(data), "original smbcontrol")
	}
	target, err := os.Readlink(inst.SmbcontrolPath)
	if err != nil {
		t.Fatalf("smbcontrol is not a symlink: %v", err)
	}
	if target != inst.BinaryPath {
		t.Errorf("smbcontrol symlink = %q, want %q", target, inst.BinaryPath)
	}
}

func TestInstall_SMB_SmbcontrolAlreadyInstalled(t *testing.T) {
	inst, _ := newTestInstaller(t)
	os.WriteFile(inst.SmbcontrolOrigPath, []byte("sentinel smbcontrol"), 0755)
	if err := inst.Install(inst.BinaryPath); err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	data, _ := os.ReadFile(inst.SmbcontrolOrigPath)
	if string(data) != "sentinel smbcontrol" {
		t.Errorf("smbcontrol.orig was overwritten; got %q, want %q", string(data), "sentinel smbcontrol")
	}
}

func TestInstall_SMB_CreatesDropIn(t *testing.T) {
	inst, _ := newTestInstaller(t)
	if err := inst.Install(inst.BinaryPath); err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if _, err := os.Stat(inst.DropInPath); err != nil {
		t.Errorf("drop-in not created: %v", err)
	}
}

func TestInstall_SMB_DropInContent(t *testing.T) {
	inst, _ := newTestInstaller(t)
	if err := inst.Install(inst.BinaryPath); err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	data, err := os.ReadFile(inst.DropInPath)
	if err != nil {
		t.Fatalf("reading drop-in: %v", err)
	}
	content := string(data)
	requiredLines := []string{
		"# Managed by unas-custom",
		"[Service]",
		"ExecStartPre=/persistent/unas-custom/unas-custom smb inject",
		"ExecReload=",
		"ExecReload=/persistent/unas-custom/unas-custom smb inject",
		"ExecReload=/bin/kill -HUP $MAINPID",
	}
	for _, line := range requiredLines {
		if !strings.Contains(content, line) {
			t.Errorf("drop-in missing line: %q\ncontent:\n%s", line, content)
		}
	}
}

func TestInstall_SMB_DropInAlreadyExists(t *testing.T) {
	inst, _ := newTestInstaller(t)
	os.MkdirAll(inst.DropInDir, 0755)
	os.WriteFile(inst.DropInPath, []byte("old content"), 0644)
	if err := inst.Install(inst.BinaryPath); err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	data, _ := os.ReadFile(inst.DropInPath)
	if string(data) == "old content" {
		t.Error("drop-in was not overwritten with correct content")
	}
	if !strings.Contains(string(data), "[Service]") {
		t.Errorf("drop-in missing [Service]; content:\n%s", string(data))
	}
}

func TestInstall_SMB_CreatesDropInDir(t *testing.T) {
	inst, _ := newTestInstaller(t)
	if err := inst.Install(inst.BinaryPath); err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if _, err := os.Stat(inst.DropInDir); err != nil {
		t.Errorf("DropInDir not created: %v", err)
	}
}

func TestInstall_SMB_InjectsIncludeLine(t *testing.T) {
	inst, _ := newTestInstaller(t)
	if err := inst.Install(inst.BinaryPath); err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	data, err := os.ReadFile(inst.SmbConfPath)
	if err != nil {
		t.Fatalf("reading smb.conf: %v", err)
	}
	if !strings.Contains(string(data), inst.IncludeLine) {
		t.Errorf("smb.conf does not contain include line; content:\n%s", string(data))
	}
}

func TestInstall_GeneratesConfigDir(t *testing.T) {
	inst, _ := newTestInstaller(t)
	os.RemoveAll(inst.ConfigDir)
	if err := inst.Install(inst.BinaryPath); err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if _, err := os.Stat(inst.ConfigDir); err != nil {
		t.Errorf("ConfigDir not created: %v", err)
	}
}

func TestInstall_DetectsOldInstallation(t *testing.T) {
	inst, _ := newTestInstaller(t)
	var buf bytes.Buffer
	inst.Stdout = &buf
	os.MkdirAll(inst.OldInstallDir, 0755)
	if err := inst.Install(inst.BinaryPath); err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if !strings.Contains(buf.String(), inst.OldInstallDir) {
		t.Errorf("expected old install dir mention in output; got:\n%s", buf.String())
	}
}

func TestInstall_DaemonReloadCalled(t *testing.T) {
	inst, _ := newTestInstaller(t)
	called := false
	inst.DaemonReloadFn = func() error { called = true; return nil }
	if err := inst.Install(inst.BinaryPath); err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if !called {
		t.Error("DaemonReloadFn was not called")
	}
}

func TestUninstall_NFS_RestoresOriginal(t *testing.T) {
	inst, _ := newTestInstaller(t)
	if err := inst.Install(inst.BinaryPath); err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if err := inst.Uninstall(); err != nil {
		t.Fatalf("Uninstall() error = %v", err)
	}
	data, err := os.ReadFile(inst.ExportfsPath)
	if err != nil {
		t.Fatalf("reading exportfs: %v", err)
	}
	if string(data) != "original exportfs" {
		t.Errorf("exportfs = %q, want %q", string(data), "original exportfs")
	}
	if _, err := os.Stat(inst.ExportfsOrigPath); !os.IsNotExist(err) {
		t.Error("exportfs.orig should have been removed")
	}
}

func TestUninstall_NFS_NoBackup(t *testing.T) {
	inst, _ := newTestInstaller(t)
	var buf bytes.Buffer
	inst.Stdout = &buf
	if err := inst.Uninstall(); err != nil {
		t.Fatalf("Uninstall() error = %v", err)
	}
	if !strings.Contains(buf.String(), "NFS wrapper not installed") {
		t.Errorf("expected NFS wrapper not installed message; got:\n%s", buf.String())
	}
}

func TestUninstall_SMB_RestoresSmbcontrol(t *testing.T) {
	inst, _ := newTestInstaller(t)
	if err := inst.Install(inst.BinaryPath); err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if err := inst.Uninstall(); err != nil {
		t.Fatalf("Uninstall() error = %v", err)
	}
	data, err := os.ReadFile(inst.SmbcontrolPath)
	if err != nil {
		t.Fatalf("reading smbcontrol: %v", err)
	}
	if string(data) != "original smbcontrol" {
		t.Errorf("smbcontrol = %q, want %q", string(data), "original smbcontrol")
	}
	if _, err := os.Stat(inst.SmbcontrolOrigPath); !os.IsNotExist(err) {
		t.Error("smbcontrol.orig should have been removed")
	}
}

func TestUninstall_SMB_SmbcontrolNoBackup(t *testing.T) {
	inst, _ := newTestInstaller(t)
	var buf bytes.Buffer
	inst.Stdout = &buf
	if err := inst.Uninstall(); err != nil {
		t.Fatalf("Uninstall() error = %v", err)
	}
	if !strings.Contains(buf.String(), "SMB wrapper not installed") {
		t.Errorf("expected SMB wrapper not installed message; got:\n%s", buf.String())
	}
}

func TestUninstall_SMB_RemovesDropIn(t *testing.T) {
	inst, _ := newTestInstaller(t)
	if err := inst.Install(inst.BinaryPath); err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if err := inst.Uninstall(); err != nil {
		t.Fatalf("Uninstall() error = %v", err)
	}
	if _, err := os.Stat(inst.DropInPath); !os.IsNotExist(err) {
		t.Error("drop-in file should have been removed")
	}
}

func TestUninstall_SMB_RemovesIncludeLine(t *testing.T) {
	inst, _ := newTestInstaller(t)
	if err := inst.Install(inst.BinaryPath); err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if err := inst.Uninstall(); err != nil {
		t.Fatalf("Uninstall() error = %v", err)
	}
	data, err := os.ReadFile(inst.SmbConfPath)
	if err != nil {
		t.Fatalf("reading smb.conf: %v", err)
	}
	if strings.Contains(string(data), inst.IncludeLine) {
		t.Errorf("smb.conf still contains include line after uninstall; content:\n%s", string(data))
	}
}

func TestUninstall_SMB_NoIncludeLine(t *testing.T) {
	inst, _ := newTestInstaller(t)
	if err := inst.Uninstall(); err != nil {
		t.Fatalf("Uninstall() error = %v (should be no-op)", err)
	}
	data, err := os.ReadFile(inst.SmbConfPath)
	if err != nil {
		t.Fatalf("reading smb.conf: %v", err)
	}
	if !strings.Contains(string(data), "[global]") {
		t.Errorf("smb.conf was corrupted; content:\n%s", string(data))
	}
}

func TestUninstall_Idempotent(t *testing.T) {
	inst, _ := newTestInstaller(t)
	if err := inst.Install(inst.BinaryPath); err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if err := inst.Uninstall(); err != nil {
		t.Fatalf("First Uninstall() error = %v", err)
	}
	if err := inst.Uninstall(); err != nil {
		t.Fatalf("Second Uninstall() error = %v (should be idempotent)", err)
	}
}

func TestNewInstaller_ReturnsDefaults(t *testing.T) {
	inst := NewInstaller()
	if inst == nil {
		t.Fatal("NewInstaller() returned nil")
	}
	if inst.ExportfsPath != "/usr/sbin/exportfs" {
		t.Errorf("ExportfsPath = %q, want %q", inst.ExportfsPath, "/usr/sbin/exportfs")
	}
	if inst.SmbcontrolPath != "/usr/bin/smbcontrol" {
		t.Errorf("SmbcontrolPath = %q, want %q", inst.SmbcontrolPath, "/usr/bin/smbcontrol")
	}
	if inst.DropInPath != "/etc/systemd/system/smbd.service.d/unas-custom.conf" {
		t.Errorf("DropInPath = %q", inst.DropInPath)
	}
	if inst.DaemonReloadFn == nil {
		t.Error("DaemonReloadFn is nil")
	}
	if inst.Stdout == nil {
		t.Error("Stdout is nil")
	}
}

func TestCopyFile_MissingSource(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "nonexistent")
	dst := filepath.Join(dir, "dst")
	err := copyFile(src, dst)
	if err == nil {
		t.Error("expected error for missing source, got nil")
	}
	if !strings.Contains(err.Error(), "opening") {
		t.Errorf("expected 'opening' in error message, got: %v", err)
	}
}

func TestInstall_DaemonReloadError(t *testing.T) {
	inst, _ := newTestInstaller(t)
	inst.DaemonReloadFn = func() error { return errors.New("daemon reload failed") }
	err := inst.Install(inst.BinaryPath)
	if err == nil {
		t.Error("expected error when daemon reload fails")
	}
	if !strings.Contains(err.Error(), "daemon-reload") {
		t.Errorf("expected 'daemon-reload' in error, got: %v", err)
	}
}

func TestUninstall_DaemonReloadError(t *testing.T) {
	inst, _ := newTestInstaller(t)
	inst.DaemonReloadFn = func() error { return errors.New("daemon reload failed") }
	err := inst.Uninstall()
	if err == nil {
		t.Error("expected error when daemon reload fails during uninstall")
	}
	if !strings.Contains(err.Error(), "daemon-reload") {
		t.Errorf("expected 'daemon-reload' in error, got: %v", err)
	}
}

func TestInstallWrapper_BackupFails(t *testing.T) {
	inst, _ := newTestInstaller(t)
	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "binary")
	os.WriteFile(binaryPath, []byte("bin"), 0755)
	err := inst.installWrapper(
		binaryPath,
		filepath.Join(dir, "nonexistent-target"),
		filepath.Join(dir, "nonexistent-orig"),
	)
	if err == nil {
		t.Error("expected error when target path doesn't exist for backup")
	}
	if !strings.Contains(err.Error(), "backing up") {
		t.Errorf("expected 'backing up' in error, got: %v", err)
	}
}

func TestInstallWrapper_InstallFails(t *testing.T) {
	inst, _ := newTestInstaller(t)
	dir := t.TempDir()
	origPath := filepath.Join(dir, "existing-orig")
	if err := os.WriteFile(origPath, []byte("orig"), 0755); err != nil {
		t.Fatalf("failed to create orig file: %v", err)
	}
	err := inst.installWrapper(
		filepath.Join(dir, "nonexistent-binary"),
		filepath.Join(dir, "target"),
		origPath,
	)
	if err == nil {
		t.Error("expected error when binary doesn't exist")
	}
	if !strings.Contains(err.Error(), "installing wrapper") {
		t.Errorf("expected 'installing wrapper' in error, got: %v", err)
	}
}

func TestNewInstaller_DaemonReloadFnCallable(t *testing.T) {
	inst := NewInstaller()
	_ = inst.DaemonReloadFn()
}

func TestCopyFile_MissingDestDir(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	if err := os.WriteFile(src, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create src: %v", err)
	}
	dst := filepath.Join(dir, "nonexistent-dir", "dst")
	err := copyFile(src, dst)
	if err == nil {
		t.Error("expected error for missing dst dir, got nil")
	}
	if !strings.Contains(err.Error(), "creating") {
		t.Errorf("expected 'creating' in error message, got: %v", err)
	}
}

func TestInstall_BinaryMissing(t *testing.T) {
	inst, _ := newTestInstaller(t)
	inst.BinaryPath = "/nonexistent/binary"
	err := inst.Install("/nonexistent/binary")
	if err == nil {
		t.Error("expected error when binary source is missing")
	}
}

func TestUninstall_AllMissing(t *testing.T) {
	dir := t.TempDir()
	inst := &Installer{
		ExportfsPath:       filepath.Join(dir, "exportfs"),
		ExportfsOrigPath:   filepath.Join(dir, "exportfs.orig"),
		SmbcontrolPath:     filepath.Join(dir, "smbcontrol"),
		SmbcontrolOrigPath: filepath.Join(dir, "smbcontrol.orig"),
		SmbConfPath:        filepath.Join(dir, "smb.conf"),
		IncludeLine:        "include = /persistent/unas-custom/smb-overrides.conf",
		DropInPath:         filepath.Join(dir, "unas-custom.conf"),
		DaemonReloadFn:     func() error { return nil },
		Stdout:             io.Discard,
	}
	err := inst.Uninstall()
	if err != nil {
		t.Errorf("Uninstall() on fresh state should return nil, got %v", err)
	}
}

func TestInstall_DropInContent_ExactMatch(t *testing.T) {
	inst, dir := newTestInstaller(t)
	os.WriteFile(filepath.Join(dir, "unas-custom"), []byte("fake binary"), 0755)

	if err := inst.Install(filepath.Join(dir, "unas-custom")); err != nil {
		t.Fatalf("Install() error: %v", err)
	}

	content, err := os.ReadFile(inst.DropInPath)
	if err != nil {
		t.Fatalf("could not read drop-in: %v", err)
	}

	s := string(content)
	required := []string{
		"ExecStartPre=/persistent/unas-custom/unas-custom smb inject",
		"ExecReload=",
		"ExecReload=/persistent/unas-custom/unas-custom smb inject || true",
		"ExecReload=/bin/kill -HUP $MAINPID",
	}
	for _, line := range required {
		if !strings.Contains(s, line) {
			t.Errorf("drop-in missing required line: %q", line)
		}
	}
}

func TestInstall_ConfigDirCreationFails(t *testing.T) {
	inst, _ := newTestInstaller(t)
	inst.ConfigDir = "/root/nonexistent/config"
	err := inst.Install(inst.BinaryPath)
	if err == nil {
		t.Error("expected error when config dir creation fails")
	}
	if !strings.Contains(err.Error(), "creating config dir") {
		t.Errorf("expected 'creating config dir' in error, got: %v", err)
	}
}

func TestInstall_DropInDirCreationFails(t *testing.T) {
	inst, _ := newTestInstaller(t)
	inst.DropInDir = "/root/nonexistent/drop-in"
	err := inst.Install(inst.BinaryPath)
	if err == nil {
		t.Error("expected error when drop-in dir creation fails")
	}
	if !strings.Contains(err.Error(), "creating systemd drop-in dir") {
		t.Errorf("expected 'creating systemd drop-in dir' in error, got: %v", err)
	}
}

func TestInstall_DropInWriteFails(t *testing.T) {
	inst, _ := newTestInstaller(t)
	inst.DropInPath = "/root/nonexistent/drop-in.conf"
	err := inst.Install(inst.BinaryPath)
	if err == nil {
		t.Error("expected error when drop-in write fails")
	}
	if !strings.Contains(err.Error(), "writing systemd drop-in") {
		t.Errorf("expected 'writing systemd drop-in' in error, got: %v", err)
	}
}

func TestCopyFile_PermissionDenied(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	if err := os.WriteFile(src, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create src: %v", err)
	}
	dst := filepath.Join(dir, "dst")
	if err := os.WriteFile(dst, []byte("existing"), 0644); err != nil {
		t.Fatalf("failed to create dst: %v", err)
	}
	if err := os.Chmod(dir, 0555); err != nil {
		t.Fatalf("failed to chmod dir: %v", err)
	}
	defer os.Chmod(dir, 0755)

	err := copyFile(src, filepath.Join(dir, "new-dst"))
	if err == nil {
		t.Error("expected error when write permission denied")
	}
}

func TestInstallWrapper_OrigAlreadyExists_SkipsBackup(t *testing.T) {
	inst, dir := newTestInstaller(t)
	targetPath := filepath.Join(dir, "exportfs")
	origPath := filepath.Join(dir, "exportfs.orig")
	binaryPath := filepath.Join(dir, "unas-custom")

	os.WriteFile(targetPath, []byte("original-content"), 0755)
	os.WriteFile(origPath, []byte("already-backed-up"), 0755)
	os.WriteFile(binaryPath, []byte("new-binary"), 0755)

	err := inst.installWrapper(binaryPath, targetPath, origPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := os.ReadFile(origPath)
	if string(got) != "already-backed-up" {
		t.Errorf("orig was overwritten: got %q", string(got))
	}

	target, err := os.Readlink(targetPath)
	if err != nil {
		t.Fatalf("target is not a symlink: %v", err)
	}
	if target != binaryPath {
		t.Errorf("target symlink = %q, want %q", target, binaryPath)
	}
}

func TestInstall_CreatesOverridesPlaceholder(t *testing.T) {
	inst, _ := newTestInstaller(t)
	// OverridesPath should not exist yet
	if _, err := os.Stat(inst.OverridesPath); !os.IsNotExist(err) {
		t.Fatal("expected OverridesPath to not exist before install")
	}
	if err := inst.Install(inst.BinaryPath); err != nil {
		t.Fatalf("Install failed: %v", err)
	}
	if _, err := os.Stat(inst.OverridesPath); err != nil {
		t.Errorf("OverridesPath not created: %v", err)
	}
}

func TestInstall_DoesNotOverwriteExistingOverrides(t *testing.T) {
	inst, _ := newTestInstaller(t)
	// Pre-create overrides file with known content
	existing := "# my custom overrides\n"
	if err := os.WriteFile(inst.OverridesPath, []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}
	if err := inst.Install(inst.BinaryPath); err != nil {
		t.Fatalf("Install failed: %v", err)
	}
	got, err := os.ReadFile(inst.OverridesPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != existing {
		t.Errorf("overrides file was overwritten: got %q, want %q", string(got), existing)
	}
}
