package devflow

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// setupTestRepo creates a temporary git repository for testing.
func setupTestRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	// Initialize git repo
	if err := runCmd(dir, "git", "init"); err != nil {
		t.Fatalf("git init: %v", err)
	}

	// Configure git user (required for commits)
	if err := runCmd(dir, "git", "config", "user.email", "test@test.com"); err != nil {
		t.Fatalf("git config email: %v", err)
	}
	if err := runCmd(dir, "git", "config", "user.name", "Test User"); err != nil {
		t.Fatalf("git config name: %v", err)
	}

	// Create initial commit (required for worktrees)
	testFile := filepath.Join(dir, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test\n"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := runCmd(dir, "git", "add", "."); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := runCmd(dir, "git", "commit", "-m", "Initial commit"); err != nil {
		t.Fatalf("git commit: %v", err)
	}

	return dir
}

// runCmd executes a command in the specified directory.
func runCmd(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	return cmd.Run()
}

func TestNewGitContext(t *testing.T) {
	t.Run("valid repo", func(t *testing.T) {
		dir := setupTestRepo(t)

		git, err := NewGitContext(dir)
		if err != nil {
			t.Fatalf("NewGitContext: %v", err)
		}
		if git == nil {
			t.Fatal("expected non-nil GitContext")
		}
		if git.RepoPath() != dir {
			t.Errorf("RepoPath = %q, want %q", git.RepoPath(), dir)
		}
	})

	t.Run("invalid path", func(t *testing.T) {
		_, err := NewGitContext("/nonexistent/path")
		if err == nil {
			t.Error("expected error for non-existent path")
		}
	})

	t.Run("non-git directory", func(t *testing.T) {
		dir := t.TempDir()
		_, err := NewGitContext(dir)
		if err != ErrNotGitRepo {
			t.Errorf("err = %v, want ErrNotGitRepo", err)
		}
	})

	t.Run("with options", func(t *testing.T) {
		dir := setupTestRepo(t)

		git, err := NewGitContext(dir,
			WithWorktreeDir(".custom-worktrees"),
		)
		if err != nil {
			t.Fatalf("NewGitContext: %v", err)
		}
		if git.worktreeDir != ".custom-worktrees" {
			t.Errorf("worktreeDir = %q, want %q", git.worktreeDir, ".custom-worktrees")
		}
	})
}

func TestGitContext_CurrentBranch(t *testing.T) {
	dir := setupTestRepo(t)
	git, _ := NewGitContext(dir)

	branch, err := git.CurrentBranch()
	if err != nil {
		t.Fatalf("CurrentBranch: %v", err)
	}

	// Default branch is usually "master" or "main" depending on git config
	if branch != "master" && branch != "main" {
		t.Errorf("unexpected branch: %q", branch)
	}
}

func TestGitContext_CreateBranch(t *testing.T) {
	dir := setupTestRepo(t)
	git, _ := NewGitContext(dir)

	t.Run("create new branch", func(t *testing.T) {
		err := git.CreateBranch("test-branch")
		if err != nil {
			t.Fatalf("CreateBranch: %v", err)
		}

		if !git.BranchExists("test-branch") {
			t.Error("branch should exist after creation")
		}
	})

	t.Run("create existing branch", func(t *testing.T) {
		err := git.CreateBranch("test-branch")
		if err != ErrBranchExists {
			t.Errorf("err = %v, want ErrBranchExists", err)
		}
	})
}

func TestGitContext_Checkout(t *testing.T) {
	dir := setupTestRepo(t)
	git, _ := NewGitContext(dir)

	// Create a branch first
	if err := git.CreateBranch("feature"); err != nil {
		t.Fatalf("CreateBranch: %v", err)
	}

	// Checkout the branch
	if err := git.Checkout("feature"); err != nil {
		t.Fatalf("Checkout: %v", err)
	}

	branch, _ := git.CurrentBranch()
	if branch != "feature" {
		t.Errorf("CurrentBranch = %q, want %q", branch, "feature")
	}
}

func TestGitContext_StageAndCommit(t *testing.T) {
	dir := setupTestRepo(t)
	git, _ := NewGitContext(dir)

	// Create a new file
	testFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	// Stage the file
	if err := git.Stage("test.txt"); err != nil {
		t.Fatalf("Stage: %v", err)
	}

	// Commit
	if err := git.Commit("Add test file"); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	// Verify clean state
	clean, _ := git.IsClean()
	if !clean {
		t.Error("expected clean working directory after commit")
	}
}

func TestGitContext_CommitNothingStaged(t *testing.T) {
	dir := setupTestRepo(t)
	git, _ := NewGitContext(dir)

	// Try to commit without staging anything
	err := git.Commit("Empty commit")
	if err != ErrNothingToCommit {
		t.Errorf("err = %v, want ErrNothingToCommit", err)
	}
}

func TestGitContext_Status(t *testing.T) {
	dir := setupTestRepo(t)
	git, _ := NewGitContext(dir)

	t.Run("clean repo", func(t *testing.T) {
		status, err := git.Status()
		if err != nil {
			t.Fatalf("Status: %v", err)
		}
		if status != "" {
			t.Errorf("Status = %q, want empty string", status)
		}
	})

	t.Run("dirty repo", func(t *testing.T) {
		// Create an untracked file
		if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("new"), 0644); err != nil {
			t.Fatalf("write file: %v", err)
		}

		status, err := git.Status()
		if err != nil {
			t.Fatalf("Status: %v", err)
		}
		if status == "" {
			t.Error("expected non-empty status")
		}
	})
}

func TestGitContext_IsClean(t *testing.T) {
	dir := setupTestRepo(t)
	git, _ := NewGitContext(dir)

	clean, err := git.IsClean()
	if err != nil {
		t.Fatalf("IsClean: %v", err)
	}
	if !clean {
		t.Error("expected clean repo")
	}

	// Make it dirty
	if err := os.WriteFile(filepath.Join(dir, "dirty.txt"), []byte("dirty"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	clean, _ = git.IsClean()
	if clean {
		t.Error("expected dirty repo")
	}
}

func TestGitContext_Worktree(t *testing.T) {
	dir := setupTestRepo(t)
	git, _ := NewGitContext(dir)

	t.Run("create worktree", func(t *testing.T) {
		wtPath, err := git.CreateWorktree("feature/test-worktree")
		if err != nil {
			t.Fatalf("CreateWorktree: %v", err)
		}

		// Verify path
		expectedPath := filepath.Join(dir, ".worktrees", "feature-test-worktree")
		if wtPath != expectedPath {
			t.Errorf("worktree path = %q, want %q", wtPath, expectedPath)
		}

		// Verify directory exists
		if _, err := os.Stat(wtPath); err != nil {
			t.Errorf("worktree directory should exist: %v", err)
		}

		// Cleanup
		if err := git.CleanupWorktree(wtPath); err != nil {
			t.Errorf("CleanupWorktree: %v", err)
		}
	})

	t.Run("create duplicate worktree", func(t *testing.T) {
		wtPath, err := git.CreateWorktree("feature/dup-test")
		if err != nil {
			t.Fatalf("CreateWorktree: %v", err)
		}
		defer git.CleanupWorktree(wtPath)

		// Try to create again
		_, err = git.CreateWorktree("feature/dup-test")
		if err != ErrWorktreeExists {
			t.Errorf("err = %v, want ErrWorktreeExists", err)
		}
	})

	t.Run("list worktrees", func(t *testing.T) {
		wt1, _ := git.CreateWorktree("feature/list-test-1")
		wt2, _ := git.CreateWorktree("feature/list-test-2")
		defer git.CleanupWorktree(wt1)
		defer git.CleanupWorktree(wt2)

		worktrees, err := git.ListWorktrees()
		if err != nil {
			t.Fatalf("ListWorktrees: %v", err)
		}

		// Should have main repo + 2 worktrees
		if len(worktrees) < 3 {
			t.Errorf("expected at least 3 worktrees, got %d", len(worktrees))
		}
	})

	t.Run("get worktree by branch", func(t *testing.T) {
		wtPath, _ := git.CreateWorktree("feature/get-test")
		defer git.CleanupWorktree(wtPath)

		wt, err := git.GetWorktree("feature/get-test")
		if err != nil {
			t.Fatalf("GetWorktree: %v", err)
		}
		if wt.Path != wtPath {
			t.Errorf("worktree path = %q, want %q", wt.Path, wtPath)
		}
		if wt.Branch != "feature/get-test" {
			t.Errorf("worktree branch = %q, want %q", wt.Branch, "feature/get-test")
		}
	})

	t.Run("get nonexistent worktree", func(t *testing.T) {
		_, err := git.GetWorktree("nonexistent")
		if err != ErrWorktreeNotFound {
			t.Errorf("err = %v, want ErrWorktreeNotFound", err)
		}
	})
}

func TestGitContext_InWorktree(t *testing.T) {
	dir := setupTestRepo(t)
	git, _ := NewGitContext(dir)

	wtPath, err := git.CreateWorktree("feature/in-worktree-test")
	if err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}
	defer git.CleanupWorktree(wtPath)

	// Get context for worktree
	wtGit := git.InWorktree(wtPath)

	// Operations should work in worktree context
	branch, err := wtGit.CurrentBranch()
	if err != nil {
		t.Fatalf("CurrentBranch in worktree: %v", err)
	}
	if branch != "feature/in-worktree-test" {
		t.Errorf("branch = %q, want %q", branch, "feature/in-worktree-test")
	}

	// Make changes in worktree
	testFile := filepath.Join(wtPath, "worktree-file.txt")
	if err := os.WriteFile(testFile, []byte("worktree content"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if err := wtGit.StageAll(); err != nil {
		t.Fatalf("StageAll: %v", err)
	}

	if err := wtGit.Commit("Commit in worktree"); err != nil {
		t.Fatalf("Commit: %v", err)
	}
}

func TestSanitizeBranchName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"feature/test", "feature-test"},
		{"Feature/Test", "feature-test"},
		{"feature/test-123", "feature-test-123"},
		{"feature/test_with_underscores", "feature-testwithunderscores"},
		{"feature/test--double", "feature-test-double"},
		{"feature/test-", "feature-test"},
		{"-feature/test", "feature-test"},
		{"feature/TEST/nested", "feature-test-nested"},
		{"feature/special!@#chars", "feature-specialchars"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeBranchName(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeBranchName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// =============================================================================
// Mock-based Unit Tests
// These tests use MockRunner to test git operations in isolation.
// =============================================================================

func TestGitContext_WithMockRunner(t *testing.T) {
	// Create a real git repo so NewGitContext passes validation
	dir := setupTestRepo(t)
	mockRunner := NewMockRunner()

	git, err := NewGitContext(dir, WithGitRunner(mockRunner))
	if err != nil {
		t.Fatalf("NewGitContext: %v", err)
	}

	// Now the mock runner should be used for operations
	mockRunner.OnCommand("git", "status", "--short").Return("M modified.go", nil)

	status, err := git.Status()
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status != "M modified.go" {
		t.Errorf("Status = %q, want %q", status, "M modified.go")
	}
}

func TestGitContext_DeleteBranch_Mock(t *testing.T) {
	tests := []struct {
		name  string
		force bool
		flag  string
	}{
		{"soft delete", false, "-d"},
		{"force delete", true, "-D"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunner := NewMockRunner()
			mockRunner.OnCommand("git", "branch", tt.flag, "old-branch").Return("", nil)

			git := &GitContext{
				repoPath: "/test/repo",
				workDir:  "/test/repo",
				runner:   mockRunner,
			}

			err := git.DeleteBranch("old-branch", tt.force)
			if err != nil {
				t.Fatalf("DeleteBranch: %v", err)
			}

			if !mockRunner.WasCalled("git", "branch", tt.flag, "old-branch") {
				t.Errorf("expected git branch %s to be called", tt.flag)
			}
		})
	}
}

func TestGitContext_Push_Mock(t *testing.T) {
	tests := []struct {
		name        string
		setUpstream bool
		expectedCmd []string
	}{
		{"without upstream", false, []string{"push", "origin", "main"}},
		{"with upstream", true, []string{"push", "-u", "origin", "main"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunner := NewMockRunner()
			mockRunner.OnCommand("git", tt.expectedCmd...).Return("", nil)

			git := &GitContext{
				repoPath: "/test/repo",
				workDir:  "/test/repo",
				runner:   mockRunner,
			}

			err := git.Push("origin", "main", tt.setUpstream)
			if err != nil {
				t.Fatalf("Push: %v", err)
			}

			if !mockRunner.WasCalled("git", tt.expectedCmd...) {
				t.Errorf("expected git %v to be called", tt.expectedCmd)
			}
		})
	}
}

func TestGitContext_Pull_Mock(t *testing.T) {
	mockRunner := NewMockRunner()
	mockRunner.OnCommand("git", "pull", "origin", "main").Return("Already up to date.", nil)

	git := &GitContext{
		repoPath: "/test/repo",
		workDir:  "/test/repo",
		runner:   mockRunner,
	}

	err := git.Pull("origin", "main")
	if err != nil {
		t.Fatalf("Pull: %v", err)
	}
}

func TestGitContext_Fetch_Mock(t *testing.T) {
	mockRunner := NewMockRunner()
	mockRunner.OnCommand("git", "fetch", "upstream").Return("", nil)

	git := &GitContext{
		repoPath: "/test/repo",
		workDir:  "/test/repo",
		runner:   mockRunner,
	}

	err := git.Fetch("upstream")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
}

func TestGitContext_Diff_Mock(t *testing.T) {
	mockRunner := NewMockRunner()
	mockRunner.OnCommand("git", "diff", "main...feature").Return(`diff --git a/file.go b/file.go
--- a/file.go
+++ b/file.go
@@ -1,3 +1,4 @@
 package main
+// new comment`, nil)

	git := &GitContext{
		repoPath: "/test/repo",
		workDir:  "/test/repo",
		runner:   mockRunner,
	}

	diff, err := git.Diff("main", "feature")
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if diff == "" {
		t.Error("expected non-empty diff")
	}
}

func TestGitContext_DiffStaged_Mock(t *testing.T) {
	mockRunner := NewMockRunner()
	mockRunner.OnCommand("git", "diff", "--cached").Return("staged changes", nil)

	git := &GitContext{
		repoPath: "/test/repo",
		workDir:  "/test/repo",
		runner:   mockRunner,
	}

	diff, err := git.DiffStaged()
	if err != nil {
		t.Fatalf("DiffStaged: %v", err)
	}
	if diff != "staged changes" {
		t.Errorf("DiffStaged = %q, want %q", diff, "staged changes")
	}
}

func TestGitContext_HeadCommit_Mock(t *testing.T) {
	mockRunner := NewMockRunner()
	mockRunner.OnCommand("git", "rev-parse", "HEAD").Return("abc123def456789", nil)

	git := &GitContext{
		repoPath: "/test/repo",
		workDir:  "/test/repo",
		runner:   mockRunner,
	}

	sha, err := git.HeadCommit()
	if err != nil {
		t.Fatalf("HeadCommit: %v", err)
	}
	if sha != "abc123def456789" {
		t.Errorf("HeadCommit = %q, want %q", sha, "abc123def456789")
	}
}

func TestGitContext_IsBranchPushed_Mock(t *testing.T) {
	tests := []struct {
		name   string
		branch string
		err    error
		pushed bool
	}{
		{"pushed branch", "main", nil, true},
		{"local only branch", "local-feature", errors.New("not found"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunner := NewMockRunner()
			mockRunner.OnCommand("git", "rev-parse", "--verify", "origin/"+tt.branch).Return("abc123", tt.err)

			git := &GitContext{
				repoPath: "/test/repo",
				workDir:  "/test/repo",
				runner:   mockRunner,
			}

			pushed := git.IsBranchPushed(tt.branch)
			if pushed != tt.pushed {
				t.Errorf("IsBranchPushed(%q) = %v, want %v", tt.branch, pushed, tt.pushed)
			}
		})
	}
}

func TestGitContext_GetRemoteURL_Mock(t *testing.T) {
	mockRunner := NewMockRunner()
	mockRunner.OnCommand("git", "remote", "get-url", "origin").Return("https://github.com/owner/repo.git", nil)

	git := &GitContext{
		repoPath: "/test/repo",
		workDir:  "/test/repo",
		runner:   mockRunner,
	}

	url, err := git.GetRemoteURL("origin")
	if err != nil {
		t.Fatalf("GetRemoteURL: %v", err)
	}
	if url != "https://github.com/owner/repo.git" {
		t.Errorf("GetRemoteURL = %q", url)
	}
}

func TestGitContext_ListWorktrees_Mock(t *testing.T) {
	mockRunner := NewMockRunner()
	mockRunner.OnCommand("git", "worktree", "list", "--porcelain").Return(`worktree /repo
HEAD abc123456789
branch refs/heads/main

worktree /repo/.worktrees/feature
HEAD def456789abc
branch refs/heads/feature/test

worktree /repo/.worktrees/detached
HEAD 789abcdef123
detached
`, nil)

	git := &GitContext{
		repoPath: "/repo",
		workDir:  "/repo",
		runner:   mockRunner,
	}

	worktrees, err := git.ListWorktrees()
	if err != nil {
		t.Fatalf("ListWorktrees: %v", err)
	}

	if len(worktrees) != 3 {
		t.Fatalf("got %d worktrees, want 3", len(worktrees))
	}

	// Check main worktree
	if worktrees[0].Path != "/repo" || worktrees[0].Branch != "main" {
		t.Errorf("worktree[0] = %+v", worktrees[0])
	}

	// Check feature worktree
	if worktrees[1].Path != "/repo/.worktrees/feature" || worktrees[1].Branch != "feature/test" {
		t.Errorf("worktree[1] = %+v", worktrees[1])
	}

	// Check detached worktree
	if worktrees[2].Branch != "(detached)" {
		t.Errorf("worktree[2] = %+v", worktrees[2])
	}
}

func TestGitContext_PruneWorktrees_Mock(t *testing.T) {
	mockRunner := NewMockRunner()
	mockRunner.OnCommand("git", "worktree", "prune").Return("", nil)

	git := &GitContext{
		repoPath: "/test/repo",
		workDir:  "/test/repo",
		runner:   mockRunner,
	}

	err := git.PruneWorktrees()
	if err != nil {
		t.Fatalf("PruneWorktrees: %v", err)
	}
}

func TestGitContext_BranchExists_Mock(t *testing.T) {
	tests := []struct {
		name   string
		branch string
		err    error
		exists bool
	}{
		{"exists", "main", nil, true},
		{"not exists", "nonexistent", errors.New("not a valid ref"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunner := NewMockRunner()
			mockRunner.OnCommand("git", "rev-parse", "--verify", tt.branch).Return("abc123", tt.err)

			git := &GitContext{
				repoPath: "/test/repo",
				workDir:  "/test/repo",
				runner:   mockRunner,
			}

			exists := git.BranchExists(tt.branch)
			if exists != tt.exists {
				t.Errorf("BranchExists(%q) = %v, want %v", tt.branch, exists, tt.exists)
			}
		})
	}
}

// =============================================================================
// PR Provider Tests (Mock)
// =============================================================================

func TestGitContext_CreatePR_NoPRProvider(t *testing.T) {
	mockRunner := NewMockRunner()
	mockRunner.OnCommand("git", "rev-parse", "--verify", "origin/feature").Return("abc123", nil)

	git := &GitContext{
		repoPath: "/test/repo",
		workDir:  "/test/repo",
		runner:   mockRunner,
	}

	_, err := git.CreatePR(context.Background(), PROptions{
		Title: "Test PR",
		Head:  "feature",
	})
	if err != ErrNoPRProvider {
		t.Errorf("CreatePR: got %v, want ErrNoPRProvider", err)
	}
}

func TestGitContext_CreatePR_BranchNotPushed(t *testing.T) {
	mockRunner := NewMockRunner()
	mockRunner.OnCommand("git", "rev-parse", "--verify", "origin/local-only").Return("", errors.New("not found"))

	git := &GitContext{
		repoPath: "/test/repo",
		workDir:  "/test/repo",
		runner:   mockRunner,
	}

	_, err := git.CreatePR(context.Background(), PROptions{
		Title: "Test PR",
		Head:  "local-only",
	})
	if err != ErrBranchNotPushed {
		t.Errorf("CreatePR: got %v, want ErrBranchNotPushed", err)
	}
}

func TestGitContext_GetPR_NoPRProvider(t *testing.T) {
	git := &GitContext{
		repoPath: "/test/repo",
		workDir:  "/test/repo",
		runner:   NewMockRunner(),
	}

	_, err := git.GetPR(context.Background(), 123)
	if err != ErrNoPRProvider {
		t.Errorf("GetPR: got %v, want ErrNoPRProvider", err)
	}
}

// =============================================================================
// MockRunner Tests
// =============================================================================

func TestMockRunner_OnCommand(t *testing.T) {
	m := NewMockRunner()
	m.OnCommand("echo", "hello").Return("hello world", nil)

	stdout, err := m.Run("/tmp", "echo", "hello")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if stdout != "hello world" {
		t.Errorf("Run = %q, want %q", stdout, "hello world")
	}
}

func TestMockRunner_OnAnyCommand(t *testing.T) {
	m := NewMockRunner()
	m.OnAnyCommand().Return("default", nil)

	stdout, err := m.Run("/tmp", "any", "command")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if stdout != "default" {
		t.Errorf("Run = %q, want %q", stdout, "default")
	}
}

func TestMockRunner_CallCount(t *testing.T) {
	m := NewMockRunner()
	m.OnAnyCommand().Return("", nil)

	m.Run("/tmp", "git", "status")
	m.Run("/tmp", "git", "diff")
	m.Run("/tmp", "echo", "test")

	if m.CallCount("git") != 2 {
		t.Errorf("CallCount(git) = %d, want 2", m.CallCount("git"))
	}
	if m.CallCount("echo") != 1 {
		t.Errorf("CallCount(echo) = %d, want 1", m.CallCount("echo"))
	}
}

func TestMockRunner_WasCalled(t *testing.T) {
	m := NewMockRunner()
	m.OnAnyCommand().Return("", nil)

	m.Run("/tmp", "git", "status", "--short")

	if !m.WasCalled("git") {
		t.Error("WasCalled(git) should be true")
	}
	if !m.WasCalled("git", "status", "--short") {
		t.Error("WasCalled(git, status, --short) should be true")
	}
	if m.WasCalled("git", "status") { // Missing --short
		t.Error("WasCalled(git, status) without --short should be false")
	}
}

func TestMockRunner_DefaultResponse(t *testing.T) {
	m := NewMockRunner()
	m.DefaultResponse = MockResponse{Stdout: "default output", Err: nil}

	stdout, err := m.Run("/tmp", "unregistered", "command")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if stdout != "default output" {
		t.Errorf("Run = %q, want %q", stdout, "default output")
	}
}

func TestMockRunner_Error(t *testing.T) {
	m := NewMockRunner()
	expectedErr := errors.New("command failed")
	m.OnCommand("git", "push").Return("", expectedErr)

	_, err := m.Run("/tmp", "git", "push")
	if err != expectedErr {
		t.Errorf("Run error = %v, want %v", err, expectedErr)
	}
}

// =============================================================================
// Additional GitContext Coverage Tests
// =============================================================================

func TestGitContext_WorkDir(t *testing.T) {
	dir := setupTestRepo(t)
	git, err := NewGitContext(dir)
	if err != nil {
		t.Fatalf("NewGitContext: %v", err)
	}

	if git.WorkDir() != dir {
		t.Errorf("WorkDir() = %q, want %q", git.WorkDir(), dir)
	}
}

func TestGitContext_WorktreeDir(t *testing.T) {
	dir := setupTestRepo(t)
	git, err := NewGitContext(dir)
	if err != nil {
		t.Fatalf("NewGitContext: %v", err)
	}

	expected := filepath.Join(dir, ".worktrees")
	if git.WorktreeDir() != expected {
		t.Errorf("WorktreeDir() = %q, want %q", git.WorktreeDir(), expected)
	}
}

func TestGitContext_WorktreeDir_Custom(t *testing.T) {
	dir := setupTestRepo(t)
	git, err := NewGitContext(dir, WithWorktreeDir("my-worktrees"))
	if err != nil {
		t.Fatalf("NewGitContext: %v", err)
	}

	expected := filepath.Join(dir, "my-worktrees")
	if git.WorktreeDir() != expected {
		t.Errorf("WorktreeDir() = %q, want %q", git.WorktreeDir(), expected)
	}
}

func TestGitContext_WithGitHub(t *testing.T) {
	dir := setupTestRepo(t)

	// Create a mock GitHub provider
	mockProvider := &mockPRProvider{name: "github"}

	git, err := NewGitContext(dir, WithGitHub(mockProvider))
	if err != nil {
		t.Fatalf("NewGitContext: %v", err)
	}

	// The github field should be set
	if git.github == nil {
		t.Error("github provider should be set")
	}
}

func TestGitContext_WithGitLab(t *testing.T) {
	dir := setupTestRepo(t)

	// Create a mock GitLab provider
	mockProvider := &mockPRProvider{name: "gitlab"}

	git, err := NewGitContext(dir, WithGitLab(mockProvider))
	if err != nil {
		t.Fatalf("NewGitContext: %v", err)
	}

	// The gitlab field should be set
	if git.gitlab == nil {
		t.Error("gitlab provider should be set")
	}
}

func TestGitContext_GetWorktreeByPath(t *testing.T) {
	// This test requires an actual worktree to be created
	dir := setupTestRepo(t)
	git, err := NewGitContext(dir)
	if err != nil {
		t.Fatalf("NewGitContext: %v", err)
	}

	// Create a worktree
	worktreePath, err := git.CreateWorktree("test-wt-by-path")
	if err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}
	defer git.CleanupWorktree(worktreePath)

	// Get worktree by path
	wt, err := git.GetWorktreeByPath(worktreePath)
	if err != nil {
		t.Fatalf("GetWorktreeByPath: %v", err)
	}

	if wt.Branch != "test-wt-by-path" {
		t.Errorf("Branch = %q, want %q", wt.Branch, "test-wt-by-path")
	}
}

func TestGitContext_GetWorktreeByPath_NotFound(t *testing.T) {
	dir := setupTestRepo(t)
	git, err := NewGitContext(dir)
	if err != nil {
		t.Fatalf("NewGitContext: %v", err)
	}

	_, err = git.GetWorktreeByPath("/nonexistent/path")
	if err != ErrWorktreeNotFound {
		t.Errorf("GetWorktreeByPath = %v, want ErrWorktreeNotFound", err)
	}
}

// mockPRProvider is a simple mock for testing WithGitHub and WithGitLab
type mockPRProvider struct {
	name string
}

func (m *mockPRProvider) CreatePR(ctx context.Context, opts PROptions) (*PullRequest, error) {
	return nil, nil
}

func (m *mockPRProvider) GetPR(ctx context.Context, id int) (*PullRequest, error) {
	return nil, nil
}

func (m *mockPRProvider) UpdatePR(ctx context.Context, id int, opts PRUpdateOptions) (*PullRequest, error) {
	return nil, nil
}

func (m *mockPRProvider) MergePR(ctx context.Context, id int, opts MergeOptions) error {
	return nil
}

func (m *mockPRProvider) ClosePR(ctx context.Context, id int) error {
	return nil
}

func (m *mockPRProvider) AddComment(ctx context.Context, id int, body string) error {
	return nil
}

func (m *mockPRProvider) RequestReview(ctx context.Context, id int, reviewers []string) error {
	return nil
}

func (m *mockPRProvider) ListPRs(ctx context.Context, filter PRFilter) ([]*PullRequest, error) {
	return nil, nil
}
