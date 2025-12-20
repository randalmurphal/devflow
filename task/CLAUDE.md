# task package

Task-based model selection for LLM operations.

## Quick Reference

| Type | Purpose |
|------|---------|
| `TaskType` | Type of task constant |
| `Selector` | Selects model based on task |
| `Config` | Model configuration per task |

## Task Types

| Constant | Purpose | Recommended Model |
|----------|---------|-------------------|
| `TaskInvestigation` | Code exploration, impact analysis | Opus |
| `TaskImplementation` | Writing code, making changes | Sonnet |
| `TaskReview` | Code review, validation | Opus |
| `TaskArchitecture` | Design decisions | Opus |
| `TaskSimple` | Quick searches, formatting | Haiku |

## Selector Configuration

```go
selector := task.NewSelector(task.Config{
    Investigation:   "claude-opus-4-20250514",
    Implementation:  "claude-sonnet-4-20250514",
    Review:          "claude-opus-4-20250514",
    Architecture:    "claude-opus-4-20250514",
    Simple:          "claude-haiku-3-5-20241022",
})
```

## Model Selection

```go
// Get model for task type
model := selector.ModelFor(task.TaskReview)
// Returns: "claude-opus-4-20250514"

// With fallback
model := selector.ModelForWithDefault(task.TaskCustom, "claude-sonnet-4-20250514")
```

## Integration with flowgraph

```go
import "github.com/rmurphy/flowgraph/pkg/flowgraph/llm"

selector := task.NewSelector(config)
model := selector.ModelFor(task.TaskImplementation)

client := llm.NewClaudeCLI(
    llm.WithModel(model),
    llm.WithWorkdir(repoPath),
)
```

## File Structure

```
task/
└── task.go  # TaskType, Selector, Config
```
