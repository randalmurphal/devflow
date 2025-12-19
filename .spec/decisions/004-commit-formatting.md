# ADR-004: Commit Formatting

## Status

Accepted

## Context

devflow creates commits as part of automated workflows. We need consistent commit message formatting that:

1. Identifies automated commits
2. Includes relevant context (ticket, workflow)
3. Follows conventional commit patterns
4. Works with existing tooling (changelog generators, etc.)

## Decision

### Commit Message Format

```
{type}({scope}): {subject}

{body}

{footer}
```

### Components

| Component | Required | Description |
|-----------|----------|-------------|
| `type` | Yes | Category of change |
| `scope` | No | Area affected |
| `subject` | Yes | Short description (imperative mood) |
| `body` | No | Detailed explanation |
| `footer` | No | Metadata (ticket refs, co-authors) |

### Standard Types

| Type | Description | Example |
|------|-------------|---------|
| `feat` | New feature | `feat(auth): add OAuth2 support` |
| `fix` | Bug fix | `fix(api): handle null response` |
| `docs` | Documentation | `docs: update API reference` |
| `refactor` | Code refactoring | `refactor(db): extract connection pool` |
| `test` | Tests | `test(auth): add login tests` |
| `chore` | Maintenance | `chore: update dependencies` |

### Automated Commit Markers

Commits created by devflow include a footer marker:

```
feat(api): implement user endpoint

Adds CRUD operations for user management.

Refs: TK-421
Generated-By: devflow
```

### Template

```go
type CommitMessage struct {
    Type        string   // feat, fix, docs, etc.
    Scope       string   // Optional: area affected
    Subject     string   // Short description
    Body        string   // Optional: detailed explanation
    TicketRefs  []string // Optional: TK-421, ISSUE-123
    CoAuthors   []string // Optional: co-author emails
    GeneratedBy string   // "devflow" for automated commits
}

func (c CommitMessage) String() string {
    var b strings.Builder

    // Subject line
    if c.Scope != "" {
        fmt.Fprintf(&b, "%s(%s): %s", c.Type, c.Scope, c.Subject)
    } else {
        fmt.Fprintf(&b, "%s: %s", c.Type, c.Subject)
    }

    // Body
    if c.Body != "" {
        fmt.Fprintf(&b, "\n\n%s", c.Body)
    }

    // Footer
    var footer []string
    for _, ref := range c.TicketRefs {
        footer = append(footer, fmt.Sprintf("Refs: %s", ref))
    }
    for _, author := range c.CoAuthors {
        footer = append(footer, fmt.Sprintf("Co-authored-by: %s", author))
    }
    if c.GeneratedBy != "" {
        footer = append(footer, fmt.Sprintf("Generated-By: %s", c.GeneratedBy))
    }

    if len(footer) > 0 {
        fmt.Fprintf(&b, "\n\n%s", strings.Join(footer, "\n"))
    }

    return b.String()
}
```

### Subject Line Rules

1. **Imperative mood**: "Add feature" not "Added feature"
2. **No period**: "Add user auth" not "Add user auth."
3. **Lowercase start**: "add user auth" not "Add user auth" (after type)
4. **Max 72 characters**: Wrap at 72 for readability
5. **Present tense**: "add" not "added"

### Body Guidelines

1. Explain **why**, not just **what**
2. Wrap at 72 characters
3. Use bullet points for multiple changes
4. Reference related commits if relevant

## Alternatives Considered

### Alternative 1: Free-Form Messages

Allow any commit message format.

**Rejected because:**
- Inconsistent history
- Hard to parse for tooling
- No structure for changelog generation

### Alternative 2: Git Trailer Only

Use only git trailers without conventional commit format.

**Rejected because:**
- Less readable
- Doesn't work with conventional-changelog tools
- Type/scope are useful categorization

### Alternative 3: Emoji Prefixes

Use emojis: `:sparkles: add feature`

**Rejected because:**
- Terminal rendering issues
- Harder to grep
- Less professional appearance

## Consequences

### Positive

- **Consistent history**: All commits follow same format
- **Tooling compatibility**: Works with conventional-changelog, semantic-release
- **Traceable**: Clear link back to tickets and workflow
- **Filterable**: Easy to find automated commits (`Generated-By: devflow`)

### Negative

- **Verbose**: Footer adds lines
- **Learning curve**: Contributors need to know format
- **Opinionated**: May conflict with existing team conventions

### Configuration

Allow customizing commit format:

```go
type CommitConfig struct {
    IncludeGeneratedBy bool   // Include "Generated-By: devflow" footer
    TicketRefPrefix    string // "Refs:", "Fixes:", "Closes:"
    RequireTicketRef   bool   // Require ticket reference
}
```

## Code Example

```go
package devflow

import (
    "fmt"
    "strings"
)

// CommitType represents the type of change
type CommitType string

const (
    CommitTypeFeat     CommitType = "feat"
    CommitTypeFix      CommitType = "fix"
    CommitTypeDocs     CommitType = "docs"
    CommitTypeRefactor CommitType = "refactor"
    CommitTypeTest     CommitType = "test"
    CommitTypeChore    CommitType = "chore"
)

// CommitMessage represents a structured commit message
type CommitMessage struct {
    Type        CommitType
    Scope       string
    Subject     string
    Body        string
    TicketRefs  []string
    CoAuthors   []string
    GeneratedBy string
}

// NewCommitMessage creates a commit message with devflow marker
func NewCommitMessage(typ CommitType, subject string) *CommitMessage {
    return &CommitMessage{
        Type:        typ,
        Subject:     subject,
        GeneratedBy: "devflow",
    }
}

// WithScope adds a scope to the commit message
func (c *CommitMessage) WithScope(scope string) *CommitMessage {
    c.Scope = scope
    return c
}

// WithBody adds a body to the commit message
func (c *CommitMessage) WithBody(body string) *CommitMessage {
    c.Body = body
    return c
}

// WithTicketRef adds a ticket reference
func (c *CommitMessage) WithTicketRef(ref string) *CommitMessage {
    c.TicketRefs = append(c.TicketRefs, ref)
    return c
}

// String formats the commit message
func (c *CommitMessage) String() string {
    var b strings.Builder

    // Subject line
    if c.Scope != "" {
        fmt.Fprintf(&b, "%s(%s): %s", c.Type, c.Scope, c.Subject)
    } else {
        fmt.Fprintf(&b, "%s: %s", c.Type, c.Subject)
    }

    // Body
    if c.Body != "" {
        fmt.Fprintf(&b, "\n\n%s", wrapText(c.Body, 72))
    }

    // Footer
    var footer []string
    for _, ref := range c.TicketRefs {
        footer = append(footer, fmt.Sprintf("Refs: %s", ref))
    }
    for _, author := range c.CoAuthors {
        footer = append(footer, fmt.Sprintf("Co-authored-by: %s", author))
    }
    if c.GeneratedBy != "" {
        footer = append(footer, fmt.Sprintf("Generated-By: %s", c.GeneratedBy))
    }

    if len(footer) > 0 {
        fmt.Fprintf(&b, "\n\n%s", strings.Join(footer, "\n"))
    }

    return b.String()
}

// wrapText wraps text at specified width
func wrapText(text string, width int) string {
    // Simple implementation - production would be more sophisticated
    var result []string
    for _, line := range strings.Split(text, "\n") {
        if len(line) <= width {
            result = append(result, line)
            continue
        }
        // Wrap long lines
        words := strings.Fields(line)
        var current string
        for _, word := range words {
            if len(current)+len(word)+1 > width {
                result = append(result, current)
                current = word
            } else if current == "" {
                current = word
            } else {
                current += " " + word
            }
        }
        if current != "" {
            result = append(result, current)
        }
    }
    return strings.Join(result, "\n")
}
```

### Usage

```go
// Simple commit
msg := devflow.NewCommitMessage(devflow.CommitTypeFeat, "add user authentication").
    WithScope("auth").
    WithTicketRef("TK-421")

// Output:
// feat(auth): add user authentication
//
// Refs: TK-421
// Generated-By: devflow

// Complex commit
msg := devflow.NewCommitMessage(devflow.CommitTypeFix, "handle null response in API").
    WithScope("api").
    WithBody("The API was returning null for empty arrays, causing client crashes. This normalizes empty arrays to [] in the response serializer.").
    WithTicketRef("TK-422")

// Output:
// fix(api): handle null response in API
//
// The API was returning null for empty arrays, causing client crashes.
// This normalizes empty arrays to [] in the response serializer.
//
// Refs: TK-422
// Generated-By: devflow
```

### Integration with GitContext

```go
func (g *GitContext) CommitWithMessage(msg *CommitMessage, files ...string) error {
    // Stage files
    if err := g.Stage(files...); err != nil {
        return err
    }

    // Commit with formatted message
    return g.Commit(msg.String(), files...)
}
```

## References

- [Conventional Commits](https://www.conventionalcommits.org/)
- [Git Commit Best Practices](https://cbea.ms/git-commit/)
- [Angular Commit Guidelines](https://github.com/angular/angular/blob/main/CONTRIBUTING.md#commit)
