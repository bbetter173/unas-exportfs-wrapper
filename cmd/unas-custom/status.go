package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/bbettridge/unas-custom/internal/config"
)

type StatusChecker struct {
	ExportfsPath       string
	ExportfsOrigPath   string
	SmbcontrolPath     string
	SmbcontrolOrigPath string
	WrapperBinaryPath  string
	DropInPath         string
	SmbConfPath        string
	IncludeLine        string
	OverridesPath      string
	ConfigPath         string
	Out                io.Writer
}

func NewStatusChecker() *StatusChecker {
	return &StatusChecker{
		ExportfsPath:       "/usr/sbin/exportfs",
		ExportfsOrigPath:   "/usr/sbin/exportfs.orig",
		SmbcontrolPath:     "/usr/bin/smbcontrol",
		SmbcontrolOrigPath: "/usr/bin/smbcontrol.orig",
		WrapperBinaryPath:  "/persistent/unas-custom/unas-custom",
		DropInPath:         "/etc/systemd/system/smbd.service.d/unas-custom.conf",
		SmbConfPath:        "/etc/samba/smb.conf",
		IncludeLine:        "include = /persistent/unas-custom/smb-overrides.conf",
		OverridesPath:      "/persistent/unas-custom/smb-overrides.conf",
		ConfigPath:         "/persistent/unas-custom/config.yaml",
		Out:                os.Stdout,
	}
}

func (s *StatusChecker) Check() int {
	fmt.Fprintf(s.Out, "unas-custom status\n")
	fmt.Fprintf(s.Out, "──────────────────\n")

	allOK := true

	// Check 1: NFS wrapper
	if isSymlinkTo(s.ExportfsPath, s.WrapperBinaryPath) && fileExists(s.ExportfsOrigPath) {
		fmt.Fprintf(s.Out, "NFS wrapper:      installed ✓ (exportfs → unas-custom, original backed up)\n")
	} else {
		fmt.Fprintf(s.Out, "NFS wrapper:      not installed ✗\n")
		allOK = false
	}

	// Check 2: SMB wrapper
	if isSymlinkTo(s.SmbcontrolPath, s.WrapperBinaryPath) && fileExists(s.SmbcontrolOrigPath) {
		fmt.Fprintf(s.Out, "SMB wrapper:      installed ✓ (smbcontrol → unas-custom, original backed up)\n")
	} else {
		fmt.Fprintf(s.Out, "SMB wrapper:      not installed ✗\n")
		allOK = false
	}

	// Check 3: systemd drop-in
	if fileExists(s.DropInPath) {
		fmt.Fprintf(s.Out, "systemd drop-in:  installed ✓ (smbd.service ExecReload override)\n")
	} else {
		fmt.Fprintf(s.Out, "systemd drop-in:  installed ✗\n")
		allOK = false
	}

	// Check 4: SMB include line
	if s.checkSmbInclude() {
		fmt.Fprintf(s.Out, "SMB include:      present ✓ (include line in smb.conf)\n")
	} else {
		fmt.Fprintf(s.Out, "SMB include:      present ✗\n")
		allOK = false
	}

	// Check 5: SMB overrides file
	if fileExists(s.OverridesPath) {
		fmt.Fprintf(s.Out, "SMB overrides:    generated ✓ (smb-overrides.conf exists)\n")
	} else {
		fmt.Fprintf(s.Out, "SMB overrides:    generated ✗\n")
		allOK = false
	}

	// Check 6: Config file
	if s.checkConfig() {
		fmt.Fprintf(s.Out, "Config:           valid ✓ (/persistent/unas-custom/config.yaml)\n")
	} else {
		fmt.Fprintf(s.Out, "Config:           valid ✗\n")
		allOK = false
	}

	if allOK {
		return 0
	}
	return 1
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func isSymlinkTo(path, expectedTarget string) bool {
	target, err := os.Readlink(path)
	if err != nil {
		return false
	}
	return target == expectedTarget
}

func (s *StatusChecker) checkSmbInclude() bool {
	content, err := os.ReadFile(s.SmbConfPath)
	if err != nil {
		return false
	}
	return strings.Contains(string(content), s.IncludeLine)
}

func (s *StatusChecker) checkConfig() bool {
	if !fileExists(s.ConfigPath) {
		return false
	}
	_, err := config.Load(s.ConfigPath)
	return err == nil
}

func runStatusImpl(args []string) int {
	checker := NewStatusChecker()
	return checker.Check()
}
