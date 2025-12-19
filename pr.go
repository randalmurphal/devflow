package devflow

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// PRState represents the state of a pull request.
type PRState string

const (
	PRStateOpen   PRState = "open"
	PRStateClosed PRState = "closed"
	PRStateMerged PRState = "merged"
)

// PRProvider is the interface for creating and managing pull requests.
// Implementations exist for GitHub and GitLab.
type PRProvider interface {
	// CreatePR creates a new pull request.
	CreatePR(ctx context.Context, opts PROptions) (*PullRequest, error)

	// GetPR retrieves a pull request by ID.
	GetPR(ctx context.Context, id int) (*PullRequest, error)

	// UpdatePR updates an existing pull request.
	UpdatePR(ctx context.Context, id int, opts PRUpdateOptions) (*PullRequest, error)

	// MergePR merges a pull request.
	MergePR(ctx context.Context, id int, opts MergeOptions) error

	// AddComment adds a comment to a pull request.
	AddComment(ctx context.Context, id int, body string) error

	// RequestReview requests review from the specified users.
	RequestReview(ctx context.Context, id int, reviewers []string) error

	// ListPRs lists pull requests matching the filter.
	ListPRs(ctx context.Context, filter PRFilter) ([]*PullRequest, error)
}

// PROptions configures pull request creation.
type PROptions struct {
	Title     string            // PR title (required)
	Body      string            // PR description (markdown)
	Base      string            // Target branch (default: "main")
	Head      string            // Source branch (auto-detected if empty)
	Labels    []string          // Labels to apply
	Reviewers []string          // Reviewer usernames
	Assignees []string          // Assignee usernames
	Draft     bool              // Create as draft
	Milestone string            // Milestone name or ID
	Metadata  map[string]string // Additional metadata
}

// PRUpdateOptions configures pull request updates.
type PRUpdateOptions struct {
	Title     *string  // New title (nil = no change)
	Body      *string  // New body (nil = no change)
	Base      *string  // New base branch (nil = no change)
	Labels    []string // Labels to set (replaces existing)
	Assignees []string // Assignees to set (replaces existing)
	Draft     *bool    // Draft status (nil = no change)
}

// MergeOptions configures pull request merging.
type MergeOptions struct {
	Method        MergeMethod // Merge method (merge, squash, rebase)
	CommitTitle   string      // Custom commit title (for squash/merge)
	CommitMessage string      // Custom commit message (for squash/merge)
	SHA           string      // Expected HEAD SHA (for optimistic locking)
	DeleteBranch  bool        // Delete source branch after merge
}

// MergeMethod specifies how to merge a pull request.
type MergeMethod string

const (
	MergeMethodMerge  MergeMethod = "merge"
	MergeMethodSquash MergeMethod = "squash"
	MergeMethodRebase MergeMethod = "rebase"
)

// PRFilter configures pull request listing.
type PRFilter struct {
	State     PRState // Filter by state (empty = all)
	Base      string  // Filter by base branch
	Head      string  // Filter by head branch
	Author    string  // Filter by author username
	Labels    []string // Filter by labels (all must match)
	Sort      string  // Sort field (created, updated)
	Direction string  // Sort direction (asc, desc)
	Limit     int     // Maximum number to return (0 = default)
}

// PullRequest represents a created pull request.
type PullRequest struct {
	ID          int        // PR number/ID
	URL         string     // Web URL
	HTMLURL     string     // Full HTML URL
	Title       string     // PR title
	Body        string     // PR description
	State       PRState    // Current state
	Draft       bool       // Whether it's a draft
	Head        string     // Source branch
	Base        string     // Target branch
	CreatedAt   time.Time  // Creation time
	UpdatedAt   time.Time  // Last update time
	MergedAt    *time.Time // Merge time (nil if not merged)
	MergedBy    string     // Username who merged
	Commits     int        // Number of commits
	Additions   int        // Lines added
	Deletions   int        // Lines deleted
	ChangedFiles int       // Number of files changed
	Labels      []string   // Applied labels
	Reviewers   []string   // Requested reviewers
	Assignees   []string   // Assigned users
}

// PRBuilder helps construct PR options using a fluent interface.
type PRBuilder struct {
	opts PROptions
}

// NewPRBuilder creates a new PR builder with the given title.
func NewPRBuilder(title string) *PRBuilder {
	return &PRBuilder{
		opts: PROptions{
			Title: title,
			Base:  "main",
		},
	}
}

// WithTicket adds a ticket reference to the title.
// Example: "Add feature" -> "[TK-421] Add feature"
func (b *PRBuilder) WithTicket(ticketID string) *PRBuilder {
	b.opts.Title = fmt.Sprintf("[%s] %s", ticketID, b.opts.Title)
	return b
}

// WithBody sets the PR body.
func (b *PRBuilder) WithBody(body string) *PRBuilder {
	b.opts.Body = body
	return b
}

// WithSummary creates a formatted body with summary, changes, and test plan.
func (b *PRBuilder) WithSummary(summary string, changes []string, testPlan string) *PRBuilder {
	var body strings.Builder

	body.WriteString("## Summary\n\n")
	body.WriteString(summary)

	if len(changes) > 0 {
		body.WriteString("\n\n## Changes\n\n")
		for _, change := range changes {
			body.WriteString("- ")
			body.WriteString(change)
			body.WriteString("\n")
		}
	}

	if testPlan != "" {
		body.WriteString("\n## Test Plan\n\n")
		body.WriteString(testPlan)
	}

	body.WriteString("\n\n---\n*Generated by devflow*")
	b.opts.Body = body.String()
	return b
}

// WithBase sets the target branch.
func (b *PRBuilder) WithBase(base string) *PRBuilder {
	b.opts.Base = base
	return b
}

// WithHead sets the source branch.
func (b *PRBuilder) WithHead(head string) *PRBuilder {
	b.opts.Head = head
	return b
}

// WithLabels adds labels.
func (b *PRBuilder) WithLabels(labels ...string) *PRBuilder {
	b.opts.Labels = append(b.opts.Labels, labels...)
	return b
}

// WithReviewers adds reviewers.
func (b *PRBuilder) WithReviewers(reviewers ...string) *PRBuilder {
	b.opts.Reviewers = append(b.opts.Reviewers, reviewers...)
	return b
}

// WithAssignees adds assignees.
func (b *PRBuilder) WithAssignees(assignees ...string) *PRBuilder {
	b.opts.Assignees = append(b.opts.Assignees, assignees...)
	return b
}

// WithMilestone sets the milestone.
func (b *PRBuilder) WithMilestone(milestone string) *PRBuilder {
	b.opts.Milestone = milestone
	return b
}

// AsDraft creates as a draft PR.
func (b *PRBuilder) AsDraft() *PRBuilder {
	b.opts.Draft = true
	return b
}

// WithMetadata adds custom metadata.
func (b *PRBuilder) WithMetadata(key, value string) *PRBuilder {
	if b.opts.Metadata == nil {
		b.opts.Metadata = make(map[string]string)
	}
	b.opts.Metadata[key] = value
	return b
}

// Build returns the constructed PR options.
func (b *PRBuilder) Build() PROptions {
	return b.opts
}

// DetectProvider attempts to detect the PR provider from a remote URL.
func DetectProvider(remoteURL string) (string, error) {
	remoteURL = strings.ToLower(remoteURL)

	if strings.Contains(remoteURL, "github.com") {
		return "github", nil
	}
	if strings.Contains(remoteURL, "gitlab.com") || strings.Contains(remoteURL, "gitlab") {
		return "gitlab", nil
	}
	if strings.Contains(remoteURL, "bitbucket") {
		return "bitbucket", nil
	}

	return "", ErrUnknownProvider
}

// ParseRepoFromURL extracts owner and repo from a git remote URL.
func ParseRepoFromURL(remoteURL string) (owner, repo string, err error) {
	// Handle SSH URLs: git@github.com:owner/repo.git
	if strings.HasPrefix(remoteURL, "git@") {
		parts := strings.Split(remoteURL, ":")
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid SSH URL format")
		}
		path := strings.TrimSuffix(parts[1], ".git")
		pathParts := strings.Split(path, "/")
		if len(pathParts) != 2 {
			return "", "", fmt.Errorf("invalid repository path")
		}
		return pathParts[0], pathParts[1], nil
	}

	// Handle HTTPS URLs: https://github.com/owner/repo.git
	remoteURL = strings.TrimPrefix(remoteURL, "https://")
	remoteURL = strings.TrimPrefix(remoteURL, "http://")
	remoteURL = strings.TrimSuffix(remoteURL, ".git")

	parts := strings.Split(remoteURL, "/")
	if len(parts) < 3 {
		return "", "", fmt.Errorf("invalid URL format")
	}

	// Last two parts are owner/repo
	return parts[len(parts)-2], parts[len(parts)-1], nil
}
