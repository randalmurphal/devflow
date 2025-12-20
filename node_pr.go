package devflow

import (
	"fmt"
	"time"

	"github.com/randalmurphal/flowgraph/pkg/flowgraph"
)

// CreatePRNode creates a pull request.
//
// Prerequisites: state.Branch must be set and pushed
// Updates: state.PR, state.PRCreated
func CreatePRNode(ctx flowgraph.Context, state DevState) (DevState, error) {
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
func commitChanges(ctx flowgraph.Context, git *GitContext, state DevState) error {
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
