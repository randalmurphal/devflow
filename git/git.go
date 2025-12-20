package git

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// Context manages git operations for a repository.
type Context struct {
	repoPath    string        // Path to the main repository
	worktreeDir string        // Directory where worktrees are created
	workDir     string        // Current working directory for commands (defaults to repoPath)
	runner      CommandRunner // Command runner (defaults to ExecRunner)
}

// Option configures Context.
type Option func(*Context)

// NewContext creates a new git context for the repository.
// It validates that the path is a git repository and applies any options.
func NewContext(repoPath string, opts ...Option) (*Context, error) {
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

	g := &Context{
		repoPath:    absPath,
		worktreeDir: ".worktrees",
		workDir:     absPath,
		runner:      NewExecRunner(),
	}

	for _, opt := range opts {
		opt(g)
	}

	return g, nil
}

// WithWorktreeDir sets the directory where worktrees are created.
// Default is ".worktrees" relative to the repository root.
func WithWorktreeDir(dir string) Option {
	return func(g *Context) {
		g.worktreeDir = dir
	}
}

// WithRunner sets a custom command runner for git operations.
// This is primarily used for testing to inject mock command execution.
func WithRunner(runner CommandRunner) Option {
	return func(g *Context) {
		g.runner = runner
	}
}

// RepoPath returns the path to the main repository.
func (g *Context) RepoPath() string {
	return g.repoPath
}

// WorkDir returns the current working directory for git commands.
// This is the repo path unless working in a worktree.
func (g *Context) WorkDir() string {
	return g.workDir
}

// WorktreeDir returns the path to the worktrees directory.
func (g *Context) WorktreeDir() string {
	return filepath.Join(g.repoPath, g.worktreeDir)
}

// InWorktree returns a new Context that operates in the specified worktree.
func (g *Context) InWorktree(worktreePath string) *Context {
	return &Context{
		repoPath:    g.repoPath,
		worktreeDir: g.worktreeDir,
		workDir:     worktreePath,
		runner:      g.runner,
	}
}

// CurrentBranch returns the current branch name.
func (g *Context) CurrentBranch() (string, error) {
	branch, err := g.runGit("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", &Error{Op: "get current branch", Err: err}
	}
	return branch, nil
}

// Checkout switches to the specified ref (branch, tag, or commit).
func (g *Context) Checkout(ref string) error {
	if _, err := g.runGit("checkout", ref); err != nil {
		return &Error{Op: "checkout", Err: err}
	}
	return nil
}

// CreateBranch creates a new branch at HEAD.
func (g *Context) CreateBranch(name string) error {
	if _, err := g.runGit("branch", name); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return ErrBranchExists
		}
		return &Error{Op: "create branch", Err: err}
	}
	return nil
}

// DeleteBranch deletes a branch. If force is true, uses -D instead of -d.
func (g *Context) DeleteBranch(name string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	if _, err := g.runGit("branch", flag, name); err != nil {
		return &Error{Op: "delete branch", Err: err}
	}
	return nil
}

// BranchExists checks if a branch exists.
func (g *Context) BranchExists(name string) bool {
	_, err := g.runGit("rev-parse", "--verify", name)
	return err == nil
}

// Stage adds files to the staging area.
func (g *Context) Stage(files ...string) error {
	if len(files) == 0 {
		return nil
	}
	args := append([]string{"add", "--"}, files...)
	if _, err := g.runGit(args...); err != nil {
		return &Error{Op: "stage files", Err: err}
	}
	return nil
}

// StageAll stages all changes (git add -A).
func (g *Context) StageAll() error {
	if _, err := g.runGit("add", "-A"); err != nil {
		return &Error{Op: "stage all", Err: err}
	}
	return nil
}

// Commit creates a commit with the given message.
// Returns ErrNothingToCommit if there are no staged changes.
func (g *Context) Commit(message string) error {
	output, err := g.runGit("commit", "-m", message)
	if err != nil {
		if strings.Contains(output, "nothing to commit") ||
			strings.Contains(err.Error(), "nothing to commit") {
			return ErrNothingToCommit
		}
		return &Error{Op: "commit", Output: output, Err: err}
	}
	return nil
}

// Push pushes the branch to the remote.
// If setUpstream is true, uses -u to set upstream tracking.
func (g *Context) Push(remote, branch string, setUpstream bool) error {
	args := []string{"push"}
	if setUpstream {
		args = append(args, "-u")
	}
	args = append(args, remote, branch)

	if _, err := g.runGit(args...); err != nil {
		return &Error{Op: "push", Err: err}
	}
	return nil
}

// Pull pulls changes from the remote.
func (g *Context) Pull(remote, branch string) error {
	if _, err := g.runGit("pull", remote, branch); err != nil {
		return &Error{Op: "pull", Err: err}
	}
	return nil
}

// Fetch fetches updates from the remote.
func (g *Context) Fetch(remote string) error {
	if _, err := g.runGit("fetch", remote); err != nil {
		return &Error{Op: "fetch", Err: err}
	}
	return nil
}

// Diff returns the diff between two refs.
func (g *Context) Diff(base, head string) (string, error) {
	diff, err := g.runGit("diff", base+"..."+head)
	if err != nil {
		return "", &Error{Op: "diff", Err: err}
	}
	return diff, nil
}

// DiffStaged returns the diff of staged changes.
func (g *Context) DiffStaged() (string, error) {
	diff, err := g.runGit("diff", "--cached")
	if err != nil {
		return "", &Error{Op: "diff staged", Err: err}
	}
	return diff, nil
}

// Status returns the working tree status in short format.
func (g *Context) Status() (string, error) {
	status, err := g.runGit("status", "--short")
	if err != nil {
		return "", &Error{Op: "status", Err: err}
	}
	return status, nil
}

// IsClean returns true if the working tree has no uncommitted changes.
func (g *Context) IsClean() (bool, error) {
	status, err := g.Status()
	if err != nil {
		return false, err
	}
	return status == "", nil
}

// HeadCommit returns the current HEAD commit SHA.
func (g *Context) HeadCommit() (string, error) {
	sha, err := g.runGit("rev-parse", "HEAD")
	if err != nil {
		return "", &Error{Op: "get HEAD commit", Err: err}
	}
	return sha, nil
}

// IsBranchPushed checks if the branch exists on the remote.
func (g *Context) IsBranchPushed(branch string) bool {
	_, err := g.runGit("rev-parse", "--verify", "origin/"+branch)
	return err == nil
}

// GetRemoteURL returns the URL of the specified remote.
func (g *Context) GetRemoteURL(remote string) (string, error) {
	url, err := g.runGit("remote", "get-url", remote)
	if err != nil {
		return "", &Error{Op: "get remote URL", Err: err}
	}
	return url, nil
}

// runGit executes a git command and returns stdout.
func (g *Context) runGit(args ...string) (string, error) {
	return g.runner.Run(g.workDir, "git", args...)
}

// SanitizeBranchName converts a branch name to a safe directory name.
func SanitizeBranchName(branch string) string {
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
