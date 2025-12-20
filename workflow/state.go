package workflow

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/randalmurphal/devflow/artifact"
	"github.com/randalmurphal/devflow/pr"
)

// =============================================================================
// Ticket Type
// =============================================================================

// Ticket represents input ticket data from an issue tracker
type Ticket struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Priority    string            `json:"priority,omitempty"`
	Type        string            `json:"type,omitempty"` // bug, feature, task, etc.
	Labels      []string          `json:"labels,omitempty"`
	Assignee    string            `json:"assignee,omitempty"`
	Reporter    string            `json:"reporter,omitempty"`
	URL         string            `json:"url,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// =============================================================================
// Embeddable State Components
// =============================================================================

// GitState tracks git workspace state
type GitState struct {
	Worktree   string `json:"worktree,omitempty"`
	Branch     string `json:"branch,omitempty"`
	BaseBranch string `json:"baseBranch,omitempty"`
}

// SpecState tracks specification generation
type SpecState struct {
	Spec            string    `json:"spec,omitempty"`
	SpecTokensIn    int       `json:"specTokensIn,omitempty"`
	SpecTokensOut   int       `json:"specTokensOut,omitempty"`
	SpecGeneratedAt time.Time `json:"specGeneratedAt,omitempty"`
}

// FileChange represents a file modification during implementation
type FileChange struct {
	Path      string `json:"path"`
	Operation string `json:"operation"` // "create", "modify", "delete"
	Content   string `json:"content,omitempty"`
}

// ImplementState tracks implementation progress
type ImplementState struct {
	Implementation     string       `json:"implementation,omitempty"`
	Files              []FileChange `json:"files,omitempty"`
	ImplementTokensIn  int          `json:"implementTokensIn,omitempty"`
	ImplementTokensOut int          `json:"implementTokensOut,omitempty"`
}

// ReviewState tracks code review
type ReviewState struct {
	Review          *artifact.ReviewResult `json:"review,omitempty"`
	ReviewAttempts  int                    `json:"reviewAttempts,omitempty"`
	ReviewTokensIn  int                    `json:"reviewTokensIn,omitempty"`
	ReviewTokensOut int                    `json:"reviewTokensOut,omitempty"`
}

// PullRequestState tracks pull request creation
// Named to avoid collision with pr.State (open/closed/merged)
type PullRequestState struct {
	PR        *pr.PullRequest `json:"pr,omitempty"`
	PRCreated time.Time       `json:"prCreated,omitempty"`
}

// TestState tracks test execution
type TestState struct {
	TestOutput *artifact.TestOutput `json:"testOutput,omitempty"`
	TestPassed bool                 `json:"testPassed,omitempty"`
	TestRunAt  time.Time            `json:"testRunAt,omitempty"`
}

// LintState tracks lint/type check execution
type LintState struct {
	LintOutput *artifact.LintOutput `json:"lintOutput,omitempty"`
	LintPassed bool                 `json:"lintPassed,omitempty"`
	LintRunAt  time.Time            `json:"lintRunAt,omitempty"`
}

// MetricsState tracks execution metrics
type MetricsState struct {
	TotalTokensIn  int           `json:"totalTokensIn"`
	TotalTokensOut int           `json:"totalTokensOut"`
	TotalCost      float64       `json:"totalCost"`
	StartTime      time.Time     `json:"startTime"`
	TotalDuration  time.Duration `json:"totalDuration"`
}

// =============================================================================
// State - Full Workflow State
// =============================================================================

// State is the complete state for dev workflows
type State struct {
	// Identification
	RunID    string `json:"runId"`
	FlowID   string `json:"flowId"`
	TicketID string `json:"ticketId,omitempty"`

	// Input
	Ticket *Ticket `json:"ticket,omitempty"`

	// Embedded state components
	GitState
	SpecState
	ImplementState
	ReviewState
	PullRequestState
	TestState
	LintState
	MetricsState

	// Error tracking
	Error string `json:"error,omitempty"`
}

// NewState creates a new dev workflow state
func NewState(flowID string) State {
	return State{
		RunID:  generateRunID(flowID),
		FlowID: flowID,
		MetricsState: MetricsState{
			StartTime: time.Now(),
		},
	}
}

// WithRunID sets a custom run ID
func (s State) WithRunID(runID string) State {
	s.RunID = runID
	return s
}

// WithTicket adds ticket information to state
func (s State) WithTicket(ticket *Ticket) State {
	s.TicketID = ticket.ID
	s.Ticket = ticket
	return s
}

// WithTicketID sets just the ticket ID (for when full ticket isn't needed)
func (s State) WithTicketID(ticketID string) State {
	s.TicketID = ticketID
	return s
}

// WithBaseBranch sets the base branch for the workflow
func (s State) WithBaseBranch(branch string) State {
	s.BaseBranch = branch
	return s
}

// AddTokens updates token metrics
func (s *State) AddTokens(in, out int) {
	s.TotalTokensIn += in
	s.TotalTokensOut += out
	// Rough cost estimate ($3/1M in, $15/1M out for Claude Opus)
	s.TotalCost += (float64(in) * 0.000003) + (float64(out) * 0.000015)
}

// AddTokensWithCost updates token metrics with explicit cost
func (s *State) AddTokensWithCost(in, out int, cost float64) {
	s.TotalTokensIn += in
	s.TotalTokensOut += out
	s.TotalCost += cost
}

// FinalizeDuration sets total duration from start time
func (s *State) FinalizeDuration() {
	s.TotalDuration = time.Since(s.StartTime)
}

// SetError sets the error state
func (s *State) SetError(err error) {
	if err != nil {
		s.Error = err.Error()
	}
}

// HasError returns true if state has an error
func (s State) HasError() bool {
	return s.Error != ""
}

// =============================================================================
// State Validation
// =============================================================================

// StateRequirement defines a state prerequisite
type StateRequirement string

const (
	RequireTicket         StateRequirement = "ticket"
	RequireWorktree       StateRequirement = "worktree"
	RequireSpec           StateRequirement = "spec"
	RequireImplementation StateRequirement = "implementation"
	RequireReview         StateRequirement = "review"
	RequireBranch         StateRequirement = "branch"
	RequireFiles          StateRequirement = "files"
)

// Validate checks if state has required fields
func (s State) Validate(requirements ...StateRequirement) error {
	for _, req := range requirements {
		switch req {
		case RequireTicket:
			if s.Ticket == nil {
				return fmt.Errorf("ticket required")
			}
		case RequireWorktree:
			if s.Worktree == "" {
				return fmt.Errorf("worktree required")
			}
		case RequireSpec:
			if s.Spec == "" {
				return fmt.Errorf("spec required")
			}
		case RequireImplementation:
			if s.Implementation == "" {
				return fmt.Errorf("implementation required")
			}
		case RequireReview:
			if s.Review == nil {
				return fmt.Errorf("review required")
			}
		case RequireBranch:
			if s.Branch == "" {
				return fmt.Errorf("branch required")
			}
		case RequireFiles:
			if len(s.Files) == 0 {
				return fmt.Errorf("files required")
			}
		default:
			return fmt.Errorf("unknown requirement: %s", req)
		}
	}
	return nil
}

// ValidateStrings validates using string requirements (for flexibility)
func (s State) ValidateStrings(requirements ...string) error {
	reqs := make([]StateRequirement, len(requirements))
	for i, r := range requirements {
		reqs[i] = StateRequirement(r)
	}
	return s.Validate(reqs...)
}

// =============================================================================
// Review Routing
// =============================================================================

// NeedsReviewFix returns true if review found issues that need fixing
func (s State) NeedsReviewFix() bool {
	if s.Review == nil {
		return false
	}
	return !s.Review.Approved
}

// CanRetryReview returns true if we haven't exceeded review attempts
func (s State) CanRetryReview(maxAttempts int) bool {
	return s.ReviewAttempts < maxAttempts
}

// ShouldCreateDraftPR returns true if we should create a draft PR
// (review found issues but we've hit max attempts)
func (s State) ShouldCreateDraftPR(maxAttempts int) bool {
	return !s.Review.Approved && s.ReviewAttempts >= maxAttempts
}

// =============================================================================
// Helper Functions
// =============================================================================

// generateRunID creates a unique run ID
func generateRunID(flowID string) string {
	timestamp := time.Now().Format("2006-01-02")
	suffix := randomSuffix(4)
	return fmt.Sprintf("%s-%s-%s", timestamp, flowID, suffix)
}

// randomSuffix generates a random hex suffix
func randomSuffix(bytes int) string {
	b := make([]byte, bytes)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based suffix on entropy failure
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// =============================================================================
// State Summary
// =============================================================================

// Summary returns a human-readable summary of the state
func (s State) Summary() string {
	var status string
	switch {
	case s.Error != "":
		status = "failed"
	case s.PR != nil:
		status = "completed"
	case s.Review != nil && s.Review.Approved:
		status = "reviewed"
	case s.Implementation != "":
		status = "implemented"
	case s.Spec != "":
		status = "specified"
	case s.Worktree != "":
		status = "started"
	default:
		status = "pending"
	}

	return fmt.Sprintf("Run %s [%s]: %s (tokens: %d in, %d out, cost: $%.4f)",
		s.RunID, status, s.FlowID,
		s.TotalTokensIn, s.TotalTokensOut, s.TotalCost)
}
