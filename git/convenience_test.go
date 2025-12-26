package git

import (
	"context"
	"testing"
)

func TestCommitAll(t *testing.T) {
	runner := NewSequentialMockRunner()
	runner.AddOutput("", nil)                 // git add -A
	runner.AddOutput("", nil)                 // git commit -m "test message"
	runner.AddOutput("abc123def456", nil)     // git rev-parse HEAD
	runner.AddOutput("feature/test", nil)     // git rev-parse --abbrev-ref HEAD

	ctx := &Context{
		repoPath: t.TempDir(),
		workDir:  t.TempDir(),
		runner:   runner,
	}

	result, err := ctx.CommitAll("test message")
	if err != nil {
		t.Fatalf("CommitAll failed: %v", err)
	}

	if result.SHA != "abc123def456" {
		t.Errorf("SHA = %q, want %q", result.SHA, "abc123def456")
	}
	if result.Branch != "feature/test" {
		t.Errorf("Branch = %q, want %q", result.Branch, "feature/test")
	}
	if result.Message != "test message" {
		t.Errorf("Message = %q, want %q", result.Message, "test message")
	}
}

func TestCommitAll_NothingToCommit(t *testing.T) {
	runner := NewSequentialMockRunner()
	runner.AddOutput("", nil)                       // git add -A
	runner.AddOutput("nothing to commit", ErrNothingToCommit) // git commit

	ctx := &Context{
		repoPath: t.TempDir(),
		workDir:  t.TempDir(),
		runner:   runner,
	}

	_, err := ctx.CommitAll("test message")
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestPushCurrent(t *testing.T) {
	runner := NewSequentialMockRunner()
	runner.AddOutput("main", nil)                    // git rev-parse --abbrev-ref HEAD
	runner.AddOutputError("", "error", nil)          // git rev-parse --verify origin/main (returns error = not pushed)
	runner.AddOutput("", nil)                        // git push -u origin main
	runner.AddOutput("abc123", nil)                  // git rev-parse HEAD
	runner.AddOutput("git@github.com:o/r.git", nil)  // git remote get-url origin

	ctx := &Context{
		repoPath: t.TempDir(),
		workDir:  t.TempDir(),
		runner:   runner,
	}

	result, err := ctx.PushCurrent()
	if err != nil {
		t.Fatalf("PushCurrent failed: %v", err)
	}

	if result.Remote != "origin" {
		t.Errorf("Remote = %q, want %q", result.Remote, "origin")
	}
	if result.Branch != "main" {
		t.Errorf("Branch = %q, want %q", result.Branch, "main")
	}
	if result.SHA != "abc123" {
		t.Errorf("SHA = %q, want %q", result.SHA, "abc123")
	}
	if result.URL != "git@github.com:o/r.git" {
		t.Errorf("URL = %q, want %q", result.URL, "git@github.com:o/r.git")
	}
}

func TestCheckoutNew(t *testing.T) {
	runner := NewSequentialMockRunner()
	runner.AddOutput("", nil) // git branch feature/new
	runner.AddOutput("", nil) // git checkout feature/new

	ctx := &Context{
		repoPath: t.TempDir(),
		workDir:  t.TempDir(),
		runner:   runner,
	}

	err := ctx.CheckoutNew("feature/new")
	if err != nil {
		t.Fatalf("CheckoutNew failed: %v", err)
	}
}

func TestCheckoutNew_BranchExists(t *testing.T) {
	runner := NewSequentialMockRunner()
	runner.AddOutput("already exists", ErrBranchExists) // git branch feature/new

	ctx := &Context{
		repoPath: t.TempDir(),
		workDir:  t.TempDir(),
		runner:   runner,
	}

	err := ctx.CheckoutNew("feature/new")
	if err != ErrBranchExists {
		t.Errorf("expected ErrBranchExists, got %v", err)
	}
}

func TestContextWithGit(t *testing.T) {
	gitCtx := &Context{
		repoPath: "/test/repo",
		workDir:  "/test/repo",
	}

	ctx := ContextWithGit(context.Background(), gitCtx)

	retrieved := GitFromContext(ctx)
	if retrieved == nil {
		t.Fatal("GitFromContext returned nil")
	}
	if retrieved.repoPath != "/test/repo" {
		t.Errorf("repoPath = %q, want %q", retrieved.repoPath, "/test/repo")
	}
}

func TestGitFromContext_Missing(t *testing.T) {
	ctx := context.Background()

	retrieved := GitFromContext(ctx)
	if retrieved != nil {
		t.Errorf("expected nil, got %v", retrieved)
	}
}

func TestMustGitFromContext_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic, got none")
		}
	}()

	ctx := context.Background()
	MustGitFromContext(ctx)
}

func TestCommitAllAndPush(t *testing.T) {
	runner := NewSequentialMockRunner()
	// CommitAll sequence
	runner.AddOutput("", nil)          // git add -A
	runner.AddOutput("", nil)          // git commit
	runner.AddOutput("abc123", nil)    // git rev-parse HEAD
	runner.AddOutput("feature/x", nil) // git rev-parse --abbrev-ref HEAD
	// PushCurrent sequence
	runner.AddOutput("feature/x", nil)              // git rev-parse --abbrev-ref HEAD
	runner.AddOutputError("", "error", nil)         // git rev-parse --verify origin/feature/x (not pushed)
	runner.AddOutput("", nil)                       // git push -u origin feature/x
	runner.AddOutput("abc123", nil)                 // git rev-parse HEAD
	runner.AddOutput("git@github.com:o/r.git", nil) // git remote get-url origin

	ctx := &Context{
		repoPath: t.TempDir(),
		workDir:  t.TempDir(),
		runner:   runner,
	}

	result, err := ctx.CommitAllAndPush("test message")
	if err != nil {
		t.Fatalf("CommitAllAndPush failed: %v", err)
	}

	if result.Commit == nil {
		t.Fatal("Commit result is nil")
	}
	if result.Push == nil {
		t.Fatal("Push result is nil")
	}

	if result.Commit.SHA != "abc123" {
		t.Errorf("Commit.SHA = %q, want %q", result.Commit.SHA, "abc123")
	}
	if result.Push.Branch != "feature/x" {
		t.Errorf("Push.Branch = %q, want %q", result.Push.Branch, "feature/x")
	}
}

func TestCheckoutNewAt(t *testing.T) {
	runner := NewSequentialMockRunner()
	runner.AddOutput("", nil) // git checkout main
	runner.AddOutput("", nil) // git branch feature/new
	runner.AddOutput("", nil) // git checkout feature/new

	ctx := &Context{
		repoPath: t.TempDir(),
		workDir:  t.TempDir(),
		runner:   runner,
	}

	err := ctx.CheckoutNewAt("feature/new", "main")
	if err != nil {
		t.Fatalf("CheckoutNewAt failed: %v", err)
	}

	// Verify all 3 commands were called: checkout ref, create branch, checkout branch
	if len(runner.Calls) != 3 {
		t.Errorf("expected 3 calls, got %d", len(runner.Calls))
	}
}
