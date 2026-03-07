package main

import (
	"testing"
)

func TestRunCLI_NoArgs(t *testing.T) {
	code := runCLI([]string{})
	if code != 0 {
		t.Errorf("runCLI([]) = %d, want 0", code)
	}
}

func TestRunCLI_HelpFlag(t *testing.T) {
	code := runCLI([]string{"--help"})
	if code != 0 {
		t.Errorf("runCLI([--help]) = %d, want 0", code)
	}
}

func TestRunCLI_ShortHelpFlag(t *testing.T) {
	code := runCLI([]string{"-h"})
	if code != 0 {
		t.Errorf("runCLI([-h]) = %d, want 0", code)
	}
}

func TestRunCLI_UnknownCmd(t *testing.T) {
	code := runCLI([]string{"badcmd"})
	if code != 1 {
		t.Errorf("runCLI([badcmd]) = %d, want 1", code)
	}
}

func TestRunCLI_Status(t *testing.T) {
	code := runCLI([]string{"status"})
	if code != 0 && code != 1 {
		t.Errorf("runCLI([status]) = %d, want 0 or 1", code)
	}
}

func TestRunCLI_SmbInject(t *testing.T) {
	code := runCLI([]string{"smb", "inject"})
	if code != 0 && code != 1 {
		t.Errorf("runCLI([smb inject]) = %d, want 0 or 1", code)
	}
}

func TestRunCLI_SmbApply(t *testing.T) {
	code := runCLI([]string{"smb", "apply"})
	if code != 0 && code != 1 {
		t.Errorf("runCLI([smb apply]) = %d, want 0 or 1", code)
	}
}

func TestRunNfsWrapper_SystemPath(t *testing.T) {
	code := runNfsWrapper(nil)
	if code != 0 && code != 1 {
		t.Errorf("runNfsWrapper(nil) = %d, want 0 or 1", code)
	}
}

func TestRunSmbcontrolWrapper_SystemPath(t *testing.T) {
	code := runSmbcontrolWrapper(nil)
	if code != 0 && code != 1 {
		t.Errorf("runSmbcontrolWrapper(nil) = %d, want 0 or 1", code)
	}
}

func TestRunCLI_Uninstall(t *testing.T) {
	code := runCLI([]string{"uninstall"})
	if code != 0 && code != 1 {
		t.Errorf("runCLI([uninstall]) = %d, want 0 or 1", code)
	}
}
