# config package

Hierarchical configuration resolution for CLI applications.

## Quick Reference

| Type | Purpose |
|------|---------|
| `ResolverConfig` | Configuration for the resolver |
| `Resolver` | Builds resolved config from all sources |
| `Resolved` | Final merged configuration with source tracking |
| `SaveConfig` | Configuration for saving values |
| `Source` | Indicates where a value came from |

## Source Priority

From highest to lowest:
1. `SourceFlag` - Command-line flags
2. `SourceEnv` - Environment variables
3. `SourceLocal` - Local config (e.g., `.myapp.yaml` in git root)
4. `SourceGlobal` - Global config (e.g., `~/.config/myapp/config.yaml`)
5. `SourceDefault` - Built-in defaults

## Resolver Functions

| Function | Purpose |
|----------|---------|
| `NewResolver(cfg)` | Create new resolver |
| `resolver.Resolve()` | Build resolved config |
| `resolver.ResolveWithFlags(flags)` | Resolve and apply flag overrides |
| `resolver.GitRoot()` | Get detected git root |
| `resolver.GlobalPath()` | Get global config path |
| `resolver.LocalPath()` | Get local config path |

## Resolved Functions

| Function | Purpose |
|----------|---------|
| `cfg.Get(key)` | Get value for key |
| `cfg.Source(key)` | Get source of key's value |
| `cfg.GetWithSource(key)` | Get both value and source |
| `cfg.All()` | Get all key-value pairs |
| `cfg.Keys()` | Get all keys |

## Save Functions

| Function | Purpose |
|----------|---------|
| `save.SaveGlobal(key, value)` | Save to global config |
| `save.SaveLocal(gitRoot, key, value)` | Save to local config |
| `save.DeleteGlobalKey(key)` | Remove key from global config |

## Usage Example

```go
resolver := config.NewResolver(config.ResolverConfig{
    EnvPrefix:       "MYAPP_",
    GlobalConfigDir: "myapp",
    LocalConfigName: ".myapp.yaml",
    Defaults: map[string]string{
        "api_url": "http://localhost:8080",
        "format":  "table",
    },
    ValidGlobalKeys: []string{"api_url", "format", "no_color"},
    ValidLocalKeys:  []string{"project_id", "api_url"},
})

cfg := resolver.Resolve()

// Use config
apiURL := cfg.Get("api_url")
source := cfg.Source("api_url")

// With flag overrides
cfg = resolver.ResolveWithFlags(map[string]string{
    "format": "json",
})
```

## Application-Specific Wrappers

Applications should create thin wrappers with their defaults:

```go
package cli

import devconfig "github.com/randalmurphal/devflow/config"

var defaultResolver = devconfig.NewResolver(devconfig.ResolverConfig{
    EnvPrefix:       "TK_",
    GlobalConfigDir: "taskkeeper",
    LocalConfigName: ".taskkeeper.yaml",
    Defaults: map[string]string{
        "api_url":       "http://localhost:8080",
        "output_format": "table",
        "no_color":      "false",
    },
    ValidGlobalKeys: []string{"api_url", "output_format", "no_color", "default_project_id"},
    ValidLocalKeys:  []string{"project_id", "api_url", "project_name"},
})

func ResolveConfig() *devconfig.Resolved {
    return defaultResolver.Resolve()
}
```

## File Structure

```
config/
├── doc.go           # Package documentation
├── source.go        # Source enum
├── config.go        # Resolver and Resolved types
├── save.go          # SaveConfig for persisting values
└── config_test.go   # Tests
```
