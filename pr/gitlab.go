package pr

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/xanzy/go-gitlab"
)

// GitLabProvider implements Provider for GitLab repositories.
type GitLabProvider struct {
	client    *gitlab.Client
	projectID string // Can be numeric ID or "namespace/project"
}

// NewGitLabProvider creates a new GitLab provider.
// token is a personal access token.
// baseURL is the GitLab instance URL (empty for gitlab.com).
// projectID can be numeric ID or "namespace/project" path.
func NewGitLabProvider(token, baseURL, projectID string) (*GitLabProvider, error) {
	if token == "" {
		return nil, fmt.Errorf("GitLab token is required")
	}
	if projectID == "" {
		return nil, fmt.Errorf("project ID is required")
	}

	var client *gitlab.Client
	var err error

	if baseURL != "" {
		client, err = gitlab.NewClient(token, gitlab.WithBaseURL(baseURL))
	} else {
		client, err = gitlab.NewClient(token)
	}

	if err != nil {
		return nil, fmt.Errorf("create GitLab client: %w", err)
	}

	return &GitLabProvider{
		client:    client,
		projectID: projectID,
	}, nil
}

// NewGitLabProviderFromURL creates a GitLab provider from a remote URL.
// Example: "https://gitlab.com/namespace/project.git"
func NewGitLabProviderFromURL(token, remoteURL string) (*GitLabProvider, error) {
	owner, repo, err := ParseRepoFromURL(remoteURL)
	if err != nil {
		return nil, fmt.Errorf("parse remote URL: %w", err)
	}

	// Extract base URL for self-hosted instances
	var baseURL string
	if !strings.Contains(remoteURL, "gitlab.com") {
		// Self-hosted GitLab
		remoteURL = strings.TrimPrefix(remoteURL, "https://")
		remoteURL = strings.TrimPrefix(remoteURL, "http://")
		parts := strings.Split(remoteURL, "/")
		if len(parts) > 0 {
			baseURL = "https://" + parts[0]
		}
	}

	projectID := owner + "/" + repo
	return NewGitLabProvider(token, baseURL, projectID)
}

// CreatePR creates a new merge request.
func (p *GitLabProvider) CreatePR(ctx context.Context, opts Options) (*PullRequest, error) {
	// Set default base branch
	targetBranch := opts.Base
	if targetBranch == "" {
		targetBranch = "main"
	}

	// Create MR options
	mrOpts := &gitlab.CreateMergeRequestOptions{
		Title:        gitlab.Ptr(opts.Title),
		Description:  gitlab.Ptr(opts.Body),
		SourceBranch: gitlab.Ptr(opts.Head),
		TargetBranch: gitlab.Ptr(targetBranch),
	}

	// GitLab doesn't support draft via API directly in older versions,
	// but newer versions support the Draft field
	// We'll prepend "Draft: " to title as a fallback
	if opts.Draft {
		mrOpts.Title = gitlab.Ptr("Draft: " + opts.Title)
	}

	// Add labels if specified
	if len(opts.Labels) > 0 {
		mrOpts.Labels = gitlab.Ptr(gitlab.LabelOptions(opts.Labels))
	}

	// Add assignees if specified
	if len(opts.Assignees) > 0 {
		// GitLab uses user IDs, but we have usernames
		// For simplicity, we'll use AssigneeIDs if they're numeric
		// or look them up if needed
		var assigneeIDs []int
		for _, a := range opts.Assignees {
			if id, err := strconv.Atoi(a); err == nil {
				assigneeIDs = append(assigneeIDs, id)
			}
		}
		if len(assigneeIDs) > 0 {
			mrOpts.AssigneeIDs = gitlab.Ptr(assigneeIDs)
		}
	}

	// Add reviewers if specified
	if len(opts.Reviewers) > 0 {
		var reviewerIDs []int
		for _, r := range opts.Reviewers {
			if id, err := strconv.Atoi(r); err == nil {
				reviewerIDs = append(reviewerIDs, id)
			}
		}
		if len(reviewerIDs) > 0 {
			mrOpts.ReviewerIDs = gitlab.Ptr(reviewerIDs)
		}
	}

	mr, resp, err := p.client.MergeRequests.CreateMergeRequest(p.projectID, mrOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusConflict {
			return nil, ErrExists
		}
		if resp != nil && resp.StatusCode == http.StatusBadRequest {
			if strings.Contains(err.Error(), "No commits between") {
				return nil, ErrNoChanges
			}
		}
		return nil, fmt.Errorf("create MR: %w", err)
	}

	return p.prFromGitLab(mr), nil
}

// GetPR retrieves a merge request by IID.
func (p *GitLabProvider) GetPR(ctx context.Context, id int) (*PullRequest, error) {
	mr, resp, err := p.client.MergeRequests.GetMergeRequest(p.projectID, id, nil)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get MR: %w", err)
	}
	return p.prFromGitLab(mr), nil
}

// UpdatePR updates an existing merge request.
func (p *GitLabProvider) UpdatePR(ctx context.Context, id int, opts UpdateOptions) (*PullRequest, error) {
	updateOpts := &gitlab.UpdateMergeRequestOptions{}

	if opts.Title != nil {
		updateOpts.Title = opts.Title
	}
	if opts.Body != nil {
		updateOpts.Description = opts.Body
	}
	if opts.Base != nil {
		updateOpts.TargetBranch = opts.Base
	}
	if opts.Labels != nil {
		updateOpts.Labels = gitlab.Ptr(gitlab.LabelOptions(opts.Labels))
	}

	mr, _, err := p.client.MergeRequests.UpdateMergeRequest(p.projectID, id, updateOpts)
	if err != nil {
		return nil, fmt.Errorf("update MR: %w", err)
	}

	return p.prFromGitLab(mr), nil
}

// MergePR merges a merge request.
func (p *GitLabProvider) MergePR(ctx context.Context, id int, opts MergeOptions) error {
	mergeOpts := &gitlab.AcceptMergeRequestOptions{}

	if opts.CommitMessage != "" {
		mergeOpts.MergeCommitMessage = gitlab.Ptr(opts.CommitMessage)
	}
	if opts.SHA != "" {
		mergeOpts.SHA = gitlab.Ptr(opts.SHA)
	}
	if opts.DeleteBranch {
		mergeOpts.ShouldRemoveSourceBranch = gitlab.Ptr(true)
	}

	switch opts.Method {
	case MergeMethodSquash:
		mergeOpts.Squash = gitlab.Ptr(true)
		if opts.CommitMessage != "" {
			mergeOpts.SquashCommitMessage = gitlab.Ptr(opts.CommitMessage)
		}
	case MergeMethodRebase:
		// GitLab doesn't have a direct rebase merge, but we can rebase before merging
		// For now, just do a regular merge
		// To properly support rebase, use RebaseMergeRequest first
	}

	_, resp, err := p.client.MergeRequests.AcceptMergeRequest(p.projectID, id, mergeOpts)
	if err != nil {
		if resp != nil {
			switch resp.StatusCode {
			case http.StatusNotFound:
				return ErrNotFound
			case http.StatusMethodNotAllowed:
				return ErrClosed
			case http.StatusNotAcceptable:
				return ErrMergeConflict
			}
		}
		return fmt.Errorf("merge MR: %w", err)
	}

	return nil
}

// AddComment adds a note to a merge request.
func (p *GitLabProvider) AddComment(ctx context.Context, id int, body string) error {
	_, _, err := p.client.Notes.CreateMergeRequestNote(p.projectID, id,
		&gitlab.CreateMergeRequestNoteOptions{Body: gitlab.Ptr(body)})
	if err != nil {
		return fmt.Errorf("add comment: %w", err)
	}
	return nil
}

// RequestReview requests review from the specified users.
// Note: GitLab uses reviewer IDs, so usernames should be numeric IDs.
func (p *GitLabProvider) RequestReview(ctx context.Context, id int, reviewers []string) error {
	var reviewerIDs []int
	for _, r := range reviewers {
		if rid, err := strconv.Atoi(r); err == nil {
			reviewerIDs = append(reviewerIDs, rid)
		}
	}

	if len(reviewerIDs) == 0 {
		return fmt.Errorf("no valid reviewer IDs provided")
	}

	_, _, err := p.client.MergeRequests.UpdateMergeRequest(p.projectID, id,
		&gitlab.UpdateMergeRequestOptions{ReviewerIDs: gitlab.Ptr(reviewerIDs)})
	if err != nil {
		return fmt.Errorf("request review: %w", err)
	}
	return nil
}

// ListPRs lists merge requests matching the filter.
func (p *GitLabProvider) ListPRs(ctx context.Context, filter Filter) ([]*PullRequest, error) {
	opts := &gitlab.ListProjectMergeRequestsOptions{
		ListOptions: gitlab.ListOptions{PerPage: 20},
	}

	if filter.State != "" {
		opts.State = gitlab.Ptr(string(filter.State))
	}
	if filter.Base != "" {
		opts.TargetBranch = gitlab.Ptr(filter.Base)
	}
	if filter.Head != "" {
		opts.SourceBranch = gitlab.Ptr(filter.Head)
	}
	if filter.Author != "" {
		opts.AuthorUsername = gitlab.Ptr(filter.Author)
	}
	if len(filter.Labels) > 0 {
		opts.Labels = gitlab.Ptr(gitlab.LabelOptions(filter.Labels))
	}
	if filter.Sort != "" {
		opts.OrderBy = gitlab.Ptr(filter.Sort)
	}
	if filter.Direction != "" {
		opts.Sort = gitlab.Ptr(filter.Direction)
	}
	if filter.Limit > 0 {
		opts.PerPage = filter.Limit
	}

	mrs, _, err := p.client.MergeRequests.ListProjectMergeRequests(p.projectID, opts)
	if err != nil {
		return nil, fmt.Errorf("list MRs: %w", err)
	}

	result := make([]*PullRequest, len(mrs))
	for i, mr := range mrs {
		result[i] = p.prFromGitLab(mr)
	}
	return result, nil
}

// prFromGitLab converts a GitLab MR to our PullRequest type.
func (p *GitLabProvider) prFromGitLab(mr *gitlab.MergeRequest) *PullRequest {
	result := &PullRequest{
		ID:      mr.IID,
		URL:     mr.WebURL,
		HTMLURL: mr.WebURL,
		Title:   mr.Title,
		Body:    mr.Description,
		Head:    mr.SourceBranch,
		Base:    mr.TargetBranch,
	}

	// Parse ChangesCount from string
	if mr.ChangesCount != "" {
		if count, err := strconv.Atoi(mr.ChangesCount); err == nil {
			result.ChangedFiles = count
		}
	}

	// Draft detection (title starts with "Draft: " or "WIP:")
	result.Draft = strings.HasPrefix(mr.Title, "Draft:") ||
		strings.HasPrefix(mr.Title, "WIP:")

	// State
	switch mr.State {
	case "opened":
		result.State = StateOpen
	case "merged":
		result.State = StateMerged
	case "closed":
		result.State = StateClosed
	}

	// Times
	if mr.CreatedAt != nil {
		result.CreatedAt = *mr.CreatedAt
	}
	if mr.UpdatedAt != nil {
		result.UpdatedAt = *mr.UpdatedAt
	}
	if mr.MergedAt != nil {
		result.MergedAt = mr.MergedAt
	}
	if mr.MergedBy != nil {
		result.MergedBy = mr.MergedBy.Username
	}

	// Stats - use Changes if available (requires additional API call for details)
	// For now, just set what we have
	// Additions/Deletions would require fetching MR changes separately

	// Labels
	result.Labels = mr.Labels

	// Reviewers
	for _, reviewer := range mr.Reviewers {
		result.Reviewers = append(result.Reviewers, reviewer.Username)
	}

	// Assignees
	for _, assignee := range mr.Assignees {
		result.Assignees = append(result.Assignees, assignee.Username)
	}

	return result
}
