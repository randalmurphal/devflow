package git

import (
	"fmt"
	"strings"
)

// CommitType represents the type of change in a commit.
type CommitType string

const (
	CommitTypeFeat     CommitType = "feat"
	CommitTypeFix      CommitType = "fix"
	CommitTypeDocs     CommitType = "docs"
	CommitTypeStyle    CommitType = "style"
	CommitTypeRefactor CommitType = "refactor"
	CommitTypePerf     CommitType = "perf"
	CommitTypeTest     CommitType = "test"
	CommitTypeBuild    CommitType = "build"
	CommitTypeCI       CommitType = "ci"
	CommitTypeChore    CommitType = "chore"
	CommitTypeRevert   CommitType = "revert"
)

// CommitMessage represents a structured commit message following conventional commits.
type CommitMessage struct {
	Type        CommitType // Required: type of change (feat, fix, etc.)
	Scope       string     // Optional: area of codebase affected
	Subject     string     // Required: short description (imperative mood)
	Body        string     // Optional: detailed explanation
	TicketRefs  []string   // Optional: ticket references (TK-421, ISSUE-123)
	CoAuthors   []string   // Optional: co-author emails
	GeneratedBy string     // Optional: tool that generated the commit
	Breaking    bool       // Whether this is a breaking change
}

// NewCommitMessage creates a commit message with the devflow marker.
func NewCommitMessage(typ CommitType, subject string) *CommitMessage {
	return &CommitMessage{
		Type:        typ,
		Subject:     subject,
		GeneratedBy: "devflow",
	}
}

// WithScope adds a scope to the commit message.
func (c *CommitMessage) WithScope(scope string) *CommitMessage {
	c.Scope = scope
	return c
}

// WithBody adds a body to the commit message.
func (c *CommitMessage) WithBody(body string) *CommitMessage {
	c.Body = body
	return c
}

// WithTicketRef adds a ticket reference.
func (c *CommitMessage) WithTicketRef(ref string) *CommitMessage {
	c.TicketRefs = append(c.TicketRefs, ref)
	return c
}

// WithTicketRefs adds multiple ticket references.
func (c *CommitMessage) WithTicketRefs(refs ...string) *CommitMessage {
	c.TicketRefs = append(c.TicketRefs, refs...)
	return c
}

// WithCoAuthor adds a co-author.
func (c *CommitMessage) WithCoAuthor(email string) *CommitMessage {
	c.CoAuthors = append(c.CoAuthors, email)
	return c
}

// WithBreaking marks this as a breaking change.
func (c *CommitMessage) WithBreaking() *CommitMessage {
	c.Breaking = true
	return c
}

// WithoutGeneratedBy removes the Generated-By footer.
func (c *CommitMessage) WithoutGeneratedBy() *CommitMessage {
	c.GeneratedBy = ""
	return c
}

// String formats the commit message following conventional commit format.
func (c *CommitMessage) String() string {
	var b strings.Builder

	// Subject line: type(scope)!: subject
	b.WriteString(string(c.Type))
	if c.Scope != "" {
		b.WriteString("(")
		b.WriteString(c.Scope)
		b.WriteString(")")
	}
	if c.Breaking {
		b.WriteString("!")
	}
	b.WriteString(": ")
	b.WriteString(c.Subject)

	// Body
	if c.Body != "" {
		b.WriteString("\n\n")
		b.WriteString(wrapText(c.Body, 72))
	}

	// Footer
	var footer []string

	// Breaking change note
	if c.Breaking && c.Body == "" {
		// If no body but breaking, add a note
		footer = append(footer, "BREAKING CHANGE: This commit introduces breaking changes")
	}

	// Ticket references
	for _, ref := range c.TicketRefs {
		footer = append(footer, fmt.Sprintf("Refs: %s", ref))
	}

	// Co-authors
	for _, author := range c.CoAuthors {
		footer = append(footer, fmt.Sprintf("Co-authored-by: %s", author))
	}

	// Generated-By marker
	if c.GeneratedBy != "" {
		footer = append(footer, fmt.Sprintf("Generated-By: %s", c.GeneratedBy))
	}

	if len(footer) > 0 {
		b.WriteString("\n\n")
		b.WriteString(strings.Join(footer, "\n"))
	}

	return b.String()
}

// Validate checks if the commit message is valid.
func (c *CommitMessage) Validate() error {
	if c.Type == "" {
		return fmt.Errorf("commit type is required")
	}
	if c.Subject == "" {
		return fmt.Errorf("commit subject is required")
	}
	if len(c.Subject) > 100 {
		return fmt.Errorf("commit subject too long (max 100 characters)")
	}
	return nil
}

// wrapText wraps text at the specified width, preserving existing newlines.
func wrapText(text string, width int) string {
	var result []string

	for _, paragraph := range strings.Split(text, "\n") {
		if len(paragraph) <= width {
			result = append(result, paragraph)
			continue
		}

		// Wrap long lines
		var line string
		for _, word := range strings.Fields(paragraph) {
			if line == "" {
				line = word
			} else if len(line)+1+len(word) > width {
				result = append(result, line)
				line = word
			} else {
				line += " " + word
			}
		}
		if line != "" {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

// CommitConfig configures commit message formatting.
type CommitConfig struct {
	IncludeGeneratedBy bool   // Include "Generated-By: devflow" footer
	TicketRefPrefix    string // Prefix for ticket refs ("Refs:", "Fixes:", "Closes:")
	RequireTicketRef   bool   // Require at least one ticket reference
}

// DefaultCommitConfig returns the default commit configuration.
func DefaultCommitConfig() CommitConfig {
	return CommitConfig{
		IncludeGeneratedBy: true,
		TicketRefPrefix:    "Refs:",
		RequireTicketRef:   false,
	}
}
