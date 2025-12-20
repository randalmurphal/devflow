package artifact

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// RetentionConfig defines retention policy
type RetentionConfig struct {
	RetentionDays        int  // Days to keep active runs
	ArchiveAfterDays     int  // Days before archiving
	ArchiveRetentionDays int  // Days to keep archived runs
	KeepFailed           bool // Keep failed runs longer
	KeepMinRuns          int  // Minimum runs to keep regardless of age
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
	Archived   []string `json:"archived"`
	Deleted    []string `json:"deleted"`
	Kept       []string `json:"kept"`
	Errors     []string `json:"errors,omitempty"`
	SpaceSaved int64    `json:"spaceSaved"`
}

// Cleanup performs retention policy
func (m *LifecycleManager) Cleanup(dryRun bool) (*CleanupResult, error) {
	result := &CleanupResult{
		Archived: make([]string, 0),
		Deleted:  make([]string, 0),
		Kept:     make([]string, 0),
		Errors:   make([]string, 0),
	}

	runsDir := filepath.Join(m.baseDir, "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return nil, err
	}

	now := time.Now()
	archiveThreshold := now.Add(-time.Duration(m.config.ArchiveAfterDays) * 24 * time.Hour)
	deleteThreshold := now.Add(-time.Duration(m.config.RetentionDays) * 24 * time.Hour)

	type runInfo struct {
		id      string
		meta    *transcriptMeta
		size    int64
		endedAt time.Time
	}

	var runs []runInfo

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		runID := entry.Name()
		runDir := filepath.Join(runsDir, runID)

		// Load metadata
		meta, err := loadRunMetadataFromDir(runDir)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("load %s: %v", runID, err))
			continue
		}

		// Calculate directory size
		size := dirSize(runDir)

		runs = append(runs, runInfo{
			id:      runID,
			meta:    meta,
			size:    size,
			endedAt: meta.EndedAt,
		})
	}

	// Sort by end time (oldest first)
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].endedAt.Before(runs[j].endedAt)
	})

	// Calculate how many we can potentially remove
	canRemove := len(runs) - m.config.KeepMinRuns
	if canRemove < 0 {
		canRemove = 0
	}

	removed := 0
	for _, run := range runs {
		// Skip failed runs if configured
		if m.config.KeepFailed && run.meta.Status == "failed" {
			result.Kept = append(result.Kept, run.id)
			continue
		}

		// Still running - keep
		if run.meta.Status == "running" {
			result.Kept = append(result.Kept, run.id)
			continue
		}

		// Ensure we keep minimum runs
		remainingAfterThis := len(runs) - removed - 1
		if remainingAfterThis < m.config.KeepMinRuns {
			result.Kept = append(result.Kept, run.id)
			continue
		}

		runDir := filepath.Join(runsDir, run.id)

		// Determine action based on age
		if run.endedAt.Before(deleteThreshold) {
			// Delete
			if !dryRun {
				if err := os.RemoveAll(runDir); err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("delete %s: %v", run.id, err))
					continue
				}
			}
			result.Deleted = append(result.Deleted, run.id)
			result.SpaceSaved += run.size
			removed++

		} else if run.endedAt.Before(archiveThreshold) {
			// Archive
			if !dryRun {
				if err := m.archiveRun(run.id); err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("archive %s: %v", run.id, err))
					continue
				}
			}
			result.Archived = append(result.Archived, run.id)
			// Archives are smaller due to compression
			result.SpaceSaved += run.size / 2 // Rough estimate
			removed++

		} else {
			result.Kept = append(result.Kept, run.id)
		}
	}

	return result, nil
}

// archiveRun compresses a run to archive
func (m *LifecycleManager) archiveRun(runID string) error {
	runDir := filepath.Join(m.baseDir, "runs", runID)

	// Determine archive path using date from runID (e.g., "2025-01-15-...")
	archiveMonth := extractMonthFromRunID(runID)
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
			_, err = io.Copy(tw, file)
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		os.Remove(archivePath)
		return err
	}

	// Close writers before removing source
	tw.Close()
	gz.Close()
	f.Close()

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

	runDir := filepath.Join(m.baseDir, "runs", runID)

	// Check if run already exists
	if _, err := os.Stat(runDir); err == nil {
		return fmt.Errorf("run already exists: %s", runID)
	}

	// Extract
	if err := m.extractArchive(archivePath, filepath.Dir(runDir)); err != nil {
		return err
	}

	return nil
}

// ListArchives returns all archived run IDs
func (m *LifecycleManager) ListArchives() ([]string, error) {
	archiveDir := filepath.Join(m.baseDir, "archive")
	var archives []string

	err := filepath.Walk(archiveDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Ignore errors, just skip
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(info.Name(), ".tar.gz") {
			runID := strings.TrimSuffix(info.Name(), ".tar.gz")
			archives = append(archives, runID)
		}
		return nil
	})

	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return archives, nil
}

// DeleteArchive removes an archived run
func (m *LifecycleManager) DeleteArchive(runID string) error {
	archivePath := m.findArchive(runID)
	if archivePath == "" {
		return fmt.Errorf("archive not found: %s", runID)
	}
	return os.Remove(archivePath)
}

// GetArchiveSize returns the size of an archive
func (m *LifecycleManager) GetArchiveSize(runID string) (int64, error) {
	archivePath := m.findArchive(runID)
	if archivePath == "" {
		return 0, fmt.Errorf("archive not found: %s", runID)
	}

	info, err := os.Stat(archivePath)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func (m *LifecycleManager) findArchive(runID string) string {
	archiveMonth := extractMonthFromRunID(runID)
	path := filepath.Join(m.baseDir, "archive", archiveMonth, runID+".tar.gz")
	if _, err := os.Stat(path); err == nil {
		return path
	}

	// Try searching all archive directories
	archiveDir := filepath.Join(m.baseDir, "archive")
	var found string
	filepath.Walk(archiveDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.Name() == runID+".tar.gz" {
			found = path
			return filepath.SkipAll
		}
		return nil
	})

	return found
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

		target := filepath.Join(destDir, header.Name)

		// Ensure target is within destDir (security check)
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)) {
			return fmt.Errorf("invalid path in archive: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			out, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()

			// Restore file permissions
			if err := os.Chmod(target, os.FileMode(header.Mode)); err != nil {
				return err
			}
		}
	}

	return nil
}

// CleanupArchives removes archives older than retention period
func (m *LifecycleManager) CleanupArchives(dryRun bool) (*CleanupResult, error) {
	result := &CleanupResult{
		Deleted: make([]string, 0),
		Kept:    make([]string, 0),
		Errors:  make([]string, 0),
	}

	archiveDir := filepath.Join(m.baseDir, "archive")
	threshold := time.Now().Add(-time.Duration(m.config.ArchiveRetentionDays) * 24 * time.Hour)

	err := filepath.Walk(archiveDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".tar.gz") {
			return nil
		}

		runID := strings.TrimSuffix(info.Name(), ".tar.gz")

		if info.ModTime().Before(threshold) {
			if !dryRun {
				if err := os.Remove(path); err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("delete archive %s: %v", runID, err))
					return nil
				}
			}
			result.Deleted = append(result.Deleted, runID)
			result.SpaceSaved += info.Size()
		} else {
			result.Kept = append(result.Kept, runID)
		}

		return nil
	})

	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return result, nil
}

// DiskUsage returns disk usage statistics
func (m *LifecycleManager) DiskUsage() (*DiskUsageStats, error) {
	stats := &DiskUsageStats{}

	runsDir := filepath.Join(m.baseDir, "runs")
	archiveDir := filepath.Join(m.baseDir, "archive")

	// Calculate runs directory size
	runEntries, err := os.ReadDir(runsDir)
	if err == nil {
		stats.RunCount = len(runEntries)
		for _, entry := range runEntries {
			if entry.IsDir() {
				stats.ActiveSize += dirSize(filepath.Join(runsDir, entry.Name()))
			}
		}
	}

	// Calculate archive directory size
	filepath.Walk(archiveDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".tar.gz") {
			stats.ArchiveSize += info.Size()
			stats.ArchiveCount++
		}
		return nil
	})

	stats.TotalSize = stats.ActiveSize + stats.ArchiveSize

	return stats, nil
}

// DiskUsageStats contains disk usage statistics
type DiskUsageStats struct {
	RunCount     int   `json:"runCount"`
	ArchiveCount int   `json:"archiveCount"`
	ActiveSize   int64 `json:"activeSize"`
	ArchiveSize  int64 `json:"archiveSize"`
	TotalSize    int64 `json:"totalSize"`
}

// Helper functions

// transcriptMeta is a minimal type for reading metadata
// This avoids circular imports with the main devflow package
type transcriptMeta struct {
	Status  string    `json:"status"`
	EndedAt time.Time `json:"endedAt"`
}

func loadRunMetadataFromDir(runDir string) (*transcriptMeta, error) {
	data, err := os.ReadFile(filepath.Join(runDir, "metadata.json"))
	if err != nil {
		return nil, err
	}
	var meta transcriptMeta
	return &meta, json.Unmarshal(data, &meta)
}

func extractMonthFromRunID(runID string) string {
	// Expected format: "2025-01-15-..."
	if len(runID) >= 7 {
		return runID[:7] // "2025-01"
	}
	return time.Now().Format("2006-01")
}

func dirSize(path string) int64 {
	var size int64
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}
