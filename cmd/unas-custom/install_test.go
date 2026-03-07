package main

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/bbettridge/unas-custom/internal/installer"
)

func newTestInstallInstaller(t *testing.T) (*installer.Installer, string) {
	t.Helper()
	dir := t.TempDir()
	smb_conf := filepath.Join(dir, "smb.conf")
	os.WriteFile(smb_conf, []byte("[global]\n\tworkgroup = WORKGROUP\n"), 0644)

	return &installer.Installer{
		BinaryPath:         filepath.Join(dir, "unas-custom"),
		ExportfsPath:       filepath.Join(dir, "exportfs"),
		ExportfsOrigPath:   filepath.Join(dir, "exportfs.orig"),
		SmbcontrolPath:     filepath.Join(dir, "smbcontrol"),
		SmbcontrolOrigPath: filepath.Join(dir, "smbcontrol.orig"),
		SmbConfPath:        smb_conf,
		IncludeLine:        "include = /persistent/unas-custom/smb-overrides.conf",
		ConfigDir:          filepath.Join(dir, "config"),
		OverridesPath:      filepath.Join(dir, "smb-overrides.conf"),
		DropInDir:          filepath.Join(dir, "smbd.service.d"),
		DropInPath:         filepath.Join(dir, "smbd.service.d", "unas-custom.conf"),
		OldInstallDir:      filepath.Join(dir, "nfs-intercept"),
		DaemonReloadFn:     func() error { return nil },
		Stdout:             io.Discard,
	}, dir
}

func TestRunInstallImplWithInstaller_Success(t *testing.T) {
	inst, dir := newTestInstallInstaller(t)
	os.WriteFile(filepath.Join(dir, "unas-custom"), []byte("fake binary"), 0755)
	os.WriteFile(filepath.Join(dir, "exportfs"), []byte("original exportfs"), 0755)
	os.WriteFile(filepath.Join(dir, "smbcontrol"), []byte("original smbcontrol"), 0755)

	result := runInstallImplWithInstaller(inst, filepath.Join(dir, "unas-custom"))
	if result != 0 {
		t.Errorf("expected exit 0, got %d", result)
	}
}

func TestRunInstallImplWithInstaller_InstallFails(t *testing.T) {
	inst, _ := newTestInstallInstaller(t)
	result := runInstallImplWithInstaller(inst, "/nonexistent/path/unas-custom")
	if result != 1 {
		t.Errorf("expected exit 1 for missing binary, got %d", result)
	}
}

func TestRunUninstallImplWithInstaller_Success(t *testing.T) {
	inst, dir := newTestInstallInstaller(t)
	os.WriteFile(filepath.Join(dir, "exportfs.orig"), []byte("original exportfs"), 0755)
	os.WriteFile(filepath.Join(dir, "exportfs"), []byte("our binary"), 0755)
	os.WriteFile(filepath.Join(dir, "smbcontrol.orig"), []byte("original smbcontrol"), 0755)
	os.WriteFile(filepath.Join(dir, "smbcontrol"), []byte("our binary"), 0755)

	result := runUninstallImplWithInstaller(inst)
	if result != 0 {
		t.Errorf("expected exit 0, got %d", result)
	}
}

func TestRunUninstallImplWithInstaller_NoBackup(t *testing.T) {
	inst, _ := newTestInstallInstaller(t)
	result := runUninstallImplWithInstaller(inst)
	if result != 0 {
		t.Errorf("expected exit 0 for uninstall with no backup, got %d", result)
	}
}

func TestRunUninstallImplWithInstaller_ErrorHandling(t *testing.T) {
	inst, _ := newTestInstallInstaller(t)
	inst.DaemonReloadFn = func() error {
		return os.ErrPermission
	}
	result := runUninstallImplWithInstaller(inst)
	if result != 1 {
		t.Errorf("expected exit 1 on uninstall error, got %d", result)
	}
}

func TestRunInstallImpl_WithGetExecutable(t *testing.T) {
	inst, dir := newTestInstallInstaller(t)
	os.WriteFile(filepath.Join(dir, "unas-custom"), []byte("fake binary"), 0755)
	os.WriteFile(filepath.Join(dir, "exportfs"), []byte("original exportfs"), 0755)
	os.WriteFile(filepath.Join(dir, "smbcontrol"), []byte("original smbcontrol"), 0755)

	oldGetExecutable := getExecutable
	oldNewInstaller := newInstallerFn
	getExecutable = func() (string, error) {
		return filepath.Join(dir, "unas-custom"), nil
	}
	newInstallerFn = func() *installer.Installer {
		return inst
	}
	defer func() {
		getExecutable = oldGetExecutable
		newInstallerFn = oldNewInstaller
	}()

	result := runInstallImpl([]string{})
	if result != 0 {
		t.Errorf("expected exit 0, got %d", result)
	}
}

func TestRunInstallImpl_GetExecutableFails(t *testing.T) {
	oldGetExecutable := getExecutable
	getExecutable = func() (string, error) {
		return "", os.ErrPermission
	}
	defer func() { getExecutable = oldGetExecutable }()

	result := runInstallImpl([]string{})
	if result != 1 {
		t.Errorf("expected exit 1 when getExecutable fails, got %d", result)
	}
}

func TestRunUninstallImpl_WithValidState(t *testing.T) {
	inst, dir := newTestInstallInstaller(t)
	os.WriteFile(filepath.Join(dir, "exportfs.orig"), []byte("original exportfs"), 0755)
	os.WriteFile(filepath.Join(dir, "exportfs"), []byte("our binary"), 0755)
	os.WriteFile(filepath.Join(dir, "smbcontrol.orig"), []byte("original smbcontrol"), 0755)
	os.WriteFile(filepath.Join(dir, "smbcontrol"), []byte("our binary"), 0755)

	oldNewInstaller := newInstallerFn
	newInstallerFn = func() *installer.Installer {
		return inst
	}
	defer func() { newInstallerFn = oldNewInstaller }()

	result := runUninstallImpl([]string{})
	if result != 0 {
		t.Errorf("expected exit 0, got %d", result)
	}
}
