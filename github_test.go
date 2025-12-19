package devflow

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v57/github"
)

// =============================================================================
// Test Helpers
// =============================================================================

// newTestGitHubProvider creates a GitHubProvider pointing to a test server.
func newTestGitHubProvider(t *testing.T, handler http.Handler) (*GitHubProvider, *httptest.Server) {
	t.Helper()

	server := httptest.NewServer(handler)

	// Create a client pointing to the test server
	client := github.NewClient(nil)
	baseURL := server.URL + "/"
	client.BaseURL, _ = client.BaseURL.Parse(baseURL)

	return &GitHubProvider{
		client: client,
		owner:  "testowner",
		repo:   "testrepo",
	}, server
}

// =============================================================================
// Provider Creation Tests
// =============================================================================

func TestNewGitHubProvider(t *testing.T) {
	t.Run("valid inputs", func(t *testing.T) {
		p, err := NewGitHubProvider("token123", "owner", "repo")
		if err != nil {
			t.Fatalf("NewGitHubProvider: %v", err)
		}
		if p == nil {
			t.Fatal("expected non-nil provider")
		}
		if p.owner != "owner" || p.repo != "repo" {
			t.Errorf("owner/repo = %s/%s", p.owner, p.repo)
		}
	})

	t.Run("missing token", func(t *testing.T) {
		_, err := NewGitHubProvider("", "owner", "repo")
		if err == nil {
			t.Error("expected error for missing token")
		}
	})

	t.Run("missing owner", func(t *testing.T) {
		_, err := NewGitHubProvider("token", "", "repo")
		if err == nil {
			t.Error("expected error for missing owner")
		}
	})

	t.Run("missing repo", func(t *testing.T) {
		_, err := NewGitHubProvider("token", "owner", "")
		if err == nil {
			t.Error("expected error for missing repo")
		}
	})
}

func TestNewGitHubProviderFromURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{
			name:      "https URL",
			url:       "https://github.com/anthropic/devflow.git",
			wantOwner: "anthropic",
			wantRepo:  "devflow",
		},
		{
			name:      "ssh URL",
			url:       "git@github.com:anthropic/devflow.git",
			wantOwner: "anthropic",
			wantRepo:  "devflow",
		},
		{
			name:    "invalid URL",
			url:     "not-a-url",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewGitHubProviderFromURL("token", tt.url)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if p.owner != tt.wantOwner {
				t.Errorf("owner = %q, want %q", p.owner, tt.wantOwner)
			}
			if p.repo != tt.wantRepo {
				t.Errorf("repo = %q, want %q", p.repo, tt.wantRepo)
			}
		})
	}
}

// =============================================================================
// CreatePR Tests
// =============================================================================

func TestGitHubProvider_CreatePR(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" && strings.Contains(r.URL.Path, "/pulls") {
				pr := &github.PullRequest{
					Number:  github.Int(42),
					Title:   github.String("Test PR"),
					State:   github.String("open"),
					HTMLURL: github.String("https://github.com/owner/repo/pull/42"),
					Head:    &github.PullRequestBranch{Ref: github.String("feature")},
					Base:    &github.PullRequestBranch{Ref: github.String("main")},
				}
				json.NewEncoder(w).Encode(pr)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		})

		provider, server := newTestGitHubProvider(t, handler)
		defer server.Close()

		pr, err := provider.CreatePR(context.Background(), PROptions{
			Title: "Test PR",
			Body:  "Description",
			Head:  "feature",
			Base:  "main",
		})
		if err != nil {
			t.Fatalf("CreatePR: %v", err)
		}
		if pr.ID != 42 {
			t.Errorf("PR ID = %d, want 42", pr.ID)
		}
		if pr.Title != "Test PR" {
			t.Errorf("PR Title = %q", pr.Title)
		}
	})

	t.Run("default base branch", func(t *testing.T) {
		var receivedBase string
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" && strings.Contains(r.URL.Path, "/pulls") {
				var req github.NewPullRequest
				json.NewDecoder(r.Body).Decode(&req)
				receivedBase = req.GetBase()

				pr := &github.PullRequest{
					Number: github.Int(1),
					Title:  github.String("Test"),
					State:  github.String("open"),
				}
				json.NewEncoder(w).Encode(pr)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		})

		provider, server := newTestGitHubProvider(t, handler)
		defer server.Close()

		_, err := provider.CreatePR(context.Background(), PROptions{
			Title: "Test",
			Head:  "feature",
			// Base not specified
		})
		if err != nil {
			t.Fatalf("CreatePR: %v", err)
		}
		if receivedBase != "main" {
			t.Errorf("base = %q, want %q", receivedBase, "main")
		}
	})

	t.Run("PR already exists", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnprocessableEntity)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"message": "A pull request already exists for feature",
			})
		})

		provider, server := newTestGitHubProvider(t, handler)
		defer server.Close()

		_, err := provider.CreatePR(context.Background(), PROptions{
			Title: "Test",
			Head:  "feature",
		})
		if err != ErrPRExists {
			t.Errorf("err = %v, want ErrPRExists", err)
		}
	})

	t.Run("no changes", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnprocessableEntity)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"message": "No commits between main and feature",
			})
		})

		provider, server := newTestGitHubProvider(t, handler)
		defer server.Close()

		_, err := provider.CreatePR(context.Background(), PROptions{
			Title: "Test",
			Head:  "feature",
		})
		if err != ErrNoChanges {
			t.Errorf("err = %v, want ErrNoChanges", err)
		}
	})

	t.Run("with labels and reviewers", func(t *testing.T) {
		var labelsAdded, reviewersAdded bool
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Log path for debugging
			path := r.URL.Path
			switch {
			case r.Method == "POST" && strings.HasSuffix(path, "/pulls"):
				pr := &github.PullRequest{
					Number: github.Int(42),
					Title:  github.String("Test"),
					State:  github.String("open"),
				}
				json.NewEncoder(w).Encode(pr)
			case r.Method == "POST" && strings.Contains(path, "/issues/") && strings.Contains(path, "/labels"):
				labelsAdded = true
				json.NewEncoder(w).Encode([]github.Label{})
			case r.Method == "POST" && strings.Contains(path, "/pulls/") && strings.Contains(path, "/requested_reviewers"):
				reviewersAdded = true
				json.NewEncoder(w).Encode(&github.PullRequest{})
			default:
				// Default OK for any other requests
				w.WriteHeader(http.StatusOK)
			}
		})

		provider, server := newTestGitHubProvider(t, handler)
		defer server.Close()

		_, err := provider.CreatePR(context.Background(), PROptions{
			Title:     "Test",
			Head:      "feature",
			Labels:    []string{"bug", "urgent"},
			Reviewers: []string{"alice", "bob"},
		})
		if err != nil {
			t.Fatalf("CreatePR: %v", err)
		}
		if !labelsAdded {
			t.Error("labels were not added")
		}
		if !reviewersAdded {
			t.Error("reviewers were not requested")
		}
	})
}

// =============================================================================
// GetPR Tests
// =============================================================================

func TestGitHubProvider_GetPR(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		createdAt := time.Now().Add(-24 * time.Hour)
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "GET" && strings.Contains(r.URL.Path, "/pulls/42") {
				pr := &github.PullRequest{
					Number:       github.Int(42),
					Title:        github.String("Feature PR"),
					Body:         github.String("Description"),
					State:        github.String("open"),
					Draft:        github.Bool(false),
					Commits:      github.Int(3),
					Additions:    github.Int(100),
					Deletions:    github.Int(50),
					ChangedFiles: github.Int(5),
					HTMLURL:      github.String("https://github.com/owner/repo/pull/42"),
					Head:         &github.PullRequestBranch{Ref: github.String("feature")},
					Base:         &github.PullRequestBranch{Ref: github.String("main")},
					CreatedAt:    &github.Timestamp{Time: createdAt},
					Labels: []*github.Label{
						{Name: github.String("bug")},
					},
					RequestedReviewers: []*github.User{
						{Login: github.String("reviewer1")},
					},
					Assignees: []*github.User{
						{Login: github.String("assignee1")},
					},
				}
				json.NewEncoder(w).Encode(pr)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		})

		provider, server := newTestGitHubProvider(t, handler)
		defer server.Close()

		pr, err := provider.GetPR(context.Background(), 42)
		if err != nil {
			t.Fatalf("GetPR: %v", err)
		}
		if pr.ID != 42 {
			t.Errorf("ID = %d", pr.ID)
		}
		if pr.Title != "Feature PR" {
			t.Errorf("Title = %q", pr.Title)
		}
		if pr.State != PRStateOpen {
			t.Errorf("State = %v", pr.State)
		}
		if pr.Commits != 3 {
			t.Errorf("Commits = %d", pr.Commits)
		}
		if len(pr.Labels) != 1 || pr.Labels[0] != "bug" {
			t.Errorf("Labels = %v", pr.Labels)
		}
		if len(pr.Reviewers) != 1 || pr.Reviewers[0] != "reviewer1" {
			t.Errorf("Reviewers = %v", pr.Reviewers)
		}
	})

	t.Run("not found", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})

		provider, server := newTestGitHubProvider(t, handler)
		defer server.Close()

		_, err := provider.GetPR(context.Background(), 999)
		if err != ErrPRNotFound {
			t.Errorf("err = %v, want ErrPRNotFound", err)
		}
	})

	t.Run("merged PR", func(t *testing.T) {
		mergedAt := time.Now()
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			pr := &github.PullRequest{
				Number:   github.Int(1),
				State:    github.String("closed"),
				Merged:   github.Bool(true),
				MergedAt: &github.Timestamp{Time: mergedAt},
				MergedBy: &github.User{Login: github.String("merger")},
			}
			json.NewEncoder(w).Encode(pr)
		})

		provider, server := newTestGitHubProvider(t, handler)
		defer server.Close()

		pr, err := provider.GetPR(context.Background(), 1)
		if err != nil {
			t.Fatalf("GetPR: %v", err)
		}
		if pr.State != PRStateMerged {
			t.Errorf("State = %v, want merged", pr.State)
		}
		if pr.MergedBy != "merger" {
			t.Errorf("MergedBy = %q", pr.MergedBy)
		}
	})
}

// =============================================================================
// UpdatePR Tests
// =============================================================================

func TestGitHubProvider_UpdatePR(t *testing.T) {
	t.Run("update title and body", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "PATCH" && strings.Contains(r.URL.Path, "/pulls/") {
				pr := &github.PullRequest{
					Number: github.Int(42),
					Title:  github.String("Updated Title"),
					Body:   github.String("Updated Body"),
					State:  github.String("open"),
				}
				json.NewEncoder(w).Encode(pr)
				return
			}
			w.WriteHeader(http.StatusOK)
		})

		provider, server := newTestGitHubProvider(t, handler)
		defer server.Close()

		newTitle := "Updated Title"
		newBody := "Updated Body"
		pr, err := provider.UpdatePR(context.Background(), 42, PRUpdateOptions{
			Title: &newTitle,
			Body:  &newBody,
		})
		if err != nil {
			t.Fatalf("UpdatePR: %v", err)
		}
		if pr.Title != "Updated Title" {
			t.Errorf("Title = %q", pr.Title)
		}
	})

	t.Run("update labels", func(t *testing.T) {
		var labelsReplaced bool
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == "PATCH" && strings.Contains(r.URL.Path, "/pulls/"):
				json.NewEncoder(w).Encode(&github.PullRequest{Number: github.Int(1)})
			case r.Method == "PUT" && strings.Contains(r.URL.Path, "/labels"):
				labelsReplaced = true
				json.NewEncoder(w).Encode([]github.Label{})
			default:
				w.WriteHeader(http.StatusOK)
			}
		})

		provider, server := newTestGitHubProvider(t, handler)
		defer server.Close()

		_, err := provider.UpdatePR(context.Background(), 1, PRUpdateOptions{
			Labels: []string{"new-label"},
		})
		if err != nil {
			t.Fatalf("UpdatePR: %v", err)
		}
		if !labelsReplaced {
			t.Error("labels were not replaced")
		}
	})
}

// =============================================================================
// MergePR Tests
// =============================================================================

func TestGitHubProvider_MergePR(t *testing.T) {
	t.Run("merge success", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "PUT" && strings.Contains(r.URL.Path, "/merge") {
				json.NewEncoder(w).Encode(&github.PullRequestMergeResult{
					Merged: github.Bool(true),
				})
				return
			}
			w.WriteHeader(http.StatusOK)
		})

		provider, server := newTestGitHubProvider(t, handler)
		defer server.Close()

		err := provider.MergePR(context.Background(), 42, MergeOptions{
			Method:        MergeMethodMerge,
			CommitMessage: "Merge PR",
		})
		if err != nil {
			t.Fatalf("MergePR: %v", err)
		}
	})

	t.Run("squash merge", func(t *testing.T) {
		// Just verify the merge request is made successfully
		// The merge method is set via the API, we trust go-github to serialize it correctly
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "PUT" && strings.Contains(r.URL.Path, "/merge") {
				json.NewEncoder(w).Encode(&github.PullRequestMergeResult{Merged: github.Bool(true)})
				return
			}
			w.WriteHeader(http.StatusOK)
		})

		provider, server := newTestGitHubProvider(t, handler)
		defer server.Close()

		err := provider.MergePR(context.Background(), 1, MergeOptions{
			Method: MergeMethodSquash,
		})
		if err != nil {
			t.Fatalf("MergePR: %v", err)
		}
	})

	t.Run("rebase merge", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "PUT" && strings.Contains(r.URL.Path, "/merge") {
				json.NewEncoder(w).Encode(&github.PullRequestMergeResult{Merged: github.Bool(true)})
				return
			}
			w.WriteHeader(http.StatusOK)
		})

		provider, server := newTestGitHubProvider(t, handler)
		defer server.Close()

		err := provider.MergePR(context.Background(), 1, MergeOptions{
			Method: MergeMethodRebase,
		})
		if err != nil {
			t.Fatalf("MergePR: %v", err)
		}
	})

	t.Run("PR not found", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})

		provider, server := newTestGitHubProvider(t, handler)
		defer server.Close()

		err := provider.MergePR(context.Background(), 999, MergeOptions{})
		if err != ErrPRNotFound {
			t.Errorf("err = %v, want ErrPRNotFound", err)
		}
	})

	t.Run("PR closed", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusMethodNotAllowed)
		})

		provider, server := newTestGitHubProvider(t, handler)
		defer server.Close()

		err := provider.MergePR(context.Background(), 1, MergeOptions{})
		if err != ErrPRClosed {
			t.Errorf("err = %v, want ErrPRClosed", err)
		}
	})

	t.Run("merge conflict", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusConflict)
		})

		provider, server := newTestGitHubProvider(t, handler)
		defer server.Close()

		err := provider.MergePR(context.Background(), 1, MergeOptions{})
		if err != ErrMergeConflict {
			t.Errorf("err = %v, want ErrMergeConflict", err)
		}
	})

	t.Run("delete branch after merge", func(t *testing.T) {
		var branchDeleted bool
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == "PUT" && strings.Contains(r.URL.Path, "/merge"):
				json.NewEncoder(w).Encode(&github.PullRequestMergeResult{Merged: github.Bool(true)})
			case r.Method == "GET" && strings.Contains(r.URL.Path, "/pulls/"):
				json.NewEncoder(w).Encode(&github.PullRequest{
					Head: &github.PullRequestBranch{Ref: github.String("feature-branch")},
				})
			case r.Method == "DELETE" && strings.Contains(r.URL.Path, "/git/refs/heads/feature-branch"):
				branchDeleted = true
				w.WriteHeader(http.StatusNoContent)
			default:
				w.WriteHeader(http.StatusOK)
			}
		})

		provider, server := newTestGitHubProvider(t, handler)
		defer server.Close()

		err := provider.MergePR(context.Background(), 1, MergeOptions{
			DeleteBranch: true,
		})
		if err != nil {
			t.Fatalf("MergePR: %v", err)
		}
		if !branchDeleted {
			t.Error("branch was not deleted")
		}
	})
}

// =============================================================================
// AddComment Tests
// =============================================================================

func TestGitHubProvider_AddComment(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var receivedBody string
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" && strings.Contains(r.URL.Path, "/comments") {
				var comment github.IssueComment
				json.NewDecoder(r.Body).Decode(&comment)
				receivedBody = comment.GetBody()
				json.NewEncoder(w).Encode(&github.IssueComment{ID: github.Int64(1)})
				return
			}
			w.WriteHeader(http.StatusOK)
		})

		provider, server := newTestGitHubProvider(t, handler)
		defer server.Close()

		err := provider.AddComment(context.Background(), 42, "Great work!")
		if err != nil {
			t.Fatalf("AddComment: %v", err)
		}
		if receivedBody != "Great work!" {
			t.Errorf("body = %q", receivedBody)
		}
	})
}

// =============================================================================
// RequestReview Tests
// =============================================================================

func TestGitHubProvider_RequestReview(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var receivedReviewers []string
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" && strings.Contains(r.URL.Path, "/requested_reviewers") {
				var req github.ReviewersRequest
				json.NewDecoder(r.Body).Decode(&req)
				receivedReviewers = req.Reviewers
				json.NewEncoder(w).Encode(&github.PullRequest{})
				return
			}
			w.WriteHeader(http.StatusOK)
		})

		provider, server := newTestGitHubProvider(t, handler)
		defer server.Close()

		err := provider.RequestReview(context.Background(), 42, []string{"alice", "bob"})
		if err != nil {
			t.Fatalf("RequestReview: %v", err)
		}
		if len(receivedReviewers) != 2 {
			t.Errorf("reviewers = %v", receivedReviewers)
		}
	})
}

// =============================================================================
// ListPRs Tests
// =============================================================================

func TestGitHubProvider_ListPRs(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "GET" && strings.Contains(r.URL.Path, "/pulls") {
				prs := []*github.PullRequest{
					{Number: github.Int(1), Title: github.String("PR 1"), State: github.String("open")},
					{Number: github.Int(2), Title: github.String("PR 2"), State: github.String("open")},
				}
				json.NewEncoder(w).Encode(prs)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		})

		provider, server := newTestGitHubProvider(t, handler)
		defer server.Close()

		prs, err := provider.ListPRs(context.Background(), PRFilter{})
		if err != nil {
			t.Fatalf("ListPRs: %v", err)
		}
		if len(prs) != 2 {
			t.Fatalf("got %d PRs, want 2", len(prs))
		}
		if prs[0].ID != 1 || prs[1].ID != 2 {
			t.Errorf("PR IDs = %d, %d", prs[0].ID, prs[1].ID)
		}
	})

	t.Run("with filters", func(t *testing.T) {
		var receivedState, receivedBase string
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedState = r.URL.Query().Get("state")
			receivedBase = r.URL.Query().Get("base")
			json.NewEncoder(w).Encode([]*github.PullRequest{})
		})

		provider, server := newTestGitHubProvider(t, handler)
		defer server.Close()

		_, err := provider.ListPRs(context.Background(), PRFilter{
			State: PRStateOpen,
			Base:  "main",
		})
		if err != nil {
			t.Fatalf("ListPRs: %v", err)
		}
		if receivedState != "open" {
			t.Errorf("state = %q", receivedState)
		}
		if receivedBase != "main" {
			t.Errorf("base = %q", receivedBase)
		}
	})
}

// =============================================================================
// PR State Conversion Tests
// =============================================================================

func TestGitHubProvider_prFromGitHub_States(t *testing.T) {
	provider := &GitHubProvider{}

	tests := []struct {
		name      string
		ghState   string
		merged    bool
		wantState PRState
	}{
		{"open", "open", false, PRStateOpen},
		{"closed not merged", "closed", false, PRStateClosed},
		{"closed merged", "closed", true, PRStateMerged},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ghPR := &github.PullRequest{
				State:  github.String(tt.ghState),
				Merged: github.Bool(tt.merged),
			}
			pr := provider.prFromGitHub(ghPR)
			if pr.State != tt.wantState {
				t.Errorf("State = %v, want %v", pr.State, tt.wantState)
			}
		})
	}
}
