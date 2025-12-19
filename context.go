package devflow

import (
	"context"

	"github.com/rmurphy/flowgraph/pkg/flowgraph/llm"
)

// =============================================================================
// Context Injection Helpers
// =============================================================================
// These helpers allow devflow services to be injected into context.Context
// for use by flowgraph nodes.

// serviceContextKey is a private type for context keys to avoid collisions
type serviceContextKey string

// Context keys for devflow services
const (
	gitServiceKey        serviceContextKey = "devflow.git"
	llmServiceKey        serviceContextKey = "devflow.llm"
	transcriptServiceKey serviceContextKey = "devflow.transcripts"
	artifactServiceKey   serviceContextKey = "devflow.artifacts"
	promptServiceKey     serviceContextKey = "devflow.prompts"
	runnerServiceKey     serviceContextKey = "devflow.runner"
)

// WithGitContext adds a GitContext to the context
func WithGitContext(ctx context.Context, git *GitContext) context.Context {
	return context.WithValue(ctx, gitServiceKey, git)
}

// GitFromContext extracts GitContext from context
func GitFromContext(ctx context.Context) *GitContext {
	if git, ok := ctx.Value(gitServiceKey).(*GitContext); ok {
		return git
	}
	return nil
}

// MustGitFromContext extracts GitContext or panics
func MustGitFromContext(ctx context.Context) *GitContext {
	git := GitFromContext(ctx)
	if git == nil {
		panic("devflow: GitContext not found in context")
	}
	return git
}

// WithLLMClient adds an LLM client to the context.
// This uses flowgraph's llm.Client interface.
func WithLLMClient(ctx context.Context, client llm.Client) context.Context {
	return context.WithValue(ctx, llmServiceKey, client)
}

// LLMFromContext extracts the LLM client from context.
func LLMFromContext(ctx context.Context) llm.Client {
	if client, ok := ctx.Value(llmServiceKey).(llm.Client); ok {
		return client
	}
	return nil
}

// MustLLMFromContext extracts the LLM client or panics.
func MustLLMFromContext(ctx context.Context) llm.Client {
	client := LLMFromContext(ctx)
	if client == nil {
		panic("devflow: llm.Client not found in context")
	}
	return client
}

// WithTranscriptManager adds a TranscriptManager to the context
func WithTranscriptManager(ctx context.Context, mgr TranscriptManager) context.Context {
	return context.WithValue(ctx, transcriptServiceKey, mgr)
}

// TranscriptManagerFromContext extracts TranscriptManager from context
func TranscriptManagerFromContext(ctx context.Context) TranscriptManager {
	if mgr, ok := ctx.Value(transcriptServiceKey).(TranscriptManager); ok {
		return mgr
	}
	return nil
}

// MustTranscriptManagerFromContext extracts TranscriptManager or panics
func MustTranscriptManagerFromContext(ctx context.Context) TranscriptManager {
	mgr := TranscriptManagerFromContext(ctx)
	if mgr == nil {
		panic("devflow: TranscriptManager not found in context")
	}
	return mgr
}

// WithArtifactManager adds an ArtifactManager to the context
func WithArtifactManager(ctx context.Context, mgr *ArtifactManager) context.Context {
	return context.WithValue(ctx, artifactServiceKey, mgr)
}

// ArtifactManagerFromContext extracts ArtifactManager from context
func ArtifactManagerFromContext(ctx context.Context) *ArtifactManager {
	if mgr, ok := ctx.Value(artifactServiceKey).(*ArtifactManager); ok {
		return mgr
	}
	return nil
}

// MustArtifactManagerFromContext extracts ArtifactManager or panics
func MustArtifactManagerFromContext(ctx context.Context) *ArtifactManager {
	mgr := ArtifactManagerFromContext(ctx)
	if mgr == nil {
		panic("devflow: ArtifactManager not found in context")
	}
	return mgr
}

// WithPromptLoader adds a PromptLoader to the context
func WithPromptLoader(ctx context.Context, loader *PromptLoader) context.Context {
	return context.WithValue(ctx, promptServiceKey, loader)
}

// PromptLoaderFromContext extracts PromptLoader from context
func PromptLoaderFromContext(ctx context.Context) *PromptLoader {
	if loader, ok := ctx.Value(promptServiceKey).(*PromptLoader); ok {
		return loader
	}
	return nil
}

// MustPromptLoaderFromContext extracts PromptLoader or panics
func MustPromptLoaderFromContext(ctx context.Context) *PromptLoader {
	loader := PromptLoaderFromContext(ctx)
	if loader == nil {
		panic("devflow: PromptLoader not found in context")
	}
	return loader
}

// WithCommandRunner adds a CommandRunner to the context.
// This allows nodes to execute shell commands through a mockable interface.
func WithCommandRunner(ctx context.Context, runner CommandRunner) context.Context {
	return context.WithValue(ctx, runnerServiceKey, runner)
}

// CommandRunnerFromContext extracts CommandRunner from context.
// Returns nil if not set - callers should fall back to ExecRunner.
func CommandRunnerFromContext(ctx context.Context) CommandRunner {
	if runner, ok := ctx.Value(runnerServiceKey).(CommandRunner); ok {
		return runner
	}
	return nil
}

// GetCommandRunner returns the CommandRunner from context, or a default ExecRunner.
// This is the preferred way for nodes to get a runner - it always returns a usable runner.
func GetCommandRunner(ctx context.Context) CommandRunner {
	if runner := CommandRunnerFromContext(ctx); runner != nil {
		return runner
	}
	return NewExecRunner()
}

// DevServices wraps all devflow services for convenient initialization
type DevServices struct {
	Git         *GitContext
	LLM         llm.Client // flowgraph llm.Client interface
	Transcripts TranscriptManager
	Artifacts   *ArtifactManager
	Prompts     *PromptLoader
	Notifier    Notifier        // Optional notification service
	Runner      CommandRunner   // Optional command runner (defaults to ExecRunner)
}

// InjectAll adds all configured services to the context
func (d *DevServices) InjectAll(ctx context.Context) context.Context {
	if d.Git != nil {
		ctx = WithGitContext(ctx, d.Git)
	}
	if d.LLM != nil {
		ctx = WithLLMClient(ctx, d.LLM)
	}
	if d.Transcripts != nil {
		ctx = WithTranscriptManager(ctx, d.Transcripts)
	}
	if d.Artifacts != nil {
		ctx = WithArtifactManager(ctx, d.Artifacts)
	}
	if d.Prompts != nil {
		ctx = WithPromptLoader(ctx, d.Prompts)
	}
	if d.Notifier != nil {
		ctx = WithNotifier(ctx, d.Notifier)
	}
	if d.Runner != nil {
		ctx = WithCommandRunner(ctx, d.Runner)
	}
	return ctx
}

// DevServicesConfig configures NewDevServices
type DevServicesConfig struct {
	RepoPath  string // Path to git repository (required)
	BaseDir   string // Base directory for storage (default: ".devflow")
	PromptDir string // Directory for prompt templates (default: ".devflow/prompts")

	// LLM configuration
	LLMModel   string // Model to use (default: "claude-sonnet-4-20250514")
	LLMWorkdir string // Working directory for LLM (default: RepoPath)
}

// NewDevServices creates DevServices with common defaults
func NewDevServices(cfg DevServicesConfig) (*DevServices, error) {
	ds := &DevServices{}

	// Create GitContext
	git, err := NewGitContext(cfg.RepoPath)
	if err != nil {
		return nil, err
	}
	ds.Git = git

	// Create LLM client using flowgraph's llm.ClaudeCLI
	model := cfg.LLMModel
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}
	workdir := cfg.LLMWorkdir
	if workdir == "" {
		workdir = cfg.RepoPath
	}
	llmClient := llm.NewClaudeCLI(
		llm.WithModel(model),
		llm.WithWorkdir(workdir),
		llm.WithDangerouslySkipPermissions(), // Non-interactive mode for automation
	)
	ds.LLM = llmClient

	// Create base directory for storage
	baseDir := cfg.BaseDir
	if baseDir == "" {
		baseDir = ".devflow"
	}

	// Create TranscriptManager
	transcripts, err := NewFileTranscriptStore(baseDir)
	if err != nil {
		return nil, err
	}
	ds.Transcripts = transcripts

	// Create ArtifactManager
	ds.Artifacts = NewArtifactManager(ArtifactConfig{
		BaseDir: baseDir,
	})

	// Create PromptLoader
	promptDir := cfg.PromptDir
	if promptDir == "" {
		promptDir = ".devflow/prompts"
	}
	ds.Prompts = NewPromptLoader(promptDir)

	return ds, nil
}
