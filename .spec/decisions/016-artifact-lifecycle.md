# ADR-016: Artifact Lifecycle

## Status

Accepted

## Context

Artifacts need lifecycle management:

1. Creation during workflow execution
2. Retention based on policy
3. Cleanup of old artifacts
4. Archival of important runs

We need to define how artifacts move through their lifecycle.

## Decision

### 1. Lifecycle States

```
Created → Active → [Archived | Deleted]
```

| State | Description | Location |
|-------|-------------|----------|
| Active | Recent runs, quick access | `.devflow/runs/` |
| Archived | Old but retained | `.devflow/archive/` |
| Deleted | Removed permanently | N/A |

### 2. Retention Policy

Default retention:
- **Active**: Keep all runs from last 30 days
- **Archive**: Keep completed runs for 90 days
- **Delete**: Remove after archive period

Configurable in `.devflow/config.json`:

```json
{
  "artifacts": {
    "retentionDays": 30,
    "archiveAfterDays": 7,
    "archiveRetentionDays": 90,
    "keepFailed": true,
    "keepMinRuns": 100
  }
}
```

### 3. Cleanup Process

Cleanup is manual or cron-triggered, not automatic:

```bash
# Manual cleanup
devflow cleanup --dry-run
devflow cleanup

# Cron (daily)
0 3 * * * cd /project && devflow cleanup --quiet
```

### 4. Archive Process

Archive moves runs to compressed storage:

```
.devflow/
├── runs/           # Active runs
│   └── 2025-01-15-run-001/
└── archive/        # Archived runs
    └── 2025-01/
        └── 2025-01-08-run-001.tar.gz
```

### 5. No Automatic Deletion

devflow never deletes without explicit action:
- Manual `devflow cleanup` command
- Explicit `--delete` flag required
- Dry-run shows what would be deleted

## Alternatives Considered

### Alternative 1: Automatic Cleanup

Clean up automatically in background.

**Rejected because:**
- Surprising data loss
- Hard to predict
- Better to be explicit

### Alternative 2: Database Tracking

Track lifecycle in database.

**Rejected because:**
- Files already have timestamps
- Unnecessary complexity
- File operations sufficient

### Alternative 3: No Archive

Just delete old runs.

**Rejected because:**
- May need old runs for audit
- Archive is low cost
- Compression saves space

## Consequences

### Positive

- **Predictable**: Clear lifecycle states
- **Safe**: No automatic deletion
- **Configurable**: Retention policy is adjustable
- **Space-efficient**: Archive compresses old runs

### Negative

- **Manual**: Requires cleanup command
- **Disk growth**: Without cleanup, grows unbounded
- **Complexity**: Archive adds another location

### Mitigations

1. **Warnings**: Warn when disk usage high
2. **CLI reminders**: Suggest cleanup in status output
3. **Documentation**: Clear cleanup instructions

## Code Example

```go
package devflow

import (
    "archive/tar"
    "compress/gzip"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "time"
)

// RetentionConfig defines retention policy
type RetentionConfig struct {
    RetentionDays       int  // Days to keep active runs
    ArchiveAfterDays    int  // Days before archiving
    ArchiveRetentionDays int // Days to keep archived runs
    KeepFailed          bool // Keep failed runs longer
    KeepMinRuns         int  // Minimum runs to keep regardless of age
}

// DefaultRetentionConfig returns sensible defaults
func DefaultRetentionConfig() RetentionConfig {
    return RetentionConfig{
        RetentionDays:        30,
        ArchiveAfterDays:     7,
        ArchiveRetentionDays: 90,
        KeepFailed:           true,
        KeepMinRuns:          100,
    }
}

// LifecycleManager handles artifact lifecycle
type LifecycleManager struct {
    baseDir string
    config  RetentionConfig
}

// NewLifecycleManager creates a lifecycle manager
func NewLifecycleManager(baseDir string, config RetentionConfig) *LifecycleManager {
    return &LifecycleManager{
        baseDir: baseDir,
        config:  config,
    }
}

// CleanupResult summarizes cleanup actions
type CleanupResult struct {
    Archived []string
    Deleted  []string
    Kept     []string
    Errors   []error
    SpaceSaved int64
}

// Cleanup performs retention policy
func (m *LifecycleManager) Cleanup(dryRun bool) (*CleanupResult, error) {
    result := &CleanupResult{}

    runsDir := filepath.Join(m.baseDir, "runs")
    entries, err := os.ReadDir(runsDir)
    if err != nil {
        return nil, err
    }

    now := time.Now()
    archiveThreshold := now.Add(-time.Duration(m.config.ArchiveAfterDays) * 24 * time.Hour)
    deleteThreshold := now.Add(-time.Duration(m.config.RetentionDays) * 24 * time.Hour)

    for _, entry := range entries {
        if !entry.IsDir() {
            continue
        }

        runID := entry.Name()
        runDir := filepath.Join(runsDir, runID)

        // Load metadata
        meta, err := loadRunMetadata(runDir)
        if err != nil {
            result.Errors = append(result.Errors, fmt.Errorf("load %s: %w", runID, err))
            continue
        }

        // Skip failed runs if configured
        if m.config.KeepFailed && meta.Status == RunStatusFailed {
            result.Kept = append(result.Kept, runID)
            continue
        }

        // Determine action
        if meta.EndedAt.Before(deleteThreshold) {
            // Delete
            if !dryRun {
                if err := os.RemoveAll(runDir); err != nil {
                    result.Errors = append(result.Errors, err)
                    continue
                }
            }
            result.Deleted = append(result.Deleted, runID)

        } else if meta.EndedAt.Before(archiveThreshold) {
            // Archive
            if !dryRun {
                if err := m.archiveRun(runID); err != nil {
                    result.Errors = append(result.Errors, err)
                    continue
                }
            }
            result.Archived = append(result.Archived, runID)

        } else {
            result.Kept = append(result.Kept, runID)
        }
    }

    // Ensure minimum runs kept
    if len(result.Kept) < m.config.KeepMinRuns {
        // Move some from deleted/archived back to kept
        // Implementation omitted for brevity
    }

    return result, nil
}

// archiveRun compresses a run to archive
func (m *LifecycleManager) archiveRun(runID string) error {
    runDir := filepath.Join(m.baseDir, "runs", runID)

    // Determine archive path
    // Extract date from runID (e.g., "2025-01-15-...")
    archiveMonth := runID[:7] // "2025-01"
    archiveDir := filepath.Join(m.baseDir, "archive", archiveMonth)
    if err := os.MkdirAll(archiveDir, 0755); err != nil {
        return err
    }

    archivePath := filepath.Join(archiveDir, runID+".tar.gz")

    // Create archive
    f, err := os.Create(archivePath)
    if err != nil {
        return err
    }
    defer f.Close()

    gz := gzip.NewWriter(f)
    defer gz.Close()

    tw := tar.NewWriter(gz)
    defer tw.Close()

    // Add all files from run directory
    err = filepath.Walk(runDir, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }

        header, err := tar.FileInfoHeader(info, "")
        if err != nil {
            return err
        }

        relPath, _ := filepath.Rel(runDir, path)
        header.Name = filepath.Join(runID, relPath)

        if err := tw.WriteHeader(header); err != nil {
            return err
        }

        if !info.IsDir() {
            file, err := os.Open(path)
            if err != nil {
                return err
            }
            defer file.Close()
            io.Copy(tw, file)
        }

        return nil
    })

    if err != nil {
        os.Remove(archivePath)
        return err
    }

    // Remove original
    return os.RemoveAll(runDir)
}

// RestoreArchive restores an archived run
func (m *LifecycleManager) RestoreArchive(runID string) error {
    // Find archive
    archivePath := m.findArchive(runID)
    if archivePath == "" {
        return fmt.Errorf("archive not found: %s", runID)
    }

    // Extract
    runDir := filepath.Join(m.baseDir, "runs", runID)
    return m.extractArchive(archivePath, runDir)
}

func (m *LifecycleManager) findArchive(runID string) string {
    archiveMonth := runID[:7]
    path := filepath.Join(m.baseDir, "archive", archiveMonth, runID+".tar.gz")
    if _, err := os.Stat(path); err == nil {
        return path
    }
    return ""
}

func (m *LifecycleManager) extractArchive(archivePath, destDir string) error {
    f, err := os.Open(archivePath)
    if err != nil {
        return err
    }
    defer f.Close()

    gz, err := gzip.NewReader(f)
    if err != nil {
        return err
    }
    defer gz.Close()

    tr := tar.NewReader(gz)

    for {
        header, err := tr.Next()
        if err == io.EOF {
            break
        }
        if err != nil {
            return err
        }

        // Strip leading directory (runID)
        parts := strings.SplitN(header.Name, "/", 2)
        if len(parts) < 2 {
            continue
        }
        target := filepath.Join(destDir, parts[1])

        if header.Typeflag == tar.TypeDir {
            os.MkdirAll(target, 0755)
        } else {
            os.MkdirAll(filepath.Dir(target), 0755)
            out, err := os.Create(target)
            if err != nil {
                return err
            }
            io.Copy(out, tr)
            out.Close()
        }
    }

    return nil
}

func loadRunMetadata(runDir string) (*TranscriptMeta, error) {
    data, err := os.ReadFile(filepath.Join(runDir, "metadata.json"))
    if err != nil {
        return nil, err
    }
    var meta TranscriptMeta
    return &meta, json.Unmarshal(data, &meta)
}
```

### CLI Commands

```bash
# Show what would be cleaned up
$ devflow cleanup --dry-run
Would archive (older than 7 days):
  2025-01-08-ticket-to-pr-TK412
  2025-01-07-ticket-to-pr-TK410

Would delete (older than 30 days):
  2024-12-15-ticket-to-pr-TK380
  2024-12-10-ticket-to-pr-TK375

Would keep: 45 runs
Space to be freed: 234 MB

# Actually clean up
$ devflow cleanup
Archived 2 runs
Deleted 2 runs
Freed 234 MB

# Restore archived run
$ devflow restore 2025-01-08-ticket-to-pr-TK412
Restored to .devflow/runs/2025-01-08-ticket-to-pr-TK412
```

## References

- ADR-015: Artifact Structure
- ADR-017: Artifact Types
- ADR-012: Transcript Storage
