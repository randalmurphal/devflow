package pr

import (
	"fmt"
	"os"
)

// ProviderFromEnv creates a provider based on remote URL and environment.
// Automatically detects GitHub vs GitLab and uses appropriate token env var.
//
// Environment variables checked:
//   - GITHUB_TOKEN for GitHub
//   - GITLAB_TOKEN for GitLab
//   - GIT_TOKEN as fallback for either
//
// Example:
//
//	remoteURL, _ := gitCtx.GetRemoteURL("origin")
//	provider, err := pr.ProviderFromEnv(remoteURL)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	pr, _ := provider.CreatePR(ctx, opts)
func ProviderFromEnv(remoteURL string) (Provider, error) {
	platform, err := DetectProvider(remoteURL)
	if err != nil {
		return nil, err
	}

	switch platform {
	case "github":
		token := os.Getenv("GITHUB_TOKEN")
		if token == "" {
			token = os.Getenv("GIT_TOKEN")
		}
		if token == "" {
			return nil, fmt.Errorf("GITHUB_TOKEN or GIT_TOKEN not set; set one of these environment variables with a valid personal access token")
		}
		return NewGitHubProviderFromURL(token, remoteURL)

	case "gitlab":
		token := os.Getenv("GITLAB_TOKEN")
		if token == "" {
			token = os.Getenv("GIT_TOKEN")
		}
		if token == "" {
			return nil, fmt.Errorf("GITLAB_TOKEN or GIT_TOKEN not set; set one of these environment variables with a valid personal access token")
		}
		return NewGitLabProviderFromURL(token, remoteURL)

	default:
		return nil, fmt.Errorf("%w: %s", ErrUnknownProvider, platform)
	}
}

// ProviderFromEnvWithToken creates a provider with an explicit token.
// Use this when you have the token from configuration rather than environment.
//
// Example:
//
//	token := config.GetGitToken()
//	provider, err := pr.ProviderFromEnvWithToken(remoteURL, token)
func ProviderFromEnvWithToken(remoteURL, token string) (Provider, error) {
	platform, err := DetectProvider(remoteURL)
	if err != nil {
		return nil, err
	}

	switch platform {
	case "github":
		return NewGitHubProviderFromURL(token, remoteURL)
	case "gitlab":
		return NewGitLabProviderFromURL(token, remoteURL)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnknownProvider, platform)
	}
}

// MustProviderFromEnv creates a provider or panics.
// Use when the provider is required and missing is a programming error.
func MustProviderFromEnv(remoteURL string) Provider {
	p, err := ProviderFromEnv(remoteURL)
	if err != nil {
		panic(fmt.Sprintf("pr.ProviderFromEnv: %v", err))
	}
	return p
}
