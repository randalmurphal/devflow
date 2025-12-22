// Package parallel provides multi-worktree orchestration for parallel branch execution.
//
// This package extends devflow's git worktree primitives to support fork/join workflows
// where multiple branches execute in parallel, each in their own isolated worktree.
package parallel

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/randalmurphal/devflow/git"
)

// Manager orchestrates multiple worktrees for parallel branch execution.
//
// It maintains a mapping from branch IDs (logical identifiers from the workflow)
// to git worktree paths. Each parallel branch gets its own isolated worktree,
// allowing concurrent git operations without conflicts.
//
// The Manager uses the "base branch" (whatever branch was checked out at fork time)
// as the starting point for all branch worktrees. Merges go back to this base branch,
// NOT to a hardcoded "main" branch.
//
// Thread-safe for concurrent worktree operations.
type Manager struct {
	baseDir    string            // Directory for worktrees
	baseRepo   string            // Path to base repository
	baseBranch string            // Branch at fork time (working branch)
	worktrees  map[string]string // branchID -> worktree path
	gitCtx     *git.Context      // Git context for base repo
	mu         sync.RWMutex
}

// NewManager creates a ParallelWorktreeManager for the given repository.
//
// baseDir is where worktrees will be created (e.g., /tmp/worktrees/run-123)
// repoPath is the path to the main repository
// baseBranch is the current branch (working branch) - merges return here
func NewManager(baseDir, repoPath, baseBranch string) (*Manager, error) {
	// Validate base repository exists
	if _, err := os.Stat(repoPath); err != nil {
		return nil, fmt.Errorf("base repository not found: %w", err)
	}

	// Create git context for the base repo
	gitCtx, gitErr := git.NewContext(repoPath)
	if gitErr != nil {
		return nil, fmt.Errorf("create git context: %w", gitErr)
	}

	// Create worktrees directory
	if mkdirErr := os.MkdirAll(baseDir, 0755); mkdirErr != nil {
		return nil, fmt.Errorf("create worktrees directory: %w", mkdirErr)
	}

	return &Manager{
		baseDir:    baseDir,
		baseRepo:   repoPath,
		baseBranch: baseBranch,
		worktrees:  make(map[string]string),
		gitCtx:     gitCtx,
	}, nil
}

// CreateBranchWorktree creates an isolated worktree for a parallel branch.
//
// branchID is the logical identifier from the workflow (e.g., "workerA")
// gitBranch is the git branch name to create/checkout (if empty, uses branchID)
//
// Returns the path to the worktree directory.
func (m *Manager) CreateBranchWorktree(branchID, gitBranch string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already exists
	if path, exists := m.worktrees[branchID]; exists {
		return path, nil // Already created
	}

	// Use branchID as git branch if not specified
	if gitBranch == "" {
		gitBranch = branchID
	}

	// Create unique worktree path
	worktreePath := filepath.Join(m.baseDir, git.SanitizeBranchName(branchID))

	// Create the worktree
	// First try creating a new branch from the base branch
	_, err := m.gitCtx.RunGit("worktree", "add", "-b", gitBranch, worktreePath, m.baseBranch)
	if err != nil {
		// Branch may already exist, try without -b
		_, err = m.gitCtx.RunGit("worktree", "add", worktreePath, gitBranch)
		if err != nil {
			return "", fmt.Errorf("create worktree for branch %s: %w", branchID, err)
		}
	}

	m.worktrees[branchID] = worktreePath
	return worktreePath, nil
}

// GetWorktreePath returns the worktree path for a branch ID.
// Returns empty string if not found.
func (m *Manager) GetWorktreePath(branchID string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.worktrees[branchID]
}

// ListBranchWorktrees returns all active branch worktrees.
func (m *Manager) ListBranchWorktrees() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]string, len(m.worktrees))
	for k, v := range m.worktrees {
		result[k] = v
	}
	return result
}

// CleanupBranch removes a single branch worktree.
func (m *Manager) CleanupBranch(branchID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	path, exists := m.worktrees[branchID]
	if !exists {
		return nil // Nothing to clean up
	}

	// Remove the worktree
	if err := m.gitCtx.CleanupWorktree(path); err != nil {
		return fmt.Errorf("cleanup worktree for branch %s: %w", branchID, err)
	}

	delete(m.worktrees, branchID)
	return nil
}

// CleanupAll removes all branch worktrees.
func (m *Manager) CleanupAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var lastErr error
	for branchID, path := range m.worktrees {
		if err := m.gitCtx.CleanupWorktree(path); err != nil {
			lastErr = fmt.Errorf("cleanup worktree for branch %s: %w", branchID, err)
			// Continue cleaning up others
		}
		delete(m.worktrees, branchID)
	}

	return lastErr
}

// BaseBranch returns the base branch name (where merges go).
func (m *Manager) BaseBranch() string {
	return m.baseBranch
}

// BaseRepo returns the path to the base repository.
func (m *Manager) BaseRepo() string {
	return m.baseRepo
}

// BaseDir returns the directory where worktrees are created.
func (m *Manager) BaseDir() string {
	return m.baseDir
}

// GitContextForBranch creates a git.Context for a specific branch's worktree.
// Returns nil if the branch doesn't have a worktree.
func (m *Manager) GitContextForBranch(branchID string) (*git.Context, error) {
	m.mu.RLock()
	path := m.worktrees[branchID]
	m.mu.RUnlock()

	if path == "" {
		return nil, fmt.Errorf("no worktree for branch %s", branchID)
	}

	return git.NewContext(path)
}

// BranchCount returns the number of active branch worktrees.
func (m *Manager) BranchCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.worktrees)
}

// MergeConfig configures how branches are merged back to base.
type MergeConfig struct {
	// CommitMessage is the message for the merge commit (if not fast-forward)
	CommitMessage string

	// NoFastForward forces a merge commit even if fast-forward is possible
	NoFastForward bool

	// SquashMerge squashes all branch commits into one
	SquashMerge bool
}

// MergeResult contains the outcome of merging a branch.
type MergeResult struct {
	BranchID  string
	Success   bool
	Conflicts []ConflictFile
	CommitSHA string
	Error     error
}

// ConflictFile describes a merge conflict.
type ConflictFile struct {
	Path          string // File path relative to repo root
	OursContent   string // Content from base branch
	TheirsContent string // Content from feature branch
	Markers       string // Full file content with conflict markers
}

// MergeBranches merges all branch worktrees back to the base working branch.
// Returns results for each branch, including any conflicts.
func (m *Manager) MergeBranches(ctx context.Context, cfg MergeConfig) ([]MergeResult, error) {
	m.mu.RLock()
	branches := make([]string, 0, len(m.worktrees))
	for branchID := range m.worktrees {
		branches = append(branches, branchID)
	}
	m.mu.RUnlock()

	results := make([]MergeResult, 0, len(branches))
	for _, branchID := range branches {
		result := m.MergeSingleBranch(ctx, branchID, cfg)
		results = append(results, *result)

		// If there are conflicts, stop merging more branches
		if len(result.Conflicts) > 0 {
			break
		}
	}

	return results, nil
}

// MergeSingleBranch merges one branch worktree back to the base working branch.
func (m *Manager) MergeSingleBranch(ctx context.Context, branchID string, cfg MergeConfig) *MergeResult {
	m.mu.RLock()
	worktreePath := m.worktrees[branchID]
	m.mu.RUnlock()

	if worktreePath == "" {
		return &MergeResult{
			BranchID: branchID,
			Success:  false,
			Error:    fmt.Errorf("no worktree for branch %s", branchID),
		}
	}

	// Get the git branch name from the worktree
	branchCtx, ctxErr := git.NewContext(worktreePath)
	if ctxErr != nil {
		return &MergeResult{
			BranchID: branchID,
			Success:  false,
			Error:    fmt.Errorf("create git context for worktree: %w", ctxErr),
		}
	}

	gitBranch, branchErr := branchCtx.CurrentBranch()
	if branchErr != nil {
		return &MergeResult{
			BranchID: branchID,
			Success:  false,
			Error:    fmt.Errorf("get current branch: %w", branchErr),
		}
	}

	// Build merge command
	args := []string{"merge"}
	if cfg.NoFastForward {
		args = append(args, "--no-ff")
	}
	if cfg.SquashMerge {
		args = append(args, "--squash")
	}
	if cfg.CommitMessage != "" {
		args = append(args, "-m", cfg.CommitMessage)
	}
	args = append(args, gitBranch)

	// Run merge in base repo
	_, mergeErr := m.gitCtx.RunGit(args...)
	if mergeErr != nil {
		// Check for conflicts
		conflicts, detectErr := m.detectConflicts()
		if detectErr == nil && len(conflicts) > 0 {
			return &MergeResult{
				BranchID:  branchID,
				Success:   false,
				Conflicts: conflicts,
			}
		}

		return &MergeResult{
			BranchID: branchID,
			Success:  false,
			Error:    fmt.Errorf("merge branch %s: %w", gitBranch, mergeErr),
		}
	}

	// Get the resulting commit SHA
	sha, shaErr := m.gitCtx.HeadCommit()
	if shaErr != nil {
		sha = ""
	}

	return &MergeResult{
		BranchID:  branchID,
		Success:   true,
		CommitSHA: sha,
	}
}

// detectConflicts checks for merge conflicts in the base repository.
func (m *Manager) detectConflicts() ([]ConflictFile, error) {
	// Get list of unmerged files
	output, err := m.gitCtx.RunGit("diff", "--name-only", "--diff-filter=U")
	if err != nil {
		return nil, fmt.Errorf("check for conflicts: %w", err)
	}

	if output == "" {
		return nil, nil
	}

	// Parse conflict files
	var conflicts []ConflictFile
	for _, path := range splitLines(output) {
		if path == "" {
			continue
		}

		// Read the conflicted file content
		fullPath := filepath.Join(m.baseRepo, path)
		content, readErr := os.ReadFile(fullPath)
		if readErr != nil {
			continue
		}

		conflicts = append(conflicts, ConflictFile{
			Path:    path,
			Markers: string(content),
		})
	}

	return conflicts, nil
}

// AbortMerge aborts an in-progress merge.
func (m *Manager) AbortMerge() error {
	_, err := m.gitCtx.RunGit("merge", "--abort")
	return err
}

// ResolveConflict writes resolved content for a conflicted file.
func (m *Manager) ResolveConflict(path, resolvedContent string) error {
	fullPath := filepath.Join(m.baseRepo, path)

	// Write resolved content
	if err := os.WriteFile(fullPath, []byte(resolvedContent), 0644); err != nil {
		return fmt.Errorf("write resolved content: %w", err)
	}

	// Stage the resolved file
	_, stageErr := m.gitCtx.RunGit("add", path)
	if stageErr != nil {
		return fmt.Errorf("stage resolved file: %w", stageErr)
	}

	return nil
}

// ContinueMerge completes a merge after conflicts have been resolved.
func (m *Manager) ContinueMerge(message string) error {
	args := []string{"commit"}
	if message != "" {
		args = append(args, "-m", message)
	}

	_, err := m.gitCtx.RunGit(args...)
	if err != nil {
		return fmt.Errorf("complete merge: %w", err)
	}

	return nil
}

// splitLines splits a string into non-empty lines.
func splitLines(s string) []string {
	var lines []string
	for _, line := range filepath.SplitList(s) {
		if line != "" {
			lines = append(lines, line)
		}
	}
	// Fallback to simple split for git output
	if len(lines) == 0 {
		for _, line := range split(s, '\n') {
			if line != "" {
				lines = append(lines, line)
			}
		}
	}
	return lines
}

func split(s string, sep byte) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		result = append(result, s[start:])
	}
	return result
}
