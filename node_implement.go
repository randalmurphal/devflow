package devflow

import (
	"fmt"
	"strings"

	"github.com/randalmurphal/flowgraph/pkg/flowgraph"
	"github.com/randalmurphal/flowgraph/pkg/flowgraph/llm"
)

// ImplementNode implements code based on the specification.
//
// Prerequisites: state.Spec, state.Worktree must be set
// Updates: state.Implementation, state.Files, state.ImplementTokensIn/Out
func ImplementNode(ctx flowgraph.Context, state DevState) (DevState, error) {
	if err := state.Validate(RequireSpec, RequireWorktree); err != nil {
		return state, err
	}

	client := LLMFromContext(ctx)
	if client == nil {
		return state, fmt.Errorf("llm.Client not found in context")
	}

	// Build prompt
	prompt := formatImplementPrompt(state.Spec, state.Ticket)

	// Load system prompt if available
	var systemPrompt string
	if loader := PromptLoaderFromContext(ctx); loader != nil {
		if sp, err := loader.Load("implement"); err == nil {
			systemPrompt = sp
		}
	}

	// Run LLM
	// Note: For implementation nodes that need to execute in a specific directory,
	// the caller should configure the LLM client with the appropriate workdir
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
	state.ImplementTokensIn = result.Usage.InputTokens
	state.ImplementTokensOut = result.Usage.OutputTokens
	state.AddTokens(result.Usage.InputTokens, result.Usage.OutputTokens)

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
