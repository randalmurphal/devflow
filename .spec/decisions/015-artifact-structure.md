# ADR-015: Artifact Directory Structure

## Status

Accepted

## Context

Workflow runs produce artifacts:

1. Generated specifications
2. Code diffs and patches
3. Review results
4. Test outputs
5. Intermediate files

We need a consistent directory structure for storing these artifacts.

## Decision

### 1. Run-Based Directory Structure

Artifacts are stored per-run in a predictable structure:

```
.devflow/
└── runs/
    └── 2025-01-15-ticket-to-pr-TK421/
        ├── metadata.json           # Run metadata
        ├── transcript.json         # Conversation transcript
        └── artifacts/              # Generated artifacts
            ├── spec.md             # Generated specification
            ├── implementation.diff # Code changes diff
            ├── review.json         # Review results
            ├── test-output.txt     # Test results
            └── files/              # Generated files
                ├── api.go
                └── api_test.go
```

### 2. Standard Artifact Names

| Artifact | Filename | Format |
|----------|----------|--------|
| Specification | `spec.md` | Markdown |
| Implementation diff | `implementation.diff` | Unified diff |
| Review results | `review.json` | JSON |
| Test output | `test-output.txt` | Text |
| Lint output | `lint-output.json` | JSON |
| Generated files | `files/` | Directory |

### 3. Artifact Metadata

Each artifact can have metadata in a `.meta.json` suffix:

```
artifacts/
├── spec.md
├── spec.md.meta.json      # Optional metadata
└── review.json
```

Metadata structure:
```json
{
  "createdAt": "2025-01-15T10:35:00Z",
  "createdBy": "generate-spec",
  "tokensUsed": 2500,
  "compressed": false,
  "originalSize": 4567
}
```

### 4. File-Based (No Database)

Artifacts are plain files:
- Easy to inspect
- Easy to backup
- Works with git
- No external dependencies

### 5. Path Conventions

```go
// RunDir returns the directory for a run
func (m *ArtifactManager) RunDir(runID string) string {
    return filepath.Join(m.baseDir, "runs", runID)
}

// ArtifactPath returns path to an artifact
func (m *ArtifactManager) ArtifactPath(runID, name string) string {
    return filepath.Join(m.RunDir(runID), "artifacts", name)
}

// FilePath returns path to a generated file
func (m *ArtifactManager) FilePath(runID, filename string) string {
    return filepath.Join(m.RunDir(runID), "artifacts", "files", filename)
}
```

## Alternatives Considered

### Alternative 1: Single Artifacts Directory

Store all artifacts in one flat directory.

**Rejected because:**
- No organization by run
- Naming conflicts
- Hard to cleanup old runs

### Alternative 2: Database Storage

Store artifacts in SQLite or similar.

**Rejected because:**
- Harder to inspect
- Binary files awkward
- Files are simpler

### Alternative 3: Content-Addressable Storage

Use hashes for deduplication.

**Deferred:**
- Good for large-scale
- Overkill for now
- Can add later

## Consequences

### Positive

- **Organized**: Clear structure per run
- **Inspectable**: Plain files
- **Portable**: Easy to copy/backup
- **Simple**: No database needed

### Negative

- **Disk usage**: No deduplication
- **Scale**: Many runs = many directories
- **Cleanup**: Manual or cron-based

## Code Example

```go
package devflow

import (
    "os"
    "path/filepath"
)

// ArtifactManager manages run artifacts
type ArtifactManager struct {
    baseDir string
}

// NewArtifactManager creates an artifact manager
func NewArtifactManager(baseDir string) *ArtifactManager {
    return &ArtifactManager{baseDir: baseDir}
}

// RunDir returns the directory for a run
func (m *ArtifactManager) RunDir(runID string) string {
    return filepath.Join(m.baseDir, "runs", runID)
}

// ArtifactDir returns the artifacts directory for a run
func (m *ArtifactManager) ArtifactDir(runID string) string {
    return filepath.Join(m.RunDir(runID), "artifacts")
}

// FilesDir returns the generated files directory
func (m *ArtifactManager) FilesDir(runID string) string {
    return filepath.Join(m.ArtifactDir(runID), "files")
}

// EnsureRunDir creates the run directory structure
func (m *ArtifactManager) EnsureRunDir(runID string) error {
    dirs := []string{
        m.RunDir(runID),
        m.ArtifactDir(runID),
        m.FilesDir(runID),
    }

    for _, dir := range dirs {
        if err := os.MkdirAll(dir, 0755); err != nil {
            return err
        }
    }

    return nil
}

// Standard artifact names
const (
    ArtifactSpec           = "spec.md"
    ArtifactImplementation = "implementation.diff"
    ArtifactReview         = "review.json"
    ArtifactTestOutput     = "test-output.txt"
    ArtifactLintOutput     = "lint-output.json"
)
```

### Usage

```go
manager := devflow.NewArtifactManager(".devflow")

// Setup run directory
runID := "2025-01-15-ticket-to-pr-TK421"
manager.EnsureRunDir(runID)

// Paths are predictable
specPath := filepath.Join(manager.ArtifactDir(runID), devflow.ArtifactSpec)
// = ".devflow/runs/2025-01-15-ticket-to-pr-TK421/artifacts/spec.md"

filesDir := manager.FilesDir(runID)
// = ".devflow/runs/2025-01-15-ticket-to-pr-TK421/artifacts/files"
```

## References

- ADR-012: Transcript Storage (shares runs directory)
- ADR-016: Artifact Lifecycle
- ADR-017: Artifact Types
