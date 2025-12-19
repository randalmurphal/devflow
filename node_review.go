package devflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rmurphy/flowgraph/pkg/flowgraph/llm"
)

// ReviewNode reviews implementation for issues.
//
// Prerequisites: state.Spec or state.Implementation must be set
// Updates: state.Review, state.ReviewAttempts, state.ReviewTokensIn/Out
func ReviewNode(ctx context.Context, state DevState) (DevState, error) {
	client := LLMFromContext(ctx)
	if client == nil {
		return state, fmt.Errorf("llm.Client not found in context")
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

	// Load system prompt if available
	var systemPrompt string
	if loader := PromptLoaderFromContext(ctx); loader != nil {
		if sp, err := loader.Load("review-code"); err == nil {
			systemPrompt = sp
		}
	}

	// Increment attempts before running
	state.ReviewAttempts++

	// Run LLM
	result, err := client.Complete(ctx, llm.CompletionRequest{
		SystemPrompt: systemPrompt,
		Messages:     []llm.Message{{Role: llm.RoleUser, Content: prompt}},
	})
	if err != nil {
		state.SetError(err)
		return state, err
	}

	// Parse review result
	review, parseErr := parseReviewOutput(result.Content)
	if parseErr != nil {
		// If parsing fails, create a basic review
		review = &ReviewResult{
			Approved: false,
			Summary:  result.Content,
		}
	}

	state.Review = review
	state.ReviewTokensIn = result.Usage.InputTokens
	state.ReviewTokensOut = result.Usage.OutputTokens
	state.AddTokens(result.Usage.InputTokens, result.Usage.OutputTokens)

	// Save review artifact
	if artifacts := ArtifactManagerFromContext(ctx); artifacts != nil {
		artifacts.SaveReview(state.RunID, review)
	}

	return state, nil
}

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

	client := LLMFromContext(ctx)
	if client == nil {
		return state, fmt.Errorf("llm.Client not found in context")
	}

	// Build prompt with findings
	prompt := formatFixPrompt(state.Review)

	// Load system prompt if available
	var systemPrompt string
	if loader := PromptLoaderFromContext(ctx); loader != nil {
		if sp, err := loader.Load("fix-findings"); err == nil {
			systemPrompt = sp
		}
	}

	// Run LLM
	result, err := client.Complete(ctx, llm.CompletionRequest{
		SystemPrompt: systemPrompt,
		Messages:     []llm.Message{{Role: llm.RoleUser, Content: prompt}},
	})
	if err != nil {
		state.SetError(err)
		return state, err
	}

	state.Implementation = result.Content
	// Note: Files tracking now happens through git diff, not LLM response
	state.AddTokens(result.Usage.InputTokens, result.Usage.OutputTokens)

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
