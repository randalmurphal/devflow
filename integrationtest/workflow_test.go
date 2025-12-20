package integrationtest

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"github.com/randalmurphal/devflow/artifact"
	devcontext "github.com/randalmurphal/devflow/context"
	"github.com/randalmurphal/devflow/git"
	"github.com/randalmurphal/devflow/notify"
	"github.com/randalmurphal/devflow/transcript"
	"github.com/randalmurphal/devflow/workflow"
	"github.com/randalmurphal/flowgraph/pkg/flowgraph"
	"github.com/randalmurphal/flowgraph/pkg/flowgraph/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSpecToImplementWorkflow tests a simple spec -> implement workflow.
func TestSpecToImplementWorkflow(t *testing.T) {
	repoPath := setupTempRepo(t)

	// Mock LLM responses
	mockLLM := mockResponses(
		`# Technical Specification

## Overview
Implement a simple greeting function.

## Requirements
- Function should accept a name parameter
- Return formatted greeting string
`,
		`package main

func Greet(name string) string {
    return "Hello, " + name + "!"
}
`,
	)

	// Build workflow graph
	graph := flowgraph.NewGraph[workflow.State]().
		AddNode("spec", workflow.GenerateSpecNode).
		AddNode("implement", workflow.ImplementNode).
		AddEdge("spec", "implement").
		AddEdge("implement", flowgraph.END).
		SetEntry("spec")

	compiled, err := graph.Compile()
	require.NoError(t, err)

	// Setup context with services
	ctx := setupContext(t, repoPath, mockLLM)

	// Initialize state
	state := workflow.NewState("spec-to-implement")
	state.TicketID = "TK-123"
	state.Ticket = &workflow.Ticket{
		ID:          "TK-123",
		Title:       "Implement greeting function",
		Description: "Create a simple greeting function that returns a personalized message",
	}
	state.Worktree = repoPath // Need worktree for implementation

	// Execute
	result, err := compiled.Run(ctx, state)
	require.NoError(t, err)

	// Verify spec was generated
	assert.Contains(t, result.Spec, "Technical Specification", "spec should be generated")

	// Verify implementation was generated
	assert.Contains(t, result.Implementation, "func Greet", "implementation should be generated")

	// Verify LLM was called correctly
	assert.GreaterOrEqual(t, mockLLM.CallCount(), 2, "LLM should be called at least twice")
}

// TestReviewLoopWorkflow tests the review -> fix -> review pattern.
func TestReviewLoopWorkflow(t *testing.T) {
	repoPath := setupTempRepo(t)

	// Track execution
	reviewCount := 0
	fixCount := 0

	// Custom review node that fails first, passes second time
	customReview := func(ctx flowgraph.Context, state workflow.State) (workflow.State, error) {
		reviewCount++
		state.Review = &artifact.ReviewResult{
			Approved: reviewCount >= 2, // Pass on second review
		}
		if reviewCount < 2 {
			state.Review.Findings = []artifact.ReviewFinding{
				{
					Severity: "warning",
					Message:  "Missing error handling",
					File:     "main.go",
					Line:     10,
				},
			}
		}
		return state, nil
	}

	// Custom fix node
	customFix := func(ctx flowgraph.Context, state workflow.State) (workflow.State, error) {
		fixCount++
		state.Implementation += "\n// Fixed!"
		return state, nil
	}

	// Router: if review failed, go to fix; otherwise end
	router := func(ctx flowgraph.Context, state workflow.State) string {
		if state.Review != nil && !state.Review.Approved {
			return "fix"
		}
		return flowgraph.END
	}

	graph := flowgraph.NewGraph[workflow.State]().
		AddNode("review", customReview).
		AddNode("fix", customFix).
		AddConditionalEdge("review", router).
		AddEdge("fix", "review"). // Loop back to review after fix
		SetEntry("review")

	compiled, err := graph.Compile()
	require.NoError(t, err)

	ctx := setupContext(t, repoPath, nil)
	state := workflow.NewState("review-loop")
	state.Implementation = "package main"

	result, err := compiled.Run(ctx, state)
	require.NoError(t, err)

	// Verify loop executed
	assert.Equal(t, 2, reviewCount, "should review twice")
	assert.Equal(t, 1, fixCount, "should fix once")
	assert.True(t, result.Review.Approved, "final review should be approved")
	assert.Contains(t, result.Implementation, "// Fixed!", "implementation should be updated")
}

// TestNotificationWorkflow tests notification integration.
func TestNotificationWorkflow(t *testing.T) {
	repoPath := setupTempRepo(t)

	// Capture notifications
	var captured []notify.Event
	captureNotifier := &notificationCapture{events: &captured}

	// Setup context with notifier
	baseCtx := context.Background()
	baseCtx = notify.WithNotifier(baseCtx, captureNotifier)

	gitCtx, err := git.NewContext(repoPath)
	require.NoError(t, err)
	baseCtx = devcontext.WithGit(baseCtx, gitCtx)

	ctx := flowgraph.NewContext(baseCtx)

	// Build graph with notification
	graph := flowgraph.NewGraph[workflow.State]().
		AddNode("work", func(ctx flowgraph.Context, state workflow.State) (workflow.State, error) {
			state.Spec = "Work completed"
			return state, nil
		}).
		AddNode("notify", workflow.NotifyNode).
		AddEdge("work", "notify").
		AddEdge("notify", flowgraph.END).
		SetEntry("work")

	compiled, err := graph.Compile()
	require.NoError(t, err)

	state := workflow.NewState("notify-test")
	_, err = compiled.Run(ctx, state)
	require.NoError(t, err)

	// Verify notification was sent
	assert.Len(t, captured, 1, "should capture one notification")
	assert.Equal(t, notify.EventRunCompleted, captured[0].Type)
}

// TestTranscriptRecording tests that transcript recording works with workflows.
func TestTranscriptRecording(t *testing.T) {
	repoPath := setupTempRepo(t)

	// Setup transcript manager
	transcriptDir := filepath.Join(repoPath, ".devflow", "transcripts")
	manager, err := transcript.NewFileStore(transcript.StoreConfig{
		BaseDir: transcriptDir,
	})
	require.NoError(t, err)

	// Start a run
	runID := "transcript-test-run"
	err = manager.StartRun(runID, transcript.RunMetadata{FlowID: "test"})
	require.NoError(t, err)

	// Setup context with transcript manager
	baseCtx := context.Background()
	baseCtx = devcontext.WithTranscript(baseCtx, manager)

	ctx := flowgraph.NewContext(baseCtx)

	// Wrap node with transcript recording
	recordingNode := flowgraph.NodeFunc[workflow.State](workflow.WithTranscript(
		func(ctx flowgraph.Context, state workflow.State) (workflow.State, error) {
			state.Spec = "Generated specification"
			return state, nil
		},
		"generate-spec",
	))

	graph := flowgraph.NewGraph[workflow.State]().
		AddNode("spec", recordingNode).
		AddEdge("spec", flowgraph.END).
		SetEntry("spec")

	compiled, err := graph.Compile()
	require.NoError(t, err)

	state := workflow.NewState("test")
	state.RunID = runID

	_, err = compiled.Run(ctx, state)
	require.NoError(t, err)

	// End the run
	err = manager.EndRun(runID, transcript.RunStatusCompleted)
	require.NoError(t, err)

	// Load and view the transcript
	tr, err := manager.Load(runID)
	require.NoError(t, err)

	var buf bytes.Buffer
	viewer := transcript.NewViewer(false) // no color for test
	err = viewer.ViewFull(&buf, tr)
	require.NoError(t, err)
	assert.NotEmpty(t, buf.String(), "transcript should have content")
}

// TestArtifactStorage tests that artifacts are saved during workflows.
func TestArtifactStorage(t *testing.T) {
	repoPath := setupTempRepo(t)

	// Setup artifact manager
	artifactDir := filepath.Join(repoPath, ".devflow", "artifacts")
	manager := artifact.NewManager(artifact.Config{
		BaseDir: artifactDir,
	})

	// Setup context with artifact manager
	baseCtx := context.Background()
	baseCtx = devcontext.WithArtifact(baseCtx, manager)
	baseCtx = devcontext.WithRunner(baseCtx, git.NewMockRunner())

	ctx := flowgraph.NewContext(baseCtx)

	// RunTestsNode saves test output as artifact
	state := workflow.NewState("artifact-test")
	state.RunID = "artifact-run-123"
	state.Worktree = repoPath

	result, err := workflow.RunTestsNode(ctx, state)
	require.NoError(t, err)

	// Verify test output is in state
	assert.NotNil(t, result.TestOutput, "test output should be set")

	// Verify artifact was saved
	output, err := manager.LoadArtifact(state.RunID, "test-output.json")
	require.NoError(t, err)
	assert.NotEmpty(t, output, "artifact should be saved")
}

// TestMockClientUsage verifies the MockClient behavior.
func TestMockClientUsage(t *testing.T) {
	// Test sequential responses
	mock := llm.NewMockClient("").WithResponses("first", "second", "third")

	resp1, _ := mock.Complete(context.Background(), llm.CompletionRequest{})
	assert.Equal(t, "first", resp1.Content)

	resp2, _ := mock.Complete(context.Background(), llm.CompletionRequest{})
	assert.Equal(t, "second", resp2.Content)

	resp3, _ := mock.Complete(context.Background(), llm.CompletionRequest{})
	assert.Equal(t, "third", resp3.Content)

	// After exhausting responses, cycles back to first (modulo behavior)
	resp4, _ := mock.Complete(context.Background(), llm.CompletionRequest{})
	assert.Equal(t, "first", resp4.Content) // cycles back

	// Check call count
	assert.Equal(t, 4, mock.CallCount())
}

// TestMockClientWithCompleteFunc verifies custom completion handler behavior.
func TestMockClientWithCompleteFunc(t *testing.T) {
	var receivedPrompt string

	mock := llm.NewMockClient("").WithCompleteFunc(func(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
		receivedPrompt = req.SystemPrompt
		return &llm.CompletionResponse{
			Content: "Handled: " + req.SystemPrompt,
		}, nil
	})

	resp, err := mock.Complete(context.Background(), llm.CompletionRequest{
		SystemPrompt: "Be helpful",
	})
	require.NoError(t, err)

	assert.Equal(t, "Be helpful", receivedPrompt)
	assert.Equal(t, "Handled: Be helpful", resp.Content)
}

// notificationCapture captures notifications for testing.
type notificationCapture struct {
	events *[]notify.Event
}

func (n *notificationCapture) Notify(ctx context.Context, event notify.Event) error {
	*n.events = append(*n.events, event)
	return nil
}
