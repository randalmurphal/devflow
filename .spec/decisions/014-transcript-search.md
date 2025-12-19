# ADR-014: Transcript Search

## Status

Accepted

## Context

Users need to find transcripts based on:

1. Content (what was said)
2. Metadata (flow, status, date)
3. Token usage / cost
4. Specific patterns (errors, tool calls)

We need to decide how search works.

## Decision

### 1. grep is the Answer

For content search, use grep (or ripgrep):

```bash
# Search transcript content
grep -r "authentication" .devflow/runs/*/transcript.json

# With ripgrep
rg "authentication" .devflow/runs/
```

This is simple, fast, and leverages existing tools.

### 2. Metadata Filtering

Programmatic filtering on metadata:

```go
// List with filter
runs, err := store.List(devflow.ListFilter{
    FlowID: "ticket-to-pr",
    Status: devflow.RunStatusCompleted,
    After:  time.Now().Add(-24 * time.Hour),
    Limit:  10,
})
```

### 3. Search Interface

```go
type TranscriptSearcher interface {
    // Content search (grep-based)
    SearchContent(query string, opts SearchOptions) ([]SearchResult, error)

    // Metadata queries
    FindByFlow(flowID string) ([]TranscriptMeta, error)
    FindByStatus(status RunStatus) ([]TranscriptMeta, error)
    FindByDateRange(start, end time.Time) ([]TranscriptMeta, error)
    FindByTokens(minIn, maxIn, minOut, maxOut int) ([]TranscriptMeta, error)

    // Aggregations
    TotalCost(filter ListFilter) (float64, error)
    TotalTokens(filter ListFilter) (int, int, error)
}
```

### 4. Search Results

```go
type SearchResult struct {
    RunID     string
    TurnID    int
    Role      string
    Content   string // Matched content with context
    MatchLine int    // Line number in content
}
```

### 5. CLI Integration

```bash
# Search content
devflow transcript search "authentication error"

# Filter by metadata
devflow transcript list --flow ticket-to-pr --status failed --since 24h

# Cost summary
devflow transcript cost --since 7d

# Token usage
devflow transcript usage --flow ticket-to-pr
```

## Alternatives Considered

### Alternative 1: Full-Text Search Engine

Use SQLite FTS, Elasticsearch, or similar.

**Deferred:**
- Overkill for typical use
- Adds complexity/dependencies
- Can add later if needed

### Alternative 2: In-Memory Index

Build search index in memory.

**Rejected because:**
- Memory usage for large histories
- Rebuild on every start
- Files work fine

### Alternative 3: No Search

Just use file tools.

**Rejected because:**
- Some queries need structure (metadata)
- Aggregations useful
- Minimal API is helpful

## Consequences

### Positive

- **Simple**: grep for content
- **Familiar**: Standard Unix tools
- **Fast**: grep is optimized
- **No dependencies**: No search engine needed

### Negative

- **Limited queries**: Complex queries harder
- **No ranking**: Results unranked
- **Manual**: Some operations require scripting

### Future Enhancements

If needed later:
1. SQLite FTS for full-text search
2. Index file for faster metadata queries
3. Elasticsearch integration for large deployments

## Code Example

```go
package devflow

import (
    "bufio"
    "encoding/json"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "time"
)

// TranscriptSearcher provides search capabilities
type TranscriptSearcher struct {
    baseDir string
}

// NewTranscriptSearcher creates a searcher
func NewTranscriptSearcher(baseDir string) *TranscriptSearcher {
    return &TranscriptSearcher{baseDir: baseDir}
}

// SearchOptions configures content search
type SearchOptions struct {
    CaseSensitive bool
    MaxResults    int
    Context       int // Lines of context around match
}

// SearchResult represents a search match
type SearchResult struct {
    RunID     string
    TurnID    int
    Role      string
    Content   string
    MatchLine int
    Match     string
}

// SearchContent searches transcript content using ripgrep
func (s *TranscriptSearcher) SearchContent(query string, opts SearchOptions) ([]SearchResult, error) {
    runsDir := filepath.Join(s.baseDir, "runs")

    // Build rg command
    args := []string{
        "--json",
        "-g", "transcript.json",
    }

    if !opts.CaseSensitive {
        args = append(args, "-i")
    }

    if opts.MaxResults > 0 {
        args = append(args, "-m", fmt.Sprintf("%d", opts.MaxResults))
    }

    args = append(args, query, runsDir)

    cmd := exec.Command("rg", args...)
    output, err := cmd.Output()
    if err != nil {
        // rg returns exit code 1 for no matches
        if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
            return nil, nil
        }
        // Fall back to grep
        return s.searchWithGrep(query, opts)
    }

    return s.parseRipgrepOutput(output)
}

func (s *TranscriptSearcher) searchWithGrep(query string, opts SearchOptions) ([]SearchResult, error) {
    runsDir := filepath.Join(s.baseDir, "runs")

    args := []string{"-r", "-l"}
    if !opts.CaseSensitive {
        args = append(args, "-i")
    }
    args = append(args, query, runsDir)

    cmd := exec.Command("grep", args...)
    output, err := cmd.Output()
    if err != nil {
        return nil, nil // No matches
    }

    // Parse matching files
    var results []SearchResult
    for _, line := range strings.Split(string(output), "\n") {
        if line == "" {
            continue
        }

        // Extract run ID from path
        parts := strings.Split(line, string(filepath.Separator))
        for i, p := range parts {
            if p == "runs" && i+1 < len(parts) {
                results = append(results, SearchResult{
                    RunID: parts[i+1],
                })
                break
            }
        }
    }

    return results, nil
}

func (s *TranscriptSearcher) parseRipgrepOutput(output []byte) ([]SearchResult, error) {
    var results []SearchResult

    scanner := bufio.NewScanner(strings.NewReader(string(output)))
    for scanner.Scan() {
        var msg struct {
            Type string `json:"type"`
            Data struct {
                Path struct {
                    Text string `json:"text"`
                } `json:"path"`
                Lines struct {
                    Text string `json:"text"`
                } `json:"lines"`
                LineNumber int `json:"line_number"`
            } `json:"data"`
        }

        if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
            continue
        }

        if msg.Type != "match" {
            continue
        }

        // Extract run ID from path
        path := msg.Data.Path.Text
        runID := extractRunID(path)

        results = append(results, SearchResult{
            RunID:     runID,
            Content:   msg.Data.Lines.Text,
            MatchLine: msg.Data.LineNumber,
        })
    }

    return results, nil
}

// FindByFlow returns transcripts for a flow
func (s *TranscriptSearcher) FindByFlow(flowID string) ([]TranscriptMeta, error) {
    return s.findByMetadata(func(m *TranscriptMeta) bool {
        return m.FlowID == flowID
    })
}

// FindByStatus returns transcripts with status
func (s *TranscriptSearcher) FindByStatus(status RunStatus) ([]TranscriptMeta, error) {
    return s.findByMetadata(func(m *TranscriptMeta) bool {
        return m.Status == status
    })
}

// FindByDateRange returns transcripts in date range
func (s *TranscriptSearcher) FindByDateRange(start, end time.Time) ([]TranscriptMeta, error) {
    return s.findByMetadata(func(m *TranscriptMeta) bool {
        return m.StartedAt.After(start) && m.StartedAt.Before(end)
    })
}

func (s *TranscriptSearcher) findByMetadata(predicate func(*TranscriptMeta) bool) ([]TranscriptMeta, error) {
    runsDir := filepath.Join(s.baseDir, "runs")
    entries, err := os.ReadDir(runsDir)
    if err != nil {
        return nil, err
    }

    var results []TranscriptMeta

    for _, entry := range entries {
        if !entry.IsDir() {
            continue
        }

        metaPath := filepath.Join(runsDir, entry.Name(), "metadata.json")
        data, err := os.ReadFile(metaPath)
        if err != nil {
            continue
        }

        var meta TranscriptMeta
        if err := json.Unmarshal(data, &meta); err != nil {
            continue
        }

        if predicate(&meta) {
            results = append(results, meta)
        }
    }

    return results, nil
}

// TotalCost calculates total cost for matching runs
func (s *TranscriptSearcher) TotalCost(filter ListFilter) (float64, error) {
    store, err := NewFileTranscriptStore(s.baseDir)
    if err != nil {
        return 0, err
    }

    runs, err := store.List(filter)
    if err != nil {
        return 0, err
    }

    var total float64
    for _, run := range runs {
        total += run.TotalCost
    }

    return total, nil
}

// TotalTokens calculates total tokens for matching runs
func (s *TranscriptSearcher) TotalTokens(filter ListFilter) (int, int, error) {
    store, err := NewFileTranscriptStore(s.baseDir)
    if err != nil {
        return 0, 0, err
    }

    runs, err := store.List(filter)
    if err != nil {
        return 0, 0, err
    }

    var totalIn, totalOut int
    for _, run := range runs {
        totalIn += run.TotalTokensIn
        totalOut += run.TotalTokensOut
    }

    return totalIn, totalOut, nil
}

func extractRunID(path string) string {
    parts := strings.Split(path, string(filepath.Separator))
    for i, p := range parts {
        if p == "runs" && i+1 < len(parts) {
            return parts[i+1]
        }
    }
    return ""
}
```

### Usage

```go
searcher := devflow.NewTranscriptSearcher(".devflow")

// Content search
results, err := searcher.SearchContent("authentication error", devflow.SearchOptions{
    CaseSensitive: false,
    MaxResults:    20,
})
for _, r := range results {
    fmt.Printf("%s: %s\n", r.RunID, r.Content[:80])
}

// Find failed runs
failed, err := searcher.FindByStatus(devflow.RunStatusFailed)
for _, m := range failed {
    fmt.Printf("%s: %s (error: %s)\n", m.RunID, m.FlowID, m.Error)
}

// Cost summary for last 7 days
cost, err := searcher.TotalCost(devflow.ListFilter{
    After: time.Now().Add(-7 * 24 * time.Hour),
})
fmt.Printf("Last 7 days cost: $%.2f\n", cost)

// Token usage by flow
tokensIn, tokensOut, err := searcher.TotalTokens(devflow.ListFilter{
    FlowID: "ticket-to-pr",
})
fmt.Printf("ticket-to-pr tokens: %d in, %d out\n", tokensIn, tokensOut)
```

### CLI Commands

```bash
# Search content
$ devflow transcript search "authentication"
run-2025-01-15-001: Turn 3: "...implementing authentication..."
run-2025-01-14-005: Turn 7: "...authentication failed because..."

# List with filters
$ devflow transcript list --flow ticket-to-pr --status failed --since 7d
RUN ID                          STATUS   STARTED              COST
2025-01-15-ticket-to-pr-TK423  failed   2025-01-15 14:30:00  $0.08
2025-01-14-ticket-to-pr-TK419  failed   2025-01-14 09:15:00  $0.12

# Cost summary
$ devflow transcript cost --since 30d
Last 30 days: $45.67 across 234 runs

# Token usage
$ devflow transcript usage --flow ticket-to-pr
ticket-to-pr (47 runs):
  Input:  523,400 tokens
  Output: 891,200 tokens
  Avg:    11,136 in / 18,961 out per run
```

## References

- ADR-011: Transcript Format
- ADR-012: Transcript Storage
- ADR-013: Transcript Replay
- [ripgrep](https://github.com/BurntSushi/ripgrep)
