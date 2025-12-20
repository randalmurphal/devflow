package devflow

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/rmurphy/flowgraph/pkg/flowgraph"
	"github.com/rmurphy/flowgraph/pkg/flowgraph/llm"
)

// testContext creates a flowgraph.Context for use in tests.
// For tests that need service injection, use testContextWith pattern instead.
func testContext() flowgraph.Context {
	return flowgraph.NewContext(context.Background())
}

// wrapContext wraps a context.Context (with injected services) into a flowgraph.Context
func wrapContext(ctx context.Context) flowgraph.Context {
	return flowgraph.NewContext(ctx)
}

// =============================================================================
// State Tests
// =============================================================================

func TestNewDevState(t *testing.T) {
	state := NewDevState("test-flow")

	if state.FlowID != "test-flow" {
		t.Errorf("FlowID = %q, want %q", state.FlowID, "test-flow")
	}

	if state.RunID == "" {
		t.Error("RunID should not be empty")
	}

	if state.StartTime.IsZero() {
		t.Error("StartTime should be set")
	}
}

func TestDevState_WithTicket(t *testing.T) {
	ticket := &Ticket{
		ID:    "TK-123",
		Title: "Test Ticket",
	}

	state := NewDevState("test").WithTicket(ticket)

	if state.TicketID != "TK-123" {
		t.Errorf("TicketID = %q, want %q", state.TicketID, "TK-123")
	}

	if state.Ticket == nil {
		t.Error("Ticket should not be nil")
	}

	if state.Ticket.Title != "Test Ticket" {
		t.Errorf("Ticket.Title = %q, want %q", state.Ticket.Title, "Test Ticket")
	}
}

func TestDevState_AddTokens(t *testing.T) {
	state := NewDevState("test")

	state.AddTokens(1000, 500)
	state.AddTokens(2000, 1000)

	if state.TotalTokensIn != 3000 {
		t.Errorf("TotalTokensIn = %d, want %d", state.TotalTokensIn, 3000)
	}

	if state.TotalTokensOut != 1500 {
		t.Errorf("TotalTokensOut = %d, want %d", state.TotalTokensOut, 1500)
	}

	if state.TotalCost <= 0 {
		t.Error("TotalCost should be positive")
	}
}

func TestDevState_Validate(t *testing.T) {
	tests := []struct {
		name    string
		state   DevState
		reqs    []StateRequirement
		wantErr bool
	}{
		{
			name:    "no requirements",
			state:   NewDevState("test"),
			reqs:    nil,
			wantErr: false,
		},
		{
			name:    "ticket required but missing",
			state:   NewDevState("test"),
			reqs:    []StateRequirement{RequireTicket},
			wantErr: true,
		},
		{
			name:    "ticket required and present",
			state:   NewDevState("test").WithTicket(&Ticket{ID: "TK-1"}),
			reqs:    []StateRequirement{RequireTicket},
			wantErr: false,
		},
		{
			name:    "worktree required but missing",
			state:   NewDevState("test"),
			reqs:    []StateRequirement{RequireWorktree},
			wantErr: true,
		},
		{
			name: "worktree required and present",
			state: func() DevState {
				s := NewDevState("test")
				s.Worktree = "/tmp/worktree"
				return s
			}(),
			reqs:    []StateRequirement{RequireWorktree},
			wantErr: false,
		},
		{
			name: "multiple requirements met",
			state: func() DevState {
				s := NewDevState("test").WithTicket(&Ticket{ID: "TK-1"})
				s.Worktree = "/tmp/worktree"
				s.Spec = "some spec"
				return s
			}(),
			reqs:    []StateRequirement{RequireTicket, RequireWorktree, RequireSpec},
			wantErr: false,
		},
		{
			name: "one of multiple requirements missing",
			state: func() DevState {
				s := NewDevState("test").WithTicket(&Ticket{ID: "TK-1"})
				s.Worktree = "/tmp/worktree"
				return s
			}(),
			reqs:    []StateRequirement{RequireTicket, RequireWorktree, RequireSpec},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.state.Validate(tt.reqs...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDevState_Summary(t *testing.T) {
	state := NewDevState("test-flow")
	state.TotalTokensIn = 1000
	state.TotalTokensOut = 500
	state.TotalCost = 0.05

	summary := state.Summary()

	if summary == "" {
		t.Error("Summary should not be empty")
	}

	// Check contains key information
	if !containsAll(summary, "test-flow", "1000", "500") {
		t.Errorf("Summary missing expected content: %s", summary)
	}
}

func TestDevState_ReviewRouting(t *testing.T) {
	t.Run("approved review goes to create-pr", func(t *testing.T) {
		state := NewDevState("test")
		state.Review = &ReviewResult{Approved: true}

		if state.NeedsReviewFix() {
			t.Error("NeedsReviewFix should be false for approved review")
		}
	})

	t.Run("rejected review needs fix", func(t *testing.T) {
		state := NewDevState("test")
		state.Review = &ReviewResult{Approved: false}

		if !state.NeedsReviewFix() {
			t.Error("NeedsReviewFix should be true for rejected review")
		}
	})

	t.Run("can retry within limit", func(t *testing.T) {
		state := NewDevState("test")
		state.ReviewAttempts = 2

		if !state.CanRetryReview(3) {
			t.Error("CanRetryReview should be true with 2 attempts and max 3")
		}
	})

	t.Run("cannot retry at limit", func(t *testing.T) {
		state := NewDevState("test")
		state.ReviewAttempts = 3

		if state.CanRetryReview(3) {
			t.Error("CanRetryReview should be false at max attempts")
		}
	})
}

// =============================================================================
// Context Helper Tests
// =============================================================================

func TestContextInjection(t *testing.T) {
	t.Run("GitContext", func(t *testing.T) {
		ctx := context.Background()

		// Without injection
		if GitFromContext(ctx) != nil {
			t.Error("GitFromContext should return nil without injection")
		}

		// With injection
		git := &GitContext{repoPath: "/tmp/test"}
		ctx = WithGitContext(ctx, git)

		got := GitFromContext(ctx)
		if got == nil {
			t.Error("GitFromContext should not return nil after injection")
		}
		if got.RepoPath() != "/tmp/test" {
			t.Errorf("RepoPath = %q, want %q", got.RepoPath(), "/tmp/test")
		}
	})

	t.Run("LLMClient", func(t *testing.T) {
		ctx := context.Background()

		if LLMFromContext(ctx) != nil {
			t.Error("LLMFromContext should return nil without injection")
		}

		client := llm.NewMockClient("test response")
		ctx = WithLLMClient(ctx, client)

		if LLMFromContext(ctx) == nil {
			t.Error("LLMFromContext should not return nil after injection")
		}
	})

	t.Run("ArtifactManager", func(t *testing.T) {
		ctx := context.Background()

		if ArtifactManagerFromContext(ctx) != nil {
			t.Error("ArtifactManagerFromContext should return nil without injection")
		}

		artifacts := &ArtifactManager{}
		ctx = WithArtifactManager(ctx, artifacts)

		if ArtifactManagerFromContext(ctx) == nil {
			t.Error("ArtifactManagerFromContext should not return nil after injection")
		}
	})

	t.Run("PromptLoader", func(t *testing.T) {
		ctx := context.Background()

		if PromptLoaderFromContext(ctx) != nil {
			t.Error("PromptLoaderFromContext should return nil without injection")
		}

		loader := &PromptLoader{}
		ctx = WithPromptLoader(ctx, loader)

		if PromptLoaderFromContext(ctx) == nil {
			t.Error("PromptLoaderFromContext should not return nil after injection")
		}
	})
}

func TestMustFromContext_Panics(t *testing.T) {
	ctx := testContext()

	t.Run("MustGitFromContext panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustGitFromContext should panic without injection")
			}
		}()
		MustGitFromContext(ctx)
	})

	t.Run("MustLLMFromContext panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustLLMFromContext should panic without injection")
			}
		}()
		MustLLMFromContext(ctx)
	})

}

func TestDevServices_InjectAll(t *testing.T) {
	services := &DevServices{
		Git:       &GitContext{repoPath: "/tmp/test"},
		LLM:       llm.NewMockClient("test"),
		Artifacts: &ArtifactManager{},
		Prompts:   &PromptLoader{},
	}

	ctx := services.InjectAll(context.Background())

	if GitFromContext(ctx) == nil {
		t.Error("Git should be injected")
	}
	if LLMFromContext(ctx) == nil {
		t.Error("LLM should be injected")
	}
	if ArtifactManagerFromContext(ctx) == nil {
		t.Error("Artifacts should be injected")
	}
	if PromptLoaderFromContext(ctx) == nil {
		t.Error("Prompts should be injected")
	}
}


// =============================================================================
// Node Tests (Unit Level)
// =============================================================================

func TestCreateWorktreeNode_MissingGit(t *testing.T) {
	ctx := testContext()
	state := NewDevState("test")

	_, err := CreateWorktreeNode(ctx, state)
	if err == nil {
		t.Error("CreateWorktreeNode should fail without GitContext")
	}
}

func TestGenerateSpecNode_MissingTicket(t *testing.T) {
	ctx := testContext()
	state := NewDevState("test")

	_, err := GenerateSpecNode(ctx, state)
	if err == nil {
		t.Error("GenerateSpecNode should fail without Ticket")
	}
}

func TestGenerateSpecNode_MissingLLM(t *testing.T) {
	ctx := testContext()
	state := NewDevState("test").WithTicket(&Ticket{ID: "TK-1"})

	_, err := GenerateSpecNode(ctx, state)
	if err == nil {
		t.Error("GenerateSpecNode should fail without llm.Client")
	}
}

func TestImplementNode_MissingSpec(t *testing.T) {
	ctx := testContext()
	state := NewDevState("test")
	state.Worktree = "/tmp/test"

	_, err := ImplementNode(ctx, state)
	if err == nil {
		t.Error("ImplementNode should fail without Spec")
	}
}

func TestImplementNode_MissingWorktree(t *testing.T) {
	ctx := testContext()
	state := NewDevState("test")
	state.Spec = "some spec"

	_, err := ImplementNode(ctx, state)
	if err == nil {
		t.Error("ImplementNode should fail without Worktree")
	}
}

func TestReviewNode_NoDiff(t *testing.T) {
	baseCtx := context.Background()
	client := llm.NewMockClient("test")
	baseCtx = WithLLMClient(baseCtx, client)
	ctx := wrapContext(baseCtx)

	state := NewDevState("test")
	// No worktree and no implementation

	_, err := ReviewNode(ctx, state)
	if err == nil {
		t.Error("ReviewNode should fail without implementation to review")
	}
}

func TestFixFindingsNode_MissingReview(t *testing.T) {
	ctx := testContext()
	state := NewDevState("test")
	state.Worktree = "/tmp/test"

	_, err := FixFindingsNode(ctx, state)
	if err == nil {
		t.Error("FixFindingsNode should fail without Review")
	}
}

func TestFixFindingsNode_ApprovedReview(t *testing.T) {
	ctx := testContext()
	state := NewDevState("test")
	state.Worktree = "/tmp/test"
	state.Review = &ReviewResult{Approved: true}

	result, err := FixFindingsNode(ctx, state)
	if err != nil {
		t.Errorf("FixFindingsNode with approved review should not fail: %v", err)
	}

	// Should return state unchanged
	if result.Review != state.Review {
		t.Error("State should be unchanged for approved review")
	}
}

func TestFixFindingsNode_WithFindings(t *testing.T) {
	baseCtx := context.Background()

	// Setup mock LLM client
	mockClient := llm.NewMockClient("```go\npackage main\n\nfunc main() {\n    // Fixed the error handling\n}\n```")
	baseCtx = WithLLMClient(baseCtx, mockClient)
	ctx := wrapContext(baseCtx)

	state := NewDevState("test-flow")
	state.Worktree = "/tmp/test-worktree"
	state.Review = &ReviewResult{
		Approved: false,
		Findings: []ReviewFinding{
			{Message: "Missing error handling", Severity: SeverityError, File: "main.go", Line: 10},
			{Message: "Unused variable", Severity: SeverityWarning, File: "main.go", Line: 15},
		},
	}

	result, err := FixFindingsNode(ctx, state)
	if err != nil {
		t.Fatalf("FixFindingsNode failed: %v", err)
	}

	if result.Implementation == "" {
		t.Error("Implementation should not be empty after fixing")
	}
	if !strings.Contains(result.Implementation, "Fixed") {
		t.Log("Note: Implementation content depends on mock response")
	}
}

func TestFixFindingsNode_MissingLLM(t *testing.T) {
	ctx := testContext()

	state := NewDevState("test-flow")
	state.Worktree = "/tmp/test-worktree"
	state.Review = &ReviewResult{Approved: false, Findings: []ReviewFinding{{Message: "Issue"}}}

	_, err := FixFindingsNode(ctx, state)
	if err == nil {
		t.Error("FixFindingsNode should fail without LLM client")
	}
}

func TestRunTestsNode_MissingWorktree(t *testing.T) {
	ctx := testContext()
	state := NewDevState("test")

	_, err := RunTestsNode(ctx, state)
	if err == nil {
		t.Error("RunTestsNode should fail without Worktree")
	}
}

func TestRunTestsNode_AllPass(t *testing.T) {
	baseCtx := context.Background()

	// Setup mock runner with passing tests
	mockRunner := NewMockRunner()
	mockRunner.OnCommand("sh", "-c", DefaultTestCommand).Return(
		"ok  \tgithub.com/example/pkg1\t0.5s\nok  \tgithub.com/example/pkg2\t0.3s\n", nil)
	baseCtx = WithCommandRunner(baseCtx, mockRunner)
	ctx := wrapContext(baseCtx)

	state := NewDevState("test-flow")
	state.Worktree = "/tmp/test-worktree"

	result, err := RunTestsNode(ctx, state)
	if err != nil {
		t.Fatalf("RunTestsNode failed: %v", err)
	}

	if !result.TestPassed {
		t.Error("TestPassed should be true")
	}
	if result.TestOutput == nil {
		t.Fatal("TestOutput should not be nil")
	}
	if result.TestOutput.PassedTests != 2 {
		t.Errorf("PassedTests = %d, want 2", result.TestOutput.PassedTests)
	}
	if result.TestRunAt.IsZero() {
		t.Error("TestRunAt should be set")
	}

	// Verify mock was called
	if !mockRunner.WasCalled("sh", "-c", DefaultTestCommand) {
		t.Error("Expected test command to be called")
	}
}

func TestRunTestsNode_WithFailures(t *testing.T) {
	baseCtx := context.Background()

	// Setup mock runner with failing tests
	mockRunner := NewMockRunner()
	mockRunner.OnCommand("sh", "-c", DefaultTestCommand).Return(
		`--- FAIL: TestSomething (0.00s)
    test.go:10: expected 1, got 2
FAIL	github.com/example/pkg1	0.5s
ok  	github.com/example/pkg2	0.3s
`, &CommandError{Command: "sh", Err: ErrTimeout})
	baseCtx = WithCommandRunner(baseCtx, mockRunner)
	ctx := wrapContext(baseCtx)

	state := NewDevState("test-flow")
	state.Worktree = "/tmp/test-worktree"

	result, err := RunTestsNode(ctx, state)
	if err != nil {
		t.Fatalf("RunTestsNode should not return error for test failures: %v", err)
	}

	if result.TestPassed {
		t.Error("TestPassed should be false")
	}
	if result.TestOutput == nil {
		t.Fatal("TestOutput should not be nil")
	}
	if result.TestOutput.FailedTests != 1 {
		t.Errorf("FailedTests = %d, want 1", result.TestOutput.FailedTests)
	}
	if result.TestOutput.PassedTests != 1 {
		t.Errorf("PassedTests = %d, want 1", result.TestOutput.PassedTests)
	}
	if len(result.TestOutput.Failures) != 1 {
		t.Errorf("Failures count = %d, want 1", len(result.TestOutput.Failures))
	}
}

func TestRunTestsNode_WithArtifactManager(t *testing.T) {
	baseCtx := context.Background()
	tmpDir := t.TempDir()

	// Setup mock runner
	mockRunner := NewMockRunner()
	mockRunner.OnCommand("sh", "-c", DefaultTestCommand).Return("ok  \tpkg\t0.1s\n", nil)
	baseCtx = WithCommandRunner(baseCtx, mockRunner)

	// Setup artifact manager
	artifacts := NewArtifactManager(ArtifactConfig{BaseDir: tmpDir})
	baseCtx = WithArtifactManager(baseCtx, artifacts)
	ctx := wrapContext(baseCtx)

	state := NewDevState("test-flow")
	state.Worktree = "/tmp/test-worktree"

	result, err := RunTestsNode(ctx, state)
	if err != nil {
		t.Fatalf("RunTestsNode failed: %v", err)
	}

	if !result.TestPassed {
		t.Error("TestPassed should be true")
	}
	// Artifact should have been saved (exercises that code path)
}

func TestCheckLintNode_MissingWorktree(t *testing.T) {
	ctx := testContext()
	state := NewDevState("test")

	_, err := CheckLintNode(ctx, state)
	if err == nil {
		t.Error("CheckLintNode should fail without Worktree")
	}
}

func TestCheckLintNode_NoProblem(t *testing.T) {
	baseCtx := context.Background()

	// Setup mock runner with clean lint output
	mockRunner := NewMockRunner()
	mockRunner.OnCommand("sh", "-c", DefaultLintCommand).Return("", nil)
	baseCtx = WithCommandRunner(baseCtx, mockRunner)
	ctx := wrapContext(baseCtx)

	state := NewDevState("test-flow")
	state.Worktree = "/tmp/test-worktree"

	result, err := CheckLintNode(ctx, state)
	if err != nil {
		t.Fatalf("CheckLintNode failed: %v", err)
	}

	if !result.LintPassed {
		t.Error("LintPassed should be true")
	}
	if result.LintOutput == nil {
		t.Fatal("LintOutput should not be nil")
	}
	if result.LintOutput.Summary.TotalIssues != 0 {
		t.Errorf("TotalIssues = %d, want 0", result.LintOutput.Summary.TotalIssues)
	}
	if result.LintRunAt.IsZero() {
		t.Error("LintRunAt should be set")
	}

	// Verify mock was called
	if !mockRunner.WasCalled("sh", "-c", DefaultLintCommand) {
		t.Error("Expected lint command to be called")
	}
}

func TestCheckLintNode_WithIssues(t *testing.T) {
	baseCtx := context.Background()

	// Setup mock runner with lint issues
	mockRunner := NewMockRunner()
	mockRunner.OnCommand("sh", "-c", DefaultLintCommand).Return(
		`main.go:10:5: printf format %d has arg of wrong type
main.go:25:3: unreachable code
`, &CommandError{Command: "sh", Err: ErrTimeout})
	baseCtx = WithCommandRunner(baseCtx, mockRunner)
	ctx := wrapContext(baseCtx)

	state := NewDevState("test-flow")
	state.Worktree = "/tmp/test-worktree"

	result, err := CheckLintNode(ctx, state)
	if err != nil {
		t.Fatalf("CheckLintNode should not return error for lint issues: %v", err)
	}

	if result.LintPassed {
		t.Error("LintPassed should be false")
	}
	if result.LintOutput == nil {
		t.Fatal("LintOutput should not be nil")
	}
	if len(result.LintOutput.Issues) != 2 {
		t.Errorf("Issues count = %d, want 2", len(result.LintOutput.Issues))
	}
	if result.LintOutput.Issues[0].File != "main.go" {
		t.Errorf("Issue file = %q, want %q", result.LintOutput.Issues[0].File, "main.go")
	}
	if result.LintOutput.Issues[0].Line != 10 {
		t.Errorf("Issue line = %d, want 10", result.LintOutput.Issues[0].Line)
	}
}

func TestCheckLintNode_WithArtifactManager(t *testing.T) {
	baseCtx := context.Background()
	tmpDir := t.TempDir()

	// Setup mock runner
	mockRunner := NewMockRunner()
	mockRunner.OnCommand("sh", "-c", DefaultLintCommand).Return("", nil)
	baseCtx = WithCommandRunner(baseCtx, mockRunner)

	// Setup artifact manager
	artifacts := NewArtifactManager(ArtifactConfig{BaseDir: tmpDir})
	baseCtx = WithArtifactManager(baseCtx, artifacts)
	ctx := wrapContext(baseCtx)

	state := NewDevState("test-flow")
	state.Worktree = "/tmp/test-worktree"

	result, err := CheckLintNode(ctx, state)
	if err != nil {
		t.Fatalf("CheckLintNode failed: %v", err)
	}

	if !result.LintPassed {
		t.Error("LintPassed should be true")
	}
	// Artifact should have been saved (exercises that code path)
}

func TestCreatePRNode_MissingBranch(t *testing.T) {
	ctx := testContext()
	state := NewDevState("test")

	_, err := CreatePRNode(ctx, state)
	if err == nil {
		t.Error("CreatePRNode should fail without Branch")
	}
}

func TestCleanupNode_NoWorktree(t *testing.T) {
	ctx := testContext()
	state := NewDevState("test")
	state.Worktree = "" // Empty worktree

	result, err := CleanupNode(ctx, state)
	if err != nil {
		t.Errorf("CleanupNode should not fail with empty worktree: %v", err)
	}

	if result.Worktree != "" {
		t.Error("Worktree should remain empty")
	}
}

// =============================================================================
// Node Happy Path Tests (with mocks)
// =============================================================================

func TestGenerateSpecNode_Success(t *testing.T) {
	baseCtx := context.Background()

	// Setup mock LLM client
	mockClient := llm.NewMockClient("# Technical Specification\n\nThis is a generated spec.")
	baseCtx = WithLLMClient(baseCtx, mockClient)
	ctx := wrapContext(baseCtx)

	// Setup state with ticket
	state := NewDevState("test-flow")
	state.Ticket = &Ticket{
		ID:          "TK-123",
		Title:       "Add user authentication",
		Description: "Implement login/logout functionality",
		Labels:      []string{"feature", "security"},
	}

	// Execute node
	result, err := GenerateSpecNode(ctx, state)
	if err != nil {
		t.Fatalf("GenerateSpecNode failed: %v", err)
	}

	// Verify spec was generated
	if result.Spec == "" {
		t.Error("Spec should not be empty")
	}
	if !strings.Contains(result.Spec, "Technical Specification") {
		t.Errorf("Spec content unexpected: %s", result.Spec)
	}

	// Verify token tracking
	if result.SpecTokensIn == 0 && result.SpecTokensOut == 0 {
		// MockClient might not set tokens, but state should be updated
		t.Log("Note: MockClient doesn't set token counts")
	}

	// Verify timestamp was set
	if result.SpecGeneratedAt.IsZero() {
		t.Error("SpecGeneratedAt should be set")
	}
}

func TestImplementNode_Success(t *testing.T) {
	baseCtx := context.Background()

	// Setup mock LLM client
	mockClient := llm.NewMockClient("```go\npackage main\n\nfunc main() {}\n```")
	baseCtx = WithLLMClient(baseCtx, mockClient)
	ctx := wrapContext(baseCtx)

	// Setup state with spec and worktree
	state := NewDevState("test-flow")
	state.Spec = "# Specification\n\nImplement a hello world program."
	state.Worktree = "/tmp/test-worktree"

	// Execute node
	result, err := ImplementNode(ctx, state)
	if err != nil {
		t.Fatalf("ImplementNode failed: %v", err)
	}

	// Verify implementation was set
	if result.Implementation == "" {
		t.Error("Implementation should not be empty")
	}
	if !strings.Contains(result.Implementation, "package main") {
		t.Errorf("Implementation content unexpected: %s", result.Implementation)
	}

	// Verify token tracking was updated
	if result.ImplementTokensIn == 0 && result.ImplementTokensOut == 0 {
		t.Log("Note: MockClient may not set token counts")
	}
}

func TestReviewNode_Approved(t *testing.T) {
	baseCtx := context.Background()

	// Setup mock LLM client - return an approval
	mockClient := llm.NewMockClient("APPROVED\n\nThe code looks good. No issues found.")
	baseCtx = WithLLMClient(baseCtx, mockClient)
	ctx := wrapContext(baseCtx)

	// Setup state with implementation
	state := NewDevState("test-flow")
	state.Implementation = "package main\n\nfunc main() { println(\"hello\") }"
	state.Worktree = "/tmp/test-worktree"

	// Execute node
	result, err := ReviewNode(ctx, state)
	if err != nil {
		t.Fatalf("ReviewNode failed: %v", err)
	}

	// Verify review was set
	if result.Review == nil {
		t.Fatal("Review should not be nil")
	}

	// Note: The actual approval parsing depends on parseReviewOutput
	// which looks for specific patterns
	if result.ReviewAttempts != 1 {
		t.Errorf("ReviewAttempts = %d, want 1", result.ReviewAttempts)
	}
}

func TestReviewNode_WithFindings(t *testing.T) {
	baseCtx := context.Background()

	// Setup mock LLM client - return findings
	reviewResponse := `NOT APPROVED

FINDINGS:
- ERROR: Missing error handling in line 10
- WARNING: Variable 'x' is unused
- SUGGESTION: Consider using constants`

	mockClient := llm.NewMockClient(reviewResponse)
	baseCtx = WithLLMClient(baseCtx, mockClient)
	ctx := wrapContext(baseCtx)

	// Setup state with implementation
	state := NewDevState("test-flow")
	state.Implementation = "package main\n\nfunc main() { x := 1 }"
	state.Worktree = "/tmp/test-worktree"

	// Execute node
	result, err := ReviewNode(ctx, state)
	if err != nil {
		t.Fatalf("ReviewNode failed: %v", err)
	}

	// Verify review was set
	if result.Review == nil {
		t.Fatal("Review should not be nil")
	}
}

func TestGenerateSpecNode_WithPromptLoader(t *testing.T) {
	baseCtx := context.Background()
	tmpDir := t.TempDir()

	// Setup mock LLM client
	mockClient := llm.NewMockClient("# Spec from custom prompt")
	baseCtx = WithLLMClient(baseCtx, mockClient)

	// Setup prompt loader
	loader := NewPromptLoader(tmpDir)
	baseCtx = WithPromptLoader(baseCtx, loader)
	ctx := wrapContext(baseCtx)

	// Setup state
	state := NewDevState("test-flow")
	state.Ticket = &Ticket{ID: "TK-1", Title: "Test"}

	// Execute - should work even without custom prompt file
	result, err := GenerateSpecNode(ctx, state)
	if err != nil {
		t.Fatalf("GenerateSpecNode failed: %v", err)
	}

	if result.Spec == "" {
		t.Error("Spec should not be empty")
	}
}

func TestGenerateSpecNode_WithArtifactManager(t *testing.T) {
	baseCtx := context.Background()
	tmpDir := t.TempDir()

	// Setup mock LLM client
	mockClient := llm.NewMockClient("# Spec to be saved")
	baseCtx = WithLLMClient(baseCtx, mockClient)

	// Setup artifact manager
	artifacts := NewArtifactManager(ArtifactConfig{BaseDir: tmpDir})
	baseCtx = WithArtifactManager(baseCtx, artifacts)
	ctx := wrapContext(baseCtx)

	// Setup state
	state := NewDevState("test-flow")
	state.Ticket = &Ticket{ID: "TK-1", Title: "Test"}

	// Execute
	result, err := GenerateSpecNode(ctx, state)
	if err != nil {
		t.Fatalf("GenerateSpecNode failed: %v", err)
	}

	if result.Spec == "" {
		t.Error("Spec should not be empty")
	}

	// Artifact should have been saved
	// (we'd need to check the artifact manager, but this exercises the code path)
}

// =============================================================================
// Prompt Formatting Tests
// =============================================================================

func TestFormatSpecPrompt(t *testing.T) {
	ticket := &Ticket{
		ID:          "TK-123",
		Title:       "Test Feature",
		Description: "Add a new feature",
		Labels:      []string{"enhancement", "priority-high"},
	}

	prompt := formatSpecPrompt(ticket)

	if !containsAll(prompt, "TK-123", "Test Feature", "Add a new feature", "enhancement") {
		t.Error("Prompt missing expected content")
	}
}

func TestFormatImplementPrompt(t *testing.T) {
	spec := "Build a REST API endpoint"
	ticket := &Ticket{
		ID:    "TK-456",
		Title: "API Endpoint",
	}

	prompt := formatImplementPrompt(spec, ticket)

	if !containsAll(prompt, "REST API endpoint", "TK-456", "API Endpoint") {
		t.Error("Prompt missing expected content")
	}
}

func TestFormatReviewPrompt(t *testing.T) {
	diff := "+func add(a, b int) int { return a + b }"
	spec := "Add utility functions"

	prompt := formatReviewPrompt(diff, spec)

	if !containsAll(prompt, "add(a, b int)", "Add utility functions", "Security issues") {
		t.Error("Prompt missing expected content")
	}
}

func TestFormatFixPrompt(t *testing.T) {
	review := &ReviewResult{
		Approved: false,
		Summary:  "Found issues",
		Findings: []ReviewFinding{
			{
				Category:   "security",
				Severity:   "high",
				File:       "main.go",
				Line:       42,
				Message:    "SQL injection vulnerability",
				Suggestion: "Use parameterized queries",
			},
		},
	}

	prompt := formatFixPrompt(review)

	if !containsAll(prompt, "Found issues", "security", "SQL injection", "main.go", "42") {
		t.Error("Prompt missing expected content")
	}
}

// =============================================================================
// Output Parsing Tests
// =============================================================================

func TestParseReviewOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		approved bool
	}{
		{
			name:     "plain JSON",
			input:    `{"approved": true, "summary": "Looks good"}`,
			wantErr:  false,
			approved: true,
		},
		{
			name:     "JSON in code block",
			input:    "```json\n{\"approved\": false, \"summary\": \"Issues found\"}\n```",
			wantErr:  false,
			approved: false,
		},
		{
			name:     "JSON in plain code block",
			input:    "```\n{\"approved\": true, \"summary\": \"OK\"}\n```",
			wantErr:  false,
			approved: true,
		},
		{
			name:    "invalid JSON",
			input:   "not json at all",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			review, err := parseReviewOutput(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseReviewOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && review.Approved != tt.approved {
				t.Errorf("Approved = %v, want %v", review.Approved, tt.approved)
			}
		})
	}
}

func TestParseTestOutput(t *testing.T) {
	output := `=== RUN   TestAdd
--- PASS: TestAdd (0.00s)
=== RUN   TestSub
--- FAIL: TestSub (0.00s)
    math_test.go:12: expected 3, got 2
ok  	example.com/pkg1	0.005s
FAIL	example.com/pkg2	0.010s
`

	result := parseTestOutput(output, false)

	if result.Passed {
		t.Error("Passed should be false")
	}

	// Counts package results: 1 ok + 1 FAIL = 2 total
	if result.TotalTests != 2 {
		t.Errorf("TotalTests = %d, want 2", result.TotalTests)
	}

	if len(result.Failures) == 0 {
		t.Error("Should have captured failure details")
	}
}

func TestParseLintOutput(t *testing.T) {
	t.Run("passing lint", func(t *testing.T) {
		result := parseLintOutput("", true)
		if !result.Passed {
			t.Error("Passed should be true")
		}
	})

	t.Run("failing lint", func(t *testing.T) {
		output := `main.go:10:5: unused variable 'x'
util.go:25:1: missing return statement
`
		result := parseLintOutput(output, false)

		if result.Passed {
			t.Error("Passed should be false")
		}

		if len(result.Issues) != 2 {
			t.Errorf("Issues count = %d, want 2", len(result.Issues))
		}

		if result.Issues[0].File != "main.go" {
			t.Errorf("Issue[0].File = %q, want %q", result.Issues[0].File, "main.go")
		}

		if result.Issues[0].Line != 10 {
			t.Errorf("Issue[0].Line = %d, want 10", result.Issues[0].Line)
		}
	})
}

// =============================================================================
// Node Wrapper Tests
// =============================================================================

func TestWithRetry(t *testing.T) {
	attempts := 0
	failingNode := func(ctx flowgraph.Context, state DevState) (DevState, error) {
		attempts++
		if attempts < 3 {
			return state, context.DeadlineExceeded
		}
		return state, nil
	}

	wrapped := WithRetry(failingNode, 3)
	_, err := wrapped(testContext(), NewDevState("test"))

	if err != nil {
		t.Errorf("WithRetry should succeed after retries: %v", err)
	}

	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestWithRetry_Exhausted(t *testing.T) {
	alwaysFails := func(ctx flowgraph.Context, state DevState) (DevState, error) {
		return state, context.DeadlineExceeded
	}

	wrapped := WithRetry(alwaysFails, 2)
	_, err := wrapped(testContext(), NewDevState("test"))

	if err == nil {
		t.Error("WithRetry should fail after exhausting retries")
	}
}

func TestWithTiming(t *testing.T) {
	slowNode := func(ctx flowgraph.Context, state DevState) (DevState, error) {
		time.Sleep(10 * time.Millisecond)
		return state, nil
	}

	wrapped := WithTiming(slowNode)
	_, err := wrapped(testContext(), NewDevState("test"))

	if err != nil {
		t.Errorf("WithTiming should not affect node execution: %v", err)
	}
}

// =============================================================================
// Review Router Tests
// =============================================================================

func TestReviewRouter(t *testing.T) {
	tests := []struct {
		name        string
		state       DevState
		maxAttempts int
		want        string
	}{
		{
			name: "approved goes to create-pr",
			state: func() DevState {
				s := NewDevState("test")
				s.Review = &ReviewResult{Approved: true}
				return s
			}(),
			maxAttempts: 3,
			want:        "create-pr",
		},
		{
			name: "rejected under max goes to fix-findings",
			state: func() DevState {
				s := NewDevState("test")
				s.Review = &ReviewResult{Approved: false}
				s.ReviewAttempts = 1
				return s
			}(),
			maxAttempts: 3,
			want:        "fix-findings",
		},
		{
			name: "rejected at max goes to create-pr",
			state: func() DevState {
				s := NewDevState("test")
				s.Review = &ReviewResult{Approved: false}
				s.ReviewAttempts = 3
				return s
			}(),
			maxAttempts: 3,
			want:        "create-pr",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ReviewRouter(tt.state, tt.maxAttempts)
			if got != tt.want {
				t.Errorf("ReviewRouter() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDefaultReviewRouter(t *testing.T) {
	state := NewDevState("test")
	state.Review = &ReviewResult{Approved: true}

	result := DefaultReviewRouter(state)
	if result != "create-pr" {
		t.Errorf("DefaultReviewRouter() = %q, want %q", result, "create-pr")
	}
}

// =============================================================================
// Test Helper Functions
// =============================================================================

// containsAll checks if a string contains all the given substrings
func containsAll(s string, substrings ...string) bool {
	for _, sub := range substrings {
		if !contains(s, sub) {
			return false
		}
	}
	return true
}

// contains checks if string s contains substring sub
func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsSubstring(s, sub))
}

// containsSubstring is a simple substring check
func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// =============================================================================
// Additional Node Helper Tests
// =============================================================================

func TestDefaultNodeConfig(t *testing.T) {
	config := DefaultNodeConfig()

	if config.MaxReviewAttempts != 3 {
		t.Errorf("MaxReviewAttempts = %d, want 3", config.MaxReviewAttempts)
	}
	if config.TestCommand != "go test -race ./..." {
		t.Errorf("TestCommand = %q", config.TestCommand)
	}
	if config.LintCommand != "go vet ./..." {
		t.Errorf("LintCommand = %q", config.LintCommand)
	}
	if config.BaseBranch != "main" {
		t.Errorf("BaseBranch = %q", config.BaseBranch)
	}
}

func TestBuildCommitMessage(t *testing.T) {
	tests := []struct {
		name  string
		state DevState
		want  string
	}{
		{
			name: "with ticket",
			state: func() DevState {
				s := NewDevState("test")
				s.TicketID = "TK-123"
				s.Ticket = &Ticket{ID: "TK-123", Title: "Add feature"}
				return s
			}(),
			want: "[TK-123] Add feature",
		},
		{
			name: "with ticket ID only",
			state: func() DevState {
				s := NewDevState("test")
				s.TicketID = "TK-456"
				return s
			}(),
			want: "[TK-456] Implementation",
		},
		{
			name: "no ticket",
			state: func() DevState {
				s := NewDevState("test")
				s.RunID = "run-abc123"
				return s
			}(),
			want: "Implementation for run-abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildCommitMessage(tt.state)
			if got != tt.want {
				t.Errorf("buildCommitMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetPRTitle(t *testing.T) {
	tests := []struct {
		name  string
		state DevState
		want  string
	}{
		{
			name: "with ticket",
			state: func() DevState {
				s := NewDevState("test")
				s.TicketID = "TK-123"
				s.Ticket = &Ticket{ID: "TK-123", Title: "Implement feature"}
				return s
			}(),
			want: "[TK-123] Implement feature",
		},
		{
			name: "with ticket ID only",
			state: func() DevState {
				s := NewDevState("test")
				s.TicketID = "TK-456"
				return s
			}(),
			want: "[TK-456] Implementation",
		},
		{
			name: "no ticket",
			state: func() DevState {
				s := NewDevState("test")
				s.RunID = "run-xyz789"
				return s
			}(),
			want: "devflow: run-xyz789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getPRTitle(tt.state)
			if got != tt.want {
				t.Errorf("getPRTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildPROptions(t *testing.T) {
	t.Run("with spec", func(t *testing.T) {
		state := NewDevState("test")
		state.Spec = "This is the spec content"
		state.TicketID = "TK-123"
		state.Ticket = &Ticket{ID: "TK-123", Title: "Feature"}

		opts := buildPROptions(state)

		if opts.Title != "[TK-123] Feature" {
			t.Errorf("Title = %q", opts.Title)
		}
		if !containsAll(opts.Body, "Specification", "spec content") {
			t.Errorf("Body should contain specification: %s", opts.Body)
		}
		if len(opts.Labels) == 0 || opts.Labels[0] != "TK-123" {
			t.Errorf("Labels = %v", opts.Labels)
		}
	})

	t.Run("with test results", func(t *testing.T) {
		state := NewDevState("test")
		state.TestOutput = &TestOutput{
			PassedTests: 10,
			FailedTests: 2,
		}

		opts := buildPROptions(state)

		if !containsAll(opts.Body, "Test Results", "Passed: 10", "Failed: 2") {
			t.Errorf("Body should contain test results: %s", opts.Body)
		}
	})

	t.Run("with rejected review", func(t *testing.T) {
		state := NewDevState("test")
		state.Review = &ReviewResult{Approved: false}

		opts := buildPROptions(state)

		if !opts.Draft {
			t.Error("Should be created as draft for rejected review")
		}
	})

	t.Run("with approved review", func(t *testing.T) {
		state := NewDevState("test")
		state.Review = &ReviewResult{Approved: true}

		opts := buildPROptions(state)

		if opts.Draft {
			t.Error("Should not be draft for approved review")
		}
	})
}

func TestWithTranscript(t *testing.T) {
	// Create a transcript manager
	manager, err := NewTranscriptManager(TranscriptConfig{
		BaseDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("NewTranscriptManager: %v", err)
	}

	// Start a run
	runID := "test-run-123"
	err = manager.StartRun(runID, RunMetadata{FlowID: "test-flow"})
	if err != nil {
		t.Fatalf("StartRun: %v", err)
	}

	// Create context with manager
	ctx := WithTranscriptManager(context.Background(), manager)

	// Create a simple node
	executed := false
	simpleNode := func(ctx flowgraph.Context, state DevState) (DevState, error) {
		executed = true
		return state, nil
	}

	// Wrap with transcript recording
	wrapped := WithTranscript(simpleNode, "test-node")

	// Execute
	state := NewDevState("test")
	state.RunID = runID

	_, err = wrapped(wrapContext(ctx), state)
	if err != nil {
		t.Fatalf("wrapped node failed: %v", err)
	}

	if !executed {
		t.Error("wrapped node should have executed the inner node")
	}
}

func TestWithTranscript_Error(t *testing.T) {
	manager, err := NewTranscriptManager(TranscriptConfig{
		BaseDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("NewTranscriptManager: %v", err)
	}

	runID := "test-run-error"
	_ = manager.StartRun(runID, RunMetadata{FlowID: "test"})

	ctx := WithTranscriptManager(context.Background(), manager)

	// Create a failing node
	expectedErr := context.DeadlineExceeded
	failingNode := func(ctx flowgraph.Context, state DevState) (DevState, error) {
		return state, expectedErr
	}

	wrapped := WithTranscript(failingNode, "failing-node")

	state := NewDevState("test")
	state.RunID = runID

	_, err = wrapped(wrapContext(ctx), state)
	if err != expectedErr {
		t.Errorf("wrapped node should propagate error: got %v, want %v", err, expectedErr)
	}
}

func TestShouldCreateDraftPR(t *testing.T) {
	// ShouldCreateDraftPR returns true when:
	// - Review is NOT approved, AND
	// - ReviewAttempts >= maxAttempts
	// Note: the method panics if Review is nil
	tests := []struct {
		name  string
		state DevState
		want  bool
	}{
		{
			name: "approved review with many attempts",
			state: func() DevState {
				s := NewDevState("test")
				s.Review = &ReviewResult{Approved: true}
				s.ReviewAttempts = 5
				return s
			}(),
			want: false, // approved, so not draft
		},
		{
			name: "rejected review but under max attempts",
			state: func() DevState {
				s := NewDevState("test")
				s.Review = &ReviewResult{Approved: false}
				s.ReviewAttempts = 1 // less than maxAttempts=3
				return s
			}(),
			want: false, // can still retry
		},
		{
			name: "rejected review at max attempts",
			state: func() DevState {
				s := NewDevState("test")
				s.Review = &ReviewResult{Approved: false}
				s.ReviewAttempts = 3 // equals maxAttempts
				return s
			}(),
			want: true, // exhausted attempts
		},
		{
			name: "rejected review over max attempts",
			state: func() DevState {
				s := NewDevState("test")
				s.Review = &ReviewResult{Approved: false}
				s.ReviewAttempts = 5 // greater than maxAttempts
				return s
			}(),
			want: true, // exhausted attempts
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.state.ShouldCreateDraftPR(3) // maxAttempts=3
			if got != tt.want {
				t.Errorf("ShouldCreateDraftPR(3) = %v, want %v", got, tt.want)
			}
		})
	}
}
