package context

import (
	"context"

	"github.com/randalmurphal/devflow/artifact"
	"github.com/randalmurphal/devflow/git"
	"github.com/randalmurphal/devflow/pr"
	"github.com/randalmurphal/devflow/prompt"
	"github.com/randalmurphal/devflow/transcript"
	"github.com/randalmurphal/llmkit/claude"
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
	prServiceKey         serviceContextKey = "devflow.pr"
)

// WithGit adds a Git context to the context
func WithGit(ctx context.Context, gitCtx *git.Context) context.Context {
	return context.WithValue(ctx, gitServiceKey, gitCtx)
}

// Git extracts Git context from context
func Git(ctx context.Context) *git.Context {
	if gitCtx, ok := ctx.Value(gitServiceKey).(*git.Context); ok {
		return gitCtx
	}
	return nil
}

// MustGit extracts Git context or panics
func MustGit(ctx context.Context) *git.Context {
	gitCtx := Git(ctx)
	if gitCtx == nil {
		panic("devflow/context: git.Context not found in context")
	}
	return gitCtx
}

// WithLLM adds an LLM client to the context.
// This uses flowgraph's claude.Client interface.
func WithLLM(ctx context.Context, client claude.Client) context.Context {
	return context.WithValue(ctx, llmServiceKey, client)
}

// LLM extracts the LLM client from context.
func LLM(ctx context.Context) claude.Client {
	if client, ok := ctx.Value(llmServiceKey).(claude.Client); ok {
		return client
	}
	return nil
}

// MustLLM extracts the LLM client or panics.
func MustLLM(ctx context.Context) claude.Client {
	client := LLM(ctx)
	if client == nil {
		panic("devflow/context: claude.Client not found in context")
	}
	return client
}

// WithTranscript adds a transcript manager to the context
func WithTranscript(ctx context.Context, mgr transcript.Manager) context.Context {
	return context.WithValue(ctx, transcriptServiceKey, mgr)
}

// Transcript extracts transcript manager from context
func Transcript(ctx context.Context) transcript.Manager {
	if mgr, ok := ctx.Value(transcriptServiceKey).(transcript.Manager); ok {
		return mgr
	}
	return nil
}

// MustTranscript extracts transcript manager or panics
func MustTranscript(ctx context.Context) transcript.Manager {
	mgr := Transcript(ctx)
	if mgr == nil {
		panic("devflow/context: transcript.Manager not found in context")
	}
	return mgr
}

// WithArtifact adds an artifact manager to the context
func WithArtifact(ctx context.Context, mgr *artifact.Manager) context.Context {
	return context.WithValue(ctx, artifactServiceKey, mgr)
}

// Artifact extracts artifact manager from context
func Artifact(ctx context.Context) *artifact.Manager {
	if mgr, ok := ctx.Value(artifactServiceKey).(*artifact.Manager); ok {
		return mgr
	}
	return nil
}

// MustArtifact extracts artifact manager or panics
func MustArtifact(ctx context.Context) *artifact.Manager {
	mgr := Artifact(ctx)
	if mgr == nil {
		panic("devflow/context: artifact.Manager not found in context")
	}
	return mgr
}

// WithPrompt adds a prompt loader to the context
func WithPrompt(ctx context.Context, loader *prompt.Loader) context.Context {
	return context.WithValue(ctx, promptServiceKey, loader)
}

// Prompt extracts prompt loader from context
func Prompt(ctx context.Context) *prompt.Loader {
	if loader, ok := ctx.Value(promptServiceKey).(*prompt.Loader); ok {
		return loader
	}
	return nil
}

// MustPrompt extracts prompt loader or panics
func MustPrompt(ctx context.Context) *prompt.Loader {
	loader := Prompt(ctx)
	if loader == nil {
		panic("devflow/context: prompt.Loader not found in context")
	}
	return loader
}

// WithRunner adds a command runner to the context.
// This allows nodes to execute shell commands through a mockable interface.
func WithRunner(ctx context.Context, runner git.CommandRunner) context.Context {
	return context.WithValue(ctx, runnerServiceKey, runner)
}

// Runner extracts command runner from context.
// Returns nil if not set - callers should fall back to ExecRunner.
func Runner(ctx context.Context) git.CommandRunner {
	if runner, ok := ctx.Value(runnerServiceKey).(git.CommandRunner); ok {
		return runner
	}
	return nil
}

// GetRunner returns the command runner from context, or a default ExecRunner.
// This is the preferred way for nodes to get a runner - it always returns a usable runner.
func GetRunner(ctx context.Context) git.CommandRunner {
	if runner := Runner(ctx); runner != nil {
		return runner
	}
	return git.NewExecRunner()
}

// WithPR adds a PR provider to the context
func WithPR(ctx context.Context, provider pr.Provider) context.Context {
	return context.WithValue(ctx, prServiceKey, provider)
}

// PR extracts PR provider from context
func PR(ctx context.Context) pr.Provider {
	if provider, ok := ctx.Value(prServiceKey).(pr.Provider); ok {
		return provider
	}
	return nil
}

// MustPR extracts PR provider or panics
func MustPR(ctx context.Context) pr.Provider {
	provider := PR(ctx)
	if provider == nil {
		panic("devflow/context: pr.Provider not found in context")
	}
	return provider
}
