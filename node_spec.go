package devflow

import (
	"fmt"
	"strings"
	"time"

	"github.com/randalmurphal/flowgraph/pkg/flowgraph"
	"github.com/randalmurphal/flowgraph/pkg/flowgraph/llm"
)

// GenerateSpecNode generates a technical specification from the ticket.
//
// Prerequisites: state.Ticket must be set
// Updates: state.Spec, state.SpecTokensIn/Out, state.SpecGeneratedAt
func GenerateSpecNode(ctx flowgraph.Context, state DevState) (DevState, error) {
	if err := state.Validate(RequireTicket); err != nil {
		return state, err
	}

	client := LLMFromContext(ctx)
	if client == nil {
		return state, fmt.Errorf("llm.Client not found in context")
	}

	// Build prompt
	prompt := formatSpecPrompt(state.Ticket)

	// Load system prompt if available
	var systemPrompt string
	if loader := PromptLoaderFromContext(ctx); loader != nil {
		if sp, err := loader.Load("generate-spec"); err == nil {
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

	state.Spec = result.Content
	state.SpecTokensIn = result.Usage.InputTokens
	state.SpecTokensOut = result.Usage.OutputTokens
	state.SpecGeneratedAt = time.Now()
	state.AddTokens(result.Usage.InputTokens, result.Usage.OutputTokens)

	// Save artifact if manager available
	if artifacts := ArtifactManagerFromContext(ctx); artifacts != nil {
		artifacts.SaveSpec(state.RunID, result.Content)
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
