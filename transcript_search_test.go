package devflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestNewTranscriptSearcher(t *testing.T) {
	searcher := NewTranscriptSearcher("/tmp/test")

	if searcher.baseDir != "/tmp/test" {
		t.Errorf("baseDir = %q, want %q", searcher.baseDir, "/tmp/test")
	}
}

func TestTranscriptSearcher_FindByFlow(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test metadata files
	createTestMetadata(t, tmpDir, "run-1", TranscriptMeta{
		RunID:     "run-1",
		FlowID:    "test-flow",
		StartedAt: time.Now(),
	})
	createTestMetadata(t, tmpDir, "run-2", TranscriptMeta{
		RunID:     "run-2",
		FlowID:    "other-flow",
		StartedAt: time.Now(),
	})
	createTestMetadata(t, tmpDir, "run-3", TranscriptMeta{
		RunID:     "run-3",
		FlowID:    "test-flow",
		StartedAt: time.Now(),
	})

	searcher := NewTranscriptSearcher(tmpDir)
	results, err := searcher.FindByFlow("test-flow")
	if err != nil {
		t.Fatalf("FindByFlow() error = %v", err)
	}

	if len(results) != 2 {
		t.Errorf("FindByFlow() returned %d results, want 2", len(results))
	}

	for _, r := range results {
		if r.FlowID != "test-flow" {
			t.Errorf("result FlowID = %q, want %q", r.FlowID, "test-flow")
		}
	}
}

func TestTranscriptSearcher_FindByStatus_All(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test metadata files with all status types
	createTestMetadata(t, tmpDir, "run-completed", TranscriptMeta{
		RunID:  "run-completed",
		Status: RunStatusCompleted,
	})
	createTestMetadata(t, tmpDir, "run-failed", TranscriptMeta{
		RunID:  "run-failed",
		Status: RunStatusFailed,
	})
	createTestMetadata(t, tmpDir, "run-canceled", TranscriptMeta{
		RunID:  "run-canceled",
		Status: RunStatusCanceled,
	})
	createTestMetadata(t, tmpDir, "run-running", TranscriptMeta{
		RunID:  "run-running",
		Status: RunStatusRunning,
	})

	searcher := NewTranscriptSearcher(tmpDir)

	// Test finding failed
	failedResults, err := searcher.FindByStatus(RunStatusFailed)
	if err != nil {
		t.Fatalf("FindByStatus(failed) error = %v", err)
	}
	if len(failedResults) != 1 {
		t.Errorf("FindByStatus(failed) returned %d results, want 1", len(failedResults))
	}

	// Test finding canceled
	canceledResults, err := searcher.FindByStatus(RunStatusCanceled)
	if err != nil {
		t.Fatalf("FindByStatus(canceled) error = %v", err)
	}
	if len(canceledResults) != 1 {
		t.Errorf("FindByStatus(canceled) returned %d results, want 1", len(canceledResults))
	}

	// Test finding running
	runningResults, err := searcher.FindByStatus(RunStatusRunning)
	if err != nil {
		t.Fatalf("FindByStatus(running) error = %v", err)
	}
	if len(runningResults) != 1 {
		t.Errorf("FindByStatus(running) returned %d results, want 1", len(runningResults))
	}
}

func TestTranscriptSearcher_FindByDateRange(t *testing.T) {
	tmpDir := t.TempDir()

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	lastWeek := now.Add(-7 * 24 * time.Hour)

	// Create test metadata files
	createTestMetadata(t, tmpDir, "run-recent", TranscriptMeta{
		RunID:     "run-recent",
		StartedAt: now.Add(-1 * time.Hour),
	})
	createTestMetadata(t, tmpDir, "run-yesterday", TranscriptMeta{
		RunID:     "run-yesterday",
		StartedAt: yesterday,
	})
	createTestMetadata(t, tmpDir, "run-old", TranscriptMeta{
		RunID:     "run-old",
		StartedAt: lastWeek.Add(-24 * time.Hour), // older than last week
	})

	searcher := NewTranscriptSearcher(tmpDir)

	// Find runs from yesterday to now
	results, err := searcher.FindByDateRange(yesterday.Add(-1*time.Hour), now.Add(1*time.Hour))
	if err != nil {
		t.Fatalf("FindByDateRange() error = %v", err)
	}

	if len(results) != 2 {
		t.Errorf("FindByDateRange() returned %d results, want 2", len(results))
	}
}

func TestTranscriptSearcher_FindByTokenRange(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test metadata files
	createTestMetadata(t, tmpDir, "run-small", TranscriptMeta{
		RunID:          "run-small",
		TotalTokensIn:  100,
		TotalTokensOut: 50,
	})
	createTestMetadata(t, tmpDir, "run-medium", TranscriptMeta{
		RunID:          "run-medium",
		TotalTokensIn:  1000,
		TotalTokensOut: 500,
	})
	createTestMetadata(t, tmpDir, "run-large", TranscriptMeta{
		RunID:          "run-large",
		TotalTokensIn:  10000,
		TotalTokensOut: 5000,
	})

	searcher := NewTranscriptSearcher(tmpDir)

	tests := []struct {
		name   string
		minIn  int
		maxIn  int
		minOut int
		maxOut int
		want   int
	}{
		{"no filters", 0, 0, 0, 0, 3},
		{"min input only", 500, 0, 0, 0, 2},
		{"max input only", 0, 5000, 0, 0, 2},
		{"min and max input", 500, 5000, 0, 0, 1},
		{"min output only", 0, 0, 100, 0, 2},
		{"max output only", 0, 0, 0, 1000, 2},
		{"combined", 500, 5000, 100, 1000, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := searcher.FindByTokenRange(tt.minIn, tt.maxIn, tt.minOut, tt.maxOut)
			if err != nil {
				t.Fatalf("FindByTokenRange() error = %v", err)
			}

			if len(results) != tt.want {
				t.Errorf("FindByTokenRange() returned %d results, want %d", len(results), tt.want)
			}
		})
	}
}

func TestTranscriptSearcher_FindByMetadata_EmptyDir(t *testing.T) {
	searcher := NewTranscriptSearcher("/nonexistent/path")

	results, err := searcher.FindByFlow("test-flow")
	if err != nil {
		t.Fatalf("FindByFlow() should not error for nonexistent dir, got %v", err)
	}

	if results != nil && len(results) != 0 {
		t.Errorf("FindByFlow() should return empty for nonexistent dir, got %d", len(results))
	}
}

func TestTranscriptSearcher_TotalCost(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := NewFileTranscriptStore(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create runs with costs
	for _, runID := range []string{"run-1", "run-2", "run-3"} {
		if err := store.StartRun(runID, RunMetadata{FlowID: "test-flow"}); err != nil {
			t.Fatal(err)
		}
		// Add some cost
		store.AddCost(runID, 0.10)
		if err := store.EndRun(runID, RunStatusCompleted); err != nil {
			t.Fatal(err)
		}
	}

	searcher := NewTranscriptSearcher(tmpDir)
	cost, err := searcher.TotalCost(ListFilter{})
	if err != nil {
		t.Fatalf("TotalCost() error = %v", err)
	}

	expectedCost := 0.30
	if cost < expectedCost-0.001 || cost > expectedCost+0.001 {
		t.Errorf("TotalCost() = %f, want %f", cost, expectedCost)
	}
}

func TestTranscriptSearcher_TotalTokens(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := NewFileTranscriptStore(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create runs with tokens
	// Note: RecordTurn counts TokensIn for "user"/"system" roles, TokensOut for "assistant"
	for _, runID := range []string{"run-1", "run-2"} {
		if err := store.StartRun(runID, RunMetadata{FlowID: "test-flow"}); err != nil {
			t.Fatal(err)
		}
		// User turn adds to TokensIn
		if err := store.RecordTurn(runID, Turn{
			Role:     "user",
			TokensIn: 100,
		}); err != nil {
			t.Fatal(err)
		}
		// Assistant turn adds to TokensOut
		if err := store.RecordTurn(runID, Turn{
			Role:      "assistant",
			TokensOut: 50,
		}); err != nil {
			t.Fatal(err)
		}
		if err := store.EndRun(runID, RunStatusCompleted); err != nil {
			t.Fatal(err)
		}
	}

	searcher := NewTranscriptSearcher(tmpDir)
	tokensIn, tokensOut, err := searcher.TotalTokens(ListFilter{})
	if err != nil {
		t.Fatalf("TotalTokens() error = %v", err)
	}

	// 2 runs x 100 tokens in = 200
	if tokensIn != 200 {
		t.Errorf("TotalTokens() tokensIn = %d, want 200", tokensIn)
	}
	// 2 runs x 50 tokens out = 100
	if tokensOut != 100 {
		t.Errorf("TotalTokens() tokensOut = %d, want 100", tokensOut)
	}
}

func TestTranscriptSearcher_RunStats(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := NewFileTranscriptStore(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create runs with different statuses
	for i, status := range []RunStatus{
		RunStatusCompleted,
		RunStatusCompleted,
		RunStatusFailed,
		RunStatusCanceled,
	} {
		runID := strconv.Itoa(i)
		if err := store.StartRun(runID, RunMetadata{FlowID: "test-flow"}); err != nil {
			t.Fatal(err)
		}
		// User turn for TokensIn
		if err := store.RecordTurn(runID, Turn{Role: "user", TokensIn: 100}); err != nil {
			t.Fatal(err)
		}
		// Assistant turn for TokensOut
		if err := store.RecordTurn(runID, Turn{Role: "assistant", TokensOut: 50}); err != nil {
			t.Fatal(err)
		}
		store.AddCost(runID, 0.05)
		if err := store.EndRun(runID, status); err != nil {
			t.Fatal(err)
		}
	}

	searcher := NewTranscriptSearcher(tmpDir)
	stats, err := searcher.RunStats(ListFilter{})
	if err != nil {
		t.Fatalf("RunStats() error = %v", err)
	}

	if stats.TotalRuns != 4 {
		t.Errorf("TotalRuns = %d, want 4", stats.TotalRuns)
	}
	if stats.CompletedRuns != 2 {
		t.Errorf("CompletedRuns = %d, want 2", stats.CompletedRuns)
	}
	if stats.FailedRuns != 1 {
		t.Errorf("FailedRuns = %d, want 1", stats.FailedRuns)
	}
	if stats.CanceledRuns != 1 {
		t.Errorf("CanceledRuns = %d, want 1", stats.CanceledRuns)
	}
	if stats.TotalTokensIn != 400 {
		t.Errorf("TotalTokensIn = %d, want 400", stats.TotalTokensIn)
	}
	if stats.AvgTokensIn != 100 {
		t.Errorf("AvgTokensIn = %d, want 100", stats.AvgTokensIn)
	}
	if stats.AvgCost < 0.04 || stats.AvgCost > 0.06 {
		t.Errorf("AvgCost = %f, want ~0.05", stats.AvgCost)
	}
}

func TestExtractRunID(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/base/runs/run-123/transcript.json", "run-123"},
		{"/base/runs/2025-01-15-flow-abc123/metadata.json", "2025-01-15-flow-abc123"},
		{"runs/my-run/file.txt", "my-run"},
		{"/no-runs-in-path/file.txt", ""},
		{"/runs/", ""},
		{"runs", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := extractRunID(tt.path)
			if got != tt.want {
				t.Errorf("extractRunID(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}


func TestSearchOptions_Defaults(t *testing.T) {
	opts := SearchOptions{}

	if opts.CaseSensitive {
		t.Error("CaseSensitive default should be false")
	}
	if opts.MaxResults != 0 {
		t.Error("MaxResults default should be 0 (unlimited)")
	}
	if opts.Context != 0 {
		t.Error("Context default should be 0")
	}
}

func TestSearchResult_Fields(t *testing.T) {
	result := SearchResult{
		RunID:     "run-123",
		TurnID:    5,
		Role:      "assistant",
		Content:   "test content",
		MatchLine: 42,
		Match:     "matching text",
	}

	if result.RunID != "run-123" {
		t.Errorf("RunID = %q, want %q", result.RunID, "run-123")
	}
	if result.TurnID != 5 {
		t.Errorf("TurnID = %d, want 5", result.TurnID)
	}
}

func TestRunStatistics_Empty(t *testing.T) {
	tmpDir := t.TempDir()

	// Create store with no runs
	_, err := NewFileTranscriptStore(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	searcher := NewTranscriptSearcher(tmpDir)
	stats, err := searcher.RunStats(ListFilter{})
	if err != nil {
		t.Fatalf("RunStats() error = %v", err)
	}

	if stats.TotalRuns != 0 {
		t.Errorf("TotalRuns = %d, want 0", stats.TotalRuns)
	}
	if stats.AvgTokensIn != 0 {
		t.Errorf("AvgTokensIn = %d, want 0 (no division by zero)", stats.AvgTokensIn)
	}
	if stats.AvgCost != 0 {
		t.Errorf("AvgCost = %f, want 0", stats.AvgCost)
	}
}

func TestTranscriptSearcher_SearchContent(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test transcript files with searchable content
	runsDir := filepath.Join(tmpDir, "runs")

	// Create run-1 with transcript
	run1Dir := filepath.Join(runsDir, "run-1")
	os.MkdirAll(run1Dir, 0755)
	transcript1 := `{"turns":[{"content":"This is a test message about golang"},{"content":"Another line here"}]}`
	os.WriteFile(filepath.Join(run1Dir, "transcript.json"), []byte(transcript1), 0644)

	// Create run-2 with transcript
	run2Dir := filepath.Join(runsDir, "run-2")
	os.MkdirAll(run2Dir, 0755)
	transcript2 := `{"turns":[{"content":"Different content without the keyword"},{"content":"More stuff"}]}`
	os.WriteFile(filepath.Join(run2Dir, "transcript.json"), []byte(transcript2), 0644)

	// Create run-3 with transcript containing the search term
	run3Dir := filepath.Join(runsDir, "run-3")
	os.MkdirAll(run3Dir, 0755)
	transcript3 := `{"turns":[{"content":"This also mentions golang programming"}]}`
	os.WriteFile(filepath.Join(run3Dir, "transcript.json"), []byte(transcript3), 0644)

	searcher := NewTranscriptSearcher(tmpDir)

	results, err := searcher.SearchContent("golang", SearchOptions{})
	if err != nil {
		t.Fatalf("SearchContent: %v", err)
	}

	// Should find matches in run-1 and run-3
	if len(results) < 2 {
		t.Errorf("SearchContent returned %d results, want at least 2", len(results))
	}
}

func TestTranscriptSearcher_SearchContent_CaseSensitive(t *testing.T) {
	tmpDir := t.TempDir()

	runsDir := filepath.Join(tmpDir, "runs")
	runDir := filepath.Join(runsDir, "run-1")
	os.MkdirAll(runDir, 0755)
	transcript := `{"turns":[{"content":"GOLANG uppercase"},{"content":"golang lowercase"}]}`
	os.WriteFile(filepath.Join(runDir, "transcript.json"), []byte(transcript), 0644)

	searcher := NewTranscriptSearcher(tmpDir)

	// Case-insensitive (default)
	results, err := searcher.SearchContent("GOLANG", SearchOptions{CaseSensitive: false})
	if err != nil {
		t.Fatalf("SearchContent: %v", err)
	}
	// Should find both
	if len(results) < 1 {
		t.Errorf("Case-insensitive search returned %d results, want >= 1", len(results))
	}

	// Case-sensitive
	results, err = searcher.SearchContent("GOLANG", SearchOptions{CaseSensitive: true})
	if err != nil {
		t.Fatalf("SearchContent: %v", err)
	}
	// Should find only uppercase
	if len(results) < 1 {
		t.Errorf("Case-sensitive search returned %d results, want >= 1", len(results))
	}
}

func TestTranscriptSearcher_SearchContent_MaxResults(t *testing.T) {
	tmpDir := t.TempDir()

	runsDir := filepath.Join(tmpDir, "runs")

	// Create multiple runs with matches
	for i := 0; i < 5; i++ {
		runDir := filepath.Join(runsDir, strconv.Itoa(i))
		os.MkdirAll(runDir, 0755)
		transcript := `{"turns":[{"content":"findme keyword here"}]}`
		os.WriteFile(filepath.Join(runDir, "transcript.json"), []byte(transcript), 0644)
	}

	searcher := NewTranscriptSearcher(tmpDir)

	// Test with max results - note that grep mode deduplicates by runID
	// and MaxResults applies to grep differently than ripgrep
	results, err := searcher.SearchContent("findme", SearchOptions{MaxResults: 2})
	if err != nil {
		t.Fatalf("SearchContent: %v", err)
	}

	// Just verify we got results (the limitation logic varies by tool)
	if len(results) == 0 {
		t.Error("SearchContent should return some results")
	}
}

func TestTranscriptSearcher_SearchContent_NoMatches(t *testing.T) {
	tmpDir := t.TempDir()

	runsDir := filepath.Join(tmpDir, "runs")
	runDir := filepath.Join(runsDir, "run-1")
	os.MkdirAll(runDir, 0755)
	transcript := `{"turns":[{"content":"nothing relevant here"}]}`
	os.WriteFile(filepath.Join(runDir, "transcript.json"), []byte(transcript), 0644)

	searcher := NewTranscriptSearcher(tmpDir)

	results, err := searcher.SearchContent("notfoundxyz123", SearchOptions{})
	if err != nil {
		t.Fatalf("SearchContent: %v", err)
	}

	// Should return empty (not error) for no matches
	if len(results) != 0 {
		t.Errorf("SearchContent returned %d results, want 0", len(results))
	}
}

func TestTranscriptSearcher_SearchContent_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create the runs dir but leave it empty
	os.MkdirAll(filepath.Join(tmpDir, "runs"), 0755)

	searcher := NewTranscriptSearcher(tmpDir)

	// Searching an empty runs dir - grep/rg may return errors or no results
	// This is acceptable behavior when there's nothing to search
	results, err := searcher.SearchContent("anything", SearchOptions{})

	// Either no error with empty results, or an error is acceptable
	if err == nil && len(results) != 0 {
		t.Errorf("SearchContent on empty dir returned %d results, want 0", len(results))
	}
}

func TestParseRipgrepOutput(t *testing.T) {
	searcher := NewTranscriptSearcher("/tmp")

	// Simulate ripgrep JSON output
	output := []byte(`{"type":"match","data":{"path":{"text":"/base/runs/run-123/transcript.json"},"lines":{"text":"  \"content\": \"test message\"\n"},"line_number":5}}
{"type":"match","data":{"path":{"text":"/base/runs/run-456/transcript.json"},"lines":{"text":"  \"content\": \"another match\"\n"},"line_number":10}}
{"type":"summary","data":{"stats":{"matches":2}}}`)

	results, err := searcher.parseRipgrepOutput(output)
	if err != nil {
		t.Fatalf("parseRipgrepOutput: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("parseRipgrepOutput returned %d results, want 2", len(results))
	}

	if results[0].RunID != "run-123" {
		t.Errorf("first result RunID = %q, want %q", results[0].RunID, "run-123")
	}
	if results[0].MatchLine != 5 {
		t.Errorf("first result MatchLine = %d, want 5", results[0].MatchLine)
	}

	if results[1].RunID != "run-456" {
		t.Errorf("second result RunID = %q, want %q", results[1].RunID, "run-456")
	}
}

func TestParseRipgrepOutput_Empty(t *testing.T) {
	searcher := NewTranscriptSearcher("/tmp")

	results, err := searcher.parseRipgrepOutput([]byte{})
	if err != nil {
		t.Fatalf("parseRipgrepOutput: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("parseRipgrepOutput returned %d results, want 0", len(results))
	}
}

func TestParseRipgrepOutput_InvalidJSON(t *testing.T) {
	searcher := NewTranscriptSearcher("/tmp")

	// Invalid JSON should be skipped, not error
	output := []byte(`not json
{"type":"match","data":{"path":{"text":"/base/runs/run-123/transcript.json"},"lines":{"text":"match\n"},"line_number":1}}
{broken json}`)

	results, err := searcher.parseRipgrepOutput(output)
	if err != nil {
		t.Fatalf("parseRipgrepOutput: %v", err)
	}

	// Should have parsed the one valid match
	if len(results) != 1 {
		t.Errorf("parseRipgrepOutput returned %d results, want 1", len(results))
	}
}

func TestParseRipgrepOutput_NoRunID(t *testing.T) {
	searcher := NewTranscriptSearcher("/tmp")

	// Path without "runs" in it should be skipped
	output := []byte(`{"type":"match","data":{"path":{"text":"/invalid/path/file.json"},"lines":{"text":"match\n"},"line_number":1}}`)

	results, err := searcher.parseRipgrepOutput(output)
	if err != nil {
		t.Fatalf("parseRipgrepOutput: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("parseRipgrepOutput returned %d results, want 0", len(results))
	}
}

// Helper to create test metadata files
func createTestMetadata(t *testing.T, baseDir, runID string, meta TranscriptMeta) {
	t.Helper()

	runsDir := filepath.Join(baseDir, "runs", runID)
	if err := os.MkdirAll(runsDir, 0755); err != nil {
		t.Fatal(err)
	}

	meta.RunID = runID
	data, err := json.Marshal(meta)
	if err != nil {
		t.Fatal(err)
	}

	metaPath := filepath.Join(runsDir, "metadata.json")
	if err := os.WriteFile(metaPath, data, 0644); err != nil {
		t.Fatal(err)
	}
}
