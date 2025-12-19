package devflow

import (
	"compress/gzip"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Artifact errors
var (
	ErrArtifactNotFound = errors.New("artifact not found")
)

// ArtifactConfig holds configuration for artifact management
type ArtifactConfig struct {
	BaseDir       string // Base directory for storage (default: ".devflow")
	CompressAbove int64  // Compress artifacts larger than this (default: 10KB)
	RetentionDays int    // Days to keep artifacts (default: 30)
}

// ArtifactManager manages run artifacts
type ArtifactManager struct {
	baseDir       string
	compressAbove int64
	retentionDays int
}

// ArtifactInfo contains metadata about a stored artifact
type ArtifactInfo struct {
	Name       string    `json:"name"`
	Size       int64     `json:"size"`
	Compressed bool      `json:"compressed"`
	CreatedAt  time.Time `json:"createdAt"`
	Type       string    `json:"type"`
}

// Standard artifact names
const (
	ArtifactSpec           = "spec.md"
	ArtifactImplementation = "implementation.diff"
	ArtifactReview         = "review.json"
	ArtifactTestOutput     = "test-output.json"
	ArtifactLintOutput     = "lint-output.json"
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
	"json":          {"json", []string{".json"}, true, true},
	"text":          {"text", []string{".txt", ".log"}, true, true},
	"code":          {"code", []string{".go", ".py", ".js", ".ts", ".java", ".rb"}, true, true},
	"binary":        {"binary", []string{".png", ".jpg", ".jpeg", ".gif", ".pdf", ".zip", ".tar", ".gz"}, false, false},
}

// NewArtifactManager creates an artifact manager with the given config
func NewArtifactManager(cfg ArtifactConfig) *ArtifactManager {
	if cfg.BaseDir == "" {
		cfg.BaseDir = ".devflow"
	}
	if cfg.CompressAbove == 0 {
		cfg.CompressAbove = 10 * 1024 // 10KB
	}
	if cfg.RetentionDays == 0 {
		cfg.RetentionDays = 30
	}

	return &ArtifactManager{
		baseDir:       cfg.BaseDir,
		compressAbove: cfg.CompressAbove,
		retentionDays: cfg.RetentionDays,
	}
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

// SaveArtifact saves an artifact with automatic compression
func (m *ArtifactManager) SaveArtifact(runID, name string, data []byte) error {
	artifactType := InferArtifactType(name)
	artifactPath := filepath.Join(m.ArtifactDir(runID), name)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0755); err != nil {
		return err
	}

	// Compress if needed
	if m.shouldCompress(artifactType, int64(len(data))) {
		// Remove uncompressed version if it exists
		os.Remove(artifactPath)
		return m.saveCompressed(artifactPath+".gz", data)
	}

	// Remove compressed version if it exists
	os.Remove(artifactPath + ".gz")
	return os.WriteFile(artifactPath, data, 0644)
}

// LoadArtifact loads an artifact (handles compression transparently)
func (m *ArtifactManager) LoadArtifact(runID, name string) ([]byte, error) {
	artifactPath := filepath.Join(m.ArtifactDir(runID), name)

	// Try compressed first
	if data, err := m.loadCompressed(artifactPath + ".gz"); err == nil {
		return data, nil
	}

	// Try uncompressed
	data, err := os.ReadFile(artifactPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrArtifactNotFound
		}
		return nil, err
	}
	return data, nil
}

// DeleteArtifact removes an artifact
func (m *ArtifactManager) DeleteArtifact(runID, name string) error {
	artifactPath := filepath.Join(m.ArtifactDir(runID), name)

	// Try to remove both compressed and uncompressed
	os.Remove(artifactPath + ".gz")
	err := os.Remove(artifactPath)
	if err != nil && os.IsNotExist(err) {
		return ErrArtifactNotFound
	}
	return err
}

// ListArtifacts returns all artifacts for a run
func (m *ArtifactManager) ListArtifacts(runID string) ([]ArtifactInfo, error) {
	artifactDir := m.ArtifactDir(runID)
	entries, err := os.ReadDir(artifactDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var artifacts []ArtifactInfo

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		compressed := false

		// Handle .gz extension
		if strings.HasSuffix(name, ".gz") {
			name = strings.TrimSuffix(name, ".gz")
			compressed = true
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		artifactType := InferArtifactType(name)

		artifacts = append(artifacts, ArtifactInfo{
			Name:       name,
			Size:       info.Size(),
			Compressed: compressed,
			CreatedAt:  info.ModTime(),
			Type:       artifactType.Name,
		})
	}

	// Sort by name
	sort.Slice(artifacts, func(i, j int) bool {
		return artifacts[i].Name < artifacts[j].Name
	})

	return artifacts, nil
}

// HasArtifact checks if an artifact exists
func (m *ArtifactManager) HasArtifact(runID, name string) bool {
	artifactPath := filepath.Join(m.ArtifactDir(runID), name)

	// Check both compressed and uncompressed
	if _, err := os.Stat(artifactPath + ".gz"); err == nil {
		return true
	}
	if _, err := os.Stat(artifactPath); err == nil {
		return true
	}
	return false
}

// GetArtifactInfo returns info about a specific artifact
func (m *ArtifactManager) GetArtifactInfo(runID, name string) (*ArtifactInfo, error) {
	artifactPath := filepath.Join(m.ArtifactDir(runID), name)

	// Try compressed first
	if info, err := os.Stat(artifactPath + ".gz"); err == nil {
		artifactType := InferArtifactType(name)
		return &ArtifactInfo{
			Name:       name,
			Size:       info.Size(),
			Compressed: true,
			CreatedAt:  info.ModTime(),
			Type:       artifactType.Name,
		}, nil
	}

	// Try uncompressed
	info, err := os.Stat(artifactPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrArtifactNotFound
		}
		return nil, err
	}

	artifactType := InferArtifactType(name)
	return &ArtifactInfo{
		Name:       name,
		Size:       info.Size(),
		Compressed: false,
		CreatedAt:  info.ModTime(),
		Type:       artifactType.Name,
	}, nil
}

// SaveFile saves a generated file to the files subdirectory
func (m *ArtifactManager) SaveFile(runID, filename string, data []byte) error {
	if err := os.MkdirAll(m.FilesDir(runID), 0755); err != nil {
		return err
	}

	filePath := filepath.Join(m.FilesDir(runID), filename)

	// Ensure any nested directories exist
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

// LoadFile loads a generated file from the files subdirectory
func (m *ArtifactManager) LoadFile(runID, filename string) ([]byte, error) {
	filePath := filepath.Join(m.FilesDir(runID), filename)
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrArtifactNotFound
		}
		return nil, err
	}
	return data, nil
}

// ListFiles returns all generated files for a run
func (m *ArtifactManager) ListFiles(runID string) ([]string, error) {
	filesDir := m.FilesDir(runID)
	var files []string

	err := filepath.Walk(filesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(filesDir, path)
		if err != nil {
			return err
		}
		files = append(files, relPath)
		return nil
	})

	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return files, nil
}

// BaseDir returns the base directory
func (m *ArtifactManager) BaseDir() string {
	return m.baseDir
}

func (m *ArtifactManager) shouldCompress(at ArtifactType, size int64) bool {
	if !at.Compressible {
		return false
	}
	return size >= m.compressAbove
}

func (m *ArtifactManager) saveCompressed(path string, data []byte) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	defer gz.Close()

	_, err = gz.Write(data)
	return err
}

func (m *ArtifactManager) loadCompressed(path string) ([]byte, error) {
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

// InferArtifactType infers the artifact type from filename
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
