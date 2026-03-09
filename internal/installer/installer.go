package installer

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/bbettridge/unas-custom/internal/config"
	"github.com/bbettridge/unas-custom/internal/smb"
)

const dropInContent = `# Managed by unas-custom — do not edit manually
[Service]
ExecStartPre=/persistent/unas-custom/unas-custom smb inject
ExecReload=
ExecReload=/persistent/unas-custom/unas-custom smb inject || true
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

	ConfigPath    string // /persistent/unas-custom/config.yaml
	ShareConfPath string // /etc/samba/share.conf

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

		ConfigPath:    "/persistent/unas-custom/config.yaml",
		ShareConfPath: "/etc/samba/share.conf",

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

	// Generate smb-overrides.conf from config if available, otherwise write placeholder.
	// Skip if overrides file already exists (preserve user/previous content).
	if _, err := os.Stat(i.OverridesPath); os.IsNotExist(err) {
		overridesContent, genErr := i.generateOverridesFromConfig()
		if genErr != nil {
			// Config load failed or no overrides; fall back to placeholder
			overridesContent, _ = smb.GenerateOverrides(nil)
		}
		if err := os.WriteFile(i.OverridesPath, []byte(overridesContent), 0644); err != nil {
			return fmt.Errorf("writing smb overrides: %w", err)
		}
	}

	return nil
}

func (i *Installer) installWrapper(binaryPath, targetPath, origPath string) error {
	if _, err := os.Stat(binaryPath); err != nil {
		return fmt.Errorf("installing wrapper to %s: %w", targetPath, err)
	}
	if _, err := os.Stat(origPath); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("checking backup %s: %w", origPath, err)
		}
		if err := copyFile(targetPath, origPath); err != nil {
			return fmt.Errorf("backing up %s: %w", targetPath, err)
		}
	}
	// Atomic symlink replacement: create temp symlink then rename over target.
	// This ensures targetPath is never missing (no window where it's absent).
	tmpLink := targetPath + ".tmp"
	// Clean up any stale temp symlink
	if err := os.Remove(tmpLink); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing stale temp symlink %s: %w", tmpLink, err)
	}
	if err := os.Symlink(binaryPath, tmpLink); err != nil {
		return fmt.Errorf("creating temp symlink for %s: %w", targetPath, err)
	}
	if err := os.Rename(tmpLink, targetPath); err != nil {
		// Clean up temp symlink on rename failure
		os.Remove(tmpLink)
		return fmt.Errorf("atomically replacing %s: %w", targetPath, err)
	}
	return nil
}

// generateOverridesFromConfig loads config.yaml and generates real overrides content.
// Returns an error if config cannot be loaded or has no SMB overrides.
func (i *Installer) generateOverridesFromConfig() (string, error) {
	cfg, err := config.Load(i.ConfigPath)
	if err != nil {
		return "", fmt.Errorf("loading config: %w", err)
	}
	if len(cfg.SMB.Overrides) == 0 {
		return "", fmt.Errorf("no SMB overrides in config")
	}

	overrides := cfg.SMB.Overrides

	// If any override uses AppendValidUsers, merge with existing share.conf valid users
	hasAppend := false
	for _, o := range overrides {
		if len(o.AppendValidUsers) > 0 {
			hasAppend = true
			break
		}
	}
	if hasAppend {
		shareData, err := os.ReadFile(i.ShareConfPath)
		if err == nil {
			existingVU, parseErr := smb.ParseShareValidUsers(shareData)
			if parseErr == nil {
				overrides = smb.MergeAppendValidUsers(overrides, existingVU)
			}
		}
		// If share.conf read or parse fails, proceed without merging
	}

	return smb.GenerateOverrides(overrides)
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
	if err := os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing %s wrapper: %w", name, err)
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
