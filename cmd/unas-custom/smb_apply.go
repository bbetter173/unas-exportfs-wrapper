package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bbettridge/unas-custom/internal/config"
	"github.com/bbettridge/unas-custom/internal/smb"
)

func hasAppendValidUsers(overrides []config.SMBOverride) bool {
	for _, o := range overrides {
		if len(o.AppendValidUsers) > 0 {
			return true
		}
	}
	return false
}

const (
	smbApplyDefaultConfigPath    = "/persistent/unas-custom/config.yaml"
	smbApplyDefaultOverridesPath = "/persistent/unas-custom/smb-overrides.conf"
	smbApplyDefaultShareConfPath = "/etc/samba/share.conf"
)

func runSmbApplyWithPaths(args []string, configPath, overridesPath, shareConfPath string) int {
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unas-custom: smb apply: failed to load config: %v\n", err)
		return 1
	}

	overrides := cfg.SMB.Overrides

	if hasAppendValidUsers(overrides) {
		shareData, err := os.ReadFile(shareConfPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "unas-custom: smb apply: failed to read share config for valid users: %v\n", err)
			return 1
		}
		existingValidUsers, err := smb.ParseShareValidUsers(shareData)
		if err != nil {
			fmt.Fprintf(os.Stderr, "unas-custom: smb apply: failed to parse share config: %v\n", err)
			return 1
		}
		overrides = smb.MergeAppendValidUsers(overrides, existingValidUsers)
	}

	content, err := smb.GenerateOverrides(overrides)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unas-custom: smb apply: failed to generate overrides: %v\n", err)
		return 1
	}

	if err := os.MkdirAll(filepath.Dir(overridesPath), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "unas-custom: smb apply: failed to create directory: %v\n", err)
		return 1
	}

	if err := os.WriteFile(overridesPath, []byte(content), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "unas-custom: smb apply: failed to write overrides file: %v\n", err)
		return 1
	}

	fmt.Printf("Generated %d share overrides to %s\n", len(overrides), overridesPath)
	return 0
}
