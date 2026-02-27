package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Rule defines a path-based NFS export modification rule
type Rule struct {
	Path    string `yaml:"path"`
	Options string `yaml:"options"`
}

// Config represents the application configuration
type Config struct {
	Rules []Rule `yaml:"rules"`
}

// Load reads and parses the configuration file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate rules
	for i, rule := range cfg.Rules {
		if rule.Path == "" {
			return nil, fmt.Errorf("rule %d: path cannot be empty", i)
		}
		if rule.Options == "" {
			return nil, fmt.Errorf("rule %d: options cannot be empty", i)
		}
	}

	return &cfg, nil
}

// FindRule returns the rule matching the given path, or nil if not found
func (c *Config) FindRule(path string) *Rule {
	for i := range c.Rules {
		if c.Rules[i].Path == path {
			return &c.Rules[i]
		}
	}
	return nil
}
