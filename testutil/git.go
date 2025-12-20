package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// SetupTestRepo creates a temporary git repository for testing.
// Returns the path to the repository.
// The repository is automatically cleaned up when the test ends.
func SetupTestRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	// Initialize git repo
	if err := runGit(t, dir, "init"); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	// Configure git user
	if err := runGit(t, dir, "config", "user.email", "test@test.com"); err != nil {
		t.Fatalf("git config email failed: %v", err)
	}
	if err := runGit(t, dir, "config", "user.name", "Test User"); err != nil {
		t.Fatalf("git config name failed: %v", err)
	}

	// Create initial commit
	readme := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readme, []byte("# Test Repository\n"), 0o644); err != nil {
		t.Fatalf("failed to create README: %v", err)
	}

	if err := runGit(t, dir, "add", "."); err != nil {
		t.Fatalf("git add failed: %v", err)
	}

	if err := runGit(t, dir, "commit", "-m", "Initial commit"); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	return dir
}

// SetupTestRepoWithFiles creates a test repo with specified files.
func SetupTestRepoWithFiles(t *testing.T, files map[string]string) string {
	t.Helper()

	dir := SetupTestRepo(t)

	for path, content := range files {
		fullPath := filepath.Join(dir, path)

		// Ensure directory exists
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatalf("failed to create directory for %s: %v", path, err)
		}

		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write file %s: %v", path, err)
		}
	}

	// Commit the files
	if err := runGit(t, dir, "add", "."); err != nil {
		t.Fatalf("git add failed: %v", err)
	}

	if err := runGit(t, dir, "commit", "-m", "Add test files"); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	return dir
}

// CreateBranch creates a new branch in the test repo.
func CreateBranch(t *testing.T, repoDir, branch string) {
	t.Helper()

	if err := runGit(t, repoDir, "checkout", "-b", branch); err != nil {
		t.Fatalf("git checkout -b %s failed: %v", branch, err)
	}
}

// SwitchBranch switches to an existing branch.
func SwitchBranch(t *testing.T, repoDir, branch string) {
	t.Helper()

	if err := runGit(t, repoDir, "checkout", branch); err != nil {
		t.Fatalf("git checkout %s failed: %v", branch, err)
	}
}

// CommitFile creates or updates a file and commits it.
func CommitFile(t *testing.T, repoDir, path, content, message string) {
	t.Helper()

	fullPath := filepath.Join(repoDir, path)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("failed to create directory for %s: %v", path, err)
	}

	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}

	if err := runGit(t, repoDir, "add", path); err != nil {
		t.Fatalf("git add %s failed: %v", path, err)
	}

	if err := runGit(t, repoDir, "commit", "-m", message); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}
}

// GetCurrentBranch returns the current branch name.
func GetCurrentBranch(t *testing.T, repoDir string) string {
	t.Helper()

	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = repoDir

	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("git branch --show-current failed: %v", err)
	}

	// Trim newline
	branch := string(output)
	if len(branch) > 0 && branch[len(branch)-1] == '\n' {
		branch = branch[:len(branch)-1]
	}

	return branch
}

// GetHeadSHA returns the current HEAD SHA.
func GetHeadSHA(t *testing.T, repoDir string) string {
	t.Helper()

	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoDir

	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("git rev-parse HEAD failed: %v", err)
	}

	// Trim newline
	sha := string(output)
	if len(sha) > 0 && sha[len(sha)-1] == '\n' {
		sha = sha[:len(sha)-1]
	}

	return sha
}

// GetShortSHA returns the short SHA for HEAD.
func GetShortSHA(t *testing.T, repoDir string) string {
	t.Helper()

	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	cmd.Dir = repoDir

	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("git rev-parse --short HEAD failed: %v", err)
	}

	sha := string(output)
	if len(sha) > 0 && sha[len(sha)-1] == '\n' {
		sha = sha[:len(sha)-1]
	}

	return sha
}

// runGit runs a git command in the specified directory.
func runGit(t *testing.T, dir string, args ...string) error {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test User",
		"GIT_AUTHOR_EMAIL=test@test.com",
		"GIT_COMMITTER_NAME=Test User",
		"GIT_COMMITTER_EMAIL=test@test.com",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("git %v output: %s", args, output)
		return err
	}

	return nil
}

// MergeBranch merges a branch into the current branch.
func MergeBranch(t *testing.T, repoDir, branch string) {
	t.Helper()

	if err := runGit(t, repoDir, "merge", "--no-ff", "-m", "Merge "+branch, branch); err != nil {
		t.Fatalf("git merge %s failed: %v", branch, err)
	}
}

// AddRemote adds a remote to the repository.
func AddRemote(t *testing.T, repoDir, name, url string) {
	t.Helper()

	if err := runGit(t, repoDir, "remote", "add", name, url); err != nil {
		t.Fatalf("git remote add %s %s failed: %v", name, url, err)
	}
}

// Tag creates a tag at HEAD.
func Tag(t *testing.T, repoDir, tag string) {
	t.Helper()

	if err := runGit(t, repoDir, "tag", tag); err != nil {
		t.Fatalf("git tag %s failed: %v", tag, err)
	}
}

// Stash creates a stash entry.
func Stash(t *testing.T, repoDir string) {
	t.Helper()

	if err := runGit(t, repoDir, "stash"); err != nil {
		t.Fatalf("git stash failed: %v", err)
	}
}

// StashPop pops the latest stash entry.
func StashPop(t *testing.T, repoDir string) {
	t.Helper()

	if err := runGit(t, repoDir, "stash", "pop"); err != nil {
		t.Fatalf("git stash pop failed: %v", err)
	}
}
