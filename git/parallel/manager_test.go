package parallel

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestNewManager(t *testing.T) {
	// Create a temp git repo
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")

	// Initialize git repo
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatalf("Failed to create repo dir: %v", err)
	}

	// Run git init with explicit main branch
	if err := runCommand(repoDir, "git", "init", "-b", "main"); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Create initial commit
	if err := runCommand(repoDir, "git", "config", "user.email", "test@test.com"); err != nil {
		t.Fatalf("Failed to set git config: %v", err)
	}
	if err := runCommand(repoDir, "git", "config", "user.name", "Test"); err != nil {
		t.Fatalf("Failed to set git config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "test.txt"), []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := runCommand(repoDir, "git", "add", "."); err != nil {
		t.Fatalf("Failed to stage: %v", err)
	}
	if err := runCommand(repoDir, "git", "commit", "-m", "initial"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Test creating manager
	worktreeDir := filepath.Join(tmpDir, "worktrees")
	mgr, err := NewManager(worktreeDir, repoDir, "main")
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	if mgr.BaseBranch() != "main" {
		t.Errorf("BaseBranch() = %q, want %q", mgr.BaseBranch(), "main")
	}

	if mgr.BaseRepo() != repoDir {
		t.Errorf("BaseRepo() = %q, want %q", mgr.BaseRepo(), repoDir)
	}

	if mgr.BranchCount() != 0 {
		t.Errorf("BranchCount() = %d, want 0", mgr.BranchCount())
	}
}

func TestCreateBranchWorktree(t *testing.T) {
	// Create a temp git repo
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	worktreeDir := filepath.Join(tmpDir, "worktrees")

	// Initialize git repo with initial commit
	setupTestRepo(t, repoDir)

	mgr, err := NewManager(worktreeDir, repoDir, "main")
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	// Create a branch worktree
	path, err := mgr.CreateBranchWorktree("feature-a", "")
	if err != nil {
		t.Fatalf("CreateBranchWorktree() error: %v", err)
	}

	// Verify path exists
	if _, statErr := os.Stat(path); statErr != nil {
		t.Errorf("Worktree path does not exist: %v", statErr)
	}

	// Verify it's tracked
	if mgr.GetWorktreePath("feature-a") != path {
		t.Error("Worktree not tracked by manager")
	}

	// Verify branch count
	if mgr.BranchCount() != 1 {
		t.Errorf("BranchCount() = %d, want 1", mgr.BranchCount())
	}

	// Creating same branch again should return existing path
	path2, err := mgr.CreateBranchWorktree("feature-a", "")
	if err != nil {
		t.Fatalf("CreateBranchWorktree() second call error: %v", err)
	}
	if path2 != path {
		t.Error("Second call should return same path")
	}

	// Cleanup
	if err := mgr.CleanupAll(); err != nil {
		t.Errorf("CleanupAll() error: %v", err)
	}
}

func TestListBranchWorktrees(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	worktreeDir := filepath.Join(tmpDir, "worktrees")

	setupTestRepo(t, repoDir)

	mgr, err := NewManager(worktreeDir, repoDir, "main")
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	// Create multiple worktrees
	_, _ = mgr.CreateBranchWorktree("branch-a", "")
	_, _ = mgr.CreateBranchWorktree("branch-b", "")

	worktrees := mgr.ListBranchWorktrees()
	if len(worktrees) != 2 {
		t.Errorf("Expected 2 worktrees, got %d", len(worktrees))
	}

	if _, ok := worktrees["branch-a"]; !ok {
		t.Error("branch-a not in list")
	}
	if _, ok := worktrees["branch-b"]; !ok {
		t.Error("branch-b not in list")
	}

	// Cleanup
	_ = mgr.CleanupAll()
}

func TestCleanupBranch(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	worktreeDir := filepath.Join(tmpDir, "worktrees")

	setupTestRepo(t, repoDir)

	mgr, err := NewManager(worktreeDir, repoDir, "main")
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	// Create worktree
	path, _ := mgr.CreateBranchWorktree("to-delete", "")

	// Verify exists
	if _, statErr := os.Stat(path); statErr != nil {
		t.Fatal("Worktree should exist")
	}

	// Cleanup
	if err := mgr.CleanupBranch("to-delete"); err != nil {
		t.Fatalf("CleanupBranch() error: %v", err)
	}

	// Verify removed from tracking
	if mgr.GetWorktreePath("to-delete") != "" {
		t.Error("Worktree still tracked after cleanup")
	}

	// Verify path removed
	if _, statErr := os.Stat(path); statErr == nil {
		t.Error("Worktree path still exists after cleanup")
	}
}

func setupTestRepo(t *testing.T, repoDir string) {
	t.Helper()

	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatalf("Failed to create repo dir: %v", err)
	}
	// Initialize with explicit main branch
	if err := runCommand(repoDir, "git", "init", "-b", "main"); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}
	if err := runCommand(repoDir, "git", "config", "user.email", "test@test.com"); err != nil {
		t.Fatalf("Failed to set git config: %v", err)
	}
	if err := runCommand(repoDir, "git", "config", "user.name", "Test"); err != nil {
		t.Fatalf("Failed to set git config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "test.txt"), []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := runCommand(repoDir, "git", "add", "."); err != nil {
		t.Fatalf("Failed to stage: %v", err)
	}
	if err := runCommand(repoDir, "git", "commit", "-m", "initial"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}
}

func runCommand(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, string(output))
	}
	return nil
}
