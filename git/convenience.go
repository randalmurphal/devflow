package git

import (
	"fmt"
	"time"
)

// CommitResult contains the result of a commit operation.
type CommitResult struct {
	SHA     string    // Full commit SHA
	Branch  string    // Branch name
	Message string    // Commit message
	Date    time.Time // Commit timestamp
}

// PushResult contains the result of a push operation.
type PushResult struct {
	Remote      string // Remote name (e.g., "origin")
	Branch      string // Branch that was pushed
	SHA         string // Commit SHA that was pushed
	SetUpstream bool   // Whether upstream tracking was set
	URL         string // Remote URL (for reference)
}

// CommitAndPushResult contains the result of a commit and push operation.
type CommitAndPushResult struct {
	Commit *CommitResult
	Push   *PushResult
}

// CommitAll stages all changes and commits with the given message.
// Returns ErrNothingToCommit if there are no changes to commit.
// This is a convenience method combining StageAll + Commit + HeadCommit + CurrentBranch.
func (g *Context) CommitAll(message string) (*CommitResult, error) {
	if err := g.StageAll(); err != nil {
		return nil, fmt.Errorf("stage all: %w", err)
	}

	if err := g.Commit(message); err != nil {
		return nil, err // Already wrapped appropriately
	}

	sha, err := g.HeadCommit()
	if err != nil {
		return nil, fmt.Errorf("get head: %w", err)
	}

	branch, err := g.CurrentBranch()
	if err != nil {
		return nil, fmt.Errorf("get branch: %w", err)
	}

	return &CommitResult{
		SHA:     sha,
		Branch:  branch,
		Message: message,
		Date:    time.Now(),
	}, nil
}

// PushCurrent pushes the current branch to origin.
// Automatically sets upstream tracking if the branch hasn't been pushed before.
// This is a convenience method that handles the common case of pushing work.
func (g *Context) PushCurrent() (*PushResult, error) {
	return g.PushCurrentTo("origin")
}

// PushCurrentTo pushes the current branch to the specified remote.
// Automatically sets upstream tracking if needed.
func (g *Context) PushCurrentTo(remote string) (*PushResult, error) {
	branch, err := g.CurrentBranch()
	if err != nil {
		return nil, fmt.Errorf("get current branch: %w", err)
	}

	setUpstream := !g.IsBranchPushed(branch)

	if err := g.Push(remote, branch, setUpstream); err != nil {
		return nil, err
	}

	sha, err := g.HeadCommit()
	if err != nil {
		return nil, fmt.Errorf("get head: %w", err)
	}

	url, _ := g.GetRemoteURL(remote) // Ignore error, URL is optional

	return &PushResult{
		Remote:      remote,
		Branch:      branch,
		SHA:         sha,
		SetUpstream: setUpstream,
		URL:         url,
	}, nil
}

// CommitAllAndPush stages all changes, commits, and pushes to origin.
// This is the most common workflow: save work and push it.
// If push fails, returns partial result with the commit info.
func (g *Context) CommitAllAndPush(message string) (*CommitAndPushResult, error) {
	commit, err := g.CommitAll(message)
	if err != nil {
		return nil, err
	}

	push, err := g.PushCurrent()
	if err != nil {
		// Return partial result so caller knows commit succeeded
		return &CommitAndPushResult{Commit: commit}, err
	}

	return &CommitAndPushResult{
		Commit: commit,
		Push:   push,
	}, nil
}

// CheckoutNew creates and checks out a new branch at the current HEAD.
// This is a convenience method combining CreateBranch + Checkout.
func (g *Context) CheckoutNew(name string) error {
	if err := g.CreateBranch(name); err != nil {
		return err
	}
	return g.Checkout(name)
}

// CheckoutNewAt creates and checks out a new branch at the specified ref.
func (g *Context) CheckoutNewAt(name, ref string) error {
	// First checkout the ref
	if err := g.Checkout(ref); err != nil {
		return fmt.Errorf("checkout %q: %w", ref, err)
	}
	// Create the branch
	if err := g.CreateBranch(name); err != nil {
		return fmt.Errorf("create branch %q: %w", name, err)
	}
	// Checkout the new branch
	if err := g.Checkout(name); err != nil {
		return fmt.Errorf("checkout new branch %q: %w", name, err)
	}
	return nil
}

// CommitAndPushTo stages all changes, commits, and pushes to the specified remote.
func (g *Context) CommitAndPushTo(message, remote string) (*CommitAndPushResult, error) {
	commit, err := g.CommitAll(message)
	if err != nil {
		return nil, err
	}

	push, err := g.PushCurrentTo(remote)
	if err != nil {
		// Return partial result so caller knows commit succeeded
		return &CommitAndPushResult{Commit: commit}, err
	}

	return &CommitAndPushResult{
		Commit: commit,
		Push:   push,
	}, nil
}
