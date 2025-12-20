package workflow

import (
	"context"
	"fmt"
	"time"

	devcontext "github.com/randalmurphal/devflow/context"
	"github.com/randalmurphal/devflow/git"
	"github.com/randalmurphal/devflow/pr"
	"github.com/randalmurphal/flowgraph/pkg/flowgraph"
)

// CreatePRNode creates a pull request.
//
// Prerequisites: state.Branch must be set and pushed
// Updates: state.PR, state.PRCreated
func CreatePRNode(ctx flowgraph.Context, state State) (State, error) {
	if err := state.Validate(RequireBranch); err != nil {
		return state, err
	}

	// Get git context using devflow context package
	gitCtx := devcontext.Git(ctx)
	if gitCtx == nil {
		return state, fmt.Errorf("git.Context not found in context")
	}

	// Ensure changes are committed
	if err := commitChanges(gitCtx, state); err != nil {
		// Not fatal - might already be committed
	}

	// Push branch
	if err := gitCtx.Push("origin", state.Branch, true); err != nil {
		state.SetError(err)
		return state, err
	}

	// Get PR provider from context
	provider := devcontext.PR(ctx)
	if provider == nil {
		return state, fmt.Errorf("pr.Provider not found in context")
	}

	// Create PR
	prOpts := buildPROptions(state)

	pullRequest, err := provider.CreatePR(context.Background(), prOpts)
	if err != nil {
		state.SetError(err)
		return state, err
	}

	state.PR = pullRequest
	state.PRCreated = time.Now()

	return state, nil
}

// commitChanges commits any uncommitted changes
func commitChanges(gitCtx *git.Context, state State) error {
	// Check for changes
	status, err := gitCtx.Status()
	if err != nil {
		return err
	}

	if status == "" {
		return nil // Nothing to commit
	}

	// Stage all changes
	if err := gitCtx.StageAll(); err != nil {
		return err
	}

	// Create commit message
	msg := buildCommitMessage(state)
	return gitCtx.Commit(msg)
}

// buildCommitMessage creates a commit message from state
func buildCommitMessage(state State) string {
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
func buildPROptions(state State) pr.Options {
	builder := pr.NewBuilder(getPRTitle(state))

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
func getPRTitle(state State) string {
	if state.Ticket != nil {
		return fmt.Sprintf("[%s] %s", state.TicketID, state.Ticket.Title)
	}
	if state.TicketID != "" {
		return fmt.Sprintf("[%s] Implementation", state.TicketID)
	}
	return fmt.Sprintf("devflow: %s", state.RunID)
}
