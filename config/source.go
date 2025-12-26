package config

// Source indicates where a configuration value came from.
type Source string

// Configuration source constants.
const (
	// SourceDefault indicates the value is a built-in default.
	SourceDefault Source = "default"

	// SourceGlobal indicates the value came from global config
	// (e.g., ~/.config/<app>/config.yaml).
	SourceGlobal Source = "global"

	// SourceLocal indicates the value came from local config
	// (e.g., .myapp.yaml in git root).
	SourceLocal Source = "local"

	// SourceEnv indicates the value came from an environment variable.
	SourceEnv Source = "env"

	// SourceFlag indicates the value was set via command-line flag.
	SourceFlag Source = "flag"
)
