package jira

import (
	"testing"
	"time"
)

func TestValidateIssueKey(t *testing.T) {
	tests := []struct {
		key   string
		valid bool
	}{
		{"PROJ-123", true},
		{"A-1", true},
		{"ABC123-9999", true},
		{"PROJECT-1", true},
		{"proj-123", false},    // lowercase not allowed
		{"123-456", false},     // must start with letter
		{"PROJ123", false},     // missing dash
		{"PROJ-", false},       // missing number
		{"-123", false},        // missing project
		{"", false},            // empty
		{"PROJ-0", true},       // zero is valid
		{"A1B2-123", true},     // alphanumeric project key
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := ValidateIssueKey(tt.key)
			if got != tt.valid {
				t.Errorf("ValidateIssueKey(%q) = %v, want %v", tt.key, got, tt.valid)
			}
		})
	}
}

func TestParseTime(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"standard format", "2025-01-15T10:30:00.000+0000", false},
		{"with Z", "2025-01-15T10:30:00.000Z", false},
		{"no milliseconds", "2025-01-15T10:30:00+0000", false},
		{"RFC3339", "2025-01-15T10:30:00Z", false},
		{"empty", "", false},
		{"invalid", "not-a-date", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseTime(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseTime(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseTime(%q) unexpected error: %v", tt.input, err)
				return
			}
			if tt.input != "" && result.IsZero() {
				t.Errorf("ParseTime(%q) returned zero time", tt.input)
			}
		})
	}
}

func TestFormatTime(t *testing.T) {
	tm := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	got := FormatTime(tm)
	want := "2025-01-15T10:30:00.000+0000"
	if got != want {
		t.Errorf("FormatTime() = %q, want %q", got, want)
	}
}

func TestUserGetID(t *testing.T) {
	tests := []struct {
		name string
		user User
		want string
	}{
		{
			name: "cloud user with accountId",
			user: User{AccountID: "cloud-123", Name: "jsmith"},
			want: "cloud-123",
		},
		{
			name: "server user with name only",
			user: User{Name: "jsmith"},
			want: "jsmith",
		},
		{
			name: "empty user",
			user: User{},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.user.GetID()
			if got != tt.want {
				t.Errorf("User.GetID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIssueFieldsCreatedTime(t *testing.T) {
	fields := IssueFields{
		Created: "2025-01-15T10:30:00.000+0000",
	}

	tm, err := fields.CreatedTime()
	if err != nil {
		t.Fatalf("CreatedTime() error = %v", err)
	}

	if tm.Year() != 2025 || tm.Month() != 1 || tm.Day() != 15 {
		t.Errorf("CreatedTime() = %v, expected 2025-01-15", tm)
	}
}

func TestIssueFieldsUpdatedTime(t *testing.T) {
	fields := IssueFields{
		Updated: "2025-06-20T14:45:30.000+0000",
	}

	tm, err := fields.UpdatedTime()
	if err != nil {
		t.Fatalf("UpdatedTime() error = %v", err)
	}

	if tm.Year() != 2025 || tm.Month() != 6 || tm.Day() != 20 {
		t.Errorf("UpdatedTime() = %v, expected 2025-06-20", tm)
	}
}

func TestCommentCreatedTime(t *testing.T) {
	comment := Comment{
		Created: "2025-03-10T08:15:00.000Z",
	}

	tm, err := comment.CreatedTime()
	if err != nil {
		t.Fatalf("CreatedTime() error = %v", err)
	}

	if tm.Year() != 2025 || tm.Month() != 3 || tm.Day() != 10 {
		t.Errorf("CreatedTime() = %v, expected 2025-03-10", tm)
	}
}
