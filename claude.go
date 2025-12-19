package devflow

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Claude CLI errors
var (
	// ErrClaudeNotFound indicates the claude CLI binary was not found.
	ErrClaudeNotFound = errors.New("claude CLI not found")

	// ErrClaudeTimeout indicates the claude CLI execution timed out.
	ErrClaudeTimeout = errors.New("claude CLI timed out")

	// ErrClaudeFailed indicates the claude CLI exited with an error.
	ErrClaudeFailed = errors.New("claude CLI failed")

	// ErrContextTooLarge indicates the context exceeds limits.
	ErrContextTooLarge = errors.New("context exceeds size limit")
)

// ClaudeCLI wraps the claude CLI binary for structured LLM invocation.
type ClaudeCLI struct {
	binaryPath string        // Path to claude binary
	model      string        // Default model (empty = use claude default)
	timeout    time.Duration // Default timeout
	maxTurns   int           // Default max conversation turns
}

// ClaudeConfig configures the Claude CLI wrapper.
type ClaudeConfig struct {
	BinaryPath string        // Path to claude binary (default: "claude")
	Model      string        // Default model (empty = use claude default)
	Timeout    time.Duration // Default timeout (default: 5m)
	MaxTurns   int           // Default max turns (default: 10)
}

// NewClaudeCLI creates a new Claude CLI wrapper.
// Returns ErrClaudeNotFound if the claude binary is not installed.
func NewClaudeCLI(cfg ClaudeConfig) (*ClaudeCLI, error) {
	binaryPath := cfg.BinaryPath
	if binaryPath == "" {
		binaryPath = "claude"
	}

	// Verify claude is installed
	if _, err := exec.LookPath(binaryPath); err != nil {
		return nil, ErrClaudeNotFound
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	maxTurns := cfg.MaxTurns
	if maxTurns == 0 {
		maxTurns = 10
	}

	return &ClaudeCLI{
		binaryPath: binaryPath,
		model:      cfg.Model,
		timeout:    timeout,
		maxTurns:   maxTurns,
	}, nil
}

// RunResult contains the output from a Claude CLI run.
type RunResult struct {
	Output    string        // Final output text
	TokensIn  int           // Input tokens consumed
	TokensOut int           // Output tokens generated
	Cost      float64       // Cost in USD
	SessionID string        // Session ID (for multi-turn conversations)
	Duration  time.Duration // Execution time
	ExitCode  int           // Process exit code
	Files     []FileChange  // Files created/modified
}

// FileChange represents a file change made by Claude.
type FileChange struct {
	Path   string         // File path
	Action FileAction     // Type of change
	Before string         // Content before (for modifications)
	After  string         // Content after (for modifications)
}

// FileAction represents the type of file change.
type FileAction string

const (
	FileActionCreate FileAction = "create"
	FileActionModify FileAction = "modify"
	FileActionDelete FileAction = "delete"
)

// runConfig holds configuration for a single run.
type runConfig struct {
	systemPrompt    string
	contextFiles    []string
	contextContent  string // Pre-built context content
	workDir         string
	maxTurns        int
	timeout         time.Duration
	model           string
	allowedTools    []string
	disallowedTools []string
	sessionID       string // Resume session
}

// RunOption configures a Run invocation.
type RunOption func(*runConfig)

// WithSystemPrompt sets the system prompt for Claude.
func WithSystemPrompt(prompt string) RunOption {
	return func(cfg *runConfig) {
		cfg.systemPrompt = prompt
	}
}

// WithContext adds context files to be read and included.
// Supports glob patterns.
func WithContext(files ...string) RunOption {
	return func(cfg *runConfig) {
		cfg.contextFiles = append(cfg.contextFiles, files...)
	}
}

// WithContextContent sets pre-built context content.
// Use this when you've already built the context with ContextBuilder.
func WithContextContent(content string) RunOption {
	return func(cfg *runConfig) {
		cfg.contextContent = content
	}
}

// WithWorkDir sets the working directory for Claude CLI.
func WithWorkDir(dir string) RunOption {
	return func(cfg *runConfig) {
		cfg.workDir = dir
	}
}

// WithMaxTurns limits the number of conversation turns.
func WithMaxTurns(n int) RunOption {
	return func(cfg *runConfig) {
		cfg.maxTurns = n
	}
}

// WithClaudeTimeout sets the timeout for this run.
func WithClaudeTimeout(d time.Duration) RunOption {
	return func(cfg *runConfig) {
		cfg.timeout = d
	}
}

// WithModel specifies the model to use for this run.
func WithModel(model string) RunOption {
	return func(cfg *runConfig) {
		cfg.model = model
	}
}

// WithAllowedTools specifies which tools Claude can use.
func WithAllowedTools(tools ...string) RunOption {
	return func(cfg *runConfig) {
		cfg.allowedTools = append(cfg.allowedTools, tools...)
	}
}

// WithDisallowedTools specifies which tools Claude cannot use.
func WithDisallowedTools(tools ...string) RunOption {
	return func(cfg *runConfig) {
		cfg.disallowedTools = append(cfg.disallowedTools, tools...)
	}
}

// WithSession resumes a previous session for multi-turn conversations.
func WithSession(sessionID string) RunOption {
	return func(cfg *runConfig) {
		cfg.sessionID = sessionID
	}
}

// Run executes Claude CLI with the given prompt and options.
func (c *ClaudeCLI) Run(ctx context.Context, prompt string, opts ...RunOption) (*RunResult, error) {
	// Build configuration with defaults
	cfg := &runConfig{
		timeout:  c.timeout,
		maxTurns: c.maxTurns,
		model:    c.model,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	// Build context if files specified
	if len(cfg.contextFiles) > 0 && cfg.contextContent == "" {
		workDir := cfg.workDir
		if workDir == "" {
			workDir = "."
		}
		builder := NewContextBuilder(workDir)
		for _, pattern := range cfg.contextFiles {
			if strings.ContainsAny(pattern, "*?[") {
				if err := builder.AddGlob(pattern); err != nil {
					return nil, fmt.Errorf("add context glob %s: %w", pattern, err)
				}
			} else {
				if err := builder.AddFile(pattern); err != nil {
					return nil, fmt.Errorf("add context file %s: %w", pattern, err)
				}
			}
		}
		content, err := builder.Build()
		if err != nil {
			return nil, fmt.Errorf("build context: %w", err)
		}
		cfg.contextContent = content
	}

	// Build the full prompt with context
	fullPrompt := prompt
	if cfg.contextContent != "" {
		fullPrompt = prompt + "\n\n## Context Files\n\n" + cfg.contextContent
	}

	// Build command arguments
	args := c.buildArgs(cfg, fullPrompt)

	// Create command with timeout
	ctx, cancel := context.WithTimeout(ctx, cfg.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.binaryPath, args...)
	if cfg.workDir != "" {
		cmd.Dir = cfg.workDir
	}

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	// Handle errors
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("%w: after %v", ErrClaudeTimeout, cfg.timeout)
		}
		if errors.Is(ctx.Err(), context.Canceled) {
			return nil, ctx.Err()
		}
		stderrStr := strings.TrimSpace(stderr.String())
		if stderrStr != "" {
			return nil, fmt.Errorf("%w: %s", ErrClaudeFailed, stderrStr)
		}
		return nil, fmt.Errorf("%w: %v", ErrClaudeFailed, err)
	}

	// Parse output
	result, err := parseClaudeOutput(stdout.Bytes())
	if err != nil {
		// Fallback to raw output
		result = &RunResult{
			Output: strings.TrimSpace(stdout.String()),
		}
	}

	result.Duration = duration
	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	}

	return result, nil
}

// buildArgs constructs command line arguments for claude CLI.
func (c *ClaudeCLI) buildArgs(cfg *runConfig, prompt string) []string {
	args := []string{"--print", "--output-format", "json"}

	if cfg.model != "" {
		args = append(args, "--model", cfg.model)
	}
	if cfg.systemPrompt != "" {
		args = append(args, "--system-prompt", cfg.systemPrompt)
	}
	if cfg.maxTurns > 0 {
		args = append(args, "--max-turns", fmt.Sprintf("%d", cfg.maxTurns))
	}
	if cfg.sessionID != "" {
		args = append(args, "--resume", cfg.sessionID)
	}
	for _, tool := range cfg.allowedTools {
		args = append(args, "--allowedTools", tool)
	}
	for _, tool := range cfg.disallowedTools {
		args = append(args, "--disallowedTools", tool)
	}

	// Add prompt (use -p for inline prompt)
	args = append(args, "-p", prompt)

	return args
}

// claudeJSONOutput represents the JSON output from claude CLI.
type claudeJSONOutput struct {
	Result       string  `json:"result"`
	TokensIn     int     `json:"tokens_in"`
	TokensOut    int     `json:"tokens_out"`
	TotalTokens  int     `json:"total_tokens"`
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	Cost         float64 `json:"cost"`
	CostUSD      float64 `json:"cost_usd"`
	SessionID    string  `json:"session_id"`
}

// parseClaudeOutput parses the JSON output from claude CLI.
func parseClaudeOutput(data []byte) (*RunResult, error) {
	// Try to find JSON in the output (it may be mixed with other content)
	data = bytes.TrimSpace(data)

	// Try direct parse first
	var output claudeJSONOutput
	if err := json.Unmarshal(data, &output); err != nil {
		// Try to find JSON object in the output
		start := bytes.Index(data, []byte("{"))
		end := bytes.LastIndex(data, []byte("}"))
		if start >= 0 && end > start {
			if err := json.Unmarshal(data[start:end+1], &output); err != nil {
				return nil, fmt.Errorf("parse json output: %w", err)
			}
		} else {
			return nil, fmt.Errorf("no json found in output")
		}
	}

	// Handle different field names for tokens
	tokensIn := output.TokensIn
	if tokensIn == 0 {
		tokensIn = output.InputTokens
	}
	tokensOut := output.TokensOut
	if tokensOut == 0 {
		tokensOut = output.OutputTokens
	}

	// Handle different field names for cost
	cost := output.Cost
	if cost == 0 {
		cost = output.CostUSD
	}

	return &RunResult{
		Output:    output.Result,
		TokensIn:  tokensIn,
		TokensOut: tokensOut,
		Cost:      cost,
		SessionID: output.SessionID,
	}, nil
}

// BinaryPath returns the path to the claude binary.
func (c *ClaudeCLI) BinaryPath() string {
	return c.binaryPath
}

// DefaultModel returns the default model.
func (c *ClaudeCLI) DefaultModel() string {
	return c.model
}

// DefaultTimeout returns the default timeout.
func (c *ClaudeCLI) DefaultTimeout() time.Duration {
	return c.timeout
}

// DefaultMaxTurns returns the default max turns.
func (c *ClaudeCLI) DefaultMaxTurns() int {
	return c.maxTurns
}
