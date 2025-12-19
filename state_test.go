package devflow

import (
	"strings"
	"testing"
	"time"
)

// Tests for state.go functionality not already covered in nodes_test.go

func TestDevState_WithRunID(t *testing.T) {
	state := NewDevState("test-flow")
	state = state.WithRunID("custom-run-id")

	if state.RunID != "custom-run-id" {
		t.Errorf("RunID = %q, want %q", state.RunID, "custom-run-id")
	}
}

func TestDevState_WithTicketID(t *testing.T) {
	state := NewDevState("test-flow")
	state = state.WithTicketID("TK-456")

	if state.TicketID != "TK-456" {
		t.Errorf("TicketID = %q, want %q", state.TicketID, "TK-456")
	}
	if state.Ticket != nil {
		t.Error("Ticket should be nil when only ID is set")
	}
}

func TestDevState_WithBaseBranch(t *testing.T) {
	state := NewDevState("test-flow")
	state = state.WithBaseBranch("develop")

	if state.BaseBranch != "develop" {
		t.Errorf("BaseBranch = %q, want %q", state.BaseBranch, "develop")
	}
}

func TestDevState_AddTokensWithCost(t *testing.T) {
	state := NewDevState("test-flow")

	state.AddTokensWithCost(1000, 500, 0.05)

	if state.TotalTokensIn != 1000 {
		t.Errorf("TotalTokensIn = %d, want %d", state.TotalTokensIn, 1000)
	}
	if state.TotalCost != 0.05 {
		t.Errorf("TotalCost = %f, want %f", state.TotalCost, 0.05)
	}

	// Test cumulative
	state.AddTokensWithCost(500, 250, 0.025)
	// Use tolerance for float comparison
	expectedCost := 0.075
	if state.TotalCost < expectedCost-0.0001 || state.TotalCost > expectedCost+0.0001 {
		t.Errorf("TotalCost after second add = %f, want %f", state.TotalCost, expectedCost)
	}
}

func TestDevState_FinalizeDuration(t *testing.T) {
	state := NewDevState("test-flow")

	// Wait a small amount of time
	time.Sleep(10 * time.Millisecond)

	state.FinalizeDuration()

	if state.TotalDuration < 10*time.Millisecond {
		t.Errorf("TotalDuration = %v, want >= 10ms", state.TotalDuration)
	}
}

func TestDevState_SetError(t *testing.T) {
	state := NewDevState("test-flow")

	// Test with nil error
	state.SetError(nil)
	if state.Error != "" {
		t.Errorf("Error = %q, want empty", state.Error)
	}

	// Test with actual error
	state.SetError(ErrWorktreeExists)
	if state.Error != ErrWorktreeExists.Error() {
		t.Errorf("Error = %q, want %q", state.Error, ErrWorktreeExists.Error())
	}
}

func TestDevState_HasError(t *testing.T) {
	state := NewDevState("test-flow")

	if state.HasError() {
		t.Error("HasError() = true, want false for new state")
	}

	state.Error = "something went wrong"
	if !state.HasError() {
		t.Error("HasError() = false, want true after setting error")
	}
}

func TestDevState_ValidateStrings(t *testing.T) {
	state := NewDevState("test-flow")
	state.Ticket = &Ticket{ID: "TK-123"}
	state.Worktree = "/path/to/worktree"

	// Should pass with string requirements
	err := state.ValidateStrings("ticket", "worktree")
	if err != nil {
		t.Errorf("ValidateStrings() error = %v, want nil", err)
	}

	// Should fail for missing requirement
	err = state.ValidateStrings("spec")
	if err == nil {
		t.Error("ValidateStrings() should fail for missing spec")
	}
}

func TestDevState_NeedsReviewFix_WithFindings(t *testing.T) {
	state := NewDevState("test-flow")
	state.Review = &ReviewResult{
		Approved: false,
		Findings: []ReviewFinding{{Message: "Issue found", Severity: SeverityError}},
	}

	if !state.NeedsReviewFix() {
		t.Error("NeedsReviewFix() = false, want true with findings")
	}
}

func TestGenerateRunID(t *testing.T) {
	runID := generateRunID("test-flow")

	if runID == "" {
		t.Error("generateRunID() returned empty string")
	}

	if !strings.Contains(runID, "test-flow") {
		t.Errorf("runID %q should contain flow ID", runID)
	}

	// Should contain date format
	today := time.Now().Format("2006-01-02")
	if !strings.Contains(runID, today) {
		t.Errorf("runID %q should contain today's date %q", runID, today)
	}

	// Generate another and check uniqueness
	runID2 := generateRunID("test-flow")
	if runID == runID2 {
		t.Error("two consecutive generateRunID() calls should produce different IDs")
	}
}

func TestRandomSuffix(t *testing.T) {
	suffix := randomSuffix(4)

	if len(suffix) != 8 {
		t.Errorf("randomSuffix(4) = %q, want 8 hex chars", suffix)
	}

	// Check it's valid hex
	for _, c := range suffix {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("randomSuffix() contains non-hex char: %c", c)
		}
	}

	// Check uniqueness
	suffix2 := randomSuffix(4)
	if suffix == suffix2 {
		t.Error("two consecutive randomSuffix() calls should produce different values")
	}
}

func TestTicketStruct(t *testing.T) {
	ticket := Ticket{
		ID:          "TK-123",
		Title:       "Test Ticket",
		Description: "Test description",
		Priority:    "high",
		Type:        "bug",
		Labels:      []string{"urgent", "backend"},
		Assignee:    "john",
		Reporter:    "jane",
		URL:         "https://jira.example.com/TK-123",
		Metadata:    map[string]string{"sprint": "42"},
	}

	if ticket.ID != "TK-123" {
		t.Errorf("Ticket.ID = %q, want %q", ticket.ID, "TK-123")
	}
	if len(ticket.Labels) != 2 {
		t.Errorf("Ticket.Labels = %v, want 2 labels", ticket.Labels)
	}
	if ticket.Metadata["sprint"] != "42" {
		t.Errorf("Ticket.Metadata[sprint] = %q, want %q", ticket.Metadata["sprint"], "42")
	}
}

func TestGitState(t *testing.T) {
	state := GitState{
		Worktree:   "/tmp/worktree",
		Branch:     "feature/test",
		BaseBranch: "main",
	}

	if state.Worktree != "/tmp/worktree" {
		t.Errorf("Worktree = %q, want %q", state.Worktree, "/tmp/worktree")
	}
}

func TestSpecState(t *testing.T) {
	now := time.Now()
	state := SpecState{
		Spec:            "# Specification",
		SpecTokensIn:    100,
		SpecTokensOut:   50,
		SpecGeneratedAt: now,
	}

	if state.Spec != "# Specification" {
		t.Errorf("Spec = %q, want %q", state.Spec, "# Specification")
	}
	if state.SpecTokensIn != 100 {
		t.Errorf("SpecTokensIn = %d, want 100", state.SpecTokensIn)
	}
}

func TestImplementState(t *testing.T) {
	state := ImplementState{
		Implementation:     "code here",
		Files:              []FileChange{{Path: "main.go", Operation: "create"}},
		ImplementTokensIn:  200,
		ImplementTokensOut: 100,
	}

	if len(state.Files) != 1 {
		t.Errorf("Files count = %d, want 1", len(state.Files))
	}
}

func TestReviewState(t *testing.T) {
	state := ReviewState{
		Review:          &ReviewResult{Approved: true},
		ReviewAttempts:  2,
		ReviewTokensIn:  150,
		ReviewTokensOut: 75,
	}

	if state.ReviewAttempts != 2 {
		t.Errorf("ReviewAttempts = %d, want 2", state.ReviewAttempts)
	}
}

func TestPullRequestState(t *testing.T) {
	now := time.Now()
	state := PullRequestState{
		PR:        &PullRequest{URL: "https://github.com/test/pr/1"},
		PRCreated: now,
	}

	if state.PR.URL != "https://github.com/test/pr/1" {
		t.Errorf("PR.URL = %q, want %q", state.PR.URL, "https://github.com/test/pr/1")
	}
}

func TestTestState(t *testing.T) {
	now := time.Now()
	state := TestState{
		TestOutput: &TestOutput{Passed: true},
		TestPassed: true,
		TestRunAt:  now,
	}

	if !state.TestPassed {
		t.Error("TestPassed should be true")
	}
}

func TestLintState(t *testing.T) {
	now := time.Now()
	state := LintState{
		LintOutput: &LintOutput{Passed: true},
		LintPassed: true,
		LintRunAt:  now,
	}

	if !state.LintPassed {
		t.Error("LintPassed should be true")
	}
}

func TestMetricsState(t *testing.T) {
	state := MetricsState{
		TotalTokensIn:  5000,
		TotalTokensOut: 2500,
		TotalCost:      0.50,
		StartTime:      time.Now(),
		TotalDuration:  30 * time.Second,
	}

	if state.TotalTokensIn != 5000 {
		t.Errorf("TotalTokensIn = %d, want 5000", state.TotalTokensIn)
	}
	if state.TotalDuration != 30*time.Second {
		t.Errorf("TotalDuration = %v, want 30s", state.TotalDuration)
	}
}

func TestStateRequirementConstants(t *testing.T) {
	requirements := []StateRequirement{
		RequireTicket,
		RequireWorktree,
		RequireSpec,
		RequireImplementation,
		RequireReview,
		RequireBranch,
		RequireFiles,
	}

	// Ensure all are non-empty strings
	for _, req := range requirements {
		if req == "" {
			t.Errorf("StateRequirement constant should not be empty")
		}
	}

	// Ensure they're all unique
	seen := make(map[StateRequirement]bool)
	for _, req := range requirements {
		if seen[req] {
			t.Errorf("Duplicate StateRequirement: %s", req)
		}
		seen[req] = true
	}
}

func TestDevState_CanRetryReview(t *testing.T) {
	tests := []struct {
		name     string
		state    DevState
		maxRetry int
		want     bool
	}{
		{
			name: "can retry when under limit",
			state: func() DevState {
				s := NewDevState("test-flow")
				s.ReviewAttempts = 1
				return s
			}(),
			maxRetry: 3,
			want:     true,
		},
		{
			name: "cannot retry at limit",
			state: func() DevState {
				s := NewDevState("test-flow")
				s.ReviewAttempts = 3
				return s
			}(),
			maxRetry: 3,
			want:     false,
		},
		{
			name: "cannot retry over limit",
			state: func() DevState {
				s := NewDevState("test-flow")
				s.ReviewAttempts = 5
				return s
			}(),
			maxRetry: 3,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.state.CanRetryReview(tt.maxRetry); got != tt.want {
				t.Errorf("CanRetryReview(%d) = %v, want %v", tt.maxRetry, got, tt.want)
			}
		})
	}
}
