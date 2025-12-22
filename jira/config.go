package jira

import (
	"time"
)

// AuthType represents the type of authentication to use.
type AuthType string

// Authentication types supported by the Jira client.
const (
	AuthAPIToken AuthType = "api_token" // Cloud: email + API token
	AuthOAuth2   AuthType = "oauth2"    // Cloud: OAuth 2.0
	AuthBasic    AuthType = "basic"     // Server: username + password
	AuthPAT      AuthType = "pat"       // Server/DC: Personal Access Token
)

// Config holds the configuration for the Jira client.
type Config struct {
	// URL is the base URL of the Jira instance.
	// For Cloud: https://your-domain.atlassian.net
	// For Server: https://jira.your-company.com
	URL string `mapstructure:"url"`

	// APIVersion specifies which API version to use.
	// "auto" (default) detects based on deployment type.
	// "v3" for Cloud, "v2" for Server/DC.
	APIVersion APIVersion `mapstructure:"api_version"`

	// Auth contains authentication configuration.
	Auth AuthConfig `mapstructure:"auth"`

	// HTTP contains HTTP client configuration.
	HTTP HTTPConfig `mapstructure:"http"`

	// RateLimit contains rate limiting configuration.
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
}

// AuthConfig holds authentication configuration.
type AuthConfig struct {
	// Type is the authentication method to use.
	Type AuthType `mapstructure:"type"`

	// Email is required for api_token auth (Cloud).
	Email string `mapstructure:"email"`

	// Token is the API token (Cloud) or PAT (Server/DC).
	Token string `mapstructure:"token"`

	// Username is required for basic auth.
	Username string `mapstructure:"username"`

	// Password is required for basic auth.
	Password string `mapstructure:"password"`

	// OAuth2 configuration (Cloud only).
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	AccessToken  string `mapstructure:"access_token"`
	RefreshToken string `mapstructure:"refresh_token"`
}

// HTTPConfig holds HTTP client configuration.
type HTTPConfig struct {
	// Timeout is the request timeout.
	Timeout time.Duration `mapstructure:"timeout"`

	// MaxIdleConns is the maximum number of idle connections.
	MaxIdleConns int `mapstructure:"max_idle_conns"`

	// IdleConnTimeout is how long to keep idle connections open.
	IdleConnTimeout time.Duration `mapstructure:"idle_conn_timeout"`
}

// RateLimitConfig holds rate limiting configuration.
type RateLimitConfig struct {
	// MaxRetries is the maximum number of retry attempts.
	MaxRetries int `mapstructure:"max_retries"`

	// RetryWaitMin is the minimum wait between retries.
	RetryWaitMin time.Duration `mapstructure:"retry_wait_min"`

	// RetryWaitMax is the maximum wait between retries.
	RetryWaitMax time.Duration `mapstructure:"retry_wait_max"`

	// RetryJitter enables randomized jitter on retry waits.
	RetryJitter bool `mapstructure:"retry_jitter"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		APIVersion: APIVersionAuto,
		HTTP: HTTPConfig{
			Timeout:         30 * time.Second,
			MaxIdleConns:    10,
			IdleConnTimeout: 90 * time.Second,
		},
		RateLimit: RateLimitConfig{
			MaxRetries:   3,
			RetryWaitMin: 1 * time.Second,
			RetryWaitMax: 30 * time.Second,
			RetryJitter:  true,
		},
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.URL == "" {
		return ErrConfigURLRequired
	}

	if c.Auth.Type == "" {
		return ErrConfigAuthTypeRequired
	}

	switch c.Auth.Type {
	case AuthAPIToken:
		if c.Auth.Email == "" || c.Auth.Token == "" {
			return ErrConfigAPITokenAuth
		}
	case AuthBasic:
		if c.Auth.Username == "" || c.Auth.Password == "" {
			return ErrConfigBasicAuth
		}
	case AuthPAT:
		if c.Auth.Token == "" {
			return ErrConfigPATAuth
		}
	case AuthOAuth2:
		if c.Auth.ClientID == "" || c.Auth.ClientSecret == "" {
			return ErrConfigOAuth2Auth
		}
	default:
		return ErrConfigAuthTypeInvalid
	}

	if c.APIVersion != "" && c.APIVersion != APIVersionAuto &&
		c.APIVersion != APIVersionV2 && c.APIVersion != APIVersionV3 {
		return ErrConfigAPIVersionInvalid
	}

	return nil
}

// GetAPIVersion returns the effective API version.
// If APIVersion is "auto" or empty, returns the default based on detection.
func (c *Config) GetAPIVersion() APIVersion {
	if c.APIVersion == "" || c.APIVersion == APIVersionAuto {
		return APIVersionV3 // Default to v3, client will adjust after detection
	}
	return c.APIVersion
}

// Clone returns a deep copy of the config.
func (c *Config) Clone() *Config {
	if c == nil {
		return nil
	}
	clone := *c
	return &clone
}
