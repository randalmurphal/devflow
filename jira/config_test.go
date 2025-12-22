package jira

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.APIVersion != APIVersionAuto {
		t.Errorf("APIVersion = %v, want %v", cfg.APIVersion, APIVersionAuto)
	}
	if cfg.HTTP.Timeout != 30*time.Second {
		t.Errorf("HTTP.Timeout = %v, want %v", cfg.HTTP.Timeout, 30*time.Second)
	}
	if cfg.HTTP.MaxIdleConns != 10 {
		t.Errorf("HTTP.MaxIdleConns = %v, want 10", cfg.HTTP.MaxIdleConns)
	}
	if cfg.RateLimit.MaxRetries != 3 {
		t.Errorf("RateLimit.MaxRetries = %v, want 3", cfg.RateLimit.MaxRetries)
	}
	if !cfg.RateLimit.RetryJitter {
		t.Error("RateLimit.RetryJitter should be true")
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr error
	}{
		{
			name: "valid api_token config",
			config: Config{
				URL: "https://example.atlassian.net",
				Auth: AuthConfig{
					Type:  AuthAPIToken,
					Email: "user@example.com",
					Token: "api-token",
				},
			},
			wantErr: nil,
		},
		{
			name: "valid basic auth config",
			config: Config{
				URL: "https://jira.example.com",
				Auth: AuthConfig{
					Type:     AuthBasic,
					Username: "admin",
					Password: "secret",
				},
			},
			wantErr: nil,
		},
		{
			name: "valid PAT config",
			config: Config{
				URL: "https://jira.example.com",
				Auth: AuthConfig{
					Type:  AuthPAT,
					Token: "pat-token",
				},
			},
			wantErr: nil,
		},
		{
			name: "valid OAuth2 config",
			config: Config{
				URL: "https://example.atlassian.net",
				Auth: AuthConfig{
					Type:         AuthOAuth2,
					ClientID:     "client-id",
					ClientSecret: "client-secret",
				},
			},
			wantErr: nil,
		},
		{
			name: "missing URL",
			config: Config{
				Auth: AuthConfig{
					Type:  AuthAPIToken,
					Email: "user@example.com",
					Token: "token",
				},
			},
			wantErr: ErrConfigURLRequired,
		},
		{
			name: "missing auth type",
			config: Config{
				URL: "https://example.atlassian.net",
			},
			wantErr: ErrConfigAuthTypeRequired,
		},
		{
			name: "api_token missing email",
			config: Config{
				URL: "https://example.atlassian.net",
				Auth: AuthConfig{
					Type:  AuthAPIToken,
					Token: "token",
				},
			},
			wantErr: ErrConfigAPITokenAuth,
		},
		{
			name: "api_token missing token",
			config: Config{
				URL: "https://example.atlassian.net",
				Auth: AuthConfig{
					Type:  AuthAPIToken,
					Email: "user@example.com",
				},
			},
			wantErr: ErrConfigAPITokenAuth,
		},
		{
			name: "basic missing username",
			config: Config{
				URL: "https://jira.example.com",
				Auth: AuthConfig{
					Type:     AuthBasic,
					Password: "secret",
				},
			},
			wantErr: ErrConfigBasicAuth,
		},
		{
			name: "PAT missing token",
			config: Config{
				URL: "https://jira.example.com",
				Auth: AuthConfig{
					Type: AuthPAT,
				},
			},
			wantErr: ErrConfigPATAuth,
		},
		{
			name: "OAuth2 missing client_id",
			config: Config{
				URL: "https://example.atlassian.net",
				Auth: AuthConfig{
					Type:         AuthOAuth2,
					ClientSecret: "secret",
				},
			},
			wantErr: ErrConfigOAuth2Auth,
		},
		{
			name: "invalid auth type",
			config: Config{
				URL: "https://example.atlassian.net",
				Auth: AuthConfig{
					Type: "invalid",
				},
			},
			wantErr: ErrConfigAuthTypeInvalid,
		},
		{
			name: "invalid API version",
			config: Config{
				URL: "https://example.atlassian.net",
				Auth: AuthConfig{
					Type:  AuthAPIToken,
					Email: "user@example.com",
					Token: "token",
				},
				APIVersion: "v4",
			},
			wantErr: ErrConfigAPIVersionInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
				return
			}
			if err != tt.wantErr {
				t.Errorf("Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigGetAPIVersion(t *testing.T) {
	tests := []struct {
		name    string
		version APIVersion
		want    APIVersion
	}{
		{"auto", APIVersionAuto, APIVersionV3},
		{"empty", "", APIVersionV3},
		{"v2", APIVersionV2, APIVersionV2},
		{"v3", APIVersionV3, APIVersionV3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{APIVersion: tt.version}
			got := cfg.GetAPIVersion()
			if got != tt.want {
				t.Errorf("GetAPIVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigClone(t *testing.T) {
	cfg := &Config{
		URL:        "https://example.atlassian.net",
		APIVersion: APIVersionV3,
		Auth: AuthConfig{
			Type:  AuthAPIToken,
			Email: "user@example.com",
			Token: "token",
		},
	}

	clone := cfg.Clone()

	if clone.URL != cfg.URL {
		t.Errorf("Clone.URL = %v, want %v", clone.URL, cfg.URL)
	}
	if clone.Auth.Token != cfg.Auth.Token {
		t.Errorf("Clone.Auth.Token = %v, want %v", clone.Auth.Token, cfg.Auth.Token)
	}

	// Verify independence
	clone.URL = "https://other.atlassian.net"
	if cfg.URL == clone.URL {
		t.Error("Clone should be independent from original")
	}
}

func TestConfigCloneNil(t *testing.T) {
	var cfg *Config
	clone := cfg.Clone()
	if clone != nil {
		t.Errorf("Clone of nil should be nil, got %v", clone)
	}
}
