# artifact package

Workflow artifact storage and lifecycle management.

## Quick Reference

| Type | Purpose |
|------|---------|
| `Manager` | Save/load artifacts for workflow runs |
| `Config` | Manager configuration |
| `Info` | Artifact metadata |
| `LifecycleManager` | Cleanup, archival, retention |
| `ReviewResult` | Code review findings |
| `TestOutput` | Test execution results |
| `LintOutput` | Linting results |
| `Specification` | Feature specification |

## Manager Operations

```go
mgr := artifact.NewManager(artifact.Config{
    BaseDir:       ".devflow/runs",
    CompressAbove: 1024, // Compress files > 1KB
})

// Save artifact
err := mgr.SaveArtifact("run-123", "output.json", data)

// Load artifact
data, err := mgr.LoadArtifact("run-123", "output.json")

// List artifacts
list, err := mgr.ListArtifacts("run-123")

// Delete artifact
err := mgr.DeleteArtifact("run-123", "output.json")
```

## Artifact Types

| Type Constant | Purpose |
|---------------|---------|
| `TypeSpec` | Feature specification |
| `TypeReview` | Code review result |
| `TypeTestOutput` | Test execution output |
| `TypeLintOutput` | Linting output |
| `TypeImplementation` | Generated code |
| `TypeDiff` | Git diff |

## ReviewResult Structure

```go
type ReviewResult struct {
    Approved     bool      `json:"approved"`
    Findings     []Finding `json:"findings"`
    Summary      string    `json:"summary"`
    ReviewedAt   time.Time `json:"reviewedAt"`
    ReviewedBy   string    `json:"reviewedBy"`
}

type Finding struct {
    Severity    string `json:"severity"` // error, warning, info
    Category    string `json:"category"` // security, performance, etc.
    Message     string `json:"message"`
    File        string `json:"file"`
    Line        int    `json:"line"`
    Suggestion  string `json:"suggestion,omitempty"`
}
```

## Lifecycle Management

```go
lifecycle := artifact.NewLifecycleManager(mgr, artifact.LifecycleConfig{
    RetentionDays:  30,
    ArchiveAfter:   7,
    CompressOnArchive: true,
})

// Cleanup old runs
deleted, err := lifecycle.Cleanup()

// Archive runs
archived, err := lifecycle.Archive()
```

## File Structure

```
artifact/
├── artifact.go   # Manager, Config, Info
├── types.go      # ReviewResult, TestOutput, etc.
└── lifecycle.go  # LifecycleManager
```
