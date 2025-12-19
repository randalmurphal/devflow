package devflow

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// =============================================================================
// Node Types
// =============================================================================

// NodeFunc is a function that processes state and returns updated state.
// This signature is compatible with flowgraph's NodeFunc[DevState].
type NodeFunc func(ctx context.Context, state DevState) (DevState, error)

// NodeConfig configures node behavior
type NodeConfig struct {
	MaxReviewAttempts int    // Max review/fix cycles (default: 3)
	TestCommand       string // Test command (default: "go test ./...")
	LintCommand       string // Lint command (default: "go vet ./...")
	BaseBranch        string // Default base branch (default: "main")
}

// DefaultNodeConfig returns sensible defaults
func DefaultNodeConfig() NodeConfig {
	return NodeConfig{
		MaxReviewAttempts: 3,
		TestCommand:       "go test -race ./...",
		LintCommand:       "go vet ./...",
		BaseBranch:        "main",
	}
}

// =============================================================================
// CreateWorktreeNode
// =============================================================================

// CreateWorktreeNode creates an isolated git worktree for the task.
//
// Prerequisites: state.TicketID or state.Branch must be set
// Updates: state.Worktree, state.Branch
func CreateWorktreeNode(ctx context.Context, state DevState) (DevState, error) {
	git := GitFromContext(ctx)
	if git == nil {
		return state, fmt.Errorf("GitContext not found in context")
	}

	// Determine branch name
	branch := state.Branch
	if branch == "" {
		if state.TicketID != "" {
			namer := DefaultBranchNamer()
			branch = namer.ForTicket(state.TicketID, "")
		} else {
			branch = fmt.Sprintf("devflow/%s", state.RunID)
		}
	}

	// Determine base branch
	baseBranch := state.BaseBranch
	if baseBranch == "" {
		baseBranch = "main"
	}

	// Create worktree
	worktreePath, err := git.CreateWorktree(branch)
	if err != nil {
		state.SetError(err)
		return state, err
	}

	state.Worktree = worktreePath
	state.Branch = branch
	state.BaseBranch = baseBranch

	return state, nil
}

// =============================================================================
// GenerateSpecNode
// =============================================================================

// GenerateSpecNode generates a technical specification from the ticket.
//
// Prerequisites: state.Ticket must be set
// Updates: state.Spec, state.SpecTokensIn/Out, state.SpecGeneratedAt
func GenerateSpecNode(ctx context.Context, state DevState) (DevState, error) {
	if err := state.Validate(RequireTicket); err != nil {
		return state, err
	}

	claude := ClaudeFromContext(ctx)
	if claude == nil {
		return state, fmt.Errorf("ClaudeCLI not found in context")
	}

	// Build prompt
	prompt := formatSpecPrompt(state.Ticket)

	// Load system prompt if available
	var opts []RunOption
	if loader := PromptLoaderFromContext(ctx); loader != nil {
		if systemPrompt, err := loader.Load("generate-spec"); err == nil {
			opts = append(opts, WithSystemPrompt(systemPrompt))
		}
	}

	// Run Claude
	result, err := claude.Run(ctx, prompt, opts...)
	if err != nil {
		state.SetError(err)
		return state, err
	}

	state.Spec = result.Output
	state.SpecTokensIn = result.TokensIn
	state.SpecTokensOut = result.TokensOut
	state.SpecGeneratedAt = time.Now()
	state.AddTokens(result.TokensIn, result.TokensOut)

	// Save artifact if manager available
	if artifacts := ArtifactManagerFromContext(ctx); artifacts != nil {
		artifacts.SaveSpec(state.RunID, result.Output)
	}

	return state, nil
}

// formatSpecPrompt creates the spec generation prompt
func formatSpecPrompt(ticket *Ticket) string {
	var b strings.Builder
	b.WriteString("Generate a technical specification for this ticket:\n\n")
	b.WriteString(fmt.Sprintf("**Ticket ID**: %s\n", ticket.ID))
	b.WriteString(fmt.Sprintf("**Title**: %s\n\n", ticket.Title))
	if ticket.Description != "" {
		b.WriteString(fmt.Sprintf("**Description**:\n%s\n\n", ticket.Description))
	}
	if len(ticket.Labels) > 0 {
		b.WriteString(fmt.Sprintf("**Labels**: %s\n\n", strings.Join(ticket.Labels, ", ")))
	}
	b.WriteString("Please provide:\n")
	b.WriteString("1. Overview of the changes needed\n")
	b.WriteString("2. Technical design approach\n")
	b.WriteString("3. Files that will need to be modified\n")
	b.WriteString("4. Testing strategy\n")
	b.WriteString("5. Potential risks or concerns\n")
	return b.String()
}

// =============================================================================
// ImplementNode
// =============================================================================

// ImplementNode implements code based on the specification.
//
// Prerequisites: state.Spec, state.Worktree must be set
// Updates: state.Implementation, state.Files, state.ImplementTokensIn/Out
func ImplementNode(ctx context.Context, state DevState) (DevState, error) {
	if err := state.Validate(RequireSpec, RequireWorktree); err != nil {
		return state, err
	}

	claude := ClaudeFromContext(ctx)
	if claude == nil {
		return state, fmt.Errorf("ClaudeCLI not found in context")
	}

	// Build prompt
	prompt := formatImplementPrompt(state.Spec, state.Ticket)

	// Configure run options
	opts := []RunOption{
		WithWorkDir(state.Worktree),
	}

	// Load system prompt if available
	if loader := PromptLoaderFromContext(ctx); loader != nil {
		if systemPrompt, err := loader.Load("implement"); err == nil {
			opts = append(opts, WithSystemPrompt(systemPrompt))
		}
	}

	// Run Claude
	result, err := claude.Run(ctx, prompt, opts...)
	if err != nil {
		state.SetError(err)
		return state, err
	}

	state.Implementation = result.Output
	state.Files = result.Files
	state.ImplementTokensIn = result.TokensIn
	state.ImplementTokensOut = result.TokensOut
	state.AddTokens(result.TokensIn, result.TokensOut)

	// Save implementation diff if artifacts available
	if artifacts := ArtifactManagerFromContext(ctx); artifacts != nil {
		// Get diff of changes (staged and unstaged vs HEAD)
		if git := GitFromContext(ctx); git != nil {
			if diff, err := git.Diff("HEAD", ""); err == nil {
				artifacts.SaveDiff(state.RunID, diff)
			}
		}
	}

	return state, nil
}

// formatImplementPrompt creates the implementation prompt
func formatImplementPrompt(spec string, ticket *Ticket) string {
	var b strings.Builder
	b.WriteString("Implement the following specification:\n\n")
	b.WriteString("## Specification\n\n")
	b.WriteString(spec)
	b.WriteString("\n\n")
	if ticket != nil {
		b.WriteString(fmt.Sprintf("Original ticket: %s - %s\n\n", ticket.ID, ticket.Title))
	}
	b.WriteString("Please implement this in the codebase. ")
	b.WriteString("Make sure to:\n")
	b.WriteString("- Follow existing code patterns\n")
	b.WriteString("- Add appropriate tests\n")
	b.WriteString("- Update documentation if needed\n")
	return b.String()
}

// =============================================================================
// ReviewNode
// =============================================================================

// ReviewNode reviews implementation for issues.
//
// Prerequisites: state.Spec or state.Implementation must be set
// Updates: state.Review, state.ReviewAttempts, state.ReviewTokensIn/Out
func ReviewNode(ctx context.Context, state DevState) (DevState, error) {
	claude := ClaudeFromContext(ctx)
	if claude == nil {
		return state, fmt.Errorf("ClaudeCLI not found in context")
	}

	// Get diff to review
	var diff string
	if git := GitFromContext(ctx); git != nil && state.Worktree != "" {
		var err error
		diff, err = git.Diff("HEAD", "")
		if err != nil {
			diff = state.Implementation // Fallback to stored implementation
		}
	} else {
		diff = state.Implementation
	}

	if diff == "" {
		return state, fmt.Errorf("no implementation to review")
	}

	// Build prompt
	prompt := formatReviewPrompt(diff, state.Spec)

	// Configure run options
	var opts []RunOption
	if loader := PromptLoaderFromContext(ctx); loader != nil {
		if systemPrompt, err := loader.Load("review-code"); err == nil {
			opts = append(opts, WithSystemPrompt(systemPrompt))
		}
	}

	// Increment attempts before running
	state.ReviewAttempts++

	// Run Claude
	result, err := claude.Run(ctx, prompt, opts...)
	if err != nil {
		state.SetError(err)
		return state, err
	}

	// Parse review result
	review, parseErr := parseReviewOutput(result.Output)
	if parseErr != nil {
		// If parsing fails, create a basic review
		review = &ReviewResult{
			Approved: false,
			Summary:  result.Output,
		}
	}

	state.Review = review
	state.ReviewTokensIn = result.TokensIn
	state.ReviewTokensOut = result.TokensOut
	state.AddTokens(result.TokensIn, result.TokensOut)

	// Save review artifact
	if artifacts := ArtifactManagerFromContext(ctx); artifacts != nil {
		artifacts.SaveReview(state.RunID, review)
	}

	return state, nil
}

// formatReviewPrompt creates the code review prompt
func formatReviewPrompt(diff string, spec string) string {
	var b strings.Builder
	b.WriteString("Please review this code change:\n\n")
	b.WriteString("## Diff\n\n```diff\n")
	b.WriteString(diff)
	b.WriteString("\n```\n\n")
	if spec != "" {
		b.WriteString("## Original Specification\n\n")
		b.WriteString(spec)
		b.WriteString("\n\n")
	}
	b.WriteString("Please review for:\n")
	b.WriteString("- Correctness and logic errors\n")
	b.WriteString("- Security issues\n")
	b.WriteString("- Performance concerns\n")
	b.WriteString("- Code style and readability\n")
	b.WriteString("- Test coverage\n\n")
	b.WriteString("Respond with a JSON object:\n")
	b.WriteString("```json\n")
	b.WriteString(`{"approved": true/false, "verdict": "APPROVE/REQUEST_CHANGES", "summary": "...", "findings": [...]}`)
	b.WriteString("\n```\n")
	return b.String()
}

// parseReviewOutput attempts to parse review JSON from Claude output
func parseReviewOutput(output string) (*ReviewResult, error) {
	// Try to extract JSON from code blocks
	output = strings.TrimSpace(output)

	// Look for JSON in code blocks
	if start := strings.Index(output, "```json"); start != -1 {
		start += 7
		if end := strings.Index(output[start:], "```"); end != -1 {
			output = strings.TrimSpace(output[start : start+end])
		}
	} else if start := strings.Index(output, "```"); start != -1 {
		start += 3
		if end := strings.Index(output[start:], "```"); end != -1 {
			output = strings.TrimSpace(output[start : start+end])
		}
	}

	// Parse JSON
	var review ReviewResult
	if err := json.Unmarshal([]byte(output), &review); err != nil {
		return nil, err
	}

	return &review, nil
}

// =============================================================================
// FixFindingsNode
// =============================================================================

// FixFindingsNode fixes issues found in review.
//
// Prerequisites: state.Review with findings, state.Worktree
// Updates: state.Implementation, state.Files
func FixFindingsNode(ctx context.Context, state DevState) (DevState, error) {
	if err := state.Validate(RequireReview, RequireWorktree); err != nil {
		return state, err
	}

	if state.Review.Approved {
		// Nothing to fix
		return state, nil
	}

	claude := ClaudeFromContext(ctx)
	if claude == nil {
		return state, fmt.Errorf("ClaudeCLI not found in context")
	}

	// Build prompt with findings
	prompt := formatFixPrompt(state.Review)

	// Configure run options
	opts := []RunOption{
		WithWorkDir(state.Worktree),
	}

	if loader := PromptLoaderFromContext(ctx); loader != nil {
		if systemPrompt, err := loader.Load("fix-findings"); err == nil {
			opts = append(opts, WithSystemPrompt(systemPrompt))
		}
	}

	// Run Claude
	result, err := claude.Run(ctx, prompt, opts...)
	if err != nil {
		state.SetError(err)
		return state, err
	}

	state.Implementation = result.Output
	state.Files = result.Files
	state.AddTokens(result.TokensIn, result.TokensOut)

	return state, nil
}

// formatFixPrompt creates the fix findings prompt
func formatFixPrompt(review *ReviewResult) string {
	var b strings.Builder
	b.WriteString("Please fix the following issues found during code review:\n\n")
	b.WriteString(fmt.Sprintf("## Review Summary\n\n%s\n\n", review.Summary))

	if len(review.Findings) > 0 {
		b.WriteString("## Findings to Address\n\n")
		for i, f := range review.Findings {
			b.WriteString(fmt.Sprintf("### %d. %s (%s)\n", i+1, f.Category, f.Severity))
			b.WriteString(fmt.Sprintf("**File**: %s", f.File))
			if f.Line > 0 {
				b.WriteString(fmt.Sprintf(":%d", f.Line))
			}
			b.WriteString("\n")
			b.WriteString(fmt.Sprintf("**Issue**: %s\n", f.Message))
			if f.Suggestion != "" {
				b.WriteString(fmt.Sprintf("**Suggestion**: %s\n", f.Suggestion))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("Please address each issue and ensure the code is correct.\n")
	return b.String()
}

// =============================================================================
// RunTestsNode
// =============================================================================

// RunTestsNode runs the test suite.
//
// Prerequisites: state.Worktree must be set
// Updates: state.TestOutput, state.TestPassed, state.TestRunAt
func RunTestsNode(ctx context.Context, state DevState) (DevState, error) {
	if err := state.Validate(RequireWorktree); err != nil {
		return state, err
	}

	// Get test command from config or use default
	testCmd := "go test -race ./..."
	// Could be configurable via context

	// Run tests
	cmd := exec.CommandContext(ctx, "sh", "-c", testCmd)
	cmd.Dir = state.Worktree

	output, err := cmd.CombinedOutput()
	passed := err == nil

	// Parse test output
	testOutput := parseTestOutput(string(output), passed)

	state.TestOutput = testOutput
	state.TestPassed = passed
	state.TestRunAt = time.Now()

	// Save test output artifact
	if artifacts := ArtifactManagerFromContext(ctx); artifacts != nil {
		artifacts.SaveTestOutput(state.RunID, testOutput)
	}

	// Don't return error for test failures - let the graph handle routing
	return state, nil
}

// parseTestOutput parses test command output
func parseTestOutput(output string, passed bool) *TestOutput {
	result := &TestOutput{
		Passed: passed,
	}

	// Count test results from output
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ok ") {
			result.PassedTests++
			result.TotalTests++
		} else if strings.HasPrefix(line, "FAIL") {
			result.FailedTests++
			result.TotalTests++
		} else if strings.HasPrefix(line, "--- FAIL:") {
			// Extract failure details
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				result.Failures = append(result.Failures, TestFailure{
					Name:   parts[2],
					Output: output, // Full output for context
				})
			}
		}
	}

	return result
}

// =============================================================================
// CheckLintNode
// =============================================================================

// CheckLintNode runs linting and type checks.
//
// Prerequisites: state.Worktree must be set
// Updates: state.LintOutput, state.LintPassed, state.LintRunAt
func CheckLintNode(ctx context.Context, state DevState) (DevState, error) {
	if err := state.Validate(RequireWorktree); err != nil {
		return state, err
	}

	// Get lint command from config or use default
	lintCmd := "go vet ./..."

	// Run linter
	cmd := exec.CommandContext(ctx, "sh", "-c", lintCmd)
	cmd.Dir = state.Worktree

	output, err := cmd.CombinedOutput()
	passed := err == nil

	// Parse lint output
	lintOutput := parseLintOutput(string(output), passed)

	state.LintOutput = lintOutput
	state.LintPassed = passed
	state.LintRunAt = time.Now()

	// Save lint output artifact
	if artifacts := ArtifactManagerFromContext(ctx); artifacts != nil {
		artifacts.SaveLintOutput(state.RunID, lintOutput)
	}

	return state, nil
}

// parseLintOutput parses linter output
func parseLintOutput(output string, passed bool) *LintOutput {
	result := &LintOutput{
		Passed: passed,
		Tool:   "go vet",
	}

	if passed {
		return result
	}

	// Parse issues from output
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Go vet format: file.go:line:col: message
		if parts := strings.SplitN(line, ":", 4); len(parts) >= 3 {
			result.Issues = append(result.Issues, LintIssue{
				File:     parts[0],
				Line:     parseIntSafe(parts[1]),
				Message:  strings.TrimSpace(strings.Join(parts[2:], ":")),
				Severity: "warning",
			})
			result.Summary.TotalIssues++
			result.Summary.Warnings++
		}
	}

	return result
}

// parseIntSafe safely parses int, returns 0 on error
func parseIntSafe(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

// =============================================================================
// CreatePRNode
// =============================================================================

// CreatePRNode creates a pull request.
//
// Prerequisites: state.Branch must be set and pushed
// Updates: state.PR, state.PRCreated
func CreatePRNode(ctx context.Context, state DevState) (DevState, error) {
	if err := state.Validate(RequireBranch); err != nil {
		return state, err
	}

	git := GitFromContext(ctx)
	if git == nil {
		return state, fmt.Errorf("GitContext not found in context")
	}

	// Ensure changes are committed
	if err := commitChanges(ctx, git, state); err != nil {
		// Not fatal - might already be committed
	}

	// Push branch
	if err := git.Push("origin", state.Branch, true); err != nil {
		state.SetError(err)
		return state, err
	}

	// Create PR
	prOpts := buildPROptions(state)

	pr, err := git.CreatePR(ctx, prOpts)
	if err != nil {
		state.SetError(err)
		return state, err
	}

	state.PR = pr
	state.PRCreated = time.Now()

	return state, nil
}

// commitChanges commits any uncommitted changes
func commitChanges(ctx context.Context, git *GitContext, state DevState) error {
	// Check for changes
	status, err := git.Status()
	if err != nil {
		return err
	}

	if status == "" {
		return nil // Nothing to commit
	}

	// Stage all changes
	if err := git.StageAll(); err != nil {
		return err
	}

	// Create commit message
	msg := buildCommitMessage(state)
	return git.Commit(msg)
}

// buildCommitMessage creates a commit message from state
func buildCommitMessage(state DevState) string {
	var title string
	if state.Ticket != nil {
		title = fmt.Sprintf("[%s] %s", state.TicketID, state.Ticket.Title)
	} else if state.TicketID != "" {
		title = fmt.Sprintf("[%s] Implementation", state.TicketID)
	} else {
		title = fmt.Sprintf("Implementation for %s", state.RunID)
	}
	return title
}

// buildPROptions creates PR options from state
func buildPROptions(state DevState) PROptions {
	builder := NewPRBuilder(getPRTitle(state))

	// Set body from spec or summary
	var body string
	if state.Spec != "" {
		body = fmt.Sprintf("## Specification\n\n%s", state.Spec)
	} else {
		body = "Implementation created by devflow."
	}

	// Add test results if available
	if state.TestOutput != nil {
		body += fmt.Sprintf("\n\n## Test Results\n\n- Passed: %d\n- Failed: %d",
			state.TestOutput.PassedTests, state.TestOutput.FailedTests)
	}

	builder.WithBody(body)

	// Set draft if review found issues
	if state.Review != nil && !state.Review.Approved {
		builder.AsDraft()
	}

	// Add ticket label
	if state.TicketID != "" {
		builder.WithLabels(state.TicketID)
	}

	return builder.Build()
}

// getPRTitle generates PR title from state
func getPRTitle(state DevState) string {
	if state.Ticket != nil {
		return fmt.Sprintf("[%s] %s", state.TicketID, state.Ticket.Title)
	}
	if state.TicketID != "" {
		return fmt.Sprintf("[%s] Implementation", state.TicketID)
	}
	return fmt.Sprintf("devflow: %s", state.RunID)
}

// =============================================================================
// CleanupNode
// =============================================================================

// CleanupNode cleans up the worktree.
//
// Prerequisites: state.Worktree must be set
// Updates: clears state.Worktree
func CleanupNode(ctx context.Context, state DevState) (DevState, error) {
	if state.Worktree == "" {
		return state, nil // Nothing to clean
	}

	git := GitFromContext(ctx)
	if git == nil {
		return state, fmt.Errorf("GitContext not found in context")
	}

	if err := git.CleanupWorktree(state.Worktree); err != nil {
		// Log but don't fail - cleanup is best effort
		state.Error = fmt.Sprintf("cleanup warning: %v", err)
	}

	state.Worktree = ""

	// Finalize metrics
	state.FinalizeDuration()

	return state, nil
}

// =============================================================================
// Node Wrappers
// =============================================================================

// WithRetry wraps a node with retry logic
func WithRetry(node NodeFunc, maxRetries int) NodeFunc {
	return func(ctx context.Context, state DevState) (DevState, error) {
		var lastErr error
		for i := 0; i < maxRetries; i++ {
			result, err := node(ctx, state)
			if err == nil {
				return result, nil
			}
			lastErr = err
			// Exponential backoff could go here
		}
		return state, fmt.Errorf("after %d retries: %w", maxRetries, lastErr)
	}
}

// WithTranscript wraps a node with transcript recording
func WithTranscript(node NodeFunc, nodeName string) NodeFunc {
	return func(ctx context.Context, state DevState) (DevState, error) {
		mgr := TranscriptManagerFromContext(ctx)

		startTime := time.Now()
		result, err := node(ctx, state)
		duration := time.Since(startTime)

		if mgr != nil {
			turn := Turn{
				Role:      "system",
				Content:   fmt.Sprintf("Node %s completed in %v", nodeName, duration),
				Timestamp: time.Now(),
			}
			if err != nil {
				turn.Content = fmt.Sprintf("Node %s failed: %v", nodeName, err)
			}
			mgr.RecordTurn(state.RunID, turn)
		}

		return result, err
	}
}

// WithTiming wraps a node with timing metrics
func WithTiming(node NodeFunc) NodeFunc {
	return func(ctx context.Context, state DevState) (DevState, error) {
		start := time.Now()
		result, err := node(ctx, state)
		// Duration could be added to state or logged
		_ = time.Since(start)
		return result, err
	}
}

// =============================================================================
// Review Router (for conditional edges)
// =============================================================================

// ReviewRouter returns the next node based on review results.
// Used with flowgraph conditional edges.
func ReviewRouter(state DevState, maxAttempts int) string {
	if state.Review != nil && state.Review.Approved {
		return "create-pr"
	}
	if state.ReviewAttempts >= maxAttempts {
		return "create-pr" // Give up, create as draft
	}
	return "fix-findings"
}

// DefaultReviewRouter uses 3 max attempts
func DefaultReviewRouter(state DevState) string {
	return ReviewRouter(state, 3)
}
