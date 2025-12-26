package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// SaveConfig provides methods to save configuration values.
type SaveConfig struct {
	// GlobalConfigDir is the directory under ~/.config/ for global config.
	GlobalConfigDir string

	// GlobalConfigFile is the filename. Defaults to "config.yaml".
	GlobalConfigFile string

	// LocalConfigName is the filename for local config in git root.
	LocalConfigName string

	// ValidGlobalKeys lists keys that can be set in global config.
	ValidGlobalKeys []string

	// ValidLocalKeys lists keys that can be set in local config.
	ValidLocalKeys []string
}

func (c SaveConfig) globalConfigFile() string {
	if c.GlobalConfigFile != "" {
		return c.GlobalConfigFile
	}
	return "config.yaml"
}

// SaveGlobal saves a key-value pair to the global config file.
func (c SaveConfig) SaveGlobal(key, value string) error {
	if c.GlobalConfigDir == "" {
		return fmt.Errorf("global config directory not configured")
	}

	// Validate key
	if len(c.ValidGlobalKeys) > 0 && !contains(c.ValidGlobalKeys, key) {
		return fmt.Errorf("unknown global config key: %s\n\nValid keys: %s",
			key, strings.Join(c.ValidGlobalKeys, ", "))
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(home, ".config", c.GlobalConfigDir, c.globalConfigFile())

	// Load existing config
	var existing map[string]interface{}
	if data, readErr := os.ReadFile(configPath); readErr == nil {
		_ = yaml.Unmarshal(data, &existing)
	}
	if existing == nil {
		existing = make(map[string]interface{})
	}

	// Update value
	existing[key] = parseValue(value)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		return err
	}

	// Write config
	data, err := yaml.Marshal(existing)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0o600)
}

// SaveLocal saves a key-value pair to the local config file in the git root.
func (c SaveConfig) SaveLocal(gitRoot, key, value string) error {
	if gitRoot == "" {
		return fmt.Errorf("git root not found")
	}
	if c.LocalConfigName == "" {
		return fmt.Errorf("local config name not configured")
	}

	// Validate key
	if len(c.ValidLocalKeys) > 0 && !contains(c.ValidLocalKeys, key) {
		return fmt.Errorf("unknown local config key: %s\n\nValid keys: %s",
			key, strings.Join(c.ValidLocalKeys, ", "))
	}

	configPath := filepath.Join(gitRoot, c.LocalConfigName)

	// Load existing config
	var existing map[string]interface{}
	if data, readErr := os.ReadFile(configPath); readErr == nil {
		_ = yaml.Unmarshal(data, &existing)
	}
	if existing == nil {
		existing = make(map[string]interface{})
	}

	// Update value
	existing[key] = parseValue(value)

	// Write config
	data, err := yaml.Marshal(existing)
	if err != nil {
		return err
	}

	// Local config is shared and should be readable
	return os.WriteFile(configPath, data, 0o644) //nolint:gosec
}

// DeleteGlobalKey removes a key from the global config.
func (c SaveConfig) DeleteGlobalKey(key string) error {
	if c.GlobalConfigDir == "" {
		return fmt.Errorf("global config directory not configured")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(home, ".config", c.GlobalConfigDir, c.globalConfigFile())

	// Load existing config
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil // Nothing to delete
	}

	var existing map[string]interface{}
	if err := yaml.Unmarshal(data, &existing); err != nil {
		return nil
	}

	delete(existing, key)

	// Write back
	data, err = yaml.Marshal(existing)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0o600)
}

// parseValue converts string values to appropriate types for YAML.
func parseValue(value string) interface{} {
	lower := strings.ToLower(value)
	if lower == "true" {
		return true
	}
	if lower == "false" {
		return false
	}
	return value
}
