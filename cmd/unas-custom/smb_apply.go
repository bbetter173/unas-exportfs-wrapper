package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bbettridge/unas-custom/internal/config"
	"github.com/bbettridge/unas-custom/internal/smb"
)

const (
	smbApplyDefaultConfigPath    = "/persistent/unas-custom/config.yaml"
	smbApplyDefaultOverridesPath = "/persistent/unas-custom/smb-overrides.conf"
)

func runSmbApplyWithPaths(args []string, configPath, overridesPath string) int {
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unas-custom: smb apply: failed to load config: %v\n", err)
		return 1
	}

	content, err := smb.GenerateOverrides(cfg.SMB.Overrides)
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

	fmt.Printf("Generated %d share overrides to %s\n", len(cfg.SMB.Overrides), overridesPath)
	return 0
}
