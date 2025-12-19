package devflow

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// GitContext manages git operations for a repository.
type GitContext struct {
	repoPath    string      // Path to the main repository
	worktreeDir string      // Directory where worktrees are created
	workDir     string      // Current working directory for commands (defaults to repoPath)
	github      PRProvider  // GitHub provider (if configured)
	gitlab      PRProvider  // GitLab provider (if configured)
}

// GitOption configures GitContext.
type GitOption func(*GitContext)

// NewGitContext creates a new git context for the repository.
// It validates that the path is a git repository and applies any options.
func NewGitContext(repoPath string, opts ...GitOption) (*GitContext, error) {
	// Resolve to absolute path
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}

	// Verify it's a git repository
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = absPath
	if err := cmd.Run(); err != nil {
		return nil, ErrNotGitRepo
	}

	g := &GitContext{
		repoPath:    absPath,
		worktreeDir: ".worktrees",
		workDir:     absPath,
	}

	for _, opt := range opts {
		opt(g)
	}

	return g, nil
}

// WithWorktreeDir sets the directory where worktrees are created.
// Default is ".worktrees" relative to the repository root.
func WithWorktreeDir(dir string) GitOption {
	return func(g *GitContext) {
		g.worktreeDir = dir
	}
}

// WithGitHub configures a GitHub PR provider.
func WithGitHub(provider PRProvider) GitOption {
	return func(g *GitContext) {
		g.github = provider
	}
}

// WithGitLab configures a GitLab PR provider.
func WithGitLab(provider PRProvider) GitOption {
	return func(g *GitContext) {
		g.gitlab = provider
	}
}

// RepoPath returns the path to the main repository.
func (g *GitContext) RepoPath() string {
	return g.repoPath
}

// WorkDir returns the current working directory for git commands.
// This is the repo path unless working in a worktree.
func (g *GitContext) WorkDir() string {
	return g.workDir
}

// WorktreeDir returns the path to the worktrees directory.
func (g *GitContext) WorktreeDir() string {
	return filepath.Join(g.repoPath, g.worktreeDir)
}

// InWorktree returns a new GitContext that operates in the specified worktree.
func (g *GitContext) InWorktree(worktreePath string) *GitContext {
	return &GitContext{
		repoPath:    g.repoPath,
		worktreeDir: g.worktreeDir,
		workDir:     worktreePath,
		github:      g.github,
		gitlab:      g.gitlab,
	}
}

// CurrentBranch returns the current branch name.
func (g *GitContext) CurrentBranch() (string, error) {
	branch, err := g.runGit("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", &GitError{Op: "get current branch", Err: err}
	}
	return branch, nil
}

// Checkout switches to the specified ref (branch, tag, or commit).
func (g *GitContext) Checkout(ref string) error {
	if _, err := g.runGit("checkout", ref); err != nil {
		return &GitError{Op: "checkout", Err: err}
	}
	return nil
}

// CreateBranch creates a new branch at HEAD.
func (g *GitContext) CreateBranch(name string) error {
	if _, err := g.runGit("branch", name); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return ErrBranchExists
		}
		return &GitError{Op: "create branch", Err: err}
	}
	return nil
}

// DeleteBranch deletes a branch. If force is true, uses -D instead of -d.
func (g *GitContext) DeleteBranch(name string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	if _, err := g.runGit("branch", flag, name); err != nil {
		return &GitError{Op: "delete branch", Err: err}
	}
	return nil
}

// BranchExists checks if a branch exists.
func (g *GitContext) BranchExists(name string) bool {
	_, err := g.runGit("rev-parse", "--verify", name)
	return err == nil
}

// Stage adds files to the staging area.
func (g *GitContext) Stage(files ...string) error {
	if len(files) == 0 {
		return nil
	}
	args := append([]string{"add", "--"}, files...)
	if _, err := g.runGit(args...); err != nil {
		return &GitError{Op: "stage files", Err: err}
	}
	return nil
}

// StageAll stages all changes (git add -A).
func (g *GitContext) StageAll() error {
	if _, err := g.runGit("add", "-A"); err != nil {
		return &GitError{Op: "stage all", Err: err}
	}
	return nil
}

// Commit creates a commit with the given message.
// Returns ErrNothingToCommit if there are no staged changes.
func (g *GitContext) Commit(message string) error {
	output, err := g.runGit("commit", "-m", message)
	if err != nil {
		if strings.Contains(output, "nothing to commit") ||
			strings.Contains(err.Error(), "nothing to commit") {
			return ErrNothingToCommit
		}
		return &GitError{Op: "commit", Output: output, Err: err}
	}
	return nil
}

// Push pushes the branch to the remote.
// If setUpstream is true, uses -u to set upstream tracking.
func (g *GitContext) Push(remote, branch string, setUpstream bool) error {
	args := []string{"push"}
	if setUpstream {
		args = append(args, "-u")
	}
	args = append(args, remote, branch)

	if _, err := g.runGit(args...); err != nil {
		return &GitError{Op: "push", Err: err}
	}
	return nil
}

// Pull pulls changes from the remote.
func (g *GitContext) Pull(remote, branch string) error {
	if _, err := g.runGit("pull", remote, branch); err != nil {
		return &GitError{Op: "pull", Err: err}
	}
	return nil
}

// Fetch fetches updates from the remote.
func (g *GitContext) Fetch(remote string) error {
	if _, err := g.runGit("fetch", remote); err != nil {
		return &GitError{Op: "fetch", Err: err}
	}
	return nil
}

// Diff returns the diff between two refs.
func (g *GitContext) Diff(base, head string) (string, error) {
	diff, err := g.runGit("diff", base+"..."+head)
	if err != nil {
		return "", &GitError{Op: "diff", Err: err}
	}
	return diff, nil
}

// DiffStaged returns the diff of staged changes.
func (g *GitContext) DiffStaged() (string, error) {
	diff, err := g.runGit("diff", "--cached")
	if err != nil {
		return "", &GitError{Op: "diff staged", Err: err}
	}
	return diff, nil
}

// Status returns the working tree status in short format.
func (g *GitContext) Status() (string, error) {
	status, err := g.runGit("status", "--short")
	if err != nil {
		return "", &GitError{Op: "status", Err: err}
	}
	return status, nil
}

// IsClean returns true if the working tree has no uncommitted changes.
func (g *GitContext) IsClean() (bool, error) {
	status, err := g.Status()
	if err != nil {
		return false, err
	}
	return status == "", nil
}

// HeadCommit returns the current HEAD commit SHA.
func (g *GitContext) HeadCommit() (string, error) {
	sha, err := g.runGit("rev-parse", "HEAD")
	if err != nil {
		return "", &GitError{Op: "get HEAD commit", Err: err}
	}
	return sha, nil
}

// IsBranchPushed checks if the branch exists on the remote.
func (g *GitContext) IsBranchPushed(branch string) bool {
	_, err := g.runGit("rev-parse", "--verify", "origin/"+branch)
	return err == nil
}

// GetRemoteURL returns the URL of the specified remote.
func (g *GitContext) GetRemoteURL(remote string) (string, error) {
	url, err := g.runGit("remote", "get-url", remote)
	if err != nil {
		return "", &GitError{Op: "get remote URL", Err: err}
	}
	return url, nil
}

// CreatePR creates a pull request using the configured provider.
func (g *GitContext) CreatePR(ctx context.Context, opts PROptions) (*PullRequest, error) {
	// Auto-detect head branch if not specified
	if opts.Head == "" {
		var err error
		opts.Head, err = g.CurrentBranch()
		if err != nil {
			return nil, fmt.Errorf("get current branch: %w", err)
		}
	}

	// Verify branch is pushed
	if !g.IsBranchPushed(opts.Head) {
		return nil, ErrBranchNotPushed
	}

	// Delegate to provider
	if g.github != nil {
		return g.github.CreatePR(ctx, opts)
	}
	if g.gitlab != nil {
		return g.gitlab.CreatePR(ctx, opts)
	}

	return nil, ErrNoPRProvider
}

// GetPR retrieves a pull request by ID.
func (g *GitContext) GetPR(ctx context.Context, id int) (*PullRequest, error) {
	if g.github != nil {
		return g.github.GetPR(ctx, id)
	}
	if g.gitlab != nil {
		return g.gitlab.GetPR(ctx, id)
	}
	return nil, ErrNoPRProvider
}

// runGit executes a git command and returns stdout.
func (g *GitContext) runGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = g.workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = strings.TrimSpace(stdout.String())
		}
		return errMsg, fmt.Errorf("%s", errMsg)
	}

	return strings.TrimSpace(stdout.String()), nil
}

// sanitizeBranchName converts a branch name to a safe directory name.
func sanitizeBranchName(branch string) string {
	// Replace / with -
	safe := strings.ReplaceAll(branch, "/", "-")
	// Lowercase
	safe = strings.ToLower(safe)
	// Remove invalid characters (keep only alphanumeric and hyphens)
	safe = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(safe, "")
	// Remove consecutive hyphens
	safe = regexp.MustCompile(`-+`).ReplaceAllString(safe, "-")
	// Trim hyphens from ends
	safe = strings.Trim(safe, "-")
	return safe
}

// WorktreeInfo represents an active git worktree.
type WorktreeInfo struct {
	Path   string // Filesystem path to the worktree
	Branch string // Branch checked out in the worktree
	Commit string // HEAD commit SHA
}

// CreateWorktree creates an isolated worktree for the branch.
// If the branch doesn't exist, it will be created.
// Returns the path to the worktree directory.
func (g *GitContext) CreateWorktree(branch string) (string, error) {
	// Sanitize branch name for filesystem
	safeName := sanitizeBranchName(branch)
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
			return "", &GitError{Op: "create worktree", Err: err}
		}
	}

	return worktreePath, nil
}

// CleanupWorktree removes a worktree and its registration.
// If force is true, removes even with uncommitted changes.
func (g *GitContext) CleanupWorktree(worktreePath string) error {
	// First try normal remove
	_, err := g.runGit("worktree", "remove", worktreePath)
	if err != nil {
		// Force remove if normal fails (uncommitted changes, etc.)
		_, err = g.runGit("worktree", "remove", "--force", worktreePath)
		if err != nil {
			return &GitError{Op: "cleanup worktree", Err: err}
		}
	}

	return nil
}

// ListWorktrees returns all active worktrees.
func (g *GitContext) ListWorktrees() ([]WorktreeInfo, error) {
	output, err := g.runGit("worktree", "list", "--porcelain")
	if err != nil {
		return nil, &GitError{Op: "list worktrees", Err: err}
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
func (g *GitContext) GetWorktree(branch string) (*WorktreeInfo, error) {
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
func (g *GitContext) GetWorktreeByPath(path string) (*WorktreeInfo, error) {
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
func (g *GitContext) PruneWorktrees() error {
	if _, err := g.runGit("worktree", "prune"); err != nil {
		return &GitError{Op: "prune worktrees", Err: err}
	}
	return nil
}
