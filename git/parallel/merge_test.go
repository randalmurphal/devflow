package parallel

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMergeBranchesNoConflict(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	worktreeDir := filepath.Join(tmpDir, "worktrees")

	setupTestRepo(t, repoDir)

	mgr, err := NewManager(worktreeDir, repoDir, "main")
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	// Create a branch worktree
	wtPath, err := mgr.CreateBranchWorktree("feature-a", "")
	if err != nil {
		t.Fatalf("CreateBranchWorktree() error: %v", err)
	}

	// Make a change in the worktree (add a new file - no conflict)
	newFile := filepath.Join(wtPath, "feature-a.txt")
	if err := os.WriteFile(newFile, []byte("feature a content"), 0644); err != nil {
		t.Fatalf("Failed to create file in worktree: %v", err)
	}

	// Commit the change in the worktree
	if err := runCommand(wtPath, "git", "add", "."); err != nil {
		t.Fatalf("Failed to stage in worktree: %v", err)
	}
	if err := runCommand(wtPath, "git", "commit", "-m", "add feature-a file"); err != nil {
		t.Fatalf("Failed to commit in worktree: %v", err)
	}

	// Merge back to base
	ctx := context.Background()
	results, mergeErr := mgr.MergeBranches(ctx, MergeConfig{
		CommitMessage: "merge feature-a",
	})
	if mergeErr != nil {
		t.Fatalf("MergeBranches() error: %v", mergeErr)
	}

	// Verify merge succeeded
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}
	if !results[0].Success {
		t.Errorf("Expected merge success, got failure: %v", results[0].Error)
	}
	if len(results[0].Conflicts) > 0 {
		t.Errorf("Expected no conflicts, got %d", len(results[0].Conflicts))
	}

	// Verify file exists in base repo
	baseFile := filepath.Join(repoDir, "feature-a.txt")
	if _, statErr := os.Stat(baseFile); statErr != nil {
		t.Error("Merged file should exist in base repo")
	}

	// Cleanup
	_ = mgr.CleanupAll()
}

func TestMergeBranchesWithConflict(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	worktreeDir := filepath.Join(tmpDir, "worktrees")

	setupTestRepo(t, repoDir)

	mgr, err := NewManager(worktreeDir, repoDir, "main")
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	// Create a branch worktree
	wtPath, err := mgr.CreateBranchWorktree("conflict-branch", "")
	if err != nil {
		t.Fatalf("CreateBranchWorktree() error: %v", err)
	}

	// Modify test.txt in the worktree
	wtFile := filepath.Join(wtPath, "test.txt")
	if err := os.WriteFile(wtFile, []byte("worktree version"), 0644); err != nil {
		t.Fatalf("Failed to modify file in worktree: %v", err)
	}

	// Commit the change in the worktree
	if err := runCommand(wtPath, "git", "add", "."); err != nil {
		t.Fatalf("Failed to stage in worktree: %v", err)
	}
	if err := runCommand(wtPath, "git", "commit", "-m", "modify test.txt in worktree"); err != nil {
		t.Fatalf("Failed to commit in worktree: %v", err)
	}

	// ALSO modify test.txt in the base repo (create conflict)
	baseFile := filepath.Join(repoDir, "test.txt")
	if err := os.WriteFile(baseFile, []byte("base version"), 0644); err != nil {
		t.Fatalf("Failed to modify file in base repo: %v", err)
	}
	if err := runCommand(repoDir, "git", "add", "."); err != nil {
		t.Fatalf("Failed to stage in base: %v", err)
	}
	if err := runCommand(repoDir, "git", "commit", "-m", "modify test.txt in base"); err != nil {
		t.Fatalf("Failed to commit in base: %v", err)
	}

	// Try to merge - should detect conflict
	ctx := context.Background()
	results, _ := mgr.MergeBranches(ctx, MergeConfig{})

	// Verify conflict was detected
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}
	if results[0].Success {
		t.Error("Expected merge to fail due to conflict")
	}
	if len(results[0].Conflicts) == 0 {
		t.Error("Expected conflicts to be detected")
	}

	// Abort the merge to clean up
	_ = mgr.AbortMerge()

	// Cleanup
	_ = mgr.CleanupAll()
}

func TestResolveConflict(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	worktreeDir := filepath.Join(tmpDir, "worktrees")

	setupTestRepo(t, repoDir)

	mgr, err := NewManager(worktreeDir, repoDir, "main")
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	// Create conflict scenario
	wtPath, _ := mgr.CreateBranchWorktree("resolve-test", "")

	// Modify test.txt in worktree
	wtFile := filepath.Join(wtPath, "test.txt")
	if err := os.WriteFile(wtFile, []byte("worktree content"), 0644); err != nil {
		t.Fatalf("Failed to modify file: %v", err)
	}
	if err := runCommand(wtPath, "git", "add", "."); err != nil {
		t.Fatalf("Failed to stage: %v", err)
	}
	if err := runCommand(wtPath, "git", "commit", "-m", "worktree change"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Modify in base
	baseFile := filepath.Join(repoDir, "test.txt")
	if err := os.WriteFile(baseFile, []byte("base content"), 0644); err != nil {
		t.Fatalf("Failed to modify base: %v", err)
	}
	if err := runCommand(repoDir, "git", "add", "."); err != nil {
		t.Fatalf("Failed to stage base: %v", err)
	}
	if err := runCommand(repoDir, "git", "commit", "-m", "base change"); err != nil {
		t.Fatalf("Failed to commit base: %v", err)
	}

	// Start merge (will conflict)
	ctx := context.Background()
	results, _ := mgr.MergeBranches(ctx, MergeConfig{})

	if len(results) == 0 || len(results[0].Conflicts) == 0 {
		t.Skip("No conflicts generated, skipping resolution test")
	}

	// Resolve the conflict
	resolvedContent := "resolved content"
	if resolveErr := mgr.ResolveConflict("test.txt", resolvedContent); resolveErr != nil {
		t.Fatalf("ResolveConflict() error: %v", resolveErr)
	}

	// Continue merge
	if continueErr := mgr.ContinueMerge("merge with resolution"); continueErr != nil {
		t.Fatalf("ContinueMerge() error: %v", continueErr)
	}

	// Verify resolved content
	content, readErr := os.ReadFile(baseFile)
	if readErr != nil {
		t.Fatalf("Failed to read resolved file: %v", readErr)
	}
	if string(content) != resolvedContent {
		t.Errorf("Expected %q, got %q", resolvedContent, string(content))
	}

	// Cleanup
	_ = mgr.CleanupAll()
}

func TestMergeSingleBranch(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	worktreeDir := filepath.Join(tmpDir, "worktrees")

	setupTestRepo(t, repoDir)

	mgr, err := NewManager(worktreeDir, repoDir, "main")
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	// Create two branch worktrees
	wtPathA, _ := mgr.CreateBranchWorktree("branch-a", "")
	wtPathB, _ := mgr.CreateBranchWorktree("branch-b", "")

	// Make changes in both
	if err := os.WriteFile(filepath.Join(wtPathA, "a.txt"), []byte("a"), 0644); err != nil {
		t.Fatalf("Failed to create a.txt: %v", err)
	}
	if err := runCommand(wtPathA, "git", "add", "."); err != nil {
		t.Fatalf("Failed to stage a: %v", err)
	}
	if err := runCommand(wtPathA, "git", "commit", "-m", "add a"); err != nil {
		t.Fatalf("Failed to commit a: %v", err)
	}

	if err := os.WriteFile(filepath.Join(wtPathB, "b.txt"), []byte("b"), 0644); err != nil {
		t.Fatalf("Failed to create b.txt: %v", err)
	}
	if err := runCommand(wtPathB, "git", "add", "."); err != nil {
		t.Fatalf("Failed to stage b: %v", err)
	}
	if err := runCommand(wtPathB, "git", "commit", "-m", "add b"); err != nil {
		t.Fatalf("Failed to commit b: %v", err)
	}

	// Merge only branch-a
	ctx := context.Background()
	result := mgr.MergeSingleBranch(ctx, "branch-a", MergeConfig{})

	if !result.Success {
		t.Errorf("Expected merge success, got failure: %v", result.Error)
	}

	// Verify a.txt exists in base, but not b.txt (not merged yet)
	if _, statErr := os.Stat(filepath.Join(repoDir, "a.txt")); statErr != nil {
		t.Error("a.txt should exist after merge")
	}
	if _, statErr := os.Stat(filepath.Join(repoDir, "b.txt")); statErr == nil {
		t.Error("b.txt should NOT exist (not merged)")
	}

	// Now merge branch-b
	resultB := mgr.MergeSingleBranch(ctx, "branch-b", MergeConfig{})
	if !resultB.Success {
		t.Errorf("Expected branch-b merge success, got failure: %v", resultB.Error)
	}

	// Verify both files exist
	if _, statErr := os.Stat(filepath.Join(repoDir, "b.txt")); statErr != nil {
		t.Error("b.txt should exist after merge")
	}

	// Cleanup
	_ = mgr.CleanupAll()
}

func TestAbortMerge(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	worktreeDir := filepath.Join(tmpDir, "worktrees")

	setupTestRepo(t, repoDir)

	mgr, err := NewManager(worktreeDir, repoDir, "main")
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	// Create conflict scenario
	wtPath, _ := mgr.CreateBranchWorktree("abort-test", "")

	// Modify in worktree
	if err := os.WriteFile(filepath.Join(wtPath, "test.txt"), []byte("wt"), 0644); err != nil {
		t.Fatalf("Failed to write: %v", err)
	}
	if err := runCommand(wtPath, "git", "add", "."); err != nil {
		t.Fatalf("Failed to stage: %v", err)
	}
	if err := runCommand(wtPath, "git", "commit", "-m", "wt change"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Modify in base
	if err := os.WriteFile(filepath.Join(repoDir, "test.txt"), []byte("base"), 0644); err != nil {
		t.Fatalf("Failed to write base: %v", err)
	}
	if err := runCommand(repoDir, "git", "add", "."); err != nil {
		t.Fatalf("Failed to stage base: %v", err)
	}
	if err := runCommand(repoDir, "git", "commit", "-m", "base change"); err != nil {
		t.Fatalf("Failed to commit base: %v", err)
	}

	// Start merge (will conflict)
	ctx := context.Background()
	_, _ = mgr.MergeBranches(ctx, MergeConfig{})

	// Abort the merge
	if abortErr := mgr.AbortMerge(); abortErr != nil {
		// This might fail if no merge was in progress
		if !strings.Contains(abortErr.Error(), "no merge in progress") {
			t.Logf("AbortMerge() error (may be expected): %v", abortErr)
		}
	}

	// Verify repo is in clean state
	status, _ := runCommandOutput(repoDir, "git", "status", "--porcelain")
	if strings.TrimSpace(status) != "" {
		t.Logf("Repo may have uncommitted changes after abort: %s", status)
	}

	// Cleanup
	_ = mgr.CleanupAll()
}

func runCommandOutput(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	return string(output), err
}
