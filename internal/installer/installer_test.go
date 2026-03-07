package installer

import (
	"bytes"
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
	data, err = os.ReadFile(inst.ExportfsPath)
	if err != nil {
		t.Fatalf("exportfs not installed: %v", err)
	}
	if string(data) != "fake binary" {
		t.Errorf("exportfs = %q, want %q", string(data), "fake binary")
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
	data, _ = os.ReadFile(inst.ExportfsPath)
	if string(data) != "fake binary" {
		t.Errorf("exportfs not updated; got %q, want %q", string(data), "fake binary")
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
	data, _ = os.ReadFile(inst.SmbcontrolPath)
	if string(data) != "fake binary" {
		t.Errorf("smbcontrol = %q, want %q", string(data), "fake binary")
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
		"ExecStartPost=/persistent/unas-custom/unas-custom smb inject",
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
