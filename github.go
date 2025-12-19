package devflow

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
)

// GitHubProvider implements PRProvider for GitHub repositories.
type GitHubProvider struct {
	client *github.Client
	owner  string
	repo   string
}

// NewGitHubProvider creates a new GitHub provider.
// token is a personal access token or GitHub App token.
// owner and repo identify the repository (e.g., "anthropic", "devflow").
func NewGitHubProvider(token, owner, repo string) (*GitHubProvider, error) {
	if token == "" {
		return nil, fmt.Errorf("GitHub token is required")
	}
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("owner and repo are required")
	}

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(context.Background(), ts)
	client := github.NewClient(tc)

	return &GitHubProvider{
		client: client,
		owner:  owner,
		repo:   repo,
	}, nil
}

// NewGitHubProviderFromURL creates a GitHub provider from a remote URL.
// Example: "https://github.com/anthropic/devflow.git"
func NewGitHubProviderFromURL(token, remoteURL string) (*GitHubProvider, error) {
	owner, repo, err := ParseRepoFromURL(remoteURL)
	if err != nil {
		return nil, fmt.Errorf("parse remote URL: %w", err)
	}
	return NewGitHubProvider(token, owner, repo)
}

// CreatePR creates a new pull request.
func (p *GitHubProvider) CreatePR(ctx context.Context, opts PROptions) (*PullRequest, error) {
	// Set default base branch
	base := opts.Base
	if base == "" {
		base = "main"
	}

	// Create the PR
	newPR := &github.NewPullRequest{
		Title: github.String(opts.Title),
		Body:  github.String(opts.Body),
		Base:  github.String(base),
		Head:  github.String(opts.Head),
		Draft: github.Bool(opts.Draft),
	}

	pr, resp, err := p.client.PullRequests.Create(ctx, p.owner, p.repo, newPR)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusUnprocessableEntity {
			// Check if PR already exists
			if strings.Contains(err.Error(), "A pull request already exists") {
				return nil, ErrPRExists
			}
			// Check if no changes
			if strings.Contains(err.Error(), "No commits between") {
				return nil, ErrNoChanges
			}
		}
		return nil, fmt.Errorf("create PR: %w", err)
	}

	// Add labels if specified
	if len(opts.Labels) > 0 {
		_, _, err = p.client.Issues.AddLabelsToIssue(ctx, p.owner, p.repo, pr.GetNumber(), opts.Labels)
		if err != nil {
			// Log but don't fail - PR was created successfully
			slog.Warn("failed to add labels to PR", "error", err, "pr", pr.GetNumber(), "labels", opts.Labels)
		}
	}

	// Request reviewers if specified
	if len(opts.Reviewers) > 0 {
		_, _, err = p.client.PullRequests.RequestReviewers(ctx, p.owner, p.repo, pr.GetNumber(),
			github.ReviewersRequest{Reviewers: opts.Reviewers})
		if err != nil {
			// Log but don't fail - PR was created successfully
			slog.Warn("failed to request reviewers", "error", err, "pr", pr.GetNumber(), "reviewers", opts.Reviewers)
		}
	}

	// Add assignees if specified
	if len(opts.Assignees) > 0 {
		_, _, err = p.client.Issues.AddAssignees(ctx, p.owner, p.repo, pr.GetNumber(), opts.Assignees)
		if err != nil {
			// Log but don't fail - PR was created successfully
			slog.Warn("failed to add assignees", "error", err, "pr", pr.GetNumber(), "assignees", opts.Assignees)
		}
	}

	return p.prFromGitHub(pr), nil
}

// GetPR retrieves a pull request by number.
func (p *GitHubProvider) GetPR(ctx context.Context, id int) (*PullRequest, error) {
	pr, resp, err := p.client.PullRequests.Get(ctx, p.owner, p.repo, id)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, ErrPRNotFound
		}
		return nil, fmt.Errorf("get PR: %w", err)
	}
	return p.prFromGitHub(pr), nil
}

// UpdatePR updates an existing pull request.
func (p *GitHubProvider) UpdatePR(ctx context.Context, id int, opts PRUpdateOptions) (*PullRequest, error) {
	update := &github.PullRequest{}

	if opts.Title != nil {
		update.Title = opts.Title
	}
	if opts.Body != nil {
		update.Body = opts.Body
	}
	if opts.Base != nil {
		update.Base = &github.PullRequestBranch{Ref: opts.Base}
	}

	pr, _, err := p.client.PullRequests.Edit(ctx, p.owner, p.repo, id, update)
	if err != nil {
		return nil, fmt.Errorf("update PR: %w", err)
	}

	// Update labels if specified
	if opts.Labels != nil {
		_, _, err = p.client.Issues.ReplaceLabelsForIssue(ctx, p.owner, p.repo, id, opts.Labels)
		if err != nil {
			slog.Warn("failed to update labels", "error", err, "pr", id, "labels", opts.Labels)
		}
	}

	// Update assignees if specified
	if opts.Assignees != nil {
		issue, _, err := p.client.Issues.Get(ctx, p.owner, p.repo, id)
		if err != nil {
			slog.Warn("failed to get issue for assignee update", "error", err, "pr", id)
		} else if issue != nil {
			// Remove existing assignees
			var existing []string
			for _, a := range issue.Assignees {
				existing = append(existing, a.GetLogin())
			}
			if len(existing) > 0 {
				if _, _, err := p.client.Issues.RemoveAssignees(ctx, p.owner, p.repo, id, existing); err != nil {
					slog.Warn("failed to remove existing assignees", "error", err, "pr", id, "assignees", existing)
				}
			}
		}
		// Add new assignees
		if len(opts.Assignees) > 0 {
			if _, _, err := p.client.Issues.AddAssignees(ctx, p.owner, p.repo, id, opts.Assignees); err != nil {
				slog.Warn("failed to add new assignees", "error", err, "pr", id, "assignees", opts.Assignees)
			}
		}
	}

	return p.prFromGitHub(pr), nil
}

// MergePR merges a pull request.
func (p *GitHubProvider) MergePR(ctx context.Context, id int, opts MergeOptions) error {
	mergeOpts := &github.PullRequestOptions{
		CommitTitle: opts.CommitTitle,
		SHA:         opts.SHA,
	}

	switch opts.Method {
	case MergeMethodSquash:
		mergeOpts.MergeMethod = "squash"
	case MergeMethodRebase:
		mergeOpts.MergeMethod = "rebase"
	default:
		mergeOpts.MergeMethod = "merge"
	}

	_, resp, err := p.client.PullRequests.Merge(ctx, p.owner, p.repo, id, opts.CommitMessage, mergeOpts)
	if err != nil {
		if resp != nil {
			switch resp.StatusCode {
			case http.StatusNotFound:
				return ErrPRNotFound
			case http.StatusMethodNotAllowed:
				return ErrPRClosed
			case http.StatusConflict:
				return ErrMergeConflict
			}
		}
		return fmt.Errorf("merge PR: %w", err)
	}

	// Delete branch if requested
	if opts.DeleteBranch {
		pr, _, err := p.client.PullRequests.Get(ctx, p.owner, p.repo, id)
		if err != nil {
			slog.Warn("failed to get PR for branch deletion", "error", err, "pr", id)
		} else if pr.Head != nil && pr.Head.Ref != nil {
			if _, err := p.client.Git.DeleteRef(ctx, p.owner, p.repo, "heads/"+*pr.Head.Ref); err != nil {
				slog.Warn("failed to delete branch after merge", "error", err, "pr", id, "branch", *pr.Head.Ref)
			}
		}
	}

	return nil
}

// AddComment adds a comment to a pull request.
func (p *GitHubProvider) AddComment(ctx context.Context, id int, body string) error {
	_, _, err := p.client.Issues.CreateComment(ctx, p.owner, p.repo, id,
		&github.IssueComment{Body: github.String(body)})
	if err != nil {
		return fmt.Errorf("add comment: %w", err)
	}
	return nil
}

// RequestReview requests review from the specified users.
func (p *GitHubProvider) RequestReview(ctx context.Context, id int, reviewers []string) error {
	_, _, err := p.client.PullRequests.RequestReviewers(ctx, p.owner, p.repo, id,
		github.ReviewersRequest{Reviewers: reviewers})
	if err != nil {
		return fmt.Errorf("request review: %w", err)
	}
	return nil
}

// ListPRs lists pull requests matching the filter.
func (p *GitHubProvider) ListPRs(ctx context.Context, filter PRFilter) ([]*PullRequest, error) {
	opts := &github.PullRequestListOptions{
		ListOptions: github.ListOptions{PerPage: 30},
	}

	if filter.State != "" {
		opts.State = string(filter.State)
	} else {
		opts.State = "all"
	}
	if filter.Base != "" {
		opts.Base = filter.Base
	}
	if filter.Head != "" {
		opts.Head = filter.Head
	}
	if filter.Sort != "" {
		opts.Sort = filter.Sort
	}
	if filter.Direction != "" {
		opts.Direction = filter.Direction
	}
	if filter.Limit > 0 {
		opts.PerPage = filter.Limit
	}

	prs, _, err := p.client.PullRequests.List(ctx, p.owner, p.repo, opts)
	if err != nil {
		return nil, fmt.Errorf("list PRs: %w", err)
	}

	result := make([]*PullRequest, len(prs))
	for i, pr := range prs {
		result[i] = p.prFromGitHub(pr)
	}
	return result, nil
}

// prFromGitHub converts a GitHub PR to our PullRequest type.
func (p *GitHubProvider) prFromGitHub(pr *github.PullRequest) *PullRequest {
	result := &PullRequest{
		ID:           pr.GetNumber(),
		URL:          pr.GetURL(),
		HTMLURL:      pr.GetHTMLURL(),
		Title:        pr.GetTitle(),
		Body:         pr.GetBody(),
		Draft:        pr.GetDraft(),
		Commits:      pr.GetCommits(),
		Additions:    pr.GetAdditions(),
		Deletions:    pr.GetDeletions(),
		ChangedFiles: pr.GetChangedFiles(),
	}

	// State
	switch pr.GetState() {
	case "open":
		result.State = PRStateOpen
	case "closed":
		if pr.GetMerged() {
			result.State = PRStateMerged
		} else {
			result.State = PRStateClosed
		}
	}

	// Branches
	if pr.Head != nil {
		result.Head = pr.Head.GetRef()
	}
	if pr.Base != nil {
		result.Base = pr.Base.GetRef()
	}

	// Times
	if pr.CreatedAt != nil {
		result.CreatedAt = pr.CreatedAt.Time
	}
	if pr.UpdatedAt != nil {
		result.UpdatedAt = pr.UpdatedAt.Time
	}
	if pr.MergedAt != nil {
		t := pr.MergedAt.Time
		result.MergedAt = &t
	}
	if pr.MergedBy != nil {
		result.MergedBy = pr.MergedBy.GetLogin()
	}

	// Labels
	for _, label := range pr.Labels {
		result.Labels = append(result.Labels, label.GetName())
	}

	// Reviewers
	for _, reviewer := range pr.RequestedReviewers {
		result.Reviewers = append(result.Reviewers, reviewer.GetLogin())
	}

	// Assignees
	for _, assignee := range pr.Assignees {
		result.Assignees = append(result.Assignees, assignee.GetLogin())
	}

	return result
}
