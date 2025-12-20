package git

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// BranchNamer generates branch names following conventions.
type BranchNamer struct {
	TypePrefix   string // Branch type prefix (e.g., "feature", "bugfix", "devflow")
	IncludeTitle bool   // Whether to include title slug in branch name
	MaxLength    int    // Maximum branch name length
}

// DefaultBranchNamer returns a namer with default settings.
func DefaultBranchNamer() *BranchNamer {
	return &BranchNamer{
		TypePrefix:   "feature",
		IncludeTitle: true,
		MaxLength:    100,
	}
}

// ForTicket generates a branch name from a ticket ID and title.
// Example: "TK-421", "Add User Authentication" -> "feature/tk-421-add-user-authentication"
func (n *BranchNamer) ForTicket(ticketID, title string) string {
	parts := []string{strings.ToLower(ticketID)}

	if n.IncludeTitle && title != "" {
		slug := Slugify(title)
		if len(slug) > 50 {
			slug = slug[:50]
			// Trim trailing hyphens after truncation
			slug = strings.TrimRight(slug, "-")
		}
		parts = append(parts, slug)
	}

	branch := n.TypePrefix + "/" + strings.Join(parts, "-")

	if n.MaxLength > 0 && len(branch) > n.MaxLength {
		branch = branch[:n.MaxLength]
	}

	return CleanBranch(branch)
}

// ForWorkflow generates a branch name for automated workflow runs.
// Example: "ticket-to-pr", "TK-421" -> "devflow/ticket-to-pr-tk-421-1734567890"
func (n *BranchNamer) ForWorkflow(workflowID, identifier string) string {
	timestamp := time.Now().Unix()
	branch := fmt.Sprintf("devflow/%s-%s-%d",
		Slugify(workflowID),
		Slugify(identifier),
		timestamp,
	)

	if n.MaxLength > 0 && len(branch) > n.MaxLength {
		branch = branch[:n.MaxLength]
	}

	return CleanBranch(branch)
}

// ForFeature generates a simple feature branch name.
// Example: "add-caching" -> "feature/add-caching"
func (n *BranchNamer) ForFeature(name string) string {
	branch := n.TypePrefix + "/" + Slugify(name)

	if n.MaxLength > 0 && len(branch) > n.MaxLength {
		branch = branch[:n.MaxLength]
	}

	return CleanBranch(branch)
}

// ForBugfix generates a bugfix branch name.
// Example: "TK-422", "auth-crash" -> "bugfix/tk-422-auth-crash"
func (n *BranchNamer) ForBugfix(ticketID, description string) string {
	namer := &BranchNamer{
		TypePrefix:   "bugfix",
		IncludeTitle: true,
		MaxLength:    n.MaxLength,
	}
	return namer.ForTicket(ticketID, description)
}

// Slugify converts a string to a URL-safe slug.
func Slugify(s string) string {
	// Lowercase
	s = strings.ToLower(s)

	// Replace spaces and underscores with hyphens
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")

	// Remove non-alphanumeric except hyphens
	s = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(s, "")

	// Remove consecutive hyphens
	s = regexp.MustCompile(`-+`).ReplaceAllString(s, "-")

	// Trim hyphens from ends
	s = strings.Trim(s, "-")

	return s
}

// CleanBranch ensures a branch name is valid.
func CleanBranch(s string) string {
	// Remove consecutive hyphens
	s = regexp.MustCompile(`-+`).ReplaceAllString(s, "-")

	// Remove trailing hyphens (but not before /)
	parts := strings.Split(s, "/")
	for i, part := range parts {
		parts[i] = strings.TrimRight(part, "-")
	}
	s = strings.Join(parts, "/")

	return s
}

// ParseBranch extracts components from a branch name.
// Returns (type, identifier, extra) where extra is any additional suffix.
func ParseBranch(branch string) (branchType, identifier, extra string) {
	// Remove refs/heads/ prefix if present
	branch = strings.TrimPrefix(branch, "refs/heads/")

	parts := strings.SplitN(branch, "/", 2)
	if len(parts) == 1 {
		// No type prefix
		return "", branch, ""
	}

	branchType = parts[0]
	rest := parts[1]

	// Try to extract identifier (usually first part before -)
	idParts := strings.SplitN(rest, "-", 2)
	identifier = idParts[0]
	if len(idParts) > 1 {
		extra = idParts[1]
	}

	return branchType, identifier, extra
}
