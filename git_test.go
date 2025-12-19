package devflow

import (
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
