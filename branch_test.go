package devflow

import (
	"strings"
	"testing"
)

func TestDefaultBranchNamer(t *testing.T) {
	namer := DefaultBranchNamer()

	if namer.TypePrefix != "feature" {
		t.Errorf("TypePrefix = %q, want %q", namer.TypePrefix, "feature")
	}
	if !namer.IncludeTitle {
		t.Error("IncludeTitle should be true by default")
	}
	if namer.MaxLength != 100 {
		t.Errorf("MaxLength = %d, want %d", namer.MaxLength, 100)
	}
}

func TestBranchNamer_ForTicket(t *testing.T) {
	tests := []struct {
		name     string
		namer    *BranchNamer
		ticketID string
		title    string
		want     string
	}{
		{
			name:     "basic ticket",
			namer:    DefaultBranchNamer(),
			ticketID: "TK-421",
			title:    "Add User Authentication",
			want:     "feature/tk-421-add-user-authentication",
		},
		{
			name:     "no title",
			namer:    &BranchNamer{TypePrefix: "feature", IncludeTitle: false, MaxLength: 100},
			ticketID: "TK-421",
			title:    "Add User Authentication",
			want:     "feature/tk-421",
		},
		{
			name:     "custom prefix",
			namer:    &BranchNamer{TypePrefix: "bugfix", IncludeTitle: true, MaxLength: 100},
			ticketID: "TK-422",
			title:    "Fix login bug",
			want:     "bugfix/tk-422-fix-login-bug",
		},
		{
			name:     "long title truncation",
			namer:    DefaultBranchNamer(),
			ticketID: "TK-421",
			title:    "This is a very long title that should be truncated because it exceeds fifty characters",
			want:     "feature/tk-421-this-is-a-very-long-title-that-should-be-truncated",
		},
		{
			name:     "special characters in title",
			namer:    DefaultBranchNamer(),
			ticketID: "TK-421",
			title:    "Fix: auth bug (critical!)",
			want:     "feature/tk-421-fix-auth-bug-critical",
		},
		{
			name:     "empty title",
			namer:    DefaultBranchNamer(),
			ticketID: "TK-421",
			title:    "",
			want:     "feature/tk-421",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.namer.ForTicket(tt.ticketID, tt.title)
			if got != tt.want {
				t.Errorf("ForTicket(%q, %q) = %q, want %q", tt.ticketID, tt.title, got, tt.want)
			}
		})
	}
}

func TestBranchNamer_ForWorkflow(t *testing.T) {
	namer := DefaultBranchNamer()

	branch := namer.ForWorkflow("ticket-to-pr", "TK-421")

	// Should start with devflow/
	if !strings.HasPrefix(branch, "devflow/") {
		t.Errorf("branch should start with 'devflow/', got %q", branch)
	}

	// Should contain workflow ID and identifier
	if !strings.Contains(branch, "ticket-to-pr") {
		t.Errorf("branch should contain workflow ID, got %q", branch)
	}
	if !strings.Contains(branch, "tk-421") {
		t.Errorf("branch should contain identifier, got %q", branch)
	}

	// Should contain timestamp (unix timestamp is 10+ digits)
	parts := strings.Split(branch, "-")
	if len(parts) < 4 {
		t.Errorf("branch should have timestamp suffix, got %q", branch)
	}
}

func TestBranchNamer_ForFeature(t *testing.T) {
	namer := DefaultBranchNamer()

	branch := namer.ForFeature("add-caching")
	if branch != "feature/add-caching" {
		t.Errorf("ForFeature = %q, want %q", branch, "feature/add-caching")
	}

	// With special characters
	branch = namer.ForFeature("Add Caching Layer!")
	if branch != "feature/add-caching-layer" {
		t.Errorf("ForFeature = %q, want %q", branch, "feature/add-caching-layer")
	}
}

func TestBranchNamer_ForBugfix(t *testing.T) {
	namer := DefaultBranchNamer()

	branch := namer.ForBugfix("TK-422", "auth crash")
	if branch != "bugfix/tk-422-auth-crash" {
		t.Errorf("ForBugfix = %q, want %q", branch, "bugfix/tk-422-auth-crash")
	}
}

func TestBranchNamer_MaxLength(t *testing.T) {
	namer := &BranchNamer{
		TypePrefix:   "feature",
		IncludeTitle: true,
		MaxLength:    30,
	}

	branch := namer.ForTicket("TK-421", "A very long feature name that exceeds the limit")
	if len(branch) > 30 {
		t.Errorf("branch length = %d, want <= 30", len(branch))
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Hello World", "hello-world"},
		{"UPPERCASE", "uppercase"},
		{"special!@#$%chars", "specialchars"},
		{"multiple   spaces", "multiple-spaces"},
		{"already-slugified", "already-slugified"},
		{"with_underscores", "with-underscores"},
		{"  leading/trailing  ", "leadingtrailing"},
		{"camelCase", "camelcase"},
		{"--double--hyphens--", "double-hyphens"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := slugify(tt.input)
			if got != tt.want {
				t.Errorf("slugify(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseBranch(t *testing.T) {
	tests := []struct {
		branch     string
		wantType   string
		wantID     string
		wantExtra  string
	}{
		{"feature/tk-421-add-auth", "feature", "tk", "421-add-auth"},
		{"bugfix/tk-422", "bugfix", "tk", "422"},
		{"devflow/ticket-to-pr-tk-421", "devflow", "ticket", "to-pr-tk-421"},
		{"main", "", "main", ""},
		{"refs/heads/feature/test", "feature", "test", ""},
	}

	for _, tt := range tests {
		t.Run(tt.branch, func(t *testing.T) {
			gotType, gotID, gotExtra := ParseBranch(tt.branch)
			if gotType != tt.wantType {
				t.Errorf("type = %q, want %q", gotType, tt.wantType)
			}
			if gotID != tt.wantID {
				t.Errorf("id = %q, want %q", gotID, tt.wantID)
			}
			if gotExtra != tt.wantExtra {
				t.Errorf("extra = %q, want %q", gotExtra, tt.wantExtra)
			}
		})
	}
}

func TestCleanBranch(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"feature/test--double", "feature/test-double"},
		{"feature/test-", "feature/test"},
		{"feature-/test", "feature/test"}, // Trailing hyphens are stripped from each segment
		{"feature/test---many---hyphens", "feature/test-many-hyphens"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := cleanBranch(tt.input)
			if got != tt.want {
				t.Errorf("cleanBranch(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
