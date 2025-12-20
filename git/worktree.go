package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WorktreeInfo represents an active git worktree.
type WorktreeInfo struct {
	Path   string // Filesystem path to the worktree
	Branch string // Branch checked out in the worktree
	Commit string // HEAD commit SHA
}

// CreateWorktree creates an isolated worktree for the branch.
// If the branch doesn't exist, it will be created.
// Returns the path to the worktree directory.
func (g *Context) CreateWorktree(branch string) (string, error) {
	// Sanitize branch name for filesystem
	safeName := SanitizeBranchName(branch)
	worktreePath := filepath.Join(g.repoPath, g.worktreeDir, safeName)

	// Check if worktree already exists
	if _, err := os.Stat(worktreePath); err == nil {
		return "", ErrWorktreeExists
	}

	// Ensure worktrees directory exists
	worktreesDir := filepath.Join(g.repoPath, g.worktreeDir)
	if err := os.MkdirAll(worktreesDir, 0755); err != nil {
		return "", fmt.Errorf("create worktrees dir: %w", err)
	}

	// Try to create worktree with new branch
	_, err := g.runGit("worktree", "add", "-b", branch, worktreePath, "HEAD")
	if err != nil {
		// Branch may already exist, try without -b
		_, err = g.runGit("worktree", "add", worktreePath, branch)
		if err != nil {
			// If branch doesn't exist either, provide clear error
			if strings.Contains(err.Error(), "not a valid reference") ||
				strings.Contains(err.Error(), "invalid reference") {
				return "", fmt.Errorf("branch %q does not exist and could not be created: %w", branch, err)
			}
			return "", &Error{Op: "create worktree", Err: err}
		}
	}

	return worktreePath, nil
}

// CleanupWorktree removes a worktree and its registration.
// If force is true, removes even with uncommitted changes.
func (g *Context) CleanupWorktree(worktreePath string) error {
	// First try normal remove
	_, err := g.runGit("worktree", "remove", worktreePath)
	if err != nil {
		// Force remove if normal fails (uncommitted changes, etc.)
		_, err = g.runGit("worktree", "remove", "--force", worktreePath)
		if err != nil {
			return &Error{Op: "cleanup worktree", Err: err}
		}
	}

	return nil
}

// ListWorktrees returns all active worktrees.
func (g *Context) ListWorktrees() ([]WorktreeInfo, error) {
	output, err := g.runGit("worktree", "list", "--porcelain")
	if err != nil {
		return nil, &Error{Op: "list worktrees", Err: err}
	}

	var worktrees []WorktreeInfo
	var current WorktreeInfo

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			if current.Path != "" {
				worktrees = append(worktrees, current)
				current = WorktreeInfo{}
			}
			continue
		}

		switch {
		case strings.HasPrefix(line, "worktree "):
			current.Path = strings.TrimPrefix(line, "worktree ")
		case strings.HasPrefix(line, "HEAD "):
			current.Commit = strings.TrimPrefix(line, "HEAD ")
		case strings.HasPrefix(line, "branch "):
			// Format: branch refs/heads/branch-name
			ref := strings.TrimPrefix(line, "branch ")
			current.Branch = strings.TrimPrefix(ref, "refs/heads/")
		case line == "detached":
			current.Branch = "(detached)"
		}
	}

	// Don't forget the last entry
	if current.Path != "" {
		worktrees = append(worktrees, current)
	}

	return worktrees, nil
}

// GetWorktree returns information about a specific worktree by branch name.
func (g *Context) GetWorktree(branch string) (*WorktreeInfo, error) {
	worktrees, err := g.ListWorktrees()
	if err != nil {
		return nil, err
	}

	for _, wt := range worktrees {
		if wt.Branch == branch {
			return &wt, nil
		}
	}

	return nil, ErrWorktreeNotFound
}

// GetWorktreeByPath returns information about a specific worktree by path.
func (g *Context) GetWorktreeByPath(path string) (*WorktreeInfo, error) {
	worktrees, err := g.ListWorktrees()
	if err != nil {
		return nil, err
	}

	// Resolve to absolute path for comparison
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}

	for _, wt := range worktrees {
		wtAbs, err := filepath.Abs(wt.Path)
		if err != nil {
			continue
		}
		if wtAbs == absPath {
			return &wt, nil
		}
	}

	return nil, ErrWorktreeNotFound
}

// PruneWorktrees removes stale worktree administrative files.
func (g *Context) PruneWorktrees() error {
	if _, err := g.runGit("worktree", "prune"); err != nil {
		return &Error{Op: "prune worktrees", Err: err}
	}
	return nil
}
