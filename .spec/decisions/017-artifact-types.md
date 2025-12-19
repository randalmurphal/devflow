# ADR-017: Artifact Types

## Status

Accepted

## Context

Different workflow stages produce different artifact types:

1. Specifications (markdown)
2. Code changes (diffs, files)
3. Review results (structured JSON)
4. Test outputs (text/JSON)
5. Binary outputs (images, PDFs)

We need to define how different artifact types are handled.

## Decision

### 1. Core Artifact Types

| Type | Extension | Description |
|------|-----------|-------------|
| Specification | `.md` | Generated specs, plans |
| Diff | `.diff`, `.patch` | Code changes |
| JSON | `.json` | Structured data (reviews, results) |
| Text | `.txt`, `.log` | Unstructured output |
| Code | `.go`, `.py`, etc. | Generated source files |
| Binary | various | Images, PDFs, etc. |

### 2. Artifact Registry

```go
type ArtifactType struct {
    Name        string   // "specification", "diff", etc.
    Extensions  []string // [".md"], [".diff", ".patch"]
    Compressible bool    // Can be gzipped
    Searchable  bool     // Include in content search
}

var ArtifactTypes = map[string]ArtifactType{
    "specification": {
        Name:        "specification",
        Extensions:  []string{".md"},
        Compressible: true,
        Searchable:  true,
    },
    "diff": {
        Name:        "diff",
        Extensions:  []string{".diff", ".patch"},
        Compressible: true,
        Searchable:  true,
    },
    "review": {
        Name:        "review",
        Extensions:  []string{".json"},
        Compressible: true,
        Searchable:  true,
    },
    "test-output": {
        Name:        "test-output",
        Extensions:  []string{".txt", ".log", ".json"},
        Compressible: true,
        Searchable:  true,
    },
    "code": {
        Name:        "code",
        Extensions:  []string{".go", ".py", ".js", ".ts"},
        Compressible: true,
        Searchable:  true,
    },
    "binary": {
        Name:        "binary",
        Extensions:  []string{".png", ".jpg", ".pdf"},
        Compressible: false,
        Searchable:  false,
    },
}
```

### 3. Standard Artifact Schemas

**Review Result** (`review.json`):
```json
{
  "approved": false,
  "summary": "Found 3 issues that need addressing",
  "findings": [
    {
      "file": "api/handler.go",
      "line": 45,
      "severity": "error",
      "category": "security",
      "message": "SQL injection vulnerability",
      "suggestion": "Use parameterized queries"
    }
  ],
  "metrics": {
    "linesReviewed": 234,
    "filesReviewed": 5,
    "tokensUsed": 3500
  }
}
```

**Specification** (`spec.md`):
```markdown
# Technical Specification: [Title]

## Overview
[Brief description]

## Requirements
- [Requirement 1]
- [Requirement 2]

## Design
[Design details]

## API Changes
[If any]

## Database Changes
[If any]

## Test Plan
[How to verify]

## Risks
[Potential issues]
```

**Test Output** (`test-output.json`):
```json
{
  "passed": true,
  "totalTests": 45,
  "passedTests": 44,
  "failedTests": 1,
  "skippedTests": 0,
  "duration": "12.5s",
  "failures": [
    {
      "name": "TestUserAuth",
      "message": "expected 200, got 401",
      "file": "auth_test.go",
      "line": 34
    }
  ]
}
```

### 4. Compression Strategy

| Artifact Size | Action |
|---------------|--------|
| < 10KB | Store as-is |
| 10KB - 100KB | Compress if text-based |
| > 100KB | Always compress (if compressible) |

```go
func (m *ArtifactManager) shouldCompress(artifactType string, size int64) bool {
    at, ok := ArtifactTypes[artifactType]
    if !ok || !at.Compressible {
        return false
    }

    if size < 10*1024 {
        return false
    }

    return true
}
```

### 5. Artifact Validation

```go
type ArtifactValidator interface {
    Validate(artifactType string, data []byte) error
}

// Validate JSON artifacts against schema
func ValidateJSON(data []byte, schema string) error {
    // Use json schema validation
}

// Validate diff format
func ValidateDiff(data []byte) error {
    // Check it's valid unified diff
}
```

## Alternatives Considered

### Alternative 1: Free-Form Artifacts

Allow any file without type system.

**Rejected because:**
- No validation possible
- Harder to search
- Inconsistent formats

### Alternative 2: Strict Type System

Require explicit type declaration for all artifacts.

**Rejected because:**
- Too rigid
- Extension-based inference is simpler
- Explicit types add ceremony

### Alternative 3: Database-Stored Artifacts

Store artifact content in database.

**Rejected because:**
- Binary files awkward
- Large files problematic
- Files are simpler

## Consequences

### Positive

- **Structured**: Standard schemas for common types
- **Searchable**: Text artifacts can be searched
- **Compressible**: Saves disk space
- **Extensible**: Easy to add new types

### Negative

- **Schema drift**: Schemas may evolve
- **Inference**: Extension-based detection can be wrong
- **Validation**: Validation adds overhead

## Code Example

```go
package devflow

import (
    "compress/gzip"
    "encoding/json"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "strings"
)

// ArtifactType describes an artifact type
type ArtifactType struct {
    Name         string
    Extensions   []string
    Compressible bool
    Searchable   bool
}

// KnownArtifactTypes maps type names to their definitions
var KnownArtifactTypes = map[string]ArtifactType{
    "specification": {"specification", []string{".md"}, true, true},
    "diff":          {"diff", []string{".diff", ".patch"}, true, true},
    "review":        {"review", []string{".json"}, true, true},
    "test-output":   {"test-output", []string{".txt", ".log", ".json"}, true, true},
    "code":          {"code", []string{".go", ".py", ".js", ".ts"}, true, true},
    "binary":        {"binary", []string{".png", ".jpg", ".pdf", ".zip"}, false, false},
}

// InferArtifactType infers type from filename
func InferArtifactType(filename string) ArtifactType {
    ext := strings.ToLower(filepath.Ext(filename))

    for _, at := range KnownArtifactTypes {
        for _, e := range at.Extensions {
            if e == ext {
                return at
            }
        }
    }

    // Default to text
    return ArtifactType{
        Name:         "unknown",
        Compressible: true,
        Searchable:   true,
    }
}

// ArtifactManager with type handling
type ArtifactManagerV2 struct {
    baseDir           string
    compressThreshold int64
}

// SaveArtifact saves an artifact with automatic compression
func (m *ArtifactManagerV2) SaveArtifact(runID, name string, data []byte) error {
    artifactType := InferArtifactType(name)
    artifactPath := filepath.Join(m.baseDir, "runs", runID, "artifacts", name)

    // Ensure directory exists
    if err := os.MkdirAll(filepath.Dir(artifactPath), 0755); err != nil {
        return err
    }

    // Compress if needed
    if m.shouldCompress(artifactType, int64(len(data))) {
        return m.saveCompressed(artifactPath, data)
    }

    return os.WriteFile(artifactPath, data, 0644)
}

func (m *ArtifactManagerV2) shouldCompress(at ArtifactType, size int64) bool {
    if !at.Compressible {
        return false
    }
    return size >= m.compressThreshold
}

func (m *ArtifactManagerV2) saveCompressed(path string, data []byte) error {
    f, err := os.Create(path + ".gz")
    if err != nil {
        return err
    }
    defer f.Close()

    gz := gzip.NewWriter(f)
    defer gz.Close()

    _, err = gz.Write(data)
    return err
}

// LoadArtifact loads an artifact (handles compression transparently)
func (m *ArtifactManagerV2) LoadArtifact(runID, name string) ([]byte, error) {
    artifactPath := filepath.Join(m.baseDir, "runs", runID, "artifacts", name)

    // Try compressed first
    if data, err := m.loadCompressed(artifactPath + ".gz"); err == nil {
        return data, nil
    }

    // Try uncompressed
    return os.ReadFile(artifactPath)
}

func (m *ArtifactManagerV2) loadCompressed(path string) ([]byte, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer f.Close()

    gz, err := gzip.NewReader(f)
    if err != nil {
        return nil, err
    }
    defer gz.Close()

    return io.ReadAll(gz)
}

// Review result types
type ReviewResult struct {
    Approved bool           `json:"approved"`
    Summary  string         `json:"summary"`
    Findings []ReviewFinding `json:"findings"`
    Metrics  ReviewMetrics  `json:"metrics"`
}

type ReviewFinding struct {
    File       string `json:"file"`
    Line       int    `json:"line"`
    Severity   string `json:"severity"` // error, warning, info
    Category   string `json:"category"` // security, performance, style
    Message    string `json:"message"`
    Suggestion string `json:"suggestion,omitempty"`
}

type ReviewMetrics struct {
    LinesReviewed int `json:"linesReviewed"`
    FilesReviewed int `json:"filesReviewed"`
    TokensUsed    int `json:"tokensUsed"`
}

// SaveReview saves a review result
func (m *ArtifactManagerV2) SaveReview(runID string, review *ReviewResult) error {
    data, err := json.MarshalIndent(review, "", "  ")
    if err != nil {
        return err
    }
    return m.SaveArtifact(runID, "review.json", data)
}

// LoadReview loads a review result
func (m *ArtifactManagerV2) LoadReview(runID string) (*ReviewResult, error) {
    data, err := m.LoadArtifact(runID, "review.json")
    if err != nil {
        return nil, err
    }

    var review ReviewResult
    if err := json.Unmarshal(data, &review); err != nil {
        return nil, err
    }

    return &review, nil
}

// Test output types
type TestOutput struct {
    Passed       bool          `json:"passed"`
    TotalTests   int           `json:"totalTests"`
    PassedTests  int           `json:"passedTests"`
    FailedTests  int           `json:"failedTests"`
    SkippedTests int           `json:"skippedTests"`
    Duration     string        `json:"duration"`
    Failures     []TestFailure `json:"failures,omitempty"`
}

type TestFailure struct {
    Name    string `json:"name"`
    Message string `json:"message"`
    File    string `json:"file,omitempty"`
    Line    int    `json:"line,omitempty"`
}

// SaveTestOutput saves test results
func (m *ArtifactManagerV2) SaveTestOutput(runID string, output *TestOutput) error {
    data, err := json.MarshalIndent(output, "", "  ")
    if err != nil {
        return err
    }
    return m.SaveArtifact(runID, "test-output.json", data)
}
```

### Usage

```go
manager := devflow.NewArtifactManagerV2(".devflow")
manager.compressThreshold = 10 * 1024 // 10KB

runID := "2025-01-15-ticket-to-pr-TK421"

// Save specification
spec := []byte("# Technical Specification\n\n...")
manager.SaveArtifact(runID, "spec.md", spec)

// Save review result
review := &devflow.ReviewResult{
    Approved: false,
    Summary:  "Found security issues",
    Findings: []devflow.ReviewFinding{
        {
            File:     "api/handler.go",
            Line:     45,
            Severity: "error",
            Category: "security",
            Message:  "SQL injection vulnerability",
        },
    },
}
manager.SaveReview(runID, review)

// Save test output
tests := &devflow.TestOutput{
    Passed:      true,
    TotalTests:  45,
    PassedTests: 45,
    Duration:    "12.5s",
}
manager.SaveTestOutput(runID, tests)

// Load review later
loaded, _ := manager.LoadReview(runID)
if !loaded.Approved {
    for _, f := range loaded.Findings {
        fmt.Printf("[%s] %s:%d - %s\n", f.Severity, f.File, f.Line, f.Message)
    }
}
```

## References

- ADR-015: Artifact Structure
- ADR-016: Artifact Lifecycle
- ADR-009: Output Parsing (review result parsing)
