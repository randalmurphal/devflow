package pr

import (
	"os"
	"strings"
	"testing"
)

func TestProviderFromEnv_GitHub(t *testing.T) {
	// Save and restore environment
	origGitHub := os.Getenv("GITHUB_TOKEN")
	origGit := os.Getenv("GIT_TOKEN")
	defer func() {
		os.Setenv("GITHUB_TOKEN", origGitHub)
		os.Setenv("GIT_TOKEN", origGit)
	}()

	os.Setenv("GITHUB_TOKEN", "test-token")
	os.Unsetenv("GIT_TOKEN")

	_, err := ProviderFromEnv("https://github.com/owner/repo.git")
	if err != nil {
		t.Fatalf("ProviderFromEnv failed: %v", err)
	}
}

func TestProviderFromEnv_GitHub_FallbackToken(t *testing.T) {
	// Save and restore environment
	origGitHub := os.Getenv("GITHUB_TOKEN")
	origGit := os.Getenv("GIT_TOKEN")
	defer func() {
		os.Setenv("GITHUB_TOKEN", origGitHub)
		os.Setenv("GIT_TOKEN", origGit)
	}()

	os.Unsetenv("GITHUB_TOKEN")
	os.Setenv("GIT_TOKEN", "fallback-token")

	_, err := ProviderFromEnv("https://github.com/owner/repo.git")
	if err != nil {
		t.Fatalf("ProviderFromEnv with GIT_TOKEN failed: %v", err)
	}
}

func TestProviderFromEnv_GitHub_NoToken(t *testing.T) {
	// Save and restore environment
	origGitHub := os.Getenv("GITHUB_TOKEN")
	origGit := os.Getenv("GIT_TOKEN")
	defer func() {
		os.Setenv("GITHUB_TOKEN", origGitHub)
		os.Setenv("GIT_TOKEN", origGit)
	}()

	os.Unsetenv("GITHUB_TOKEN")
	os.Unsetenv("GIT_TOKEN")

	_, err := ProviderFromEnv("https://github.com/owner/repo.git")
	if err == nil {
		t.Fatal("expected error when no token, got nil")
	}
	if !strings.Contains(err.Error(), "GITHUB_TOKEN") || !strings.Contains(err.Error(), "not set") {
		t.Errorf("error should mention GITHUB_TOKEN not set, got: %v", err)
	}
}

func TestProviderFromEnv_GitLab(t *testing.T) {
	// Save and restore environment
	origGitLab := os.Getenv("GITLAB_TOKEN")
	origGit := os.Getenv("GIT_TOKEN")
	defer func() {
		os.Setenv("GITLAB_TOKEN", origGitLab)
		os.Setenv("GIT_TOKEN", origGit)
	}()

	os.Setenv("GITLAB_TOKEN", "test-token")
	os.Unsetenv("GIT_TOKEN")

	_, err := ProviderFromEnv("https://gitlab.com/owner/repo.git")
	if err != nil {
		t.Fatalf("ProviderFromEnv for GitLab failed: %v", err)
	}
}

func TestProviderFromEnv_GitLab_FallbackToken(t *testing.T) {
	// Save and restore environment
	origGitLab := os.Getenv("GITLAB_TOKEN")
	origGit := os.Getenv("GIT_TOKEN")
	defer func() {
		os.Setenv("GITLAB_TOKEN", origGitLab)
		os.Setenv("GIT_TOKEN", origGit)
	}()

	os.Unsetenv("GITLAB_TOKEN")
	os.Setenv("GIT_TOKEN", "fallback-token")

	_, err := ProviderFromEnv("https://gitlab.com/owner/repo.git")
	if err != nil {
		t.Fatalf("ProviderFromEnv for GitLab with GIT_TOKEN failed: %v", err)
	}
}

func TestProviderFromEnv_GitLab_NoToken(t *testing.T) {
	// Save and restore environment
	origGitLab := os.Getenv("GITLAB_TOKEN")
	origGit := os.Getenv("GIT_TOKEN")
	defer func() {
		os.Setenv("GITLAB_TOKEN", origGitLab)
		os.Setenv("GIT_TOKEN", origGit)
	}()

	os.Unsetenv("GITLAB_TOKEN")
	os.Unsetenv("GIT_TOKEN")

	_, err := ProviderFromEnv("https://gitlab.com/owner/repo.git")
	if err == nil {
		t.Fatal("expected error when no token, got nil")
	}
	if !strings.Contains(err.Error(), "GITLAB_TOKEN") || !strings.Contains(err.Error(), "not set") {
		t.Errorf("error should mention GITLAB_TOKEN not set, got: %v", err)
	}
}

func TestProviderFromEnv_UnknownProvider(t *testing.T) {
	_, err := ProviderFromEnv("https://unknown.com/owner/repo.git")
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
	if !strings.Contains(err.Error(), "unknown") {
		t.Errorf("error should indicate unknown provider, got: %v", err)
	}
}

func TestProviderFromEnvWithToken_GitHub(t *testing.T) {
	_, err := ProviderFromEnvWithToken("https://github.com/owner/repo.git", "explicit-token")
	if err != nil {
		t.Fatalf("ProviderFromEnvWithToken failed: %v", err)
	}
}

func TestProviderFromEnvWithToken_GitLab(t *testing.T) {
	_, err := ProviderFromEnvWithToken("https://gitlab.com/owner/repo.git", "explicit-token")
	if err != nil {
		t.Fatalf("ProviderFromEnvWithToken for GitLab failed: %v", err)
	}
}

func TestProviderFromEnvWithToken_Unknown(t *testing.T) {
	_, err := ProviderFromEnvWithToken("https://unknown.com/owner/repo.git", "token")
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

func TestMustProviderFromEnv_Success(t *testing.T) {
	// Save and restore environment
	origGitHub := os.Getenv("GITHUB_TOKEN")
	defer os.Setenv("GITHUB_TOKEN", origGitHub)

	os.Setenv("GITHUB_TOKEN", "test-token")

	// Should not panic
	provider := MustProviderFromEnv("https://github.com/owner/repo.git")
	if provider == nil {
		t.Error("provider should not be nil")
	}
}

func TestMustProviderFromEnv_Panics(t *testing.T) {
	// Save and restore environment
	origGitHub := os.Getenv("GITHUB_TOKEN")
	origGit := os.Getenv("GIT_TOKEN")
	defer func() {
		os.Setenv("GITHUB_TOKEN", origGitHub)
		os.Setenv("GIT_TOKEN", origGit)
	}()

	os.Unsetenv("GITHUB_TOKEN")
	os.Unsetenv("GIT_TOKEN")

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic, got none")
		}
	}()

	MustProviderFromEnv("https://github.com/owner/repo.git")
}
