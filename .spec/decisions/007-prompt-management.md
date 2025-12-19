# ADR-007: Prompt Management

## Status

Accepted

## Context

devflow workflows use prompts to guide Claude's behavior. We need to decide:

1. Where prompts are stored
2. How prompts are loaded
3. Whether prompts support templating
4. How prompts are versioned

## Decision

### 1. Prompts as Files

Prompts are stored as plain text files in a `prompts/` directory:

```
project/
├── .devflow/
│   └── prompts/
│       ├── generate-spec.txt
│       ├── implement.txt
│       ├── review-code.txt
│       └── fix-findings.txt
└── prompts/  (alternative location)
```

**Rationale:**
- Easy to version control
- Easy to edit and review
- No build step required
- Can be overridden per-project

### 2. Prompt Loading

```go
type PromptLoader interface {
    Load(name string) (string, error)
    LoadWithVars(name string, vars map[string]any) (string, error)
}

// FilePromptLoader loads prompts from filesystem
type FilePromptLoader struct {
    dirs []string // Directories to search (first wins)
}
```

Search order:
1. `.devflow/prompts/` in current project
2. `prompts/` in current project
3. Embedded prompts in devflow binary

### 3. Go Template Support

Prompts support Go template syntax for variable interpolation:

```text
You are implementing a feature for {{.ProjectName}}.

## Ticket
ID: {{.TicketID}}
Title: {{.TicketTitle}}
Description:
{{.TicketDescription}}

## Requirements
{{range .Requirements}}
- {{.}}
{{end}}

## Context Files
{{range .ContextFiles}}
- {{.Path}}: {{.Description}}
{{end}}
```

### 4. No Separate Versioning

Prompts are versioned with the codebase (git), not separately.

**Rationale:**
- Simplicity - no version management system
- Prompts change with code
- Git history provides versioning
- Can tag releases if needed

### 5. Embedded Defaults

devflow ships with default prompts embedded in the binary:

```go
//go:embed prompts/*.txt
var embeddedPrompts embed.FS

func loadEmbeddedPrompt(name string) (string, error) {
    data, err := embeddedPrompts.ReadFile("prompts/" + name + ".txt")
    if err != nil {
        return "", fmt.Errorf("embedded prompt not found: %s", name)
    }
    return string(data), nil
}
```

## Alternatives Considered

### Alternative 1: Prompts in Code

Define prompts as Go string constants.

**Rejected because:**
- Hard to edit (escape characters)
- Requires recompilation to change
- Can't be customized per-project

### Alternative 2: YAML/JSON Prompt Files

Store prompts in structured files with metadata.

**Rejected because:**
- Adds complexity
- Plain text is sufficient
- Metadata can be in filename or comments

### Alternative 3: Database-Backed Prompts

Store prompts in database with versioning.

**Rejected because:**
- Overkill for devflow (this is task-keeper territory)
- Adds database dependency
- Complexity without benefit

### Alternative 4: Jinja2 Templates

Use Jinja2-style templating.

**Rejected because:**
- Would need to embed Python or use a port
- Go templates are built-in and sufficient
- Team already knows Go templates

## Consequences

### Positive

- **Simple**: Plain text files
- **Customizable**: Override defaults per-project
- **Versionable**: Git tracks changes
- **Portable**: Files can be shared/copied

### Negative

- **No validation**: Template errors discovered at runtime
- **No metadata**: Can't attach schema to prompts
- **File management**: Users must manage prompt files

### Mitigations

1. **Validate on load**: Check template syntax when loading
2. **Provide examples**: Ship with well-documented default prompts
3. **CLI tooling**: Add commands to list/validate prompts

## Code Example

```go
package devflow

import (
    "bytes"
    "embed"
    "fmt"
    "os"
    "path/filepath"
    "text/template"
)

//go:embed prompts/*.txt
var embeddedPrompts embed.FS

// PromptLoader loads and renders prompt templates
type PromptLoader struct {
    dirs     []string
    cache    map[string]*template.Template
    funcMap  template.FuncMap
}

// NewPromptLoader creates a prompt loader
func NewPromptLoader(projectDir string) *PromptLoader {
    return &PromptLoader{
        dirs: []string{
            filepath.Join(projectDir, ".devflow", "prompts"),
            filepath.Join(projectDir, "prompts"),
        },
        cache:   make(map[string]*template.Template),
        funcMap: defaultFuncMap(),
    }
}

// Load loads a prompt by name
func (l *PromptLoader) Load(name string) (string, error) {
    return l.LoadWithVars(name, nil)
}

// LoadWithVars loads and renders a prompt with variables
func (l *PromptLoader) LoadWithVars(name string, vars map[string]any) (string, error) {
    tmpl, err := l.getTemplate(name)
    if err != nil {
        return "", err
    }

    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, vars); err != nil {
        return "", fmt.Errorf("render prompt %s: %w", name, err)
    }

    return buf.String(), nil
}

// getTemplate loads and caches a template
func (l *PromptLoader) getTemplate(name string) (*template.Template, error) {
    if tmpl, ok := l.cache[name]; ok {
        return tmpl, nil
    }

    content, err := l.loadRaw(name)
    if err != nil {
        return nil, err
    }

    tmpl, err := template.New(name).Funcs(l.funcMap).Parse(content)
    if err != nil {
        return nil, fmt.Errorf("parse prompt template %s: %w", name, err)
    }

    l.cache[name] = tmpl
    return tmpl, nil
}

// loadRaw loads raw prompt content
func (l *PromptLoader) loadRaw(name string) (string, error) {
    filename := name + ".txt"

    // Search directories
    for _, dir := range l.dirs {
        path := filepath.Join(dir, filename)
        if data, err := os.ReadFile(path); err == nil {
            return string(data), nil
        }
    }

    // Fall back to embedded
    data, err := embeddedPrompts.ReadFile("prompts/" + filename)
    if err != nil {
        return "", fmt.Errorf("prompt not found: %s", name)
    }

    return string(data), nil
}

// defaultFuncMap returns default template functions
func defaultFuncMap() template.FuncMap {
    return template.FuncMap{
        "join":   strings.Join,
        "indent": indentString,
        "trim":   strings.TrimSpace,
        "upper":  strings.ToUpper,
        "lower":  strings.ToLower,
    }
}

func indentString(indent int, s string) string {
    prefix := strings.Repeat(" ", indent)
    lines := strings.Split(s, "\n")
    for i, line := range lines {
        if line != "" {
            lines[i] = prefix + line
        }
    }
    return strings.Join(lines, "\n")
}
```

### Example Prompt File

```text
{{/* prompts/generate-spec.txt */}}
You are a senior software architect generating a technical specification.

## Task
Create a detailed technical specification for implementing:
{{.TicketTitle}}

## Description
{{.TicketDescription}}

## Project Context
Project: {{.ProjectName}}
Language: {{.Language}}
Framework: {{.Framework}}

## Relevant Files
{{range .ContextFiles}}
### {{.Path}}
{{.Content | indent 4}}
{{end}}

## Requirements
1. Generate a complete technical specification
2. Include:
   - Overview of the approach
   - Data structures/types needed
   - API changes (if any)
   - Database changes (if any)
   - Test plan
3. Be specific about implementation details
4. Consider edge cases and error handling

## Output Format
Output the specification in markdown format with clear sections.
```

### Usage

```go
loader := devflow.NewPromptLoader("/path/to/project")

// Simple load
prompt, err := loader.Load("generate-spec")

// With variables
prompt, err := loader.LoadWithVars("generate-spec", map[string]any{
    "TicketTitle":       "Add user authentication",
    "TicketDescription": "Implement OAuth2 login with Google and GitHub",
    "ProjectName":       "myapp",
    "Language":          "Go",
    "Framework":         "Echo",
    "ContextFiles": []map[string]string{
        {"Path": "auth/handler.go", "Content": "..."},
        {"Path": "models/user.go", "Content": "..."},
    },
})

// Use with Claude
result, err := claude.Run(ctx, prompt,
    devflow.WithSystemPrompt("You are an expert architect"),
)
```

### Integration with Nodes

```go
func GenerateSpecNode(ctx flowgraph.Context, state DevState) (DevState, error) {
    claude := ctx.Value(claudeKey).(*ClaudeCLI)
    loader := ctx.Value(promptLoaderKey).(*PromptLoader)

    prompt, err := loader.LoadWithVars("generate-spec", map[string]any{
        "TicketTitle":       state.Ticket.Title,
        "TicketDescription": state.Ticket.Description,
        "ProjectName":       state.ProjectName,
        "ContextFiles":      state.ContextFiles,
    })
    if err != nil {
        return state, fmt.Errorf("load prompt: %w", err)
    }

    result, err := claude.Run(ctx, prompt)
    if err != nil {
        return state, fmt.Errorf("generate spec: %w", err)
    }

    state.Spec = &Spec{
        Content:  result.Output,
        TokensIn: result.TokensIn,
        TokensOut: result.TokensOut,
    }

    return state, nil
}
```

## References

- [Go Templates Documentation](https://pkg.go.dev/text/template)
- [Go embed Documentation](https://pkg.go.dev/embed)
- ADR-006: Claude CLI Wrapper
