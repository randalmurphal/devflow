package devflow

import (
	"fmt"

	"github.com/randalmurphal/flowgraph/pkg/flowgraph"
)

// CreateWorktreeNode creates an isolated git worktree for the task.
//
// Prerequisites: state.TicketID or state.Branch must be set
// Updates: state.Worktree, state.Branch
func CreateWorktreeNode(ctx flowgraph.Context, state DevState) (DevState, error) {
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

// CleanupNode cleans up the worktree.
//
// Prerequisites: state.Worktree must be set
// Updates: clears state.Worktree
func CleanupNode(ctx flowgraph.Context, state DevState) (DevState, error) {
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
