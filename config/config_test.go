package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolver_Defaults(t *testing.T) {
	resolver := NewResolver(ResolverConfig{
		Defaults: map[string]string{
			"api_url": "http://localhost:8080",
			"format":  "table",
		},
	})

	cfg := resolver.Resolve()

	if got := cfg.Get("api_url"); got != "http://localhost:8080" {
		t.Errorf("api_url = %q, want %q", got, "http://localhost:8080")
	}
	if got := cfg.Source("api_url"); got != SourceDefault {
		t.Errorf("source = %q, want %q", got, SourceDefault)
	}
}

func TestResolver_EnvOverridesDefaults(t *testing.T) {
	os.Setenv("MYAPP_API_URL", "http://env-server:9000")
	defer os.Unsetenv("MYAPP_API_URL")

	resolver := NewResolver(ResolverConfig{
		EnvPrefix: "MYAPP_",
		Defaults: map[string]string{
			"api_url": "http://localhost:8080",
		},
	})

	cfg := resolver.Resolve()

	if got := cfg.Get("api_url"); got != "http://env-server:9000" {
		t.Errorf("api_url = %q, want %q", got, "http://env-server:9000")
	}
	if got := cfg.Source("api_url"); got != SourceEnv {
		t.Errorf("source = %q, want %q", got, SourceEnv)
	}
}

func TestResolver_GlobalConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".config", "testapp")
	os.MkdirAll(configDir, 0755)

	configPath := filepath.Join(configDir, "config.yaml")
	os.WriteFile(configPath, []byte("api_url: http://global-server:8080\n"), 0644)

	resolver := NewResolver(ResolverConfig{
		GlobalConfigDir: "testapp",
		Defaults: map[string]string{
			"api_url": "http://localhost:8080",
		},
	})
	// Override the global path for testing
	resolver.globalPath = configPath

	cfg := resolver.Resolve()

	if got := cfg.Get("api_url"); got != "http://global-server:8080" {
		t.Errorf("api_url = %q, want %q", got, "http://global-server:8080")
	}
	if got := cfg.Source("api_url"); got != SourceGlobal {
		t.Errorf("source = %q, want %q", got, SourceGlobal)
	}
}

func TestResolver_LocalConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create git directory
	gitDir := filepath.Join(tmpDir, ".git")
	os.MkdirAll(gitDir, 0755)

	// Create local config
	localConfig := filepath.Join(tmpDir, ".myapp.yaml")
	os.WriteFile(localConfig, []byte("project_id: proj_123\n"), 0644)

	resolver := NewResolver(ResolverConfig{
		LocalConfigName: ".myapp.yaml",
		GitRootFinder: func(_ string) (string, error) {
			return tmpDir, nil
		},
		Defaults: map[string]string{
			"project_id": "",
		},
	})

	cfg := resolver.Resolve()

	if got := cfg.Get("project_id"); got != "proj_123" {
		t.Errorf("project_id = %q, want %q", got, "proj_123")
	}
	if got := cfg.Source("project_id"); got != SourceLocal {
		t.Errorf("source = %q, want %q", got, SourceLocal)
	}
}

func TestResolver_Priority(t *testing.T) {
	tmpDir := t.TempDir()

	// Create global config
	globalDir := filepath.Join(tmpDir, "global")
	os.MkdirAll(globalDir, 0755)
	globalConfig := filepath.Join(globalDir, "config.yaml")
	os.WriteFile(globalConfig, []byte("api_url: http://global\n"), 0644)

	// Create local config
	localDir := filepath.Join(tmpDir, "local")
	os.MkdirAll(filepath.Join(localDir, ".git"), 0755)
	localConfig := filepath.Join(localDir, ".myapp.yaml")
	os.WriteFile(localConfig, []byte("api_url: http://local\n"), 0644)

	// Set env var
	os.Setenv("TEST_API_URL", "http://env")
	defer os.Unsetenv("TEST_API_URL")

	resolver := NewResolver(ResolverConfig{
		EnvPrefix:       "TEST_",
		LocalConfigName: ".myapp.yaml",
		GitRootFinder: func(_ string) (string, error) {
			return localDir, nil
		},
		Defaults: map[string]string{
			"api_url": "http://default",
		},
	})
	resolver.globalPath = globalConfig

	cfg := resolver.Resolve()

	// Env should win
	if got := cfg.Get("api_url"); got != "http://env" {
		t.Errorf("api_url = %q, want %q (env should have highest priority)", got, "http://env")
	}
}

func TestResolver_ResolveWithFlags(t *testing.T) {
	resolver := NewResolver(ResolverConfig{
		Defaults: map[string]string{
			"format": "table",
		},
	})

	cfg := resolver.ResolveWithFlags(map[string]string{
		"format": "json",
	})

	if got := cfg.Get("format"); got != "json" {
		t.Errorf("format = %q, want %q", got, "json")
	}
	if got := cfg.Source("format"); got != SourceFlag {
		t.Errorf("source = %q, want %q", got, SourceFlag)
	}
}

func TestResolver_ValidKeys(t *testing.T) {
	tmpDir := t.TempDir()

	// Create global config with valid and invalid keys
	configDir := filepath.Join(tmpDir, ".config", "testapp")
	os.MkdirAll(configDir, 0755)
	configPath := filepath.Join(configDir, "config.yaml")
	os.WriteFile(configPath, []byte("api_url: http://test\ninvalid_key: value\n"), 0644)

	resolver := NewResolver(ResolverConfig{
		GlobalConfigDir: "testapp",
		ValidGlobalKeys: []string{"api_url", "format"},
		Defaults: map[string]string{
			"api_url": "http://default",
		},
	})
	resolver.globalPath = configPath

	cfg := resolver.Resolve()

	// Valid key should be loaded
	if got := cfg.Get("api_url"); got != "http://test" {
		t.Errorf("api_url = %q, want %q", got, "http://test")
	}

	// Invalid key should be ignored
	if got := cfg.Get("invalid_key"); got != "" {
		t.Errorf("invalid_key = %q, want empty", got)
	}
}

func TestResolved_All(t *testing.T) {
	resolver := NewResolver(ResolverConfig{
		Defaults: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	})

	cfg := resolver.Resolve()
	all := cfg.All()

	if len(all) != 2 {
		t.Errorf("got %d keys, want 2", len(all))
	}
	if all["key1"] != "value1" {
		t.Errorf("key1 = %q, want %q", all["key1"], "value1")
	}
}

func TestResolved_Keys(t *testing.T) {
	resolver := NewResolver(ResolverConfig{
		Defaults: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	})

	cfg := resolver.Resolve()
	keys := cfg.Keys()

	if len(keys) != 2 {
		t.Errorf("got %d keys, want 2", len(keys))
	}
}

func TestResolver_NoColorEnv(t *testing.T) {
	os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")

	resolver := NewResolver(ResolverConfig{
		Defaults: map[string]string{
			"no_color": "false",
		},
	})

	cfg := resolver.Resolve()

	if got := cfg.Get("no_color"); got != "true" {
		t.Errorf("no_color = %q, want %q (NO_COLOR env should set to true)", got, "true")
	}
}

func TestFindGitRoot(t *testing.T) {
	tmpDir := t.TempDir()

	// Create nested directories
	nested := filepath.Join(tmpDir, "a", "b", "c")
	os.MkdirAll(nested, 0755)

	// Create .git directory in root
	gitDir := filepath.Join(tmpDir, ".git")
	os.MkdirAll(gitDir, 0755)

	// Find from nested directory
	root := findGitRoot(nested)
	if root != tmpDir {
		t.Errorf("findGitRoot() = %q, want %q", root, tmpDir)
	}
}

func TestFindGitRoot_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	root := findGitRoot(tmpDir)
	if root != "" {
		t.Errorf("findGitRoot() = %q, want empty", root)
	}
}

func TestResolver_BoolValues(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config with bool values
	configDir := filepath.Join(tmpDir, ".config", "testapp")
	os.MkdirAll(configDir, 0755)
	configPath := filepath.Join(configDir, "config.yaml")
	os.WriteFile(configPath, []byte("no_color: true\n"), 0644)

	resolver := NewResolver(ResolverConfig{
		GlobalConfigDir: "testapp",
		Defaults: map[string]string{
			"no_color": "false",
		},
	})
	resolver.globalPath = configPath

	cfg := resolver.Resolve()

	if got := cfg.Get("no_color"); got != "true" {
		t.Errorf("no_color = %q, want %q", got, "true")
	}
}
