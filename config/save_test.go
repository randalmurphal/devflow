package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSaveConfig_SaveGlobal(t *testing.T) {
	// Create temp home directory
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	cfg := SaveConfig{
		GlobalConfigDir: "testapp",
		ValidGlobalKeys: []string{"api_url", "no_color"},
	}

	t.Run("creates config file", func(t *testing.T) {
		err := cfg.SaveGlobal("api_url", "http://example.com")
		if err != nil {
			t.Fatalf("SaveGlobal() error = %v", err)
		}

		// Verify file exists
		configPath := filepath.Join(tmpHome, ".config", "testapp", "config.yaml")
		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("ReadFile() error = %v", err)
		}

		var saved map[string]interface{}
		if err := yaml.Unmarshal(data, &saved); err != nil {
			t.Fatalf("Unmarshal() error = %v", err)
		}

		if saved["api_url"] != "http://example.com" {
			t.Errorf("api_url = %v, want http://example.com", saved["api_url"])
		}
	})

	t.Run("updates existing config", func(t *testing.T) {
		err := cfg.SaveGlobal("no_color", "true")
		if err != nil {
			t.Fatalf("SaveGlobal() error = %v", err)
		}

		configPath := filepath.Join(tmpHome, ".config", "testapp", "config.yaml")
		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("ReadFile() error = %v", err)
		}

		var saved map[string]interface{}
		if err := yaml.Unmarshal(data, &saved); err != nil {
			t.Fatalf("Unmarshal() error = %v", err)
		}

		// Should have both keys
		if saved["api_url"] != "http://example.com" {
			t.Errorf("api_url = %v, want http://example.com", saved["api_url"])
		}
		if saved["no_color"] != true {
			t.Errorf("no_color = %v, want true", saved["no_color"])
		}
	})

	t.Run("rejects invalid key", func(t *testing.T) {
		err := cfg.SaveGlobal("invalid_key", "value")
		if err == nil {
			t.Error("expected error for invalid key")
		}
		if !strings.Contains(err.Error(), "unknown global config key") {
			t.Errorf("error = %v, want to contain 'unknown global config key'", err)
		}
	})

	t.Run("no global config dir", func(t *testing.T) {
		emptyCfg := SaveConfig{}
		err := emptyCfg.SaveGlobal("key", "value")
		if err == nil {
			t.Error("expected error when GlobalConfigDir not set")
		}
	})

	t.Run("allows any key when ValidGlobalKeys empty", func(t *testing.T) {
		noValidationCfg := SaveConfig{
			GlobalConfigDir: "novalidation",
		}
		err := noValidationCfg.SaveGlobal("any_key", "any_value")
		if err != nil {
			t.Fatalf("SaveGlobal() error = %v", err)
		}
	})

	t.Run("custom config filename", func(t *testing.T) {
		customCfg := SaveConfig{
			GlobalConfigDir:  "customfile",
			GlobalConfigFile: "settings.yaml",
		}
		err := customCfg.SaveGlobal("key", "value")
		if err != nil {
			t.Fatalf("SaveGlobal() error = %v", err)
		}

		// Verify custom filename used
		configPath := filepath.Join(tmpHome, ".config", "customfile", "settings.yaml")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Error("expected settings.yaml to be created")
		}
	})
}

func TestSaveConfig_SaveLocal(t *testing.T) {
	cfg := SaveConfig{
		LocalConfigName: ".testapp.yaml",
		ValidLocalKeys:  []string{"project_id", "api_url"},
	}

	t.Run("creates local config", func(t *testing.T) {
		tmpDir := t.TempDir()
		err := cfg.SaveLocal(tmpDir, "project_id", "proj-123")
		if err != nil {
			t.Fatalf("SaveLocal() error = %v", err)
		}

		configPath := filepath.Join(tmpDir, ".testapp.yaml")
		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("ReadFile() error = %v", err)
		}

		var saved map[string]interface{}
		if err := yaml.Unmarshal(data, &saved); err != nil {
			t.Fatalf("Unmarshal() error = %v", err)
		}

		if saved["project_id"] != "proj-123" {
			t.Errorf("project_id = %v, want proj-123", saved["project_id"])
		}
	})

	t.Run("updates existing config", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create initial config
		err := cfg.SaveLocal(tmpDir, "project_id", "proj-123")
		if err != nil {
			t.Fatalf("SaveLocal() error = %v", err)
		}

		// Update with second key
		err = cfg.SaveLocal(tmpDir, "api_url", "http://local.dev")
		if err != nil {
			t.Fatalf("SaveLocal() error = %v", err)
		}

		configPath := filepath.Join(tmpDir, ".testapp.yaml")
		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("ReadFile() error = %v", err)
		}

		var saved map[string]interface{}
		if err := yaml.Unmarshal(data, &saved); err != nil {
			t.Fatalf("Unmarshal() error = %v", err)
		}

		if saved["project_id"] != "proj-123" {
			t.Errorf("project_id = %v, want proj-123", saved["project_id"])
		}
		if saved["api_url"] != "http://local.dev" {
			t.Errorf("api_url = %v, want http://local.dev", saved["api_url"])
		}
	})

	t.Run("rejects invalid key", func(t *testing.T) {
		tmpDir := t.TempDir()
		err := cfg.SaveLocal(tmpDir, "invalid_key", "value")
		if err == nil {
			t.Error("expected error for invalid key")
		}
		if !strings.Contains(err.Error(), "unknown local config key") {
			t.Errorf("error = %v, want to contain 'unknown local config key'", err)
		}
	})

	t.Run("empty git root", func(t *testing.T) {
		err := cfg.SaveLocal("", "project_id", "value")
		if err == nil {
			t.Error("expected error when git root empty")
		}
	})

	t.Run("no local config name", func(t *testing.T) {
		emptyCfg := SaveConfig{}
		err := emptyCfg.SaveLocal("/tmp", "key", "value")
		if err == nil {
			t.Error("expected error when LocalConfigName not set")
		}
	})
}

func TestSaveConfig_DeleteGlobalKey(t *testing.T) {
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	cfg := SaveConfig{
		GlobalConfigDir: "testdelete",
	}

	t.Run("deletes existing key", func(t *testing.T) {
		// Create config with multiple keys
		err := cfg.SaveGlobal("key1", "value1")
		if err != nil {
			t.Fatalf("SaveGlobal() error = %v", err)
		}
		err = cfg.SaveGlobal("key2", "value2")
		if err != nil {
			t.Fatalf("SaveGlobal() error = %v", err)
		}

		// Delete one key
		err = cfg.DeleteGlobalKey("key1")
		if err != nil {
			t.Fatalf("DeleteGlobalKey() error = %v", err)
		}

		// Verify key1 deleted, key2 remains
		configPath := filepath.Join(tmpHome, ".config", "testdelete", "config.yaml")
		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("ReadFile() error = %v", err)
		}

		var saved map[string]interface{}
		if err := yaml.Unmarshal(data, &saved); err != nil {
			t.Fatalf("Unmarshal() error = %v", err)
		}

		if _, exists := saved["key1"]; exists {
			t.Error("key1 should have been deleted")
		}
		if saved["key2"] != "value2" {
			t.Errorf("key2 = %v, want value2", saved["key2"])
		}
	})

	t.Run("no error when file doesn't exist", func(t *testing.T) {
		newCfg := SaveConfig{
			GlobalConfigDir: "nonexistent",
		}
		err := newCfg.DeleteGlobalKey("any_key")
		if err != nil {
			t.Errorf("DeleteGlobalKey() error = %v, want nil", err)
		}
	})

	t.Run("no global config dir", func(t *testing.T) {
		emptyCfg := SaveConfig{}
		err := emptyCfg.DeleteGlobalKey("key")
		if err == nil {
			t.Error("expected error when GlobalConfigDir not set")
		}
	})
}

func TestParseValue(t *testing.T) {
	tests := []struct {
		input string
		want  interface{}
	}{
		{"true", true},
		{"TRUE", true},
		{"True", true},
		{"false", false},
		{"FALSE", false},
		{"False", false},
		{"hello", "hello"},
		{"123", "123"}, // Numbers stay as strings
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseValue(tt.input)
			if got != tt.want {
				t.Errorf("parseValue(%q) = %v (%T), want %v (%T)",
					tt.input, got, got, tt.want, tt.want)
			}
		})
	}
}

func TestSaveConfig_globalConfigFile(t *testing.T) {
	t.Run("default filename", func(t *testing.T) {
		cfg := SaveConfig{}
		if got := cfg.globalConfigFile(); got != "config.yaml" {
			t.Errorf("globalConfigFile() = %q, want %q", got, "config.yaml")
		}
	})

	t.Run("custom filename", func(t *testing.T) {
		cfg := SaveConfig{GlobalConfigFile: "custom.yaml"}
		if got := cfg.globalConfigFile(); got != "custom.yaml" {
			t.Errorf("globalConfigFile() = %q, want %q", got, "custom.yaml")
		}
	})
}

func TestSaveConfig_MalformedYAML(t *testing.T) {
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	cfg := SaveConfig{
		GlobalConfigDir: "malformed",
	}

	t.Run("overwrites malformed global config", func(t *testing.T) {
		// Create malformed config file
		configDir := filepath.Join(tmpHome, ".config", "malformed")
		os.MkdirAll(configDir, 0o700)
		configPath := filepath.Join(configDir, "config.yaml")
		os.WriteFile(configPath, []byte("not: valid: yaml: [[["), 0o600)

		// SaveGlobal should still work (overwrites bad config)
		err := cfg.SaveGlobal("key", "value")
		if err != nil {
			t.Fatalf("SaveGlobal() error = %v", err)
		}

		// Verify new config is valid
		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("ReadFile() error = %v", err)
		}

		var saved map[string]interface{}
		if err := yaml.Unmarshal(data, &saved); err != nil {
			t.Errorf("saved config is invalid YAML: %v", err)
		}
		if saved["key"] != "value" {
			t.Errorf("key = %v, want value", saved["key"])
		}
	})

	t.Run("delete ignores malformed config", func(t *testing.T) {
		// Create malformed config file
		configDir := filepath.Join(tmpHome, ".config", "malformed2")
		os.MkdirAll(configDir, 0o700)
		configPath := filepath.Join(configDir, "config.yaml")
		os.WriteFile(configPath, []byte("not: valid: yaml: [[["), 0o600)

		newCfg := SaveConfig{GlobalConfigDir: "malformed2"}
		err := newCfg.DeleteGlobalKey("key")
		// Should not error, but also doesn't fix the file
		if err != nil {
			t.Errorf("DeleteGlobalKey() error = %v, want nil", err)
		}
	})
}

func TestSaveConfig_SaveLocal_MalformedYAML(t *testing.T) {
	cfg := SaveConfig{
		LocalConfigName: ".testapp.yaml",
	}

	t.Run("overwrites malformed local config", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".testapp.yaml")
		os.WriteFile(configPath, []byte("not: valid: yaml: [[["), 0o644)

		err := cfg.SaveLocal(tmpDir, "key", "value")
		if err != nil {
			t.Fatalf("SaveLocal() error = %v", err)
		}

		// Verify new config is valid
		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("ReadFile() error = %v", err)
		}

		var saved map[string]interface{}
		if err := yaml.Unmarshal(data, &saved); err != nil {
			t.Errorf("saved config is invalid YAML: %v", err)
		}
		if saved["key"] != "value" {
			t.Errorf("key = %v, want value", saved["key"])
		}
	})
}
