// Package config provides hierarchical configuration resolution for CLI applications.
//
// This package supports layered configuration with clear precedence:
//  1. Environment variables (highest priority)
//  2. Local config (e.g., .myapp.yaml in git root)
//  3. Global config (e.g., ~/.config/myapp/config.yaml)
//  4. Built-in defaults (lowest priority)
//
// # Basic Usage
//
// Create a resolver with your application's settings:
//
//	resolver := config.NewResolver(config.ResolverConfig{
//	    EnvPrefix:       "MYAPP_",
//	    GlobalConfigDir: "myapp",
//	    LocalConfigName: ".myapp.yaml",
//	    Defaults: map[string]string{
//	        "api_url": "http://localhost:8080",
//	        "format":  "table",
//	    },
//	})
//
//	cfg := resolver.Resolve()
//	fmt.Println(cfg.Get("api_url"))        // "http://localhost:8080"
//	fmt.Println(cfg.Source("api_url"))     // "default"
//
// # Environment Variables
//
// Environment variables are automatically detected using the configured prefix:
//
//	# With EnvPrefix: "MYAPP_"
//	MYAPP_API_URL=https://api.example.com  # sets "api_url"
//	MYAPP_FORMAT=json                       # sets "format"
//
// # Config Sources
//
// Each resolved value tracks where it came from:
//   - "default": Built-in default value
//   - "global": ~/.config/<app>/config.yaml
//   - "local": .myapp.yaml in git root
//   - "env": Environment variable
//   - "flag": Command-line flag (set via SetFlagValue)
//
// # Git Root Detection
//
// By default, the resolver looks for the local config in the git repository root.
// You can customize this by providing a GitRootFinder function:
//
//	resolver := config.NewResolver(config.ResolverConfig{
//	    GitRootFinder: func(dir string) (string, error) {
//	        // Custom logic to find git root
//	        return myGitRoot(), nil
//	    },
//	})
package config
