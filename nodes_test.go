package devflow

import (
	"context"
	"testing"
	"time"
)

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

	t.Run("ClaudeCLI", func(t *testing.T) {
		ctx := context.Background()

		if ClaudeFromContext(ctx) != nil {
			t.Error("ClaudeFromContext should return nil without injection")
		}

		claude := &ClaudeCLI{}
		ctx = WithClaudeCLI(ctx, claude)

		if ClaudeFromContext(ctx) == nil {
			t.Error("ClaudeFromContext should not return nil after injection")
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
	ctx := context.Background()

	t.Run("MustGitFromContext panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustGitFromContext should panic without injection")
			}
		}()
		MustGitFromContext(ctx)
	})

	t.Run("MustClaudeFromContext panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustClaudeFromContext should panic without injection")
			}
		}()
		MustClaudeFromContext(ctx)
	})
}

func TestDevServices_InjectAll(t *testing.T) {
	services := &DevServices{
		Git:       &GitContext{repoPath: "/tmp/test"},
		Claude:    &ClaudeCLI{},
		Artifacts: &ArtifactManager{},
		Prompts:   &PromptLoader{},
	}

	ctx := services.InjectAll(context.Background())

	if GitFromContext(ctx) == nil {
		t.Error("Git should be injected")
	}
	if ClaudeFromContext(ctx) == nil {
		t.Error("Claude should be injected")
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
	ctx := context.Background()
	state := NewDevState("test")

	_, err := CreateWorktreeNode(ctx, state)
	if err == nil {
		t.Error("CreateWorktreeNode should fail without GitContext")
	}
}

func TestGenerateSpecNode_MissingTicket(t *testing.T) {
	ctx := context.Background()
	state := NewDevState("test")

	_, err := GenerateSpecNode(ctx, state)
	if err == nil {
		t.Error("GenerateSpecNode should fail without Ticket")
	}
}

func TestGenerateSpecNode_MissingClaude(t *testing.T) {
	ctx := context.Background()
	state := NewDevState("test").WithTicket(&Ticket{ID: "TK-1"})

	_, err := GenerateSpecNode(ctx, state)
	if err == nil {
		t.Error("GenerateSpecNode should fail without ClaudeCLI")
	}
}

func TestImplementNode_MissingSpec(t *testing.T) {
	ctx := context.Background()
	state := NewDevState("test")
	state.Worktree = "/tmp/test"

	_, err := ImplementNode(ctx, state)
	if err == nil {
		t.Error("ImplementNode should fail without Spec")
	}
}

func TestImplementNode_MissingWorktree(t *testing.T) {
	ctx := context.Background()
	state := NewDevState("test")
	state.Spec = "some spec"

	_, err := ImplementNode(ctx, state)
	if err == nil {
		t.Error("ImplementNode should fail without Worktree")
	}
}

func TestReviewNode_NoDiff(t *testing.T) {
	ctx := context.Background()
	claude := &ClaudeCLI{}
	ctx = WithClaudeCLI(ctx, claude)

	state := NewDevState("test")
	// No worktree and no implementation

	_, err := ReviewNode(ctx, state)
	if err == nil {
		t.Error("ReviewNode should fail without implementation to review")
	}
}

func TestFixFindingsNode_MissingReview(t *testing.T) {
	ctx := context.Background()
	state := NewDevState("test")
	state.Worktree = "/tmp/test"

	_, err := FixFindingsNode(ctx, state)
	if err == nil {
		t.Error("FixFindingsNode should fail without Review")
	}
}

func TestFixFindingsNode_ApprovedReview(t *testing.T) {
	ctx := context.Background()
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

func TestRunTestsNode_MissingWorktree(t *testing.T) {
	ctx := context.Background()
	state := NewDevState("test")

	_, err := RunTestsNode(ctx, state)
	if err == nil {
		t.Error("RunTestsNode should fail without Worktree")
	}
}

func TestCheckLintNode_MissingWorktree(t *testing.T) {
	ctx := context.Background()
	state := NewDevState("test")

	_, err := CheckLintNode(ctx, state)
	if err == nil {
		t.Error("CheckLintNode should fail without Worktree")
	}
}

func TestCreatePRNode_MissingBranch(t *testing.T) {
	ctx := context.Background()
	state := NewDevState("test")

	_, err := CreatePRNode(ctx, state)
	if err == nil {
		t.Error("CreatePRNode should fail without Branch")
	}
}

func TestCleanupNode_NoWorktree(t *testing.T) {
	ctx := context.Background()
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
			name: "plain JSON",
			input: `{"approved": true, "summary": "Looks good"}`,
			wantErr:  false,
			approved: true,
		},
		{
			name: "JSON in code block",
			input: "```json\n{\"approved\": false, \"summary\": \"Issues found\"}\n```",
			wantErr:  false,
			approved: false,
		},
		{
			name: "JSON in plain code block",
			input: "```\n{\"approved\": true, \"summary\": \"OK\"}\n```",
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
	failingNode := func(ctx context.Context, state DevState) (DevState, error) {
		attempts++
		if attempts < 3 {
			return state, context.DeadlineExceeded
		}
		return state, nil
	}

	wrapped := WithRetry(failingNode, 3)
	_, err := wrapped(context.Background(), NewDevState("test"))

	if err != nil {
		t.Errorf("WithRetry should succeed after retries: %v", err)
	}

	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestWithRetry_Exhausted(t *testing.T) {
	alwaysFails := func(ctx context.Context, state DevState) (DevState, error) {
		return state, context.DeadlineExceeded
	}

	wrapped := WithRetry(alwaysFails, 2)
	_, err := wrapped(context.Background(), NewDevState("test"))

	if err == nil {
		t.Error("WithRetry should fail after exhausting retries")
	}
}

func TestWithTiming(t *testing.T) {
	slowNode := func(ctx context.Context, state DevState) (DevState, error) {
		time.Sleep(10 * time.Millisecond)
		return state, nil
	}

	wrapped := WithTiming(slowNode)
	_, err := wrapped(context.Background(), NewDevState("test"))

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

// Helper functions containsAll and contains are defined in claude_test.go
