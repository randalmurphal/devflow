package devflow

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewTranscript(t *testing.T) {
	transcript := NewTranscript("run-001", "ticket-to-pr")

	if transcript.RunID != "run-001" {
		t.Errorf("RunID = %q, want %q", transcript.RunID, "run-001")
	}
	if transcript.Metadata.FlowID != "ticket-to-pr" {
		t.Errorf("FlowID = %q, want %q", transcript.Metadata.FlowID, "ticket-to-pr")
	}
	if transcript.Metadata.Status != RunStatusRunning {
		t.Errorf("Status = %q, want %q", transcript.Metadata.Status, RunStatusRunning)
	}
	if len(transcript.Turns) != 0 {
		t.Errorf("Turns = %d, want 0", len(transcript.Turns))
	}
}

func TestTranscript_AddTurn(t *testing.T) {
	transcript := NewTranscript("run-001", "test")

	// Add system turn
	turn1 := transcript.AddTurn("system", "You are helpful", 100)
	if turn1.ID != 1 {
		t.Errorf("turn1.ID = %d, want 1", turn1.ID)
	}
	if turn1.TokensIn != 100 {
		t.Errorf("turn1.TokensIn = %d, want 100", turn1.TokensIn)
	}

	// Add user turn
	turn2 := transcript.AddTurn("user", "Hello", 50)
	if turn2.ID != 2 {
		t.Errorf("turn2.ID = %d, want 2", turn2.ID)
	}

	// Add assistant turn
	turn3 := transcript.AddTurn("assistant", "Hi there!", 75)
	if turn3.ID != 3 {
		t.Errorf("turn3.ID = %d, want 3", turn3.ID)
	}
	if turn3.TokensOut != 75 {
		t.Errorf("turn3.TokensOut = %d, want 75", turn3.TokensOut)
	}

	// Check token accumulation
	if transcript.Metadata.TotalTokensIn != 150 {
		t.Errorf("TotalTokensIn = %d, want 150", transcript.Metadata.TotalTokensIn)
	}
	if transcript.Metadata.TotalTokensOut != 75 {
		t.Errorf("TotalTokensOut = %d, want 75", transcript.Metadata.TotalTokensOut)
	}
	if transcript.Metadata.TurnCount != 3 {
		t.Errorf("TurnCount = %d, want 3", transcript.Metadata.TurnCount)
	}
}

func TestTranscript_AddToolCall(t *testing.T) {
	transcript := NewTranscript("run-001", "test")

	// Add assistant turn
	transcript.AddTurn("assistant", "Let me read that file", 100)

	// Add tool call
	transcript.AddToolCall("read_file", map[string]any{"path": "main.go"}, "package main")

	if len(transcript.Turns[0].ToolCalls) != 1 {
		t.Errorf("ToolCalls = %d, want 1", len(transcript.Turns[0].ToolCalls))
	}

	tc := transcript.Turns[0].ToolCalls[0]
	if tc.Name != "read_file" {
		t.Errorf("Name = %q, want %q", tc.Name, "read_file")
	}
	if tc.Output != "package main" {
		t.Errorf("Output = %q, want %q", tc.Output, "package main")
	}
}

func TestTranscript_Complete(t *testing.T) {
	transcript := NewTranscript("run-001", "test")
	transcript.Complete()

	if transcript.Metadata.Status != RunStatusCompleted {
		t.Errorf("Status = %q, want %q", transcript.Metadata.Status, RunStatusCompleted)
	}
	if transcript.Metadata.EndedAt.IsZero() {
		t.Error("EndedAt should be set")
	}
}

func TestTranscript_Fail(t *testing.T) {
	transcript := NewTranscript("run-001", "test")
	transcript.Fail(ErrClaudeTimeout)

	if transcript.Metadata.Status != RunStatusFailed {
		t.Errorf("Status = %q, want %q", transcript.Metadata.Status, RunStatusFailed)
	}
	if transcript.Metadata.Error != ErrClaudeTimeout.Error() {
		t.Errorf("Error = %q, want %q", transcript.Metadata.Error, ErrClaudeTimeout.Error())
	}
}

func TestTranscript_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()

	// Create and save transcript
	transcript := NewTranscript("run-001", "ticket-to-pr")
	transcript.AddTurn("system", "You are helpful", 100)
	transcript.AddTurn("user", "Hello", 50)
	transcript.AddTurn("assistant", "Hi!", 75)
	transcript.Complete()

	if err := transcript.Save(dir); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Load and verify
	loaded, err := LoadTranscript(dir, "run-001")
	if err != nil {
		t.Fatalf("LoadTranscript: %v", err)
	}

	if loaded.RunID != transcript.RunID {
		t.Errorf("RunID = %q, want %q", loaded.RunID, transcript.RunID)
	}
	if len(loaded.Turns) != len(transcript.Turns) {
		t.Errorf("Turns = %d, want %d", len(loaded.Turns), len(transcript.Turns))
	}
	if loaded.Metadata.TotalTokensIn != transcript.Metadata.TotalTokensIn {
		t.Errorf("TotalTokensIn = %d, want %d",
			loaded.Metadata.TotalTokensIn, transcript.Metadata.TotalTokensIn)
	}
}

func TestTranscript_Compression(t *testing.T) {
	dir := t.TempDir()

	// Create a large transcript
	transcript := NewTranscript("run-large", "test")
	largeContent := strings.Repeat("This is a long content. ", 5000) // ~120KB
	transcript.AddTurn("assistant", largeContent, len(largeContent))
	transcript.Complete()

	if err := transcript.Save(dir); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Check compressed file exists
	gzPath := filepath.Join(dir, "runs", "run-large", "transcript.json.gz")
	if _, err := os.Stat(gzPath); os.IsNotExist(err) {
		t.Error("compressed file should exist")
	}

	// Check uncompressed doesn't exist
	jsonPath := filepath.Join(dir, "runs", "run-large", "transcript.json")
	if _, err := os.Stat(jsonPath); !os.IsNotExist(err) {
		t.Error("uncompressed file should not exist")
	}

	// Load and verify
	loaded, err := LoadTranscript(dir, "run-large")
	if err != nil {
		t.Fatalf("LoadTranscript: %v", err)
	}

	if loaded.Turns[0].Content != largeContent {
		t.Error("content mismatch after compression roundtrip")
	}
}

func TestFileTranscriptStore_Lifecycle(t *testing.T) {
	dir := t.TempDir()
	store, err := NewFileTranscriptStore(dir)
	if err != nil {
		t.Fatalf("NewFileTranscriptStore: %v", err)
	}

	// Start run
	err = store.StartRun("run-001", RunMetadata{
		FlowID: "ticket-to-pr",
		Input:  map[string]any{"ticket": "TK-421"},
	})
	if err != nil {
		t.Fatalf("StartRun: %v", err)
	}

	// Record turns
	err = store.RecordTurn("run-001", Turn{
		Role:     "user",
		Content:  "Hello",
		TokensIn: 50,
	})
	if err != nil {
		t.Fatalf("RecordTurn: %v", err)
	}

	err = store.RecordTurn("run-001", Turn{
		Role:      "assistant",
		Content:   "Hi!",
		TokensOut: 75,
	})
	if err != nil {
		t.Fatalf("RecordTurn: %v", err)
	}

	// End run
	err = store.EndRun("run-001", RunStatusCompleted)
	if err != nil {
		t.Fatalf("EndRun: %v", err)
	}

	// Load and verify
	loaded, err := store.Load("run-001")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded.Turns) != 2 {
		t.Errorf("Turns = %d, want 2", len(loaded.Turns))
	}
	if loaded.Metadata.Status != RunStatusCompleted {
		t.Errorf("Status = %q, want %q", loaded.Metadata.Status, RunStatusCompleted)
	}
}

func TestFileTranscriptStore_List(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewFileTranscriptStore(dir)

	// Create multiple runs
	for i := 0; i < 5; i++ {
		runID := "run-" + itoa(i)
		store.StartRun(runID, RunMetadata{FlowID: "test"})
		store.RecordTurn(runID, Turn{Role: "user", Content: "Hello"})
		status := RunStatusCompleted
		if i == 2 {
			status = RunStatusFailed
		}
		store.EndRun(runID, status)
	}

	// List all
	all, err := store.List(ListFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 5 {
		t.Errorf("List all = %d, want 5", len(all))
	}

	// List by status
	failed, err := store.List(ListFilter{Status: RunStatusFailed})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(failed) != 1 {
		t.Errorf("List failed = %d, want 1", len(failed))
	}

	// List with limit
	limited, err := store.List(ListFilter{Limit: 3})
	if err != nil {
		t.Fatalf("List limited: %v", err)
	}
	if len(limited) != 3 {
		t.Errorf("List limited = %d, want 3", len(limited))
	}
}

func TestFileTranscriptStore_Delete(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewFileTranscriptStore(dir)

	// Create run
	store.StartRun("run-001", RunMetadata{FlowID: "test"})
	store.EndRun("run-001", RunStatusCompleted)

	// Delete
	err := store.Delete("run-001")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Verify deleted
	_, err = store.Load("run-001")
	if err != ErrRunNotFound {
		t.Errorf("Load after delete = %v, want ErrRunNotFound", err)
	}
}

func TestFileTranscriptStore_Errors(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewFileTranscriptStore(dir)

	// Start same run twice
	store.StartRun("run-001", RunMetadata{FlowID: "test"})
	err := store.StartRun("run-001", RunMetadata{FlowID: "test"})
	if err != ErrRunAlreadyExists {
		t.Errorf("StartRun duplicate = %v, want ErrRunAlreadyExists", err)
	}

	// Record turn for non-existent run
	err = store.RecordTurn("nonexistent", Turn{Role: "user", Content: "Hello"})
	if err != ErrRunNotStarted {
		t.Errorf("RecordTurn nonexistent = %v, want ErrRunNotStarted", err)
	}

	// End non-existent run
	err = store.EndRun("nonexistent", RunStatusCompleted)
	if err != ErrRunNotStarted {
		t.Errorf("EndRun nonexistent = %v, want ErrRunNotStarted", err)
	}
}

func TestTranscriptSearcher_FindByStatus(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewFileTranscriptStore(dir)

	// Create runs with different statuses
	store.StartRun("run-1", RunMetadata{FlowID: "test"})
	store.EndRun("run-1", RunStatusCompleted)

	store.StartRun("run-2", RunMetadata{FlowID: "test"})
	store.EndRun("run-2", RunStatusFailed)

	store.StartRun("run-3", RunMetadata{FlowID: "test"})
	store.EndRun("run-3", RunStatusCompleted)

	searcher := NewTranscriptSearcher(dir)

	// Find completed
	completed, err := searcher.FindByStatus(RunStatusCompleted)
	if err != nil {
		t.Fatalf("FindByStatus: %v", err)
	}
	if len(completed) != 2 {
		t.Errorf("FindByStatus completed = %d, want 2", len(completed))
	}

	// Find failed
	failed, err := searcher.FindByStatus(RunStatusFailed)
	if err != nil {
		t.Fatalf("FindByStatus: %v", err)
	}
	if len(failed) != 1 {
		t.Errorf("FindByStatus failed = %d, want 1", len(failed))
	}
}

func TestTranscriptSearcher_TotalCostAndTokens(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewFileTranscriptStore(dir)

	// Create runs
	store.StartRun("run-1", RunMetadata{FlowID: "test"})
	store.RecordTurn("run-1", Turn{Role: "user", TokensIn: 100})
	store.RecordTurn("run-1", Turn{Role: "assistant", TokensOut: 200})
	store.AddCost("run-1", 0.05)
	store.EndRun("run-1", RunStatusCompleted)

	store.StartRun("run-2", RunMetadata{FlowID: "test"})
	store.RecordTurn("run-2", Turn{Role: "user", TokensIn: 150})
	store.RecordTurn("run-2", Turn{Role: "assistant", TokensOut: 300})
	store.AddCost("run-2", 0.10)
	store.EndRun("run-2", RunStatusCompleted)

	searcher := NewTranscriptSearcher(dir)

	// Total cost
	cost, err := searcher.TotalCost(ListFilter{})
	if err != nil {
		t.Fatalf("TotalCost: %v", err)
	}
	// Use tolerance for float comparison
	if cost < 0.14 || cost > 0.16 {
		t.Errorf("TotalCost = %f, want ~0.15", cost)
	}

	// Total tokens
	tokensIn, tokensOut, err := searcher.TotalTokens(ListFilter{})
	if err != nil {
		t.Fatalf("TotalTokens: %v", err)
	}
	if tokensIn != 250 {
		t.Errorf("TotalTokensIn = %d, want 250", tokensIn)
	}
	if tokensOut != 500 {
		t.Errorf("TotalTokensOut = %d, want 500", tokensOut)
	}
}

func TestTranscriptViewer_ViewFull(t *testing.T) {
	transcript := NewTranscript("run-001", "ticket-to-pr")
	transcript.AddTurn("system", "You are helpful", 100)
	transcript.AddTurn("user", "Hello", 50)
	transcript.AddTurn("assistant", "Hi there!", 75)
	transcript.Complete()

	viewer := NewTranscriptViewer(false)
	var buf bytes.Buffer
	err := viewer.ViewFull(&buf, transcript)
	if err != nil {
		t.Fatalf("ViewFull: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "run-001") {
		t.Error("output should contain run ID")
	}
	if !strings.Contains(output, "ticket-to-pr") {
		t.Error("output should contain flow ID")
	}
	if !strings.Contains(output, "SYSTEM") {
		t.Error("output should contain SYSTEM turn")
	}
	if !strings.Contains(output, "Hi there!") {
		t.Error("output should contain assistant content")
	}
}

func TestTranscriptViewer_ExportMarkdown(t *testing.T) {
	transcript := NewTranscript("run-001", "ticket-to-pr")
	transcript.AddTurn("system", "You are helpful", 100)
	transcript.AddTurn("user", "Hello", 50)
	turn := transcript.AddTurn("assistant", "Let me help", 75)
	turn.ToolCalls = append(turn.ToolCalls, ToolCall{
		Name:   "read_file",
		Input:  map[string]any{"path": "main.go"},
		Output: "package main",
	})
	transcript.Complete()

	viewer := NewTranscriptViewer(false)
	var buf bytes.Buffer
	err := viewer.ExportMarkdown(&buf, transcript)
	if err != nil {
		t.Fatalf("ExportMarkdown: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "# Transcript: run-001") {
		t.Error("output should contain markdown header")
	}
	if !strings.Contains(output, "## Metadata") {
		t.Error("output should contain metadata section")
	}
	if !strings.Contains(output, "## Conversation") {
		t.Error("output should contain conversation section")
	}
	if !strings.Contains(output, "Tool Call: `read_file`") {
		t.Error("output should contain tool call")
	}
}

func TestTranscriptViewer_Diff(t *testing.T) {
	a := NewTranscript("run-001", "test")
	a.AddTurn("user", "Hello", 50)
	a.AddTurn("assistant", "Hi!", 75)
	a.Complete()

	b := NewTranscript("run-002", "test")
	b.AddTurn("user", "Hello", 50)
	b.AddTurn("assistant", "Hello there! How can I help?", 150)
	b.Complete()

	viewer := NewTranscriptViewer(false)
	var buf bytes.Buffer
	err := viewer.Diff(&buf, a, b)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "run-001") {
		t.Error("output should contain first run ID")
	}
	if !strings.Contains(output, "run-002") {
		t.Error("output should contain second run ID")
	}
	if !strings.Contains(output, "different") {
		t.Error("output should show differences")
	}
}

func TestTranscript_TurnsByRole(t *testing.T) {
	transcript := NewTranscript("run-001", "test")
	transcript.AddTurn("system", "System prompt", 100)
	transcript.AddTurn("user", "User 1", 50)
	transcript.AddTurn("assistant", "Assistant 1", 75)
	transcript.AddTurn("user", "User 2", 50)
	transcript.AddTurn("assistant", "Assistant 2", 75)

	userTurns := transcript.TurnsByRole("user")
	if len(userTurns) != 2 {
		t.Errorf("user turns = %d, want 2", len(userTurns))
	}

	assistantTurns := transcript.TurnsByRole("assistant")
	if len(assistantTurns) != 2 {
		t.Errorf("assistant turns = %d, want 2", len(assistantTurns))
	}
}

func TestTranscript_Duration(t *testing.T) {
	transcript := NewTranscript("run-001", "test")

	// Active run
	time.Sleep(10 * time.Millisecond)
	if transcript.Duration() < 10*time.Millisecond {
		t.Error("Duration should be > 10ms for active run")
	}

	// Completed run
	transcript.Complete()
	duration := transcript.Duration()
	time.Sleep(10 * time.Millisecond)
	if transcript.Duration() != duration {
		t.Error("Duration should be fixed after completion")
	}
}

func TestTranscript_JSON_Roundtrip(t *testing.T) {
	original := NewTranscript("run-001", "ticket-to-pr")
	original.Metadata.NodeID = "generate-spec"
	original.Metadata.Input = map[string]any{"ticket": "TK-421"}
	original.AddTurn("system", "You are helpful", 100)
	original.AddTurn("user", "Hello", 50)
	turn := original.AddTurn("assistant", "Let me help", 75)
	turn.ToolCalls = append(turn.ToolCalls, ToolCall{
		Name:   "read_file",
		Input:  map[string]any{"path": "main.go"},
		Output: "package main",
	})
	original.SetCost(0.05)
	original.Complete()

	// Marshal
	data, err := json.MarshalIndent(original, "", "  ")
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	// Unmarshal
	var loaded Transcript
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	// Verify
	if loaded.RunID != original.RunID {
		t.Errorf("RunID = %q, want %q", loaded.RunID, original.RunID)
	}
	if loaded.Metadata.FlowID != original.Metadata.FlowID {
		t.Errorf("FlowID = %q, want %q", loaded.Metadata.FlowID, original.Metadata.FlowID)
	}
	if len(loaded.Turns) != len(original.Turns) {
		t.Errorf("Turns = %d, want %d", len(loaded.Turns), len(original.Turns))
	}
	if len(loaded.Turns[2].ToolCalls) != 1 {
		t.Errorf("ToolCalls = %d, want 1", len(loaded.Turns[2].ToolCalls))
	}
}

func TestRunStatistics(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewFileTranscriptStore(dir)

	// Create various runs
	for i := 0; i < 3; i++ {
		runID := "completed-" + itoa(i)
		store.StartRun(runID, RunMetadata{FlowID: "test"})
		store.RecordTurn(runID, Turn{Role: "user", TokensIn: 100})
		store.RecordTurn(runID, Turn{Role: "assistant", TokensOut: 200})
		store.AddCost(runID, 0.05)
		store.EndRun(runID, RunStatusCompleted)
	}

	store.StartRun("failed-1", RunMetadata{FlowID: "test"})
	store.AddCost("failed-1", 0.02)
	store.EndRun("failed-1", RunStatusFailed)

	searcher := NewTranscriptSearcher(dir)
	stats, err := searcher.RunStats(ListFilter{})
	if err != nil {
		t.Fatalf("RunStats: %v", err)
	}

	if stats.TotalRuns != 4 {
		t.Errorf("TotalRuns = %d, want 4", stats.TotalRuns)
	}
	if stats.CompletedRuns != 3 {
		t.Errorf("CompletedRuns = %d, want 3", stats.CompletedRuns)
	}
	if stats.FailedRuns != 1 {
		t.Errorf("FailedRuns = %d, want 1", stats.FailedRuns)
	}
	if stats.TotalTokensIn != 300 {
		t.Errorf("TotalTokensIn = %d, want 300", stats.TotalTokensIn)
	}
}
