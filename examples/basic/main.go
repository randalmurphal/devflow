// Package main demonstrates basic usage of the devflow library.
//
// This example shows how to:
// - Set up a GitContext for repository operations
// - Create and manage transcripts for conversation logging
// - Use the artifact manager for storing outputs
// - Configure notifications via Slack or webhooks
package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/randalmurphal/devflow/artifact"
	devcontext "github.com/randalmurphal/devflow/context"
	"github.com/randalmurphal/devflow/git"
	"github.com/randalmurphal/devflow/notify"
	"github.com/randalmurphal/devflow/transcript"
)

func main() {
	// Create a temporary directory for the example
	baseDir, err := os.MkdirTemp("", "devflow-example-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(baseDir)

	fmt.Println("DevFlow Example")
	fmt.Println("===============")
	fmt.Printf("Working directory: %s\n\n", baseDir)

	// Initialize a git repository for the example
	repoPath := filepath.Join(baseDir, "repo")
	if err := initGitRepo(repoPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Example 1: Git Operations
	fmt.Println("1. Git Operations")
	fmt.Println("-----------------")
	gitExample(repoPath)

	// Example 2: Transcript Management
	fmt.Println("\n2. Transcript Management")
	fmt.Println("------------------------")
	transcriptExample(baseDir)

	// Example 3: Artifact Storage
	fmt.Println("\n3. Artifact Storage")
	fmt.Println("-------------------")
	artifactExample(baseDir)

	// Example 4: Context Injection
	fmt.Println("\n4. Context Injection")
	fmt.Println("--------------------")
	contextExample(baseDir, repoPath)

	// Example 5: Commit Messages
	fmt.Println("\n5. Commit Messages")
	fmt.Println("------------------")
	commitMessageExample()

	fmt.Println("\nExample completed successfully!")
}

func initGitRepo(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}

	runner := git.NewExecRunner()
	_, err := runner.Run(path, "git", "init")
	if err != nil {
		return fmt.Errorf("git init: %w", err)
	}

	// Create initial commit
	_, _ = runner.Run(path, "git", "config", "user.email", "example@devflow.io")
	_, _ = runner.Run(path, "git", "config", "user.name", "DevFlow Example")

	if err := os.WriteFile(filepath.Join(path, "README.md"), []byte("# Example\n"), 0644); err != nil {
		return err
	}

	_, err = runner.Run(path, "git", "add", ".")
	if err != nil {
		return fmt.Errorf("git add: %w", err)
	}

	_, err = runner.Run(path, "git", "commit", "-m", "Initial commit")
	if err != nil {
		return fmt.Errorf("git commit: %w", err)
	}

	return nil
}

func gitExample(repoPath string) {
	gitCtx, err := git.NewContext(repoPath)
	if err != nil {
		fmt.Printf("Error creating GitContext: %v\n", err)
		return
	}

	// Get current branch
	branch, err := gitCtx.CurrentBranch()
	if err != nil {
		fmt.Printf("Error getting branch: %v\n", err)
		return
	}
	fmt.Printf("Current branch: %s\n", branch)

	// Check if working directory is clean
	clean, err := gitCtx.IsClean()
	if err != nil {
		fmt.Printf("Error checking clean: %v\n", err)
		return
	}
	fmt.Printf("Working directory clean: %v\n", clean)

	// Get repo path
	repoPathResult := gitCtx.RepoPath()
	fmt.Printf("Repo path: %s\n", repoPathResult)

	// Create a branch (but don't switch to it)
	newBranch := "feature/example-branch"
	if err := gitCtx.CreateBranch(newBranch); err != nil {
		fmt.Printf("Error creating branch: %v\n", err)
	} else {
		fmt.Printf("Created branch: %s\n", newBranch)
	}
}

func transcriptExample(baseDir string) {
	// Create transcript store directly for full functionality
	store, err := transcript.NewFileStore(transcript.StoreConfig{
		BaseDir: baseDir,
	})
	if err != nil {
		fmt.Printf("Error creating store: %v\n", err)
		return
	}

	// Start a new run
	runID := fmt.Sprintf("example-run-%d", time.Now().Unix())
	err = store.StartRun(runID, transcript.RunMetadata{
		FlowID: "example-flow",
		Input:  map[string]any{"task": "demo"},
	})
	if err != nil {
		fmt.Printf("Error starting run: %v\n", err)
		return
	}
	fmt.Printf("Started run: %s\n", runID)

	// Record some turns
	err = store.RecordTurn(runID, transcript.Turn{
		Role:     "user",
		Content:  "Hello, can you help me with Go?",
		TokensIn: 10,
	})
	if err != nil {
		fmt.Printf("Error recording turn: %v\n", err)
	}

	err = store.RecordTurn(runID, transcript.Turn{
		Role:      "assistant",
		Content:   "Of course! I'd be happy to help you with Go programming.",
		TokensOut: 15,
	})
	if err != nil {
		fmt.Printf("Error recording turn: %v\n", err)
	}

	// Add cost tracking (available on FileStore)
	_ = store.AddCost(runID, 0.001)

	// End the run
	err = store.EndRun(runID, transcript.RunStatusCompleted)
	if err != nil {
		fmt.Printf("Error ending run: %v\n", err)
		return
	}
	fmt.Printf("Ended run: %s\n", runID)

	// Get run info
	meta, err := store.LoadMetadata(runID)
	if err != nil {
		fmt.Printf("Error getting metadata: %v\n", err)
		return
	}
	fmt.Printf("Run status: %s, turns: %d\n", meta.Status, meta.TurnCount)
}

func artifactExample(baseDir string) {
	// Create artifact manager
	artifacts := artifact.NewManager(artifact.Config{
		BaseDir:       baseDir,
		CompressAbove: 1024, // Compress files > 1KB
	})

	runID := "artifact-example"

	// Save a JSON artifact (must be []byte)
	data := map[string]any{
		"output": "example data",
		"count":  42,
	}
	jsonData, _ := json.Marshal(data)
	err := artifacts.SaveArtifact(runID, "output.json", jsonData)
	if err != nil {
		fmt.Printf("Error saving artifact: %v\n", err)
		return
	}
	fmt.Println("Saved artifact: output.json")

	// Save a text artifact
	err = artifacts.SaveArtifact(runID, "notes.txt", []byte("This is a note"))
	if err != nil {
		fmt.Printf("Error saving text: %v\n", err)
		return
	}
	fmt.Println("Saved artifact: notes.txt")

	// List artifacts
	list, err := artifacts.ListArtifacts(runID)
	if err != nil {
		fmt.Printf("Error listing artifacts: %v\n", err)
		return
	}
	fmt.Printf("Artifacts: %v\n", list)

	// Load artifact back
	loadedData, err := artifacts.LoadArtifact(runID, "output.json")
	if err != nil {
		fmt.Printf("Error loading artifact: %v\n", err)
		return
	}
	var loaded map[string]any
	json.Unmarshal(loadedData, &loaded)
	fmt.Printf("Loaded data: %v\n", loaded)
}

func contextExample(baseDir, repoPath string) {
	gitCtx, _ := git.NewContext(repoPath)
	transcripts, _ := transcript.NewFileStore(transcript.StoreConfig{
		BaseDir: baseDir,
	})
	artifacts := artifact.NewManager(artifact.Config{
		BaseDir: baseDir,
	})

	// Create a log notifier (uses slog.Logger)
	logger := slog.Default()
	notifier := notify.NewLogNotifier(logger)

	// Inject services into context using the services struct
	services := &devcontext.Services{
		Git:         gitCtx,
		Transcripts: transcripts,
		Artifacts:   artifacts,
		Notifier:    notifier,
	}

	ctx := services.InjectAll(nil)

	// Retrieve from context (as workflow nodes would)
	if g := devcontext.Git(ctx); g != nil {
		fmt.Println("GitContext injected successfully")
	}
	if t := devcontext.Transcript(ctx); t != nil {
		fmt.Println("TranscriptManager injected successfully")
	}
	if a := devcontext.Artifact(ctx); a != nil {
		fmt.Println("ArtifactManager injected successfully")
	}
	if n := notify.NotifierFromContext(ctx); n != nil {
		fmt.Println("Notifier injected successfully")
	}
}

func commitMessageExample() {
	// Create a conventional commit message
	msg := git.NewCommitMessage(git.CommitTypeFeat, "add user authentication").
		WithScope("auth").
		WithBody("Implements OAuth2 authentication flow with support for multiple providers.").
		WithTicketRef("TK-123")

	fmt.Println("Generated commit message:")
	fmt.Println("---")
	fmt.Println(msg.String())
	fmt.Println("---")

	// Validate the message
	if err := msg.Validate(); err != nil {
		fmt.Printf("Validation error: %v\n", err)
	} else {
		fmt.Println("Message is valid!")
	}
}
