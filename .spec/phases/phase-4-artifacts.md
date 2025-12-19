# Phase 4: Artifact Management

## Overview

Implement artifact storage for workflow outputs.

**Duration**: Week 4
**Dependencies**: Phase 3 (Transcripts - shares directory structure)
**Deliverables**: `ArtifactManager` with save/load and lifecycle management

---

## Goals

1. Store workflow artifacts (specs, diffs, reviews)
2. Support compression for large files
3. Implement retention and cleanup policies
4. Define standard artifact types

---

## Components

### ArtifactManager

Central type for artifact operations:

```go
type ArtifactManager struct {
    baseDir       string
    compressAbove int64
    retentionDays int
}
```

### ArtifactInfo

Metadata about a stored artifact:

```go
type ArtifactInfo struct {
    Name       string
    Size       int64
    Compressed bool
    CreatedAt  time.Time
    Type       string
}
```

---

## Implementation Tasks

### Task 4.1: ArtifactManager Constructor

```go
func NewArtifactManager(cfg ArtifactConfig) *ArtifactManager

type ArtifactConfig struct {
    BaseDir       string // Default: ".devflow"
    CompressAbove int64  // Default: 10KB
    RetentionDays int    // Default: 30
}
```

**Acceptance Criteria**:
- [ ] Creates directory structure
- [ ] Sets sensible defaults
- [ ] Validates configuration

### Task 4.2: Save/Load Operations

```go
func (m *ArtifactManager) SaveArtifact(runID, name string, data []byte) error
func (m *ArtifactManager) LoadArtifact(runID, name string) ([]byte, error)
```

**Acceptance Criteria**:
- [ ] Saves to `runs/{runID}/artifacts/{name}`
- [ ] Compresses large files transparently
- [ ] Decompresses on load transparently
- [ ] Returns ErrArtifactNotFound if missing

### Task 4.3: Listing

```go
func (m *ArtifactManager) ListArtifacts(runID string) ([]ArtifactInfo, error)
```

**Acceptance Criteria**:
- [ ] Lists all artifacts for a run
- [ ] Includes size and compression status
- [ ] Sorted by name

### Task 4.4: Compression

```go
func (m *ArtifactManager) shouldCompress(name string, size int64) bool
func (m *ArtifactManager) saveCompressed(path string, data []byte) error
func (m *ArtifactManager) loadCompressed(path string) ([]byte, error)
```

**Acceptance Criteria**:
- [ ] Compresses if size > threshold
- [ ] Only compresses compressible types (not images)
- [ ] Uses gzip format
- [ ] Appends `.gz` extension

### Task 4.5: Standard Artifact Types

```go
const (
    ArtifactSpec           = "spec.md"
    ArtifactImplementation = "implementation.diff"
    ArtifactReview         = "review.json"
    ArtifactTestOutput     = "test-output.json"
    ArtifactLintOutput     = "lint-output.json"
)
```

Type helpers:

```go
func (m *ArtifactManager) SaveSpec(runID string, spec string) error
func (m *ArtifactManager) SaveReview(runID string, review *ReviewResult) error
func (m *ArtifactManager) LoadReview(runID string) (*ReviewResult, error)
```

**Acceptance Criteria**:
- [ ] Standard names are used consistently
- [ ] Type helpers handle serialization
- [ ] JSON types validated on load

### Task 4.6: Lifecycle Manager

```go
type LifecycleManager struct {
    baseDir string
    config  RetentionConfig
}

type RetentionConfig struct {
    RetentionDays       int  // Days to keep active runs
    ArchiveAfterDays    int  // Days before archiving
    ArchiveRetentionDays int // Days to keep archives
    KeepFailed          bool // Keep failed runs longer
}

func (m *LifecycleManager) Cleanup(dryRun bool) (*CleanupResult, error)
```

**Acceptance Criteria**:
- [ ] Identifies runs to archive/delete
- [ ] Respects retention policy
- [ ] Dry-run shows what would happen
- [ ] Archives compress to tar.gz

### Task 4.7: Archive Operations

```go
func (m *LifecycleManager) archiveRun(runID string) error
func (m *LifecycleManager) RestoreArchive(runID string) error
```

**Acceptance Criteria**:
- [ ] Creates `.devflow/archive/YYYY-MM/{runID}.tar.gz`
- [ ] Removes original after successful archive
- [ ] Restore extracts to runs directory

---

## Directory Structure

```
.devflow/
├── runs/
│   └── 2025-01-15-ticket-to-pr-TK421/
│       ├── metadata.json
│       ├── transcript.json
│       └── artifacts/
│           ├── spec.md
│           ├── implementation.diff
│           ├── review.json
│           └── files/
│               ├── api.go
│               └── api_test.go
└── archive/
    └── 2025-01/
        └── 2025-01-08-run-old.tar.gz
```

---

## Testing Strategy

### Unit Tests

| Test | Description |
|------|-------------|
| `TestArtifactManager_SaveLoad` | Basic save/load |
| `TestArtifactManager_Compression` | Compresses large files |
| `TestArtifactManager_List` | Lists artifacts |
| `TestLifecycleManager_Cleanup` | Applies retention |
| `TestLifecycleManager_Archive` | Archives correctly |

### Integration Tests

```go
func TestArtifactManager_Integration(t *testing.T) {
    dir := t.TempDir()
    manager := NewArtifactManager(ArtifactConfig{BaseDir: dir})

    runID := "test-run-001"

    // Save
    manager.SaveArtifact(runID, "spec.md", []byte("# Spec\n..."))

    // Load
    data, err := manager.LoadArtifact(runID, "spec.md")
    require.NoError(t, err)
    assert.Contains(t, string(data), "# Spec")

    // List
    artifacts, _ := manager.ListArtifacts(runID)
    assert.Len(t, artifacts, 1)
}
```

---

## Error Handling

| Error | Condition | Recovery |
|-------|-----------|----------|
| `ErrArtifactNotFound` | Artifact doesn't exist | Check name |
| `ErrRunNotFound` | Run doesn't exist | Check run ID |

---

## File Structure

```
devflow/
├── artifact.go          # ArtifactManager
├── artifact_types.go    # Standard types and helpers
├── artifact_lifecycle.go # LifecycleManager
└── artifact_test.go     # Tests
```

---

## Completion Criteria

- [ ] All tasks implemented
- [ ] Unit test coverage > 80%
- [ ] Integration tests pass
- [ ] Compression works correctly
- [ ] Lifecycle management functional
- [ ] Archive/restore works

---

## References

- ADR-015: Artifact Structure
- ADR-016: Artifact Lifecycle
- ADR-017: Artifact Types
