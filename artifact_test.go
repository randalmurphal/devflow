package devflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewArtifactManager(t *testing.T) {
	m := NewArtifactManager(ArtifactConfig{})

	if m.baseDir != ".devflow" {
		t.Errorf("baseDir = %q, want %q", m.baseDir, ".devflow")
	}
	if m.compressAbove != 10*1024 {
		t.Errorf("compressAbove = %d, want %d", m.compressAbove, 10*1024)
	}
	if m.retentionDays != 30 {
		t.Errorf("retentionDays = %d, want 30", m.retentionDays)
	}
}

func TestArtifactManager_SaveLoad(t *testing.T) {
	dir := t.TempDir()
	m := NewArtifactManager(ArtifactConfig{BaseDir: dir})

	runID := "test-run-001"
	content := []byte("# Test Specification\n\nThis is a test.")

	// Save
	err := m.SaveArtifact(runID, "spec.md", content)
	if err != nil {
		t.Fatalf("SaveArtifact: %v", err)
	}

	// Load
	loaded, err := m.LoadArtifact(runID, "spec.md")
	if err != nil {
		t.Fatalf("LoadArtifact: %v", err)
	}

	if string(loaded) != string(content) {
		t.Errorf("content mismatch:\ngot:  %q\nwant: %q", string(loaded), string(content))
	}
}

func TestArtifactManager_Compression(t *testing.T) {
	dir := t.TempDir()
	m := NewArtifactManager(ArtifactConfig{
		BaseDir:       dir,
		CompressAbove: 100, // Very low threshold for testing
	})

	runID := "test-run-001"
	// Create content larger than threshold
	content := []byte(strings.Repeat("Test content. ", 50)) // ~700 bytes

	// Save (should compress)
	err := m.SaveArtifact(runID, "large.txt", content)
	if err != nil {
		t.Fatalf("SaveArtifact: %v", err)
	}

	// Check compressed file exists
	compressedPath := filepath.Join(dir, "runs", runID, "artifacts", "large.txt.gz")
	if _, err := os.Stat(compressedPath); os.IsNotExist(err) {
		t.Error("compressed file should exist")
	}

	// Load (should decompress transparently)
	loaded, err := m.LoadArtifact(runID, "large.txt")
	if err != nil {
		t.Fatalf("LoadArtifact: %v", err)
	}

	if string(loaded) != string(content) {
		t.Error("content mismatch after compression roundtrip")
	}
}

func TestArtifactManager_ListArtifacts(t *testing.T) {
	dir := t.TempDir()
	m := NewArtifactManager(ArtifactConfig{BaseDir: dir})

	runID := "test-run-001"

	// Save multiple artifacts
	m.SaveArtifact(runID, "spec.md", []byte("# Spec"))
	m.SaveArtifact(runID, "review.json", []byte(`{"approved": true}`))
	m.SaveArtifact(runID, "test-output.txt", []byte("Tests passed"))

	// List
	artifacts, err := m.ListArtifacts(runID)
	if err != nil {
		t.Fatalf("ListArtifacts: %v", err)
	}

	if len(artifacts) != 3 {
		t.Errorf("artifact count = %d, want 3", len(artifacts))
	}

	// Check sorted by name
	if artifacts[0].Name != "review.json" {
		t.Errorf("first artifact = %q, want 'review.json'", artifacts[0].Name)
	}
}

func TestArtifactManager_HasArtifact(t *testing.T) {
	dir := t.TempDir()
	m := NewArtifactManager(ArtifactConfig{BaseDir: dir})

	runID := "test-run-001"

	if m.HasArtifact(runID, "spec.md") {
		t.Error("HasArtifact should return false for non-existent artifact")
	}

	m.SaveArtifact(runID, "spec.md", []byte("# Spec"))

	if !m.HasArtifact(runID, "spec.md") {
		t.Error("HasArtifact should return true for existing artifact")
	}
}

func TestArtifactManager_DeleteArtifact(t *testing.T) {
	dir := t.TempDir()
	m := NewArtifactManager(ArtifactConfig{BaseDir: dir})

	runID := "test-run-001"
	m.SaveArtifact(runID, "spec.md", []byte("# Spec"))

	// Delete
	err := m.DeleteArtifact(runID, "spec.md")
	if err != nil {
		t.Fatalf("DeleteArtifact: %v", err)
	}

	// Verify deleted
	if m.HasArtifact(runID, "spec.md") {
		t.Error("artifact should be deleted")
	}
}

func TestArtifactManager_SaveLoadFile(t *testing.T) {
	dir := t.TempDir()
	m := NewArtifactManager(ArtifactConfig{BaseDir: dir})

	runID := "test-run-001"
	content := []byte("package main\n\nfunc main() {}")

	// Save file
	err := m.SaveFile(runID, "main.go", content)
	if err != nil {
		t.Fatalf("SaveFile: %v", err)
	}

	// Load file
	loaded, err := m.LoadFile(runID, "main.go")
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}

	if string(loaded) != string(content) {
		t.Error("file content mismatch")
	}
}

func TestArtifactManager_ListFiles(t *testing.T) {
	dir := t.TempDir()
	m := NewArtifactManager(ArtifactConfig{BaseDir: dir})

	runID := "test-run-001"

	// Save multiple files
	m.SaveFile(runID, "main.go", []byte("package main"))
	m.SaveFile(runID, "utils/helper.go", []byte("package utils"))

	// List
	files, err := m.ListFiles(runID)
	if err != nil {
		t.Fatalf("ListFiles: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("file count = %d, want 2", len(files))
	}
}

func TestArtifactManager_SaveLoadReview(t *testing.T) {
	dir := t.TempDir()
	m := NewArtifactManager(ArtifactConfig{BaseDir: dir})

	runID := "test-run-001"
	review := &ReviewResult{
		Approved: false,
		Summary:  "Found issues",
		Findings: []ReviewFinding{
			{
				File:     "main.go",
				Line:     10,
				Severity: SeverityError,
				Category: CategorySecurity,
				Message:  "SQL injection vulnerability",
			},
		},
		Metrics: ReviewMetrics{
			LinesReviewed: 100,
			FilesReviewed: 5,
			TokensUsed:    1500,
		},
	}

	// Save
	err := m.SaveReview(runID, review)
	if err != nil {
		t.Fatalf("SaveReview: %v", err)
	}

	// Load
	loaded, err := m.LoadReview(runID)
	if err != nil {
		t.Fatalf("LoadReview: %v", err)
	}

	if loaded.Approved != review.Approved {
		t.Errorf("Approved = %v, want %v", loaded.Approved, review.Approved)
	}
	if len(loaded.Findings) != 1 {
		t.Errorf("Findings count = %d, want 1", len(loaded.Findings))
	}
	if loaded.Findings[0].Message != review.Findings[0].Message {
		t.Error("Finding message mismatch")
	}
}

func TestArtifactManager_SaveLoadTestOutput(t *testing.T) {
	dir := t.TempDir()
	m := NewArtifactManager(ArtifactConfig{BaseDir: dir})

	runID := "test-run-001"
	output := &TestOutput{
		Passed:       true,
		TotalTests:   50,
		PassedTests:  48,
		FailedTests:  2,
		SkippedTests: 0,
		Duration:     "12.5s",
		Failures: []TestFailure{
			{Name: "TestAuth", Message: "timeout"},
			{Name: "TestDB", Message: "connection refused"},
		},
	}

	// Save
	err := m.SaveTestOutput(runID, output)
	if err != nil {
		t.Fatalf("SaveTestOutput: %v", err)
	}

	// Load
	loaded, err := m.LoadTestOutput(runID)
	if err != nil {
		t.Fatalf("LoadTestOutput: %v", err)
	}

	if loaded.TotalTests != output.TotalTests {
		t.Errorf("TotalTests = %d, want %d", loaded.TotalTests, output.TotalTests)
	}
	if len(loaded.Failures) != 2 {
		t.Errorf("Failures count = %d, want 2", len(loaded.Failures))
	}
}

func TestInferArtifactType(t *testing.T) {
	tests := []struct {
		filename string
		wantType string
	}{
		{"spec.md", "specification"},
		{"changes.diff", "diff"},
		{"changes.patch", "diff"},
		{"review.json", "json"},
		{"output.txt", "text"},
		{"output.log", "text"},
		{"main.go", "code"},
		{"script.py", "code"},
		{"image.png", "binary"},
		{"document.pdf", "binary"},
		{"unknown.xyz", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			at := InferArtifactType(tt.filename)
			if at.Name != tt.wantType {
				t.Errorf("InferArtifactType(%q) = %q, want %q", tt.filename, at.Name, tt.wantType)
			}
		})
	}
}

func TestReviewResult_Helpers(t *testing.T) {
	review := &ReviewResult{
		Findings: []ReviewFinding{
			{File: "a.go", Severity: SeverityCritical},
			{File: "a.go", Severity: SeverityWarning},
			{File: "b.go", Severity: SeverityError},
			{File: "b.go", Severity: SeverityInfo},
		},
	}

	if !review.HasCriticalFindings() {
		t.Error("HasCriticalFindings should return true")
	}

	if !review.HasErrors() {
		t.Error("HasErrors should return true")
	}

	byFile := review.FindingsByFile()
	if len(byFile["a.go"]) != 2 {
		t.Errorf("FindingsByFile a.go = %d, want 2", len(byFile["a.go"]))
	}

	bySeverity := review.FindingsBySeverity()
	if len(bySeverity[SeverityError]) != 1 {
		t.Errorf("FindingsBySeverity error = %d, want 1", len(bySeverity[SeverityError]))
	}
}

func TestTestOutput_SuccessRate(t *testing.T) {
	tests := []struct {
		name   string
		output TestOutput
		want   float64
	}{
		{"all pass", TestOutput{TotalTests: 10, PassedTests: 10}, 100},
		{"half pass", TestOutput{TotalTests: 10, PassedTests: 5}, 50},
		{"none pass", TestOutput{TotalTests: 10, PassedTests: 0}, 0},
		{"no tests", TestOutput{TotalTests: 0, PassedTests: 0}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.output.SuccessRate()
			if got != tt.want {
				t.Errorf("SuccessRate() = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestLifecycleManager_ArchiveRestore(t *testing.T) {
	dir := t.TempDir()

	// Create a run with artifacts
	runID := "2025-01-15-test-run"
	runDir := filepath.Join(dir, "runs", runID)
	os.MkdirAll(filepath.Join(runDir, "artifacts"), 0755)
	os.WriteFile(filepath.Join(runDir, "metadata.json"), []byte(`{"status":"completed","endedAt":"2025-01-15T10:00:00Z"}`), 0644)
	os.WriteFile(filepath.Join(runDir, "transcript.json"), []byte(`{"runId":"test"}`), 0644)
	os.WriteFile(filepath.Join(runDir, "artifacts", "spec.md"), []byte("# Spec"), 0644)

	lm := NewLifecycleManager(dir, DefaultRetentionConfig())

	// Archive
	err := lm.archiveRun(runID)
	if err != nil {
		t.Fatalf("archiveRun: %v", err)
	}

	// Verify run directory is gone
	if _, err := os.Stat(runDir); !os.IsNotExist(err) {
		t.Error("run directory should be removed after archive")
	}

	// Verify archive exists
	archives, _ := lm.ListArchives()
	found := false
	for _, a := range archives {
		if a == runID {
			found = true
			break
		}
	}
	if !found {
		t.Error("archive should exist")
	}

	// Restore
	err = lm.RestoreArchive(runID)
	if err != nil {
		t.Fatalf("RestoreArchive: %v", err)
	}

	// Verify restored
	if _, err := os.Stat(runDir); os.IsNotExist(err) {
		t.Error("run directory should be restored")
	}

	// Verify content
	content, err := os.ReadFile(filepath.Join(runDir, "artifacts", "spec.md"))
	if err != nil {
		t.Fatalf("read restored file: %v", err)
	}
	if string(content) != "# Spec" {
		t.Error("restored content mismatch")
	}
}

func TestLifecycleManager_Cleanup_DryRun(t *testing.T) {
	dir := t.TempDir()

	// Create runs of different ages
	now := time.Now()

	// Recent run (should keep)
	createTestRun(t, dir, "2025-01-15-recent", now.Add(-1*24*time.Hour), RunStatusCompleted)

	// Old run (should archive)
	createTestRun(t, dir, "2025-01-10-old", now.Add(-10*24*time.Hour), RunStatusCompleted)

	// Very old run (should delete)
	createTestRun(t, dir, "2024-12-01-ancient", now.Add(-50*24*time.Hour), RunStatusCompleted)

	lm := NewLifecycleManager(dir, RetentionConfig{
		RetentionDays:    30,
		ArchiveAfterDays: 7,
		KeepMinRuns:      0, // Don't enforce minimum for this test
	})

	// Dry run
	result, err := lm.Cleanup(true)
	if err != nil {
		t.Fatalf("Cleanup: %v", err)
	}

	if len(result.Kept) != 1 {
		t.Errorf("Kept = %d, want 1", len(result.Kept))
	}
	if len(result.Archived) != 1 {
		t.Errorf("Archived = %d, want 1", len(result.Archived))
	}
	if len(result.Deleted) != 1 {
		t.Errorf("Deleted = %d, want 1", len(result.Deleted))
	}

	// Verify nothing actually changed (dry run)
	runsDir := filepath.Join(dir, "runs")
	entries, _ := os.ReadDir(runsDir)
	if len(entries) != 3 {
		t.Errorf("runs should not be modified in dry run, got %d", len(entries))
	}
}

func TestLifecycleManager_KeepFailed(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()

	// Old failed run
	createTestRun(t, dir, "2024-12-01-failed", now.Add(-50*24*time.Hour), RunStatusFailed)

	lm := NewLifecycleManager(dir, RetentionConfig{
		RetentionDays:    30,
		ArchiveAfterDays: 7,
		KeepFailed:       true,
		KeepMinRuns:      0,
	})

	result, _ := lm.Cleanup(true)

	// Failed run should be kept
	if len(result.Kept) != 1 {
		t.Errorf("Failed run should be kept, Kept = %d", len(result.Kept))
	}
}

func TestDiskUsage(t *testing.T) {
	dir := t.TempDir()

	// Create some runs
	createTestRun(t, dir, "2025-01-15-run1", time.Now(), RunStatusCompleted)
	createTestRun(t, dir, "2025-01-15-run2", time.Now(), RunStatusCompleted)

	lm := NewLifecycleManager(dir, DefaultRetentionConfig())

	stats, err := lm.DiskUsage()
	if err != nil {
		t.Fatalf("DiskUsage: %v", err)
	}

	if stats.RunCount != 2 {
		t.Errorf("RunCount = %d, want 2", stats.RunCount)
	}
	if stats.ActiveSize == 0 {
		t.Error("ActiveSize should be > 0")
	}
}

func TestArtifactManager_SaveLoadSpec(t *testing.T) {
	dir := t.TempDir()
	m := NewArtifactManager(ArtifactConfig{BaseDir: dir})

	runID := "test-run-spec"
	spec := "# Feature Specification\n\n## Overview\n\nImplement authentication."

	// Save
	err := m.SaveSpec(runID, spec)
	if err != nil {
		t.Fatalf("SaveSpec: %v", err)
	}

	// Load
	loaded, err := m.LoadSpec(runID)
	if err != nil {
		t.Fatalf("LoadSpec: %v", err)
	}

	if loaded != spec {
		t.Errorf("LoadSpec() = %q, want %q", loaded, spec)
	}
}

func TestArtifactManager_SaveLoadLintOutput(t *testing.T) {
	dir := t.TempDir()
	m := NewArtifactManager(ArtifactConfig{BaseDir: dir})

	runID := "test-run-lint"
	output := &LintOutput{
		Passed: false,
		Tool:   "ruff",
		Issues: []LintIssue{
			{File: "main.go", Line: 10, Column: 5, Rule: "errcheck", Severity: SeverityError, Message: "error not checked"},
			{File: "util.go", Line: 20, Column: 1, Rule: "unused", Severity: SeverityWarning, Message: "unused variable"},
		},
		Summary: LintSummary{
			TotalIssues:  5,
			Errors:       2,
			Warnings:     3,
			FilesChecked: 10,
		},
	}

	// Save
	err := m.SaveLintOutput(runID, output)
	if err != nil {
		t.Fatalf("SaveLintOutput: %v", err)
	}

	// Load
	loaded, err := m.LoadLintOutput(runID)
	if err != nil {
		t.Fatalf("LoadLintOutput: %v", err)
	}

	if loaded.Summary.TotalIssues != output.Summary.TotalIssues {
		t.Errorf("TotalIssues = %d, want %d", loaded.Summary.TotalIssues, output.Summary.TotalIssues)
	}
	if len(loaded.Issues) != 2 {
		t.Errorf("Issues count = %d, want 2", len(loaded.Issues))
	}
}

func TestArtifactManager_SaveLoadDiff(t *testing.T) {
	dir := t.TempDir()
	m := NewArtifactManager(ArtifactConfig{BaseDir: dir})

	runID := "test-run-diff"
	diff := `diff --git a/main.go b/main.go
index abc1234..def5678 100644
--- a/main.go
+++ b/main.go
@@ -1,3 +1,5 @@
 package main

+import "fmt"
+
 func main() {
`

	// Save
	err := m.SaveDiff(runID, diff)
	if err != nil {
		t.Fatalf("SaveDiff: %v", err)
	}

	// Load
	loaded, err := m.LoadDiff(runID)
	if err != nil {
		t.Fatalf("LoadDiff: %v", err)
	}

	if loaded != diff {
		t.Errorf("LoadDiff() mismatch")
	}
}

func TestArtifactManager_SaveLoadJSON(t *testing.T) {
	dir := t.TempDir()
	m := NewArtifactManager(ArtifactConfig{BaseDir: dir})

	runID := "test-run-json"
	data := map[string]interface{}{
		"name":    "test",
		"count":   42,
		"enabled": true,
		"items":   []string{"a", "b", "c"},
	}

	// Save
	err := m.SaveJSON(runID, "custom-data.json", data)
	if err != nil {
		t.Fatalf("SaveJSON: %v", err)
	}

	// Load
	var loaded map[string]interface{}
	err = m.LoadJSON(runID, "custom-data.json", &loaded)
	if err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}

	if loaded["name"] != "test" {
		t.Errorf("loaded[name] = %v, want %q", loaded["name"], "test")
	}
	// JSON unmarshals numbers as float64
	if loaded["count"].(float64) != 42 {
		t.Errorf("loaded[count] = %v, want 42", loaded["count"])
	}
}

func TestArtifactManager_EnsureRunDir(t *testing.T) {
	dir := t.TempDir()
	m := NewArtifactManager(ArtifactConfig{BaseDir: dir})

	runID := "test-ensure-dir"
	err := m.EnsureRunDir(runID)
	if err != nil {
		t.Fatalf("EnsureRunDir: %v", err)
	}

	// Check directory was created
	runDir := m.RunDir(runID)
	info, err := os.Stat(runDir)
	if err != nil {
		t.Fatalf("runDir stat: %v", err)
	}
	if !info.IsDir() {
		t.Error("runDir should be a directory")
	}

	// Check artifacts dir was created
	artifactDir := m.ArtifactDir(runID)
	info, err = os.Stat(artifactDir)
	if err != nil {
		t.Fatalf("artifactDir stat: %v", err)
	}
	if !info.IsDir() {
		t.Error("artifactDir should be a directory")
	}

	// Second call should succeed (idempotent)
	err = m.EnsureRunDir(runID)
	if err != nil {
		t.Fatalf("EnsureRunDir second call: %v", err)
	}
}

func TestArtifactManager_GetArtifactInfo(t *testing.T) {
	dir := t.TempDir()
	m := NewArtifactManager(ArtifactConfig{BaseDir: dir})

	runID := "test-info"
	content := []byte("test content for info")

	err := m.SaveArtifact(runID, "test.txt", content)
	if err != nil {
		t.Fatalf("SaveArtifact: %v", err)
	}

	info, err := m.GetArtifactInfo(runID, "test.txt")
	if err != nil {
		t.Fatalf("GetArtifactInfo: %v", err)
	}

	if info.Name != "test.txt" {
		t.Errorf("Name = %q, want %q", info.Name, "test.txt")
	}
	if info.Size != int64(len(content)) {
		t.Errorf("Size = %d, want %d", info.Size, len(content))
	}
}

func TestArtifactManager_GetArtifactInfo_NotFound(t *testing.T) {
	dir := t.TempDir()
	m := NewArtifactManager(ArtifactConfig{BaseDir: dir})

	_, err := m.GetArtifactInfo("nonexistent", "file.txt")
	if err == nil {
		t.Error("GetArtifactInfo should fail for nonexistent artifact")
	}
}

func TestArtifactManager_BaseDir(t *testing.T) {
	dir := t.TempDir()
	m := NewArtifactManager(ArtifactConfig{BaseDir: dir})

	if m.BaseDir() != dir {
		t.Errorf("BaseDir() = %q, want %q", m.BaseDir(), dir)
	}
}

// Helper to create test runs
func createTestRun(t *testing.T, baseDir, runID string, endedAt time.Time, status RunStatus) {
	t.Helper()

	runDir := filepath.Join(baseDir, "runs", runID)
	os.MkdirAll(filepath.Join(runDir, "artifacts"), 0755)

	meta := TranscriptMeta{
		RunID:   runID,
		FlowID:  "test",
		Status:  status,
		EndedAt: endedAt,
	}
	data, _ := json.MarshalIndent(meta, "", "  ")
	os.WriteFile(filepath.Join(runDir, "metadata.json"), data, 0644)
	os.WriteFile(filepath.Join(runDir, "transcript.json"), []byte(`{}`), 0644)
	os.WriteFile(filepath.Join(runDir, "artifacts", "test.txt"), []byte("test content"), 0644)
}
