package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ResolverConfig configures the hierarchical config resolver.
type ResolverConfig struct {
	// EnvPrefix is prepended to key names for environment variable lookup.
	// For example, with EnvPrefix "MYAPP_", key "api_url" maps to MYAPP_API_URL.
	EnvPrefix string

	// GlobalConfigDir is the name of the directory under ~/.config/
	// where the global config is stored.
	// For example, "myapp" results in ~/.config/myapp/config.yaml.
	GlobalConfigDir string

	// GlobalConfigFile is the filename for global config.
	// Defaults to "config.yaml" if empty.
	GlobalConfigFile string

	// LocalConfigName is the filename for local config in the git root.
	// For example, ".myapp.yaml".
	LocalConfigName string

	// Defaults provides the default values for configuration keys.
	Defaults map[string]string

	// ValidGlobalKeys lists keys that can be set in global config.
	// If nil, all keys are valid.
	ValidGlobalKeys []string

	// ValidLocalKeys lists keys that can be set in local config.
	// If nil, all keys are valid.
	ValidLocalKeys []string

	// GitRootFinder is a function that finds the git root directory.
	// If nil, uses a simple git root detection.
	GitRootFinder func(startDir string) (string, error)

	// ErrWriter is where warnings are written.
	// Defaults to os.Stderr if nil.
	ErrWriter io.Writer
}

func (c ResolverConfig) globalConfigFile() string {
	if c.GlobalConfigFile != "" {
		return c.GlobalConfigFile
	}
	return "config.yaml"
}

// Resolver handles hierarchical configuration resolution.
type Resolver struct {
	config     ResolverConfig
	globalPath string
	localPath  string
	gitRoot    string

	// Warnings collects non-fatal issues during resolution.
	Warnings []string
}

// NewResolver creates a new configuration resolver.
func NewResolver(cfg ResolverConfig) *Resolver {
	resolver := &Resolver{
		config: cfg,
	}

	// Set default error writer
	if cfg.ErrWriter == nil {
		resolver.config.ErrWriter = os.Stderr
	}

	// Find git root and local config
	if cfg.GitRootFinder != nil {
		if root, err := cfg.GitRootFinder("."); err == nil && root != "" {
			resolver.gitRoot = root
			if cfg.LocalConfigName != "" {
				resolver.localPath = filepath.Join(root, cfg.LocalConfigName)
			}
		}
	} else {
		// Use simple git root detection
		if root := findGitRoot("."); root != "" {
			resolver.gitRoot = root
			if cfg.LocalConfigName != "" {
				resolver.localPath = filepath.Join(root, cfg.LocalConfigName)
			}
		}
	}

	// Set global config path
	if cfg.GlobalConfigDir != "" {
		if home, err := os.UserHomeDir(); err == nil {
			resolver.globalPath = filepath.Join(
				home, ".config", cfg.GlobalConfigDir, cfg.globalConfigFile(),
			)
		}
	}

	return resolver
}

// NewResolverWithPaths creates a resolver with explicit global and local paths.
// This is useful for testing or when paths are known ahead of time.
func NewResolverWithPaths(cfg ResolverConfig, globalPath, localPath string) *Resolver {
	resolver := &Resolver{
		config:     cfg,
		globalPath: globalPath,
		localPath:  localPath,
	}

	// Set default error writer
	if cfg.ErrWriter == nil {
		resolver.config.ErrWriter = os.Stderr
	}

	return resolver
}

// warn adds a warning and optionally prints it.
func (r *Resolver) warn(msg string) {
	r.Warnings = append(r.Warnings, msg)
	if r.config.ErrWriter != nil {
		fmt.Fprintf(r.config.ErrWriter, "Warning: %s\n", msg)
	}
}

// Resolved holds the final merged configuration.
type Resolved struct {
	values  map[string]string
	sources map[string]Source
}

// Get returns the value for a key, or empty string if not set.
func (c *Resolved) Get(key string) string {
	return c.values[key]
}

// Source returns the source of a key's value.
func (c *Resolved) Source(key string) Source {
	return c.sources[key]
}

// GetWithSource returns both the value and its source.
func (c *Resolved) GetWithSource(key string) (string, Source) {
	return c.values[key], c.sources[key]
}

// All returns a copy of all key-value pairs.
func (c *Resolved) All() map[string]string {
	result := make(map[string]string, len(c.values))
	for k, v := range c.values {
		result[k] = v
	}
	return result
}

// Keys returns all configuration keys.
func (c *Resolved) Keys() []string {
	keys := make([]string, 0, len(c.values))
	for k := range c.values {
		keys = append(keys, k)
	}
	return keys
}

// Resolve builds the final config by merging all sources.
// Priority (highest to lowest): flags > env > local > global > defaults.
func (r *Resolver) Resolve() *Resolved {
	cfg := &Resolved{
		values:  make(map[string]string),
		sources: make(map[string]Source),
	}

	// 1. Apply defaults (lowest priority)
	r.applyDefaults(cfg)

	// 2. Apply global config
	r.applyGlobal(cfg)

	// 3. Apply local config
	r.applyLocal(cfg)

	// 4. Apply environment variables (highest priority for now)
	r.applyEnv(cfg)

	return cfg
}

// ResolveWithFlags resolves config and applies flag overrides.
func (r *Resolver) ResolveWithFlags(flags map[string]string) *Resolved {
	cfg := r.Resolve()

	for key, value := range flags {
		if value != "" {
			cfg.values[key] = value
			cfg.sources[key] = SourceFlag
		}
	}

	return cfg
}

func (r *Resolver) applyDefaults(cfg *Resolved) {
	for key, value := range r.config.Defaults {
		cfg.values[key] = value
		cfg.sources[key] = SourceDefault
	}
}

func (r *Resolver) applyGlobal(cfg *Resolved) {
	if r.globalPath == "" {
		return
	}

	data, err := os.ReadFile(r.globalPath)
	if err != nil {
		return // File doesn't exist - not an error
	}

	var parsed map[string]interface{}
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		r.warn(fmt.Sprintf("could not parse %s: %v", r.globalPath, err))
		return
	}

	for key, value := range parsed {
		// Skip if not a valid global key (when validation is enabled)
		if len(r.config.ValidGlobalKeys) > 0 && !contains(r.config.ValidGlobalKeys, key) {
			continue
		}
		if strVal := toString(value); strVal != "" {
			cfg.values[key] = strVal
			cfg.sources[key] = SourceGlobal
		}
	}
}

func (r *Resolver) applyLocal(cfg *Resolved) {
	if r.localPath == "" {
		return
	}

	data, err := os.ReadFile(r.localPath)
	if err != nil {
		return
	}

	var parsed map[string]interface{}
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		r.warn(fmt.Sprintf("could not parse %s: %v", r.localPath, err))
		return
	}

	for key, value := range parsed {
		// Skip if not a valid local key (when validation is enabled)
		if len(r.config.ValidLocalKeys) > 0 && !contains(r.config.ValidLocalKeys, key) {
			continue
		}
		if strVal := toString(value); strVal != "" {
			cfg.values[key] = strVal
			cfg.sources[key] = SourceLocal
		}
	}
}

func (r *Resolver) applyEnv(cfg *Resolved) {
	// Check environment for each known key (if prefix is set)
	if r.config.EnvPrefix != "" {
		allKeys := make(map[string]bool)
		for k := range r.config.Defaults {
			allKeys[k] = true
		}
		for k := range cfg.values {
			allKeys[k] = true
		}

		for key := range allKeys {
			envKey := r.config.EnvPrefix + strings.ToUpper(strings.ReplaceAll(key, "-", "_"))
			if value := os.Getenv(envKey); value != "" {
				cfg.values[key] = value
				cfg.sources[key] = SourceEnv
			}
		}
	}

	// Also check standard NO_COLOR env var (always, regardless of prefix)
	if _, hasNoColor := os.LookupEnv("NO_COLOR"); hasNoColor {
		cfg.values["no_color"] = "true"
		cfg.sources["no_color"] = SourceEnv
	}
}

// GitRoot returns the detected git root directory.
func (r *Resolver) GitRoot() string {
	return r.gitRoot
}

// GlobalPath returns the path to the global config file.
func (r *Resolver) GlobalPath() string {
	return r.globalPath
}

// LocalPath returns the path to the local config file.
func (r *Resolver) LocalPath() string {
	return r.localPath
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func toString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case bool:
		if val {
			return "true"
		}
		return "false"
	case int, int64, float64:
		return fmt.Sprintf("%v", val)
	default:
		return ""
	}
}

// findGitRoot finds the git root by looking for .git directory.
func findGitRoot(startDir string) string {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return ""
	}

	for {
		gitDir := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break // Reached root
		}
		dir = parent
	}

	return ""
}
