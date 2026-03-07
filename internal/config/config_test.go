package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadUnifiedConfig tests loading YAML with both nfs: and smb: sections
func TestLoadUnifiedConfig(t *testing.T) {
	yaml := `nfs:
  rules:
    - path: "/var/nfs/shared/Backups"
      options: "rw,sync,no_subtree_check"
    - path: "/var/nfs/shared/Media"
      options: "rw,sync,insecure"
smb:
  overrides:
    - share: "Media"
      directives:
        guest ok: "yes"
        create mask: "0664"
    - share: "Backups"
      directives:
        read only: "no"
`
	tmpFile := createTempConfig(t, yaml)
	defer os.Remove(tmpFile)

	cfg, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify NFS section
	if len(cfg.NFS.Rules) != 2 {
		t.Errorf("expected 2 NFS rules, got %d", len(cfg.NFS.Rules))
	}
	if cfg.NFS.Rules[0].Path != "/var/nfs/shared/Backups" {
		t.Errorf("expected path '/var/nfs/shared/Backups', got '%s'", cfg.NFS.Rules[0].Path)
	}
	if cfg.NFS.Rules[0].Options != "rw,sync,no_subtree_check" {
		t.Errorf("expected options 'rw,sync,no_subtree_check', got '%s'", cfg.NFS.Rules[0].Options)
	}

	// Verify SMB section
	if len(cfg.SMB.Overrides) != 2 {
		t.Errorf("expected 2 SMB overrides, got %d", len(cfg.SMB.Overrides))
	}
	if cfg.SMB.Overrides[0].Share != "Media" {
		t.Errorf("expected share 'Media', got '%s'", cfg.SMB.Overrides[0].Share)
	}
	if cfg.SMB.Overrides[0].Directives["guest ok"] != "yes" {
		t.Errorf("expected directive 'guest ok: yes', got '%s'", cfg.SMB.Overrides[0].Directives["guest ok"])
	}
}

// TestLoadNFSOnly tests loading YAML with only nfs: section
func TestLoadNFSOnly(t *testing.T) {
	yaml := `nfs:
  rules:
    - path: "/var/nfs/shared/Data"
      options: "rw,sync"
`
	tmpFile := createTempConfig(t, yaml)
	defer os.Remove(tmpFile)

	cfg, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(cfg.NFS.Rules) != 1 {
		t.Errorf("expected 1 NFS rule, got %d", len(cfg.NFS.Rules))
	}
	if len(cfg.SMB.Overrides) != 0 {
		t.Errorf("expected 0 SMB overrides, got %d", len(cfg.SMB.Overrides))
	}
}

// TestLoadSMBOnly tests loading YAML with only smb: section
func TestLoadSMBOnly(t *testing.T) {
	yaml := `smb:
  overrides:
    - share: "Documents"
      directives:
        read only: "no"
`
	tmpFile := createTempConfig(t, yaml)
	defer os.Remove(tmpFile)

	cfg, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(cfg.NFS.Rules) != 0 {
		t.Errorf("expected 0 NFS rules, got %d", len(cfg.NFS.Rules))
	}
	if len(cfg.SMB.Overrides) != 1 {
		t.Errorf("expected 1 SMB override, got %d", len(cfg.SMB.Overrides))
	}
}

// TestLoadEmptySections tests that empty sections are valid
func TestLoadEmptySections(t *testing.T) {
	yaml := `nfs:
  rules: []
smb:
  overrides: []
`
	tmpFile := createTempConfig(t, yaml)
	defer os.Remove(tmpFile)

	cfg, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(cfg.NFS.Rules) != 0 {
		t.Errorf("expected 0 NFS rules, got %d", len(cfg.NFS.Rules))
	}
	if len(cfg.SMB.Overrides) != 0 {
		t.Errorf("expected 0 SMB overrides, got %d", len(cfg.SMB.Overrides))
	}
}

// TestLoadMissingFile tests that missing file returns error
func TestLoadMissingFile(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

// TestLoadMalformedYAML tests that malformed YAML returns error
func TestLoadMalformedYAML(t *testing.T) {
	yaml := `nfs:
  rules:
    - path: "/var/nfs"
      options: "rw"
    invalid yaml here: [
`
	tmpFile := createTempConfig(t, yaml)
	defer os.Remove(tmpFile)

	_, err := Load(tmpFile)
	if err == nil {
		t.Fatal("expected error for malformed YAML, got nil")
	}
}

// TestFindRule_Match tests that FindRule returns matching NFSRule
func TestFindRule_Match(t *testing.T) {
	yaml := `nfs:
  rules:
    - path: "/var/nfs/shared/Backups"
      options: "rw,sync"
    - path: "/var/nfs/shared/Media"
      options: "rw,insecure"
`
	tmpFile := createTempConfig(t, yaml)
	defer os.Remove(tmpFile)

	cfg, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	rule := cfg.FindRule("/var/nfs/shared/Media")
	if rule == nil {
		t.Fatal("expected to find rule, got nil")
	}
	if rule.Path != "/var/nfs/shared/Media" {
		t.Errorf("expected path '/var/nfs/shared/Media', got '%s'", rule.Path)
	}
	if rule.Options != "rw,insecure" {
		t.Errorf("expected options 'rw,insecure', got '%s'", rule.Options)
	}
}

// TestFindRule_NoMatch tests that FindRule returns nil for non-matching path
func TestFindRule_NoMatch(t *testing.T) {
	yaml := `nfs:
  rules:
    - path: "/var/nfs/shared/Backups"
      options: "rw,sync"
`
	tmpFile := createTempConfig(t, yaml)
	defer os.Remove(tmpFile)

	cfg, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	rule := cfg.FindRule("/nonexistent/path")
	if rule != nil {
		t.Errorf("expected nil for non-matching path, got %v", rule)
	}
}

// TestFindSMBOverride_Match tests that FindSMBOverride returns matching SMBOverride
func TestFindSMBOverride_Match(t *testing.T) {
	yaml := `smb:
  overrides:
    - share: "Media"
      directives:
        guest ok: "yes"
    - share: "Backups"
      directives:
        read only: "no"
`
	tmpFile := createTempConfig(t, yaml)
	defer os.Remove(tmpFile)

	cfg, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	override := cfg.FindSMBOverride("Backups")
	if override == nil {
		t.Fatal("expected to find override, got nil")
	}
	if override.Share != "Backups" {
		t.Errorf("expected share 'Backups', got '%s'", override.Share)
	}
	if override.Directives["read only"] != "no" {
		t.Errorf("expected directive 'read only: no', got '%s'", override.Directives["read only"])
	}
}

// TestFindSMBOverride_NoMatch tests that FindSMBOverride returns nil for non-matching share
func TestFindSMBOverride_NoMatch(t *testing.T) {
	yaml := `smb:
  overrides:
    - share: "Media"
      directives:
        guest ok: "yes"
`
	tmpFile := createTempConfig(t, yaml)
	defer os.Remove(tmpFile)

	cfg, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	override := cfg.FindSMBOverride("NonExistent")
	if override != nil {
		t.Errorf("expected nil for non-matching share, got %v", override)
	}
}

// TestConfigValidation_EmptyShareName tests that Load returns error for empty share name
func TestConfigValidation_EmptyShareName(t *testing.T) {
	yaml := `smb:
  overrides:
    - share: ""
      directives:
        guest ok: "yes"
`
	tmpFile := createTempConfig(t, yaml)
	defer os.Remove(tmpFile)

	_, err := Load(tmpFile)
	if err == nil {
		t.Fatal("expected error for empty share name, got nil")
	}
}

// Helper function to create temporary config file
func createTempConfig(t *testing.T, content string) string {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create temp config file: %v", err)
	}
	return tmpFile
}
