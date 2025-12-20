package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAPIError(t *testing.T) {
	tests := []struct {
		name       string
		err        *APIError
		wantMsg    string
		wantUnwrap error
	}{
		{
			name: "basic error",
			err: &APIError{
				Service:    "jira",
				StatusCode: 404,
				Message:    "Issue not found",
				Endpoint:   "/rest/api/2/issue/TEST-1",
			},
			wantMsg:    "jira API error (404) at /rest/api/2/issue/TEST-1: Issue not found",
			wantUnwrap: ErrNotFound,
		},
		{
			name: "with request ID",
			err: &APIError{
				Service:    "gitlab",
				StatusCode: 500,
				Message:    "Internal error",
				Endpoint:   "/api/v4/projects",
				RequestID:  "abc123",
			},
			wantMsg:    "gitlab API error (500) at /api/v4/projects [abc123]: Internal error",
			wantUnwrap: ErrServerError,
		},
		{
			name: "unauthorized",
			err: &APIError{
				Service:    "jira",
				StatusCode: 401,
				Message:    "Invalid credentials",
				Endpoint:   "/rest/api/2/myself",
			},
			wantMsg:    "jira API error (401) at /rest/api/2/myself: Invalid credentials",
			wantUnwrap: ErrUnauthorized,
		},
		{
			name: "forbidden",
			err: &APIError{
				Service:    "jira",
				StatusCode: 403,
				Message:    "Access denied",
				Endpoint:   "/rest/api/2/issue/SECRET-1",
			},
			wantMsg:    "jira API error (403) at /rest/api/2/issue/SECRET-1: Access denied",
			wantUnwrap: ErrForbidden,
		},
		{
			name: "rate limited",
			err: &APIError{
				Service:    "gitlab",
				StatusCode: 429,
				Message:    "Too many requests",
				Endpoint:   "/api/v4/issues",
			},
			wantMsg:    "gitlab API error (429) at /api/v4/issues: Too many requests",
			wantUnwrap: ErrRateLimited,
		},
		{
			name: "bad request",
			err: &APIError{
				Service:    "jira",
				StatusCode: 400,
				Message:    "Invalid JQL",
				Endpoint:   "/rest/api/2/search",
			},
			wantMsg:    "jira API error (400) at /rest/api/2/search: Invalid JQL",
			wantUnwrap: ErrBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.wantMsg {
				t.Errorf("Error() = %q, want %q", got, tt.wantMsg)
			}
			if got := tt.err.Unwrap(); !errors.Is(got, tt.wantUnwrap) {
				t.Errorf("Unwrap() = %v, want %v", got, tt.wantUnwrap)
			}
		})
	}
}

func TestAuthError(t *testing.T) {
	err := &AuthError{
		Service: "jira",
		Reason:  "token expired",
	}

	want := "jira authentication failed: token expired"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}

	if !errors.Is(err, ErrUnauthorized) {
		t.Error("AuthError should unwrap to ErrUnauthorized")
	}
}

func TestRateLimitError(t *testing.T) {
	tests := []struct {
		name    string
		err     *RateLimitError
		wantMsg string
	}{
		{
			name: "with retry after",
			err: &RateLimitError{
				Service:    "gitlab",
				RetryAfter: 30 * time.Second,
			},
			wantMsg: "gitlab rate limit exceeded, retry after 30s",
		},
		{
			name: "without retry after",
			err: &RateLimitError{
				Service: "jira",
			},
			wantMsg: "jira rate limit exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.wantMsg {
				t.Errorf("Error() = %q, want %q", got, tt.wantMsg)
			}
			if !errors.Is(tt.err, ErrRateLimited) {
				t.Error("RateLimitError should unwrap to ErrRateLimited")
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	tests := []struct {
		name    string
		err     *ValidationError
		wantMsg string
	}{
		{
			name: "with field",
			err: &ValidationError{
				Service: "jira",
				Field:   "summary",
				Message: "Summary is required",
			},
			wantMsg: "jira validation error on summary: Summary is required",
		},
		{
			name: "without field",
			err: &ValidationError{
				Service: "jira",
				Message: "Request body is invalid",
			},
			wantMsg: "jira validation error: Request body is invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.wantMsg {
				t.Errorf("Error() = %q, want %q", got, tt.wantMsg)
			}
			if !errors.Is(tt.err, ErrBadRequest) {
				t.Error("ValidationError should unwrap to ErrBadRequest")
			}
		})
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "rate limited",
			err:  ErrRateLimited,
			want: true,
		},
		{
			name: "server error",
			err:  ErrServerError,
			want: true,
		},
		{
			name: "5xx API error",
			err: &APIError{
				StatusCode: 503,
				Service:    "test",
			},
			want: true,
		},
		{
			name: "not found",
			err:  ErrNotFound,
			want: false,
		},
		{
			name: "bad request",
			err:  ErrBadRequest,
			want: false,
		},
		{
			name: "4xx API error",
			err: &APIError{
				StatusCode: 400,
				Service:    "test",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRetryable(tt.err); got != tt.want {
				t.Errorf("IsRetryable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPageIterator(t *testing.T) {
	t.Run("iterates through pages", func(t *testing.T) {
		data := [][]int{{1, 2, 3}, {4, 5, 6}, {7}}
		pageIdx := 0

		fetch := func(_ context.Context, _ int) ([]int, bool, error) {
			if pageIdx >= len(data) {
				return nil, false, nil
			}
			items := data[pageIdx]
			hasMore := pageIdx < len(data)-1
			pageIdx++
			return items, hasMore, nil
		}

		iter := NewPageIterator(fetch)
		got, err := iter.All(context.Background())
		if err != nil {
			t.Fatalf("All() error = %v", err)
		}

		want := []int{1, 2, 3, 4, 5, 6, 7}
		if len(got) != len(want) {
			t.Fatalf("got %d items, want %d", len(got), len(want))
		}
		for i, v := range got {
			if v != want[i] {
				t.Errorf("item %d = %d, want %d", i, v, want[i])
			}
		}
	})

	t.Run("handles empty result", func(t *testing.T) {
		fetch := func(_ context.Context, _ int) ([]string, bool, error) {
			return nil, false, nil
		}

		iter := NewPageIterator(fetch)
		got, err := iter.All(context.Background())
		if err != nil {
			t.Fatalf("All() error = %v", err)
		}
		if len(got) != 0 {
			t.Errorf("got %d items, want 0", len(got))
		}
	})

	t.Run("propagates error", func(t *testing.T) {
		wantErr := errors.New("fetch failed")
		fetch := func(_ context.Context, _ int) ([]int, bool, error) {
			return nil, false, wantErr
		}

		iter := NewPageIterator(fetch)
		_, err := iter.All(context.Background())
		if !errors.Is(err, wantErr) {
			t.Errorf("got error %v, want %v", err, wantErr)
		}
	})

	t.Run("Take limits results", func(t *testing.T) {
		fetch := func(_ context.Context, _ int) ([]int, bool, error) {
			return []int{1, 2, 3, 4, 5}, true, nil
		}

		iter := NewPageIterator(fetch)
		got, err := iter.Take(context.Background(), 3)
		if err != nil {
			t.Fatalf("Take() error = %v", err)
		}
		if len(got) != 3 {
			t.Errorf("got %d items, want 3", len(got))
		}
	})

	t.Run("ForEach processes all items", func(t *testing.T) {
		data := []int{1, 2, 3}
		idx := 0
		fetch := func(_ context.Context, _ int) ([]int, bool, error) {
			if idx > 0 {
				return nil, false, nil
			}
			idx++
			return data, false, nil
		}

		iter := NewPageIterator(fetch)
		var sum int
		err := iter.ForEach(context.Background(), func(i int) error {
			sum += i
			return nil
		})
		if err != nil {
			t.Fatalf("ForEach() error = %v", err)
		}
		if sum != 6 {
			t.Errorf("sum = %d, want 6", sum)
		}
	})
}

func TestClient(t *testing.T) {
	t.Run("successful GET", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"name": "test"})
		}))
		defer server.Close()

		client := NewClient(ClientConfig{
			BaseURL:     server.URL,
			ServiceName: "test",
		})

		var result map[string]string
		err := client.Get(context.Background(), "/test", &result)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if result["name"] != "test" {
			t.Errorf("got name = %q, want %q", result["name"], "test")
		}
	})

	t.Run("successful POST", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("got method %s, want POST", r.Method)
			}
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["key"] != "value" {
				t.Errorf("got body key = %q, want %q", body["key"], "value")
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"id": "123"})
		}))
		defer server.Close()

		client := NewClient(ClientConfig{
			BaseURL:     server.URL,
			ServiceName: "test",
		})

		var result map[string]string
		err := client.Post(context.Background(), "/create", map[string]string{"key": "value"}, &result)
		if err != nil {
			t.Fatalf("Post() error = %v", err)
		}
		if result["id"] != "123" {
			t.Errorf("got id = %q, want %q", result["id"], "123")
		}
	})

	t.Run("handles 404", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "Not found"})
		}))
		defer server.Close()

		client := NewClient(ClientConfig{
			BaseURL:     server.URL,
			ServiceName: "test",
		})

		var result map[string]string
		err := client.Get(context.Background(), "/missing", &result)
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("got error %v, want ErrNotFound", err)
		}
	})

	t.Run("applies beforeRequest hook", func(t *testing.T) {
		var gotAuth string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotAuth = r.Header.Get("Authorization")
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{})
		}))
		defer server.Close()

		client := NewClient(ClientConfig{
			BaseURL:     server.URL,
			ServiceName: "test",
			BeforeRequest: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer token123")
			},
		})

		_ = client.Get(context.Background(), "/test", nil)
		if gotAuth != "Bearer token123" {
			t.Errorf("got Authorization = %q, want %q", gotAuth, "Bearer token123")
		}
	})

	t.Run("retries on 5xx", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			attempts++
			if attempts < 3 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
		}))
		defer server.Close()

		client := NewClient(ClientConfig{
			BaseURL:     server.URL,
			ServiceName: "test",
			MaxRetries:  3,
			RetryWait:   1 * time.Millisecond,
		})

		var result map[string]string
		err := client.Get(context.Background(), "/test", &result)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if attempts != 3 {
			t.Errorf("got %d attempts, want 3", attempts)
		}
	})
}
