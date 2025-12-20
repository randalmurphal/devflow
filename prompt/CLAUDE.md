# prompt package

Prompt template loading and variable substitution.

## Quick Reference

| Type | Purpose |
|------|---------|
| `Loader` | Loads prompt templates from files/embed |
| `Config` | Loader configuration |
| `Template` | Loaded template with substitution |

## Loader Configuration

```go
loader := prompt.NewLoader(prompt.Config{
    TemplateDir: ".devflow/prompts",  // Filesystem location
    EmbedFS:     embeddedPrompts,      // Fallback embedded FS
})
```

## Loading Templates

```go
// Load by name (searches TemplateDir, then EmbedFS)
tmpl, err := loader.Load("generate-spec")

// Load with extension
tmpl, err := loader.Load("generate-spec.txt")
```

## Variable Substitution

```go
result := tmpl.Execute(map[string]string{
    "ticket":      "TK-421",
    "title":       "Add authentication",
    "description": "Implement OAuth2 flow",
})
```

## Template Format

Templates use `{{variable}}` syntax:

```
Generate a specification for ticket {{ticket}}.

Title: {{title}}
Description: {{description}}

Output a detailed specification...
```

## Default Templates

Located in `prompts/` directory:

| Template | Purpose |
|----------|---------|
| `generate-spec.txt` | Specification generation |
| `implement.txt` | Code implementation |
| `review-code.txt` | Code review |

## File Structure

```
prompt/
├── prompt.go  # Loader, Config, Template
└── prompts/   # Default embedded templates
```
