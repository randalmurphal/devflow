package testutil

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSetupTestRepo(t *testing.T) {
	dir := SetupTestRepo(t)

	// Check git directory exists
	gitDir := filepath.Join(dir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		t.Error(".git directory does not exist")
	}

	// Check README exists
	readme := filepath.Join(dir, "README.md")
	if _, err := os.Stat(readme); os.IsNotExist(err) {
		t.Error("README.md does not exist")
	}

	// Check we can get current branch
	branch := GetCurrentBranch(t, dir)
	if branch == "" {
		t.Error("GetCurrentBranch returned empty string")
	}

	// Check we can get HEAD SHA
	sha := GetHeadSHA(t, dir)
	if sha == "" {
		t.Error("GetHeadSHA returned empty string")
	}
	if len(sha) != 40 {
		t.Errorf("SHA length = %d, want 40", len(sha))
	}
}

func TestSetupTestRepoWithFiles(t *testing.T) {
	files := map[string]string{
		"src/main.go":     "package main\n",
		"src/lib/util.go": "package lib\n",
		"config.yaml":     "key: value\n",
	}

	dir := SetupTestRepoWithFiles(t, files)

	// Check all files exist
	for path := range files {
		fullPath := filepath.Join(dir, path)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("file %s does not exist", path)
		}
	}
}

func TestCreateBranch(t *testing.T) {
	dir := SetupTestRepo(t)

	CreateBranch(t, dir, "feature-branch")

	branch := GetCurrentBranch(t, dir)
	if branch != "feature-branch" {
		t.Errorf("current branch = %q, want %q", branch, "feature-branch")
	}
}

func TestCommitFile(t *testing.T) {
	dir := SetupTestRepo(t)

	initialSHA := GetHeadSHA(t, dir)

	CommitFile(t, dir, "new-file.txt", "content", "Add new file")

	newSHA := GetHeadSHA(t, dir)
	if newSHA == initialSHA {
		t.Error("SHA did not change after commit")
	}

	// Check file exists
	filePath := filepath.Join(dir, "new-file.txt")
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(content) != "content" {
		t.Errorf("file content = %q, want %q", string(content), "content")
	}
}

func TestSwitchBranch(t *testing.T) {
	dir := SetupTestRepo(t)

	// Create a branch
	CreateBranch(t, dir, "test-branch")

	// Switch back to original
	SwitchBranch(t, dir, "master")

	branch := GetCurrentBranch(t, dir)
	if branch != "master" {
		// Some git versions use "main" as default
		if branch != "main" {
			t.Errorf("current branch = %q, want %q or %q", branch, "master", "main")
		}
	}
}

func TestTempFile(t *testing.T) {
	content := "test content"
	path := TempFileString(t, "test.txt", content)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read temp file: %v", err)
	}

	if string(data) != content {
		t.Errorf("content = %q, want %q", string(data), content)
	}
}

func TestTempDir(t *testing.T) {
	dir := TempDir(t)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("temp directory does not exist")
	}

	// Create a file in it
	filePath := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(filePath, []byte("test"), 0o644); err != nil {
		t.Errorf("failed to write to temp directory: %v", err)
	}
}

func TestTestContext(t *testing.T) {
	ctx := TestContext(t)

	// Context should not be done
	select {
	case <-ctx.Done():
		t.Error("context is already done")
	default:
		// OK
	}
}

func TestTestContextWithTimeout(t *testing.T) {
	ctx := TestContextWithTimeout(t, 100*time.Millisecond)

	// Context should not be done immediately
	select {
	case <-ctx.Done():
		t.Error("context is already done")
	default:
		// OK
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Context should be done
	select {
	case <-ctx.Done():
		// OK
	default:
		t.Error("context should be done after timeout")
	}
}

func TestCancelableContext(t *testing.T) {
	ctx, cancel := CancelableContext(t)

	// Context should not be done
	select {
	case <-ctx.Done():
		t.Error("context is already done")
	default:
		// OK
	}

	// Cancel it
	cancel()

	// Context should be done
	select {
	case <-ctx.Done():
		// OK
	default:
		t.Error("context should be done after cancel")
	}
}

func TestWithTestName(t *testing.T) {
	ctx := WithTestName(context.Background(), t)

	name := TestNameFromContext(ctx)
	if name != t.Name() {
		t.Errorf("name = %q, want %q", name, t.Name())
	}
}

func TestGetShortSHA(t *testing.T) {
	dir := SetupTestRepo(t)

	sha := GetShortSHA(t, dir)
	if sha == "" {
		t.Error("GetShortSHA returned empty string")
	}
	if len(sha) > 12 {
		t.Errorf("short SHA length = %d, expected <= 12", len(sha))
	}

	// Short SHA should be prefix of full SHA
	fullSHA := GetHeadSHA(t, dir)
	if fullSHA[:len(sha)] != sha {
		t.Errorf("short SHA %q is not prefix of full SHA %q", sha, fullSHA)
	}
}

func TestTag(t *testing.T) {
	dir := SetupTestRepo(t)

	Tag(t, dir, "v1.0.0")

	// Verify tag exists by trying to switch to it
	// This should succeed
	SwitchBranch(t, dir, "v1.0.0")
}
