package devflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultRetentionConfig(t *testing.T) {
	cfg := DefaultRetentionConfig()

	if cfg.RetentionDays != 30 {
		t.Errorf("RetentionDays = %d, want 30", cfg.RetentionDays)
	}
	if cfg.ArchiveAfterDays != 7 {
		t.Errorf("ArchiveAfterDays = %d, want 7", cfg.ArchiveAfterDays)
	}
	if cfg.ArchiveRetentionDays != 90 {
		t.Errorf("ArchiveRetentionDays = %d, want 90", cfg.ArchiveRetentionDays)
	}
	if !cfg.KeepFailed {
		t.Error("KeepFailed should be true")
	}
	if cfg.KeepMinRuns != 100 {
		t.Errorf("KeepMinRuns = %d, want 100", cfg.KeepMinRuns)
	}
}

func TestNewLifecycleManager(t *testing.T) {
	manager := NewLifecycleManager("/base", DefaultRetentionConfig())

	if manager.baseDir != "/base" {
		t.Errorf("baseDir = %q, want %q", manager.baseDir, "/base")
	}
}

func createTestRunWithMetadata(t *testing.T, baseDir, runID string, status RunStatus, endedAt time.Time) {
	t.Helper()
	runDir := filepath.Join(baseDir, "runs", runID)
	if err := os.MkdirAll(runDir, 0755); err != nil {
		t.Fatalf("create run dir: %v", err)
	}

	meta := TranscriptMeta{
		RunID:   runID,
		Status:  status,
		EndedAt: endedAt,
	}

	data, _ := json.Marshal(meta)
	if err := os.WriteFile(filepath.Join(runDir, "metadata.json"), data, 0644); err != nil {
		t.Fatalf("write metadata: %v", err)
	}

	// Create a transcript file
	os.WriteFile(filepath.Join(runDir, "transcript.json"), []byte(`{"turns":[]}`), 0644)
}

func createTestArchive(t *testing.T, baseDir, runID string, modTime time.Time) {
	t.Helper()
	month := extractMonthFromRunID(runID)
	archiveDir := filepath.Join(baseDir, "archive", month)
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		t.Fatalf("create archive dir: %v", err)
	}

	archivePath := filepath.Join(archiveDir, runID+".tar.gz")
	// Create a minimal valid gzip file
	content := []byte{0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	if err := os.WriteFile(archivePath, content, 0644); err != nil {
		t.Fatalf("write archive: %v", err)
	}

	// Set modification time
	os.Chtimes(archivePath, modTime, modTime)
}

func TestLifecycleManager_Cleanup_EmptyDir(t *testing.T) {
	baseDir := t.TempDir()
	manager := NewLifecycleManager(baseDir, DefaultRetentionConfig())

	result, err := manager.Cleanup(true)
	if err != nil {
		t.Fatalf("Cleanup: %v", err)
	}

	if len(result.Deleted) != 0 {
		t.Errorf("Deleted = %v, want empty", result.Deleted)
	}
}

func TestLifecycleManager_Cleanup_KeepsMinRuns(t *testing.T) {
	baseDir := t.TempDir()
	config := RetentionConfig{
		RetentionDays:    1,
		ArchiveAfterDays: 1,
		KeepMinRuns:      2, // Keep at least 2
	}
	manager := NewLifecycleManager(baseDir, config)

	// Create 3 old runs
	oldTime := time.Now().Add(-48 * time.Hour)
	createTestRunWithMetadata(t, baseDir, "2025-01-01-run-1", RunStatusCompleted, oldTime)
	createTestRunWithMetadata(t, baseDir, "2025-01-01-run-2", RunStatusCompleted, oldTime)
	createTestRunWithMetadata(t, baseDir, "2025-01-01-run-3", RunStatusCompleted, oldTime)

	result, err := manager.Cleanup(true)
	if err != nil {
		t.Fatalf("Cleanup: %v", err)
	}

	// Should keep at least 2
	if len(result.Kept) < 2 {
		t.Errorf("Kept = %d, want at least 2", len(result.Kept))
	}
}

func TestLifecycleManager_Cleanup_KeepsFailed(t *testing.T) {
	baseDir := t.TempDir()
	config := RetentionConfig{
		RetentionDays:    1,
		ArchiveAfterDays: 1,
		KeepFailed:       true,
		KeepMinRuns:      0,
	}
	manager := NewLifecycleManager(baseDir, config)

	oldTime := time.Now().Add(-48 * time.Hour)
	createTestRunWithMetadata(t, baseDir, "2025-01-01-failed-run", RunStatusFailed, oldTime)

	result, err := manager.Cleanup(true)
	if err != nil {
		t.Fatalf("Cleanup: %v", err)
	}

	// Failed runs should be kept
	if len(result.Kept) != 1 {
		t.Errorf("Kept = %d, want 1", len(result.Kept))
	}
}

func TestLifecycleManager_Cleanup_KeepsRunning(t *testing.T) {
	baseDir := t.TempDir()
	config := RetentionConfig{
		RetentionDays:    1,
		ArchiveAfterDays: 1,
		KeepMinRuns:      0,
	}
	manager := NewLifecycleManager(baseDir, config)

	oldTime := time.Now().Add(-48 * time.Hour)
	createTestRunWithMetadata(t, baseDir, "2025-01-01-running-run", RunStatusRunning, oldTime)

	result, err := manager.Cleanup(true)
	if err != nil {
		t.Fatalf("Cleanup: %v", err)
	}

	// Running runs should be kept
	if len(result.Kept) != 1 {
		t.Errorf("Kept = %d, want 1", len(result.Kept))
	}
}

func TestLifecycleManager_ListArchives_Empty(t *testing.T) {
	baseDir := t.TempDir()
	manager := NewLifecycleManager(baseDir, DefaultRetentionConfig())

	archives, err := manager.ListArchives()
	if err != nil {
		t.Fatalf("ListArchives: %v", err)
	}

	if len(archives) != 0 {
		t.Errorf("archives = %v, want empty", archives)
	}
}

func TestLifecycleManager_ListArchives(t *testing.T) {
	baseDir := t.TempDir()
	manager := NewLifecycleManager(baseDir, DefaultRetentionConfig())

	createTestArchive(t, baseDir, "2025-01-15-test-run", time.Now())

	archives, err := manager.ListArchives()
	if err != nil {
		t.Fatalf("ListArchives: %v", err)
	}

	if len(archives) != 1 {
		t.Errorf("archives = %d, want 1", len(archives))
	}
	if archives[0] != "2025-01-15-test-run" {
		t.Errorf("archive = %q, want %q", archives[0], "2025-01-15-test-run")
	}
}

func TestLifecycleManager_DeleteArchive(t *testing.T) {
	baseDir := t.TempDir()
	manager := NewLifecycleManager(baseDir, DefaultRetentionConfig())

	runID := "2025-01-15-test-run"
	createTestArchive(t, baseDir, runID, time.Now())

	// Verify archive exists
	archives, _ := manager.ListArchives()
	if len(archives) != 1 {
		t.Fatalf("archive not created")
	}

	// Delete it
	err := manager.DeleteArchive(runID)
	if err != nil {
		t.Fatalf("DeleteArchive: %v", err)
	}

	// Verify it's gone
	archives, _ = manager.ListArchives()
	if len(archives) != 0 {
		t.Errorf("archive should be deleted, got %v", archives)
	}
}

func TestLifecycleManager_DeleteArchive_NotFound(t *testing.T) {
	baseDir := t.TempDir()
	manager := NewLifecycleManager(baseDir, DefaultRetentionConfig())

	err := manager.DeleteArchive("nonexistent-run")
	if err == nil {
		t.Error("expected error for nonexistent archive")
	}
}

func TestLifecycleManager_GetArchiveSize(t *testing.T) {
	baseDir := t.TempDir()
	manager := NewLifecycleManager(baseDir, DefaultRetentionConfig())

	runID := "2025-01-15-test-run"
	createTestArchive(t, baseDir, runID, time.Now())

	size, err := manager.GetArchiveSize(runID)
	if err != nil {
		t.Fatalf("GetArchiveSize: %v", err)
	}

	if size <= 0 {
		t.Errorf("size = %d, want > 0", size)
	}
}

func TestLifecycleManager_GetArchiveSize_NotFound(t *testing.T) {
	baseDir := t.TempDir()
	manager := NewLifecycleManager(baseDir, DefaultRetentionConfig())

	_, err := manager.GetArchiveSize("nonexistent-run")
	if err == nil {
		t.Error("expected error for nonexistent archive")
	}
}

func TestLifecycleManager_CleanupArchives_Empty(t *testing.T) {
	baseDir := t.TempDir()
	manager := NewLifecycleManager(baseDir, DefaultRetentionConfig())

	result, err := manager.CleanupArchives(true)
	if err != nil {
		t.Fatalf("CleanupArchives: %v", err)
	}

	if len(result.Deleted) != 0 {
		t.Errorf("Deleted = %v, want empty", result.Deleted)
	}
}

func TestLifecycleManager_CleanupArchives_DeletesOld(t *testing.T) {
	baseDir := t.TempDir()
	config := RetentionConfig{
		ArchiveRetentionDays: 1, // 1 day retention
	}
	manager := NewLifecycleManager(baseDir, config)

	// Create an old archive (2 days ago)
	oldTime := time.Now().Add(-48 * time.Hour)
	createTestArchive(t, baseDir, "2025-01-15-old-run", oldTime)

	result, err := manager.CleanupArchives(true) // dry run
	if err != nil {
		t.Fatalf("CleanupArchives: %v", err)
	}

	if len(result.Deleted) != 1 {
		t.Errorf("Deleted = %d, want 1", len(result.Deleted))
	}
}

func TestLifecycleManager_CleanupArchives_KeepsRecent(t *testing.T) {
	baseDir := t.TempDir()
	config := RetentionConfig{
		ArchiveRetentionDays: 30, // 30 day retention
	}
	manager := NewLifecycleManager(baseDir, config)

	// Create a recent archive
	createTestArchive(t, baseDir, "2025-01-15-recent-run", time.Now())

	result, err := manager.CleanupArchives(true)
	if err != nil {
		t.Fatalf("CleanupArchives: %v", err)
	}

	if len(result.Kept) != 1 {
		t.Errorf("Kept = %d, want 1", len(result.Kept))
	}
	if len(result.Deleted) != 0 {
		t.Errorf("Deleted = %d, want 0", len(result.Deleted))
	}
}

func TestLifecycleManager_CleanupArchives_ActualDelete(t *testing.T) {
	baseDir := t.TempDir()
	config := RetentionConfig{
		ArchiveRetentionDays: 1,
	}
	manager := NewLifecycleManager(baseDir, config)

	oldTime := time.Now().Add(-48 * time.Hour)
	runID := "2025-01-15-old-run"
	createTestArchive(t, baseDir, runID, oldTime)

	// Verify archive exists
	archives, _ := manager.ListArchives()
	if len(archives) != 1 {
		t.Fatalf("archive not created")
	}

	result, err := manager.CleanupArchives(false) // actual delete
	if err != nil {
		t.Fatalf("CleanupArchives: %v", err)
	}

	if len(result.Deleted) != 1 {
		t.Errorf("Deleted = %d, want 1", len(result.Deleted))
	}

	// Verify archive is gone
	archives, _ = manager.ListArchives()
	if len(archives) != 0 {
		t.Errorf("archive should be deleted")
	}
}

func TestLifecycleManager_DiskUsage(t *testing.T) {
	baseDir := t.TempDir()
	manager := NewLifecycleManager(baseDir, DefaultRetentionConfig())

	// Create a run
	createTestRunWithMetadata(t, baseDir, "2025-01-15-test-run", RunStatusCompleted, time.Now())

	// Create an archive
	createTestArchive(t, baseDir, "2025-01-14-archived-run", time.Now())

	stats, err := manager.DiskUsage()
	if err != nil {
		t.Fatalf("DiskUsage: %v", err)
	}

	if stats.RunCount != 1 {
		t.Errorf("RunCount = %d, want 1", stats.RunCount)
	}
	if stats.ArchiveCount != 1 {
		t.Errorf("ArchiveCount = %d, want 1", stats.ArchiveCount)
	}
	if stats.ActiveSize <= 0 {
		t.Errorf("ActiveSize = %d, want > 0", stats.ActiveSize)
	}
	if stats.ArchiveSize <= 0 {
		t.Errorf("ArchiveSize = %d, want > 0", stats.ArchiveSize)
	}
	if stats.TotalSize != stats.ActiveSize+stats.ArchiveSize {
		t.Errorf("TotalSize = %d, want %d", stats.TotalSize, stats.ActiveSize+stats.ArchiveSize)
	}
}

func TestExtractMonthFromRunID(t *testing.T) {
	tests := []struct {
		runID string
		want  string
	}{
		{"2025-01-15-test-run", "2025-01"},
		{"2024-12-01-another-run", "2024-12"},
		{"short", time.Now().Format("2006-01")}, // Falls back to current month
		{"", time.Now().Format("2006-01")},      // Falls back to current month
	}

	for _, tt := range tests {
		t.Run(tt.runID, func(t *testing.T) {
			got := extractMonthFromRunID(tt.runID)
			if got != tt.want {
				t.Errorf("extractMonthFromRunID(%q) = %q, want %q", tt.runID, got, tt.want)
			}
		})
	}
}

func TestDirSize(t *testing.T) {
	dir := t.TempDir()

	// Empty dir
	size := dirSize(dir)
	if size != 0 {
		t.Errorf("empty dir size = %d, want 0", size)
	}

	// Add some files
	os.WriteFile(filepath.Join(dir, "file1.txt"), []byte("hello"), 0644)
	os.WriteFile(filepath.Join(dir, "file2.txt"), []byte("world!"), 0644)

	size = dirSize(dir)
	if size != 11 { // 5 + 6 bytes
		t.Errorf("dir size = %d, want 11", size)
	}
}

func TestLifecycleManager_RestoreArchive_NotFound(t *testing.T) {
	baseDir := t.TempDir()
	manager := NewLifecycleManager(baseDir, DefaultRetentionConfig())

	err := manager.RestoreArchive("nonexistent-run")
	if err == nil {
		t.Error("expected error for nonexistent archive")
	}
}

func TestLifecycleManager_RestoreArchive_AlreadyExists(t *testing.T) {
	baseDir := t.TempDir()
	manager := NewLifecycleManager(baseDir, DefaultRetentionConfig())

	runID := "2025-01-15-test-run"

	// Create archive
	createTestArchive(t, baseDir, runID, time.Now())

	// Create existing run dir
	runDir := filepath.Join(baseDir, "runs", runID)
	os.MkdirAll(runDir, 0755)

	err := manager.RestoreArchive(runID)
	if err == nil {
		t.Error("expected error when run already exists")
	}
}
