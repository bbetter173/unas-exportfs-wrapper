package installer

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/bbettridge/unas-custom/internal/smb"
)

const dropInContent = `# Managed by unas-custom — do not edit manually
[Service]
ExecStartPost=/persistent/unas-custom/unas-custom smb inject
ExecReload=
ExecReload=/persistent/unas-custom/unas-custom smb inject
ExecReload=/bin/kill -HUP $MAINPID
`

type Installer struct {
	BinaryPath string

	ExportfsPath     string // /usr/sbin/exportfs
	ExportfsOrigPath string // /usr/sbin/exportfs.orig

	SmbcontrolPath     string // /usr/bin/smbcontrol
	SmbcontrolOrigPath string // /usr/bin/smbcontrol.orig

	SmbConfPath   string // /etc/samba/smb.conf
	IncludeLine   string // "include = /persistent/unas-custom/smb-overrides.conf"
	ConfigDir     string // /persistent/unas-custom
	OverridesPath string // /persistent/unas-custom/smb-overrides.conf

	DropInDir  string // /etc/systemd/system/smbd.service.d
	DropInPath string // /etc/systemd/system/smbd.service.d/unas-custom.conf

	OldInstallDir string // /persistent/nfs-intercept

	DaemonReloadFn func() error
	Stdout         io.Writer
}

func NewInstaller() *Installer {
	return &Installer{
		ExportfsPath:     "/usr/sbin/exportfs",
		ExportfsOrigPath: "/usr/sbin/exportfs.orig",

		SmbcontrolPath:     "/usr/bin/smbcontrol",
		SmbcontrolOrigPath: "/usr/bin/smbcontrol.orig",

		SmbConfPath:   "/etc/samba/smb.conf",
		IncludeLine:   "include = /persistent/unas-custom/smb-overrides.conf",
		ConfigDir:     "/persistent/unas-custom",
		OverridesPath: "/persistent/unas-custom/smb-overrides.conf",

		DropInDir:  "/etc/systemd/system/smbd.service.d",
		DropInPath: "/etc/systemd/system/smbd.service.d/unas-custom.conf",

		OldInstallDir: "/persistent/nfs-intercept",

		DaemonReloadFn: func() error {
			return exec.Command("systemctl", "daemon-reload").Run()
		},
		Stdout: os.Stdout,
	}
}

func (i *Installer) Install(binaryPath string) error {
	if err := os.MkdirAll(i.ConfigDir, 0755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	if _, err := os.Stat(i.OldInstallDir); err == nil {
		fmt.Fprintf(i.Stdout, "unas-custom: notice: old installation found at %s\n", i.OldInstallDir)
		fmt.Fprintf(i.Stdout, "unas-custom: notice: manual migration may be required; old directory not removed\n")
	}

	if err := i.installWrapper(binaryPath, i.ExportfsPath, i.ExportfsOrigPath); err != nil {
		return fmt.Errorf("installing NFS wrapper: %w", err)
	}

	if err := i.installWrapper(binaryPath, i.SmbcontrolPath, i.SmbcontrolOrigPath); err != nil {
		return fmt.Errorf("installing SMB wrapper: %w", err)
	}

	if err := os.MkdirAll(i.DropInDir, 0755); err != nil {
		return fmt.Errorf("creating systemd drop-in dir: %w", err)
	}

	if err := os.WriteFile(i.DropInPath, []byte(dropInContent), 0644); err != nil {
		return fmt.Errorf("writing systemd drop-in: %w", err)
	}

	if err := i.DaemonReloadFn(); err != nil {
		return fmt.Errorf("daemon-reload: %w", err)
	}

	if err := smb.Inject(i.SmbConfPath, i.IncludeLine); err != nil {
		return fmt.Errorf("injecting smb include: %w", err)
	}

	return nil
}

func (i *Installer) installWrapper(binaryPath, targetPath, origPath string) error {
	if _, err := os.Stat(origPath); os.IsNotExist(err) {
		if err := copyFile(targetPath, origPath); err != nil {
			return fmt.Errorf("backing up %s: %w", targetPath, err)
		}
	}
	if err := copyFile(binaryPath, targetPath); err != nil {
		return fmt.Errorf("installing wrapper to %s: %w", targetPath, err)
	}
	return nil
}

func (i *Installer) Uninstall() error {
	if err := i.uninstallWrapper(i.ExportfsPath, i.ExportfsOrigPath, "NFS"); err != nil {
		return err
	}

	if err := i.uninstallWrapper(i.SmbcontrolPath, i.SmbcontrolOrigPath, "SMB"); err != nil {
		return err
	}

	if err := os.Remove(i.DropInPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing systemd drop-in: %w", err)
	}

	if err := i.DaemonReloadFn(); err != nil {
		return fmt.Errorf("daemon-reload: %w", err)
	}

	if err := smb.Remove(i.SmbConfPath, i.IncludeLine); err != nil {
		return fmt.Errorf("removing smb include: %w", err)
	}

	return nil
}

func (i *Installer) uninstallWrapper(targetPath, origPath, name string) error {
	if _, err := os.Stat(origPath); os.IsNotExist(err) {
		fmt.Fprintf(i.Stdout, "unas-custom: %s wrapper not installed (no backup found at %s)\n", name, origPath)
		return nil
	}
	if err := copyFile(origPath, targetPath); err != nil {
		return fmt.Errorf("restoring %s original: %w", name, err)
	}
	if err := os.Remove(origPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing %s backup: %w", name, err)
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening %s: %w", src, err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("creating %s: %w", dst, err)
	}

	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return fmt.Errorf("copying %s to %s: %w", src, dst, err)
	}

	if err := out.Close(); err != nil {
		return fmt.Errorf("closing %s: %w", dst, err)
	}

	if err := os.Chmod(dst, 0755); err != nil {
		return fmt.Errorf("setting permissions on %s: %w", dst, err)
	}

	return nil
}
