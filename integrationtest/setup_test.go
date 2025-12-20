package integrationtest

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/randalmurphal/devflow"
	"github.com/randalmurphal/flowgraph/pkg/flowgraph"
	"github.com/randalmurphal/flowgraph/pkg/flowgraph/llm"
)

// setupTempRepo creates a temporary git repository for testing.
// Returns the repo path and a cleanup function.
func setupTempRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	// Configure git user for commits
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = dir
	cmd.Run()

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = dir
	cmd.Run()

	// Create initial commit
	readme := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readme, []byte("# Test Repo\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = dir
	cmd.Run()

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = dir
	cmd.Run()

	return dir
}

// setupContext creates a flowgraph.Context with all devflow services configured.
func setupContext(t *testing.T, repoPath string, mockLLM llm.Client) flowgraph.Context {
	t.Helper()

	baseCtx := context.Background()

	// Setup git context
	git, err := devflow.NewGitContext(repoPath)
	if err != nil {
		t.Fatalf("NewGitContext: %v", err)
	}
	baseCtx = devflow.WithGitContext(baseCtx, git)

	// Setup LLM client
	if mockLLM != nil {
		baseCtx = devflow.WithLLMClient(baseCtx, mockLLM)
	}

	// Setup mock command runner for test isolation
	runner := devflow.NewMockRunner()
	baseCtx = devflow.WithCommandRunner(baseCtx, runner)

	// Setup artifact manager
	artifacts := devflow.NewArtifactManager(devflow.ArtifactConfig{
		BaseDir: filepath.Join(repoPath, ".devflow", "artifacts"),
	})
	baseCtx = devflow.WithArtifactManager(baseCtx, artifacts)

	// Setup transcript manager
	transcripts, err := devflow.NewTranscriptManager(devflow.TranscriptConfig{
		BaseDir: filepath.Join(repoPath, ".devflow", "transcripts"),
	})
	if err == nil {
		baseCtx = devflow.WithTranscriptManager(baseCtx, transcripts)
	}

	return flowgraph.NewContext(baseCtx, flowgraph.WithLLM(mockLLM))
}

// mockResponses creates a MockClient with sequential responses.
func mockResponses(responses ...string) *llm.MockClient {
	return llm.NewMockClient("").WithResponses(responses...)
}
