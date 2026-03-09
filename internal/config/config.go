package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type NFSRule struct {
	Path    string `yaml:"path"`
	Options string `yaml:"options"`
}

type NFSConfig struct {
	Rules []NFSRule `yaml:"rules"`
}

type SMBOverride struct {
	Share            string            `yaml:"share"`
	Directives       map[string]string `yaml:"directives"`
	AppendValidUsers []string          `yaml:"append_valid_users"`
}

type SMBConfig struct {
	Overrides []SMBOverride `yaml:"overrides"`
}

type Config struct {
	NFS NFSConfig `yaml:"nfs"`
	SMB SMBConfig `yaml:"smb"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	for i, rule := range cfg.NFS.Rules {
		if rule.Path == "" {
			return nil, fmt.Errorf("nfs rule %d: path cannot be empty", i)
		}
		if rule.Options == "" {
			return nil, fmt.Errorf("nfs rule %d: options cannot be empty", i)
		}
	}

	for i, override := range cfg.SMB.Overrides {
		if override.Share == "" {
			return nil, fmt.Errorf("smb override %d: share cannot be empty", i)
		}
	}

	return &cfg, nil
}

func (c *Config) FindRule(path string) *NFSRule {
	for i := range c.NFS.Rules {
		if c.NFS.Rules[i].Path == path {
			return &c.NFS.Rules[i]
		}
	}
	return nil
}

func (c *Config) FindSMBOverride(share string) *SMBOverride {
	for i := range c.SMB.Overrides {
		if c.SMB.Overrides[i].Share == share {
			return &c.SMB.Overrides[i]
		}
	}
	return nil
}
