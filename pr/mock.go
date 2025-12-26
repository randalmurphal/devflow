package pr

import "context"

// MockProvider is a mock implementation of Provider for testing.
type MockProvider struct {
	CreatePRFunc      func(ctx context.Context, opts Options) (*PullRequest, error)
	GetPRFunc         func(ctx context.Context, id int) (*PullRequest, error)
	UpdatePRFunc      func(ctx context.Context, id int, opts UpdateOptions) (*PullRequest, error)
	MergePRFunc       func(ctx context.Context, id int, opts MergeOptions) error
	AddCommentFunc    func(ctx context.Context, id int, body string) error
	RequestReviewFunc func(ctx context.Context, id int, reviewers []string) error
	ListPRsFunc       func(ctx context.Context, filter Filter) ([]*PullRequest, error)
}

// CreatePR implements Provider.
func (m *MockProvider) CreatePR(ctx context.Context, opts Options) (*PullRequest, error) {
	if m.CreatePRFunc != nil {
		return m.CreatePRFunc(ctx, opts)
	}
	return &PullRequest{ID: 1, URL: "https://example.com/pr/1"}, nil
}

// GetPR implements Provider.
func (m *MockProvider) GetPR(ctx context.Context, id int) (*PullRequest, error) {
	if m.GetPRFunc != nil {
		return m.GetPRFunc(ctx, id)
	}
	return &PullRequest{ID: id}, nil
}

// UpdatePR implements Provider.
func (m *MockProvider) UpdatePR(ctx context.Context, id int, opts UpdateOptions) (*PullRequest, error) {
	if m.UpdatePRFunc != nil {
		return m.UpdatePRFunc(ctx, id, opts)
	}
	return &PullRequest{ID: id}, nil
}

// MergePR implements Provider.
func (m *MockProvider) MergePR(ctx context.Context, id int, opts MergeOptions) error {
	if m.MergePRFunc != nil {
		return m.MergePRFunc(ctx, id, opts)
	}
	return nil
}

// AddComment implements Provider.
func (m *MockProvider) AddComment(ctx context.Context, id int, body string) error {
	if m.AddCommentFunc != nil {
		return m.AddCommentFunc(ctx, id, body)
	}
	return nil
}

// RequestReview implements Provider.
func (m *MockProvider) RequestReview(ctx context.Context, id int, reviewers []string) error {
	if m.RequestReviewFunc != nil {
		return m.RequestReviewFunc(ctx, id, reviewers)
	}
	return nil
}

// ListPRs implements Provider.
func (m *MockProvider) ListPRs(ctx context.Context, filter Filter) ([]*PullRequest, error) {
	if m.ListPRsFunc != nil {
		return m.ListPRsFunc(ctx, filter)
	}
	return []*PullRequest{}, nil
}
