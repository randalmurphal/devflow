package context

import (
	"context"

	"github.com/randalmurphal/devflow/artifact"
	"github.com/randalmurphal/devflow/git"
	"github.com/randalmurphal/devflow/notify"
	"github.com/randalmurphal/devflow/prompt"
	"github.com/randalmurphal/devflow/transcript"
	"github.com/randalmurphal/flowgraph/pkg/flowgraph/llm"
)

// Services wraps all devflow services for convenient initialization
type Services struct {
	Git         *git.Context
	LLM         llm.Client // flowgraph llm.Client interface
	Transcripts transcript.Manager
	Artifacts   *artifact.Manager
	Prompts     *prompt.Loader
	Notifier    notify.Notifier    // Optional notification service
	Runner      git.CommandRunner  // Optional command runner (defaults to ExecRunner)
}

// InjectAll adds all configured services to the context
func (s *Services) InjectAll(ctx context.Context) context.Context {
	if s.Git != nil {
		ctx = WithGit(ctx, s.Git)
	}
	if s.LLM != nil {
		ctx = WithLLM(ctx, s.LLM)
	}
	if s.Transcripts != nil {
		ctx = WithTranscript(ctx, s.Transcripts)
	}
	if s.Artifacts != nil {
		ctx = WithArtifact(ctx, s.Artifacts)
	}
	if s.Prompts != nil {
		ctx = WithPrompt(ctx, s.Prompts)
	}
	if s.Notifier != nil {
		ctx = notify.WithNotifier(ctx, s.Notifier)
	}
	if s.Runner != nil {
		ctx = WithRunner(ctx, s.Runner)
	}
	return ctx
}

// Config configures NewServices
type Config struct {
	RepoPath  string // Path to git repository (required)
	BaseDir   string // Base directory for storage (default: ".devflow")
	PromptDir string // Directory for prompt templates (default: ".devflow/prompts")

	// LLM configuration
	LLMModel   string // Model to use (default: "claude-sonnet-4-20250514")
	LLMWorkdir string // Working directory for LLM (default: RepoPath)
}

// NewServices creates Services with common defaults
func NewServices(cfg Config) (*Services, error) {
	s := &Services{}

	// Create Git context
	gitCtx, err := git.NewContext(cfg.RepoPath)
	if err != nil {
		return nil, err
	}
	s.Git = gitCtx

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
	s.LLM = llmClient

	// Create base directory for storage
	baseDir := cfg.BaseDir
	if baseDir == "" {
		baseDir = ".devflow"
	}

	// Create transcript manager
	transcripts, err := transcript.NewFileStore(transcript.StoreConfig{
		BaseDir: baseDir,
	})
	if err != nil {
		return nil, err
	}
	s.Transcripts = transcripts

	// Create artifact manager
	s.Artifacts = artifact.NewManager(artifact.Config{
		BaseDir: baseDir,
	})

	// Create prompt loader
	promptDir := cfg.PromptDir
	if promptDir == "" {
		promptDir = ".devflow/prompts"
	}
	s.Prompts = prompt.NewLoader(promptDir)

	return s, nil
}
