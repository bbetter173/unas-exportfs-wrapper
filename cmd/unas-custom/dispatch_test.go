package main

import (
	"os"
	"testing"
)

// TestDetectMode_Exportfs tests that basename "exportfs" returns "nfs-wrapper"
func TestDetectMode_Exportfs(t *testing.T) {
	// Save original os.Args[0]
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"/usr/sbin/exportfs"}
	mode := detectMode()
	if mode != "nfs-wrapper" {
		t.Errorf("detectMode() = %q, want %q", mode, "nfs-wrapper")
	}
}

// TestDetectMode_Smbcontrol tests that basename "smbcontrol" returns "smb-wrapper"
func TestDetectMode_Smbcontrol(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"/usr/sbin/smbcontrol"}
	mode := detectMode()
	if mode != "smb-wrapper" {
		t.Errorf("detectMode() = %q, want %q", mode, "smb-wrapper")
	}
}

// TestDetectMode_UnasCustom tests that basename "unas-custom" returns "cli"
func TestDetectMode_UnasCustom(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"/usr/local/bin/unas-custom"}
	mode := detectMode()
	if mode != "cli" {
		t.Errorf("detectMode() = %q, want %q", mode, "cli")
	}
}

// TestDetectMode_Unknown tests that any other basename returns "cli"
func TestDetectMode_Unknown(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"/some/path/unknown-tool"}
	mode := detectMode()
	if mode != "cli" {
		t.Errorf("detectMode() = %q, want %q", mode, "cli")
	}
}

// TestParseSubcommand_Install tests that args ["install"] returns "install"
func TestParseSubcommand_Install(t *testing.T) {
	cmd := parseSubcommand([]string{"install"})
	if cmd != "install" {
		t.Errorf("parseSubcommand([\"install\"]) = %q, want %q", cmd, "install")
	}
}

// TestParseSubcommand_Uninstall tests that args ["uninstall"] returns "uninstall"
func TestParseSubcommand_Uninstall(t *testing.T) {
	cmd := parseSubcommand([]string{"uninstall"})
	if cmd != "uninstall" {
		t.Errorf("parseSubcommand([\"uninstall\"]) = %q, want %q", cmd, "uninstall")
	}
}

// TestParseSubcommand_Status tests that args ["status"] returns "status"
func TestParseSubcommand_Status(t *testing.T) {
	cmd := parseSubcommand([]string{"status"})
	if cmd != "status" {
		t.Errorf("parseSubcommand([\"status\"]) = %q, want %q", cmd, "status")
	}
}

// TestParseSubcommand_SmbApply tests that args ["smb", "apply"] returns "smb-apply"
func TestParseSubcommand_SmbApply(t *testing.T) {
	cmd := parseSubcommand([]string{"smb", "apply"})
	if cmd != "smb-apply" {
		t.Errorf("parseSubcommand([\"smb\", \"apply\"]) = %q, want %q", cmd, "smb-apply")
	}
}

// TestParseSubcommand_SmbInject tests that args ["smb", "inject"] returns "smb-inject"
func TestParseSubcommand_SmbInject(t *testing.T) {
	cmd := parseSubcommand([]string{"smb", "inject"})
	if cmd != "smb-inject" {
		t.Errorf("parseSubcommand([\"smb\", \"inject\"]) = %q, want %q", cmd, "smb-inject")
	}
}

// TestParseSubcommand_Help tests that args [] returns "help"
func TestParseSubcommand_Help(t *testing.T) {
	cmd := parseSubcommand([]string{})
	if cmd != "help" {
		t.Errorf("parseSubcommand([]) = %q, want %q", cmd, "help")
	}
}

// TestParseSubcommand_HelpFlag tests that args ["--help"] returns "help"
func TestParseSubcommand_HelpFlag(t *testing.T) {
	cmd := parseSubcommand([]string{"--help"})
	if cmd != "help" {
		t.Errorf("parseSubcommand([\"--help\"]) = %q, want %q", cmd, "help")
	}
}

// TestParseSubcommand_Unknown tests that args ["foobar"] returns "unknown"
func TestParseSubcommand_Unknown(t *testing.T) {
	cmd := parseSubcommand([]string{"foobar"})
	if cmd != "unknown" {
		t.Errorf("parseSubcommand([\"foobar\"]) = %q, want %q", cmd, "unknown")
	}
}
