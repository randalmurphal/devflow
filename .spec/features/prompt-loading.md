# Feature: Prompt Loading

## Overview

Load and render prompt templates from files with Go template support.

## Use Cases

1. **Workflow prompts**: Standard prompts for spec/implement/review
2. **Custom prompts**: Project-specific prompt customization
3. **Template variables**: Dynamic content in prompts

## API

### Basic Loading

```go
loader := devflow.NewPromptLoader("/path/to/project")

prompt, err := loader.Load("generate-spec")
// Loads from: .devflow/prompts/generate-spec.txt
//         or: prompts/generate-spec.txt
//         or: embedded defaults
```

### With Variables

```go
prompt, err := loader.LoadWithVars("generate-spec", map[string]any{
    "TicketTitle":       "Add authentication",
    "TicketDescription": "Implement OAuth2...",
    "ProjectName":       "myapp",
})
```

## Prompt File Format

```text
{{/* prompts/generate-spec.txt */}}
You are a senior software architect generating a technical specification.

## Task
Create a specification for: {{.TicketTitle}}

## Description
{{.TicketDescription}}

## Project
{{.ProjectName}}

## Context Files
{{range .ContextFiles}}
### {{.Path}}
{{.Content | indent 4}}
{{end}}

## Output Format
Generate a detailed specification in markdown.
```

## Search Order

1. `.devflow/prompts/{name}.txt` (project override)
2. `prompts/{name}.txt` (project)
3. Embedded in devflow binary (defaults)

## Template Functions

| Function | Description |
|----------|-------------|
| `join` | Join slice with separator |
| `indent` | Indent text by N spaces |
| `trim` | Trim whitespace |
| `upper` | Uppercase |
| `lower` | Lowercase |

## Standard Prompts

devflow ships with these embedded prompts:

| Prompt | Purpose |
|--------|---------|
| `generate-spec` | Generate technical specification |
| `implement` | Implement code from spec |
| `review-code` | Review code for issues |
| `fix-findings` | Fix review findings |

## Example

### Prompt File

```text
{{/* .devflow/prompts/generate-spec.txt */}}
Generate a technical specification for the following feature.

# Feature Request
- **Title**: {{.Title}}
- **Priority**: {{.Priority}}

# Description
{{.Description}}

# Requirements
{{range .Requirements}}
- {{.}}
{{end}}

# Acceptance Criteria
Generate clear, testable acceptance criteria.
```

### Usage

```go
loader := devflow.NewPromptLoader(".")

prompt, err := loader.LoadWithVars("generate-spec", map[string]any{
    "Title":       "User Authentication",
    "Priority":    "High",
    "Description": "Add OAuth2 support for Google and GitHub",
    "Requirements": []string{
        "Users can log in with Google",
        "Users can log in with GitHub",
        "Sessions persist across browser restarts",
    },
})
if err != nil {
    log.Fatal(err)
}

result, err := claude.Run(ctx, prompt)
```

### Custom Project Prompts

Override defaults by placing in `.devflow/prompts/`:

```
myproject/
├── .devflow/
│   └── prompts/
│       └── generate-spec.txt    # Overrides default
└── src/
```

## Testing

```go
func TestPromptLoader(t *testing.T) {
    // Create temp directory with prompt file
    dir := t.TempDir()
    promptDir := filepath.Join(dir, "prompts")
    os.MkdirAll(promptDir, 0755)

    os.WriteFile(
        filepath.Join(promptDir, "test.txt"),
        []byte("Hello {{.Name}}!"),
        0644,
    )

    loader := devflow.NewPromptLoader(dir)
    result, err := loader.LoadWithVars("test", map[string]any{
        "Name": "World",
    })

    require.NoError(t, err)
    assert.Equal(t, "Hello World!", result)
}
```

## References

- ADR-007: Prompt Management
