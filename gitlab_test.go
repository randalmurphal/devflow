package devflow

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/xanzy/go-gitlab"
)

// =============================================================================
// Test Helpers
// =============================================================================

// newTestGitLabProvider creates a GitLabProvider pointing to a test server.
func newTestGitLabProvider(t *testing.T, handler http.Handler) (*GitLabProvider, *httptest.Server) {
	t.Helper()

	server := httptest.NewServer(handler)

	// Create a client pointing to the test server
	client, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(server.URL+"/api/v4"))
	if err != nil {
		t.Fatalf("create gitlab client: %v", err)
	}

	return &GitLabProvider{
		client:    client,
		projectID: "testowner/testrepo",
	}, server
}

// =============================================================================
// Provider Creation Tests
// =============================================================================

func TestNewGitLabProvider(t *testing.T) {
	t.Run("valid inputs", func(t *testing.T) {
		p, err := NewGitLabProvider("token123", "", "owner/repo")
		if err != nil {
			t.Fatalf("NewGitLabProvider: %v", err)
		}
		if p == nil {
			t.Fatal("expected non-nil provider")
		}
		if p.projectID != "owner/repo" {
			t.Errorf("projectID = %s", p.projectID)
		}
	})

	t.Run("with base URL", func(t *testing.T) {
		p, err := NewGitLabProvider("token", "https://gitlab.example.com", "myproject")
		if err != nil {
			t.Fatalf("NewGitLabProvider: %v", err)
		}
		if p == nil {
			t.Fatal("expected non-nil provider")
		}
	})

	t.Run("missing token", func(t *testing.T) {
		_, err := NewGitLabProvider("", "", "project")
		if err == nil {
			t.Error("expected error for missing token")
		}
	})

	t.Run("missing project ID", func(t *testing.T) {
		_, err := NewGitLabProvider("token", "", "")
		if err == nil {
			t.Error("expected error for missing project ID")
		}
	})
}

func TestNewGitLabProviderFromURL(t *testing.T) {
	tests := []struct {
		name          string
		url           string
		wantProjectID string
		wantErr       bool
	}{
		{
			name:          "https URL",
			url:           "https://gitlab.com/namespace/project.git",
			wantProjectID: "namespace/project",
		},
		{
			name:          "ssh URL",
			url:           "git@gitlab.com:namespace/project.git",
			wantProjectID: "namespace/project",
		},
		{
			name:    "invalid URL",
			url:     "not-a-url",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewGitLabProviderFromURL("token", tt.url)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if p.projectID != tt.wantProjectID {
				t.Errorf("projectID = %q, want %q", p.projectID, tt.wantProjectID)
			}
		})
	}
}

// =============================================================================
// CreatePR Tests
// =============================================================================

func TestGitLabProvider_CreatePR(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" && strings.Contains(r.URL.Path, "/merge_requests") {
				mr := &gitlab.MergeRequest{
					IID:          42,
					Title:        "Test MR",
					State:        "opened",
					WebURL:       "https://gitlab.com/owner/repo/-/merge_requests/42",
					SourceBranch: "feature",
					TargetBranch: "main",
				}
				json.NewEncoder(w).Encode(mr)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		})

		provider, server := newTestGitLabProvider(t, handler)
		defer server.Close()

		pr, err := provider.CreatePR(context.Background(), PROptions{
			Title: "Test MR",
			Body:  "Description",
			Head:  "feature",
			Base:  "main",
		})
		if err != nil {
			t.Fatalf("CreatePR: %v", err)
		}
		if pr.ID != 42 {
			t.Errorf("MR ID = %d, want 42", pr.ID)
		}
		if pr.Title != "Test MR" {
			t.Errorf("MR Title = %q", pr.Title)
		}
	})

	t.Run("default base branch", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" && strings.Contains(r.URL.Path, "/merge_requests") {
				mr := &gitlab.MergeRequest{
					IID:          1,
					Title:        "Test",
					State:        "opened",
					TargetBranch: "main",
				}
				json.NewEncoder(w).Encode(mr)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		})

		provider, server := newTestGitLabProvider(t, handler)
		defer server.Close()

		pr, err := provider.CreatePR(context.Background(), PROptions{
			Title: "Test",
			Head:  "feature",
			// Base not specified - should default to "main"
		})
		if err != nil {
			t.Fatalf("CreatePR: %v", err)
		}
		if pr.Base != "main" {
			t.Errorf("base = %q, want %q", pr.Base, "main")
		}
	})

	t.Run("draft MR", func(t *testing.T) {
		var receivedTitle string
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" && strings.Contains(r.URL.Path, "/merge_requests") {
				var req map[string]interface{}
				json.NewDecoder(r.Body).Decode(&req)
				if title, ok := req["title"].(string); ok {
					receivedTitle = title
				}

				mr := &gitlab.MergeRequest{
					IID:   1,
					Title: receivedTitle,
					State: "opened",
				}
				json.NewEncoder(w).Encode(mr)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		})

		provider, server := newTestGitLabProvider(t, handler)
		defer server.Close()

		_, err := provider.CreatePR(context.Background(), PROptions{
			Title: "Feature",
			Head:  "feature",
			Draft: true,
		})
		if err != nil {
			t.Fatalf("CreatePR: %v", err)
		}
		if !strings.HasPrefix(receivedTitle, "Draft:") {
			t.Errorf("title = %q, should start with 'Draft:'", receivedTitle)
		}
	})

	t.Run("MR already exists", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"message": "Another open merge request already exists",
			})
		})

		provider, server := newTestGitLabProvider(t, handler)
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
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"message": "No commits between main and feature",
			})
		})

		provider, server := newTestGitLabProvider(t, handler)
		defer server.Close()

		_, err := provider.CreatePR(context.Background(), PROptions{
			Title: "Test",
			Head:  "feature",
		})
		if err != ErrNoChanges {
			t.Errorf("err = %v, want ErrNoChanges", err)
		}
	})

	t.Run("with labels", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" && strings.Contains(r.URL.Path, "/merge_requests") {
				mr := &gitlab.MergeRequest{
					IID:    42,
					Title:  "Test",
					State:  "opened",
					Labels: gitlab.Labels{"bug", "urgent"},
				}
				json.NewEncoder(w).Encode(mr)
				return
			}
			w.WriteHeader(http.StatusOK)
		})

		provider, server := newTestGitLabProvider(t, handler)
		defer server.Close()

		pr, err := provider.CreatePR(context.Background(), PROptions{
			Title:  "Test",
			Head:   "feature",
			Labels: []string{"bug", "urgent"},
		})
		if err != nil {
			t.Fatalf("CreatePR: %v", err)
		}
		if len(pr.Labels) != 2 {
			t.Errorf("labels = %v", pr.Labels)
		}
	})
}

// =============================================================================
// GetPR Tests
// =============================================================================

func TestGitLabProvider_GetPR(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		createdAt := time.Now().Add(-24 * time.Hour)
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "GET" && strings.Contains(r.URL.Path, "/merge_requests/42") {
				mr := &gitlab.MergeRequest{
					IID:          42,
					Title:        "Feature MR",
					Description:  "Description",
					State:        "opened",
					WebURL:       "https://gitlab.com/owner/repo/-/merge_requests/42",
					SourceBranch: "feature",
					TargetBranch: "main",
					CreatedAt:    &createdAt,
					Labels:       gitlab.Labels{"enhancement"},
					Reviewers: []*gitlab.BasicUser{
						{Username: "reviewer1"},
					},
					Assignees: []*gitlab.BasicUser{
						{Username: "assignee1"},
					},
				}
				json.NewEncoder(w).Encode(mr)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		})

		provider, server := newTestGitLabProvider(t, handler)
		defer server.Close()

		pr, err := provider.GetPR(context.Background(), 42)
		if err != nil {
			t.Fatalf("GetPR: %v", err)
		}
		if pr.ID != 42 {
			t.Errorf("ID = %d", pr.ID)
		}
		if pr.Title != "Feature MR" {
			t.Errorf("Title = %q", pr.Title)
		}
		if pr.State != PRStateOpen {
			t.Errorf("State = %v", pr.State)
		}
		if len(pr.Labels) != 1 || pr.Labels[0] != "enhancement" {
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

		provider, server := newTestGitLabProvider(t, handler)
		defer server.Close()

		_, err := provider.GetPR(context.Background(), 999)
		if err != ErrPRNotFound {
			t.Errorf("err = %v, want ErrPRNotFound", err)
		}
	})

	t.Run("merged MR", func(t *testing.T) {
		mergedAt := time.Now()
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mr := &gitlab.MergeRequest{
				IID:      1,
				State:    "merged",
				MergedAt: &mergedAt,
				MergedBy: &gitlab.BasicUser{Username: "merger"},
			}
			json.NewEncoder(w).Encode(mr)
		})

		provider, server := newTestGitLabProvider(t, handler)
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

	t.Run("closed MR", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mr := &gitlab.MergeRequest{
				IID:   1,
				State: "closed",
			}
			json.NewEncoder(w).Encode(mr)
		})

		provider, server := newTestGitLabProvider(t, handler)
		defer server.Close()

		pr, err := provider.GetPR(context.Background(), 1)
		if err != nil {
			t.Fatalf("GetPR: %v", err)
		}
		if pr.State != PRStateClosed {
			t.Errorf("State = %v, want closed", pr.State)
		}
	})
}

// =============================================================================
// UpdatePR Tests
// =============================================================================

func TestGitLabProvider_UpdatePR(t *testing.T) {
	t.Run("update title and description", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "PUT" && strings.Contains(r.URL.Path, "/merge_requests/") {
				mr := &gitlab.MergeRequest{
					IID:         42,
					Title:       "Updated Title",
					Description: "Updated Description",
					State:       "opened",
				}
				json.NewEncoder(w).Encode(mr)
				return
			}
			w.WriteHeader(http.StatusOK)
		})

		provider, server := newTestGitLabProvider(t, handler)
		defer server.Close()

		newTitle := "Updated Title"
		newBody := "Updated Description"
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
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "PUT" && strings.Contains(r.URL.Path, "/merge_requests/") {
				mr := &gitlab.MergeRequest{
					IID:    1,
					Labels: gitlab.Labels{"new-label"},
				}
				json.NewEncoder(w).Encode(mr)
				return
			}
			w.WriteHeader(http.StatusOK)
		})

		provider, server := newTestGitLabProvider(t, handler)
		defer server.Close()

		_, err := provider.UpdatePR(context.Background(), 1, PRUpdateOptions{
			Labels: []string{"new-label"},
		})
		if err != nil {
			t.Fatalf("UpdatePR: %v", err)
		}
	})
}

// =============================================================================
// MergePR Tests
// =============================================================================

func TestGitLabProvider_MergePR(t *testing.T) {
	t.Run("merge success", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "PUT" && strings.Contains(r.URL.Path, "/merge") {
				mr := &gitlab.MergeRequest{
					IID:   42,
					State: "merged",
				}
				json.NewEncoder(w).Encode(mr)
				return
			}
			w.WriteHeader(http.StatusOK)
		})

		provider, server := newTestGitLabProvider(t, handler)
		defer server.Close()

		err := provider.MergePR(context.Background(), 42, MergeOptions{
			CommitMessage: "Merge MR",
		})
		if err != nil {
			t.Fatalf("MergePR: %v", err)
		}
	})

	t.Run("squash merge", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "PUT" && strings.Contains(r.URL.Path, "/merge") {
				mr := &gitlab.MergeRequest{IID: 1, State: "merged"}
				json.NewEncoder(w).Encode(mr)
				return
			}
			w.WriteHeader(http.StatusOK)
		})

		provider, server := newTestGitLabProvider(t, handler)
		defer server.Close()

		err := provider.MergePR(context.Background(), 1, MergeOptions{
			Method: MergeMethodSquash,
		})
		if err != nil {
			t.Fatalf("MergePR: %v", err)
		}
	})

	t.Run("MR not found", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})

		provider, server := newTestGitLabProvider(t, handler)
		defer server.Close()

		err := provider.MergePR(context.Background(), 999, MergeOptions{})
		if err != ErrPRNotFound {
			t.Errorf("err = %v, want ErrPRNotFound", err)
		}
	})

	t.Run("MR closed", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusMethodNotAllowed)
		})

		provider, server := newTestGitLabProvider(t, handler)
		defer server.Close()

		err := provider.MergePR(context.Background(), 1, MergeOptions{})
		if err != ErrPRClosed {
			t.Errorf("err = %v, want ErrPRClosed", err)
		}
	})

	t.Run("merge conflict", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotAcceptable)
		})

		provider, server := newTestGitLabProvider(t, handler)
		defer server.Close()

		err := provider.MergePR(context.Background(), 1, MergeOptions{})
		if err != ErrMergeConflict {
			t.Errorf("err = %v, want ErrMergeConflict", err)
		}
	})

	t.Run("delete branch after merge", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "PUT" && strings.Contains(r.URL.Path, "/merge") {
				mr := &gitlab.MergeRequest{IID: 1, State: "merged"}
				json.NewEncoder(w).Encode(mr)
				return
			}
			w.WriteHeader(http.StatusOK)
		})

		provider, server := newTestGitLabProvider(t, handler)
		defer server.Close()

		err := provider.MergePR(context.Background(), 1, MergeOptions{
			DeleteBranch: true, // GitLab handles this via should_remove_source_branch
		})
		if err != nil {
			t.Fatalf("MergePR: %v", err)
		}
	})
}

// =============================================================================
// AddComment Tests
// =============================================================================

func TestGitLabProvider_AddComment(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" && strings.Contains(r.URL.Path, "/notes") {
				note := &gitlab.Note{ID: 1, Body: "Great work!"}
				json.NewEncoder(w).Encode(note)
				return
			}
			w.WriteHeader(http.StatusOK)
		})

		provider, server := newTestGitLabProvider(t, handler)
		defer server.Close()

		err := provider.AddComment(context.Background(), 42, "Great work!")
		if err != nil {
			t.Fatalf("AddComment: %v", err)
		}
	})
}

// =============================================================================
// RequestReview Tests
// =============================================================================

func TestGitLabProvider_RequestReview(t *testing.T) {
	t.Run("success with numeric IDs", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "PUT" && strings.Contains(r.URL.Path, "/merge_requests/") {
				mr := &gitlab.MergeRequest{
					IID: 42,
					Reviewers: []*gitlab.BasicUser{
						{ID: 1, Username: "user1"},
						{ID: 2, Username: "user2"},
					},
				}
				json.NewEncoder(w).Encode(mr)
				return
			}
			w.WriteHeader(http.StatusOK)
		})

		provider, server := newTestGitLabProvider(t, handler)
		defer server.Close()

		// GitLab requires numeric IDs for reviewers
		err := provider.RequestReview(context.Background(), 42, []string{"1", "2"})
		if err != nil {
			t.Fatalf("RequestReview: %v", err)
		}
	})

	t.Run("no valid reviewer IDs", func(t *testing.T) {
		provider := &GitLabProvider{projectID: "test"}

		err := provider.RequestReview(context.Background(), 42, []string{"alice", "bob"})
		if err == nil {
			t.Error("expected error for non-numeric reviewer IDs")
		}
	})
}

// =============================================================================
// ListPRs Tests
// =============================================================================

func TestGitLabProvider_ListPRs(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "GET" && strings.Contains(r.URL.Path, "/merge_requests") {
				mrs := []*gitlab.MergeRequest{
					{IID: 1, Title: "MR 1", State: "opened"},
					{IID: 2, Title: "MR 2", State: "opened"},
				}
				json.NewEncoder(w).Encode(mrs)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		})

		provider, server := newTestGitLabProvider(t, handler)
		defer server.Close()

		prs, err := provider.ListPRs(context.Background(), PRFilter{})
		if err != nil {
			t.Fatalf("ListPRs: %v", err)
		}
		if len(prs) != 2 {
			t.Fatalf("got %d MRs, want 2", len(prs))
		}
		if prs[0].ID != 1 || prs[1].ID != 2 {
			t.Errorf("MR IIDs = %d, %d", prs[0].ID, prs[1].ID)
		}
	})

	t.Run("with filters", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode([]*gitlab.MergeRequest{})
		})

		provider, server := newTestGitLabProvider(t, handler)
		defer server.Close()

		_, err := provider.ListPRs(context.Background(), PRFilter{
			State:  PRStateOpen,
			Base:   "main",
			Head:   "feature",
			Author: "alice",
			Labels: []string{"bug"},
		})
		if err != nil {
			t.Fatalf("ListPRs: %v", err)
		}
	})
}

// =============================================================================
// MR State Conversion Tests
// =============================================================================

func TestGitLabProvider_prFromGitLab_States(t *testing.T) {
	provider := &GitLabProvider{}

	tests := []struct {
		name      string
		glState   string
		wantState PRState
	}{
		{"opened", "opened", PRStateOpen},
		{"closed", "closed", PRStateClosed},
		{"merged", "merged", PRStateMerged},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			glMR := &gitlab.MergeRequest{
				State: tt.glState,
			}
			pr := provider.prFromGitLab(glMR)
			if pr.State != tt.wantState {
				t.Errorf("State = %v, want %v", pr.State, tt.wantState)
			}
		})
	}
}

func TestGitLabProvider_prFromGitLab_Draft(t *testing.T) {
	provider := &GitLabProvider{}

	tests := []struct {
		name      string
		title     string
		wantDraft bool
	}{
		{"normal title", "Feature implementation", false},
		{"Draft prefix", "Draft: Feature implementation", true},
		{"WIP prefix", "WIP: Feature implementation", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			glMR := &gitlab.MergeRequest{Title: tt.title}
			pr := provider.prFromGitLab(glMR)
			if pr.Draft != tt.wantDraft {
				t.Errorf("Draft = %v, want %v", pr.Draft, tt.wantDraft)
			}
		})
	}
}
