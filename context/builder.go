package context

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

// ContextLimits configures file context limits.
type ContextLimits struct {
	MaxFileSize  int64 // Max size per file in bytes
	MaxTotalSize int64 // Max total size in bytes
	MaxFileCount int   // Max number of files
}

// DefaultContextLimits returns sensible default limits.
func DefaultContextLimits() ContextLimits {
	return ContextLimits{
		MaxFileSize:  100 * 1024, // 100KB per file
		MaxTotalSize: 500 * 1024, // 500KB total
		MaxFileCount: 50,         // 50 files max
	}
}

// ContextBuilder builds file context for Claude.
type ContextBuilder struct {
	workDir string
	limits  ContextLimits
	files   []contextFile
}

type contextFile struct {
	path    string
	content []byte
	binary  bool
}

// NewContextBuilder creates a context builder for the given working directory.
func NewContextBuilder(workDir string) *ContextBuilder {
	return &ContextBuilder{
		workDir: workDir,
		limits:  DefaultContextLimits(),
	}
}

// WithLimits sets custom context limits.
func (b *ContextBuilder) WithLimits(limits ContextLimits) *ContextBuilder {
	b.limits = limits
	return b
}

// AddFile adds a single file to the context.
func (b *ContextBuilder) AddFile(path string) error {
	fullPath := filepath.Join(b.workDir, path)

	info, err := os.Stat(fullPath)
	if err != nil {
		return fmt.Errorf("stat %s: %w", path, err)
	}

	if info.IsDir() {
		return fmt.Errorf("%s is a directory", path)
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	b.files = append(b.files, contextFile{
		path:    path,
		content: content,
		binary:  isBinary(content),
	})

	return nil
}

// AddGlob adds files matching a glob pattern.
func (b *ContextBuilder) AddGlob(pattern string) error {
	matches, err := filepath.Glob(filepath.Join(b.workDir, pattern))
	if err != nil {
		return fmt.Errorf("glob %s: %w", pattern, err)
	}

	for _, match := range matches {
		relPath, err := filepath.Rel(b.workDir, match)
		if err != nil {
			slog.Debug("skipping file with invalid path",
				slog.String("path", match),
				slog.String("error", err.Error()))
			continue
		}

		info, err := os.Stat(match)
		if err != nil {
			slog.Debug("skipping file with stat error",
				slog.String("path", match),
				slog.String("error", err.Error()))
			continue
		}

		if !info.IsDir() {
			if err := b.AddFile(relPath); err != nil {
				slog.Debug("skipping unreadable file",
					slog.String("path", relPath),
					slog.String("error", err.Error()))
				continue
			}
		}
	}

	return nil
}

// AddContent adds pre-loaded content with a virtual path.
func (b *ContextBuilder) AddContent(path string, content []byte) {
	b.files = append(b.files, contextFile{
		path:    path,
		content: content,
		binary:  isBinary(content),
	})
}

// Build generates the formatted context string.
func (b *ContextBuilder) Build() (string, error) {
	// Check file count
	if len(b.files) > b.limits.MaxFileCount {
		return "", fmt.Errorf("%w: %d files > max %d",
			ErrContextTooLarge, len(b.files), b.limits.MaxFileCount)
	}

	var buf bytes.Buffer
	var totalSize int64

	for _, f := range b.files {
		content := f.content

		// Handle binary files
		if f.binary {
			mimeType := detectMimeType(content)
			fmt.Fprintf(&buf, "<file path=%q>\n", f.path)
			fmt.Fprintf(&buf, "[Binary file: %d bytes, type: %s]\n", len(content), mimeType)
			buf.WriteString("</file>\n\n")
			continue
		}

		// Truncate large files
		if int64(len(content)) > b.limits.MaxFileSize {
			content = content[:b.limits.MaxFileSize]
			content = append(content, []byte("\n\n[... truncated ...]")...)
		}

		// Check total size
		totalSize += int64(len(content))
		if totalSize > b.limits.MaxTotalSize {
			return "", fmt.Errorf("%w: total size %d > max %d",
				ErrContextTooLarge, totalSize, b.limits.MaxTotalSize)
		}

		// Format file with XML-style tags
		fmt.Fprintf(&buf, "<file path=%q>\n", f.path)
		buf.Write(content)
		if !bytes.HasSuffix(content, []byte("\n")) {
			buf.WriteByte('\n')
		}
		buf.WriteString("</file>\n\n")
	}

	return buf.String(), nil
}

// FileCount returns the number of files added.
func (b *ContextBuilder) FileCount() int {
	return len(b.files)
}

// TotalSize returns the total size of all files.
func (b *ContextBuilder) TotalSize() int64 {
	var total int64
	for _, f := range b.files {
		total += int64(len(f.content))
	}
	return total
}

// Clear removes all files from the builder.
func (b *ContextBuilder) Clear() {
	b.files = nil
}

// isBinary detects if content is binary by checking for null bytes.
func isBinary(data []byte) bool {
	sample := data
	if len(sample) > 8192 {
		sample = sample[:8192]
	}
	return bytes.Contains(sample, []byte{0})
}

// detectMimeType detects MIME type from content using magic bytes.
func detectMimeType(data []byte) string {
	if len(data) < 4 {
		return "application/octet-stream"
	}

	switch {
	case bytes.HasPrefix(data, []byte{0x89, 'P', 'N', 'G'}):
		return "image/png"
	case bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF}):
		return "image/jpeg"
	case bytes.HasPrefix(data, []byte("GIF8")):
		return "image/gif"
	case bytes.HasPrefix(data, []byte("PK")):
		return "application/zip"
	case bytes.HasPrefix(data, []byte("%PDF")):
		return "application/pdf"
	case bytes.HasPrefix(data, []byte{0x7F, 'E', 'L', 'F'}):
		return "application/x-elf"
	case bytes.HasPrefix(data, []byte("MZ")):
		return "application/x-msdownload"
	default:
		return "application/octet-stream"
	}
}

// FileSelector helps select relevant files for context.
type FileSelector struct {
	workDir  string
	includes []string
	excludes []string
}

// NewFileSelector creates a file selector for the given directory.
func NewFileSelector(workDir string) *FileSelector {
	return &FileSelector{
		workDir: workDir,
	}
}

// Include adds include patterns.
func (s *FileSelector) Include(patterns ...string) *FileSelector {
	s.includes = append(s.includes, patterns...)
	return s
}

// Exclude adds exclude patterns.
func (s *FileSelector) Exclude(patterns ...string) *FileSelector {
	s.excludes = append(s.excludes, patterns...)
	return s
}

// Select returns files matching the include patterns but not the exclude patterns.
func (s *FileSelector) Select() ([]string, error) {
	matches := make(map[string]bool)

	// Process includes
	for _, pattern := range s.includes {
		paths, err := filepath.Glob(filepath.Join(s.workDir, pattern))
		if err != nil {
			return nil, fmt.Errorf("glob %s: %w", pattern, err)
		}
		for _, path := range paths {
			relPath, _ := filepath.Rel(s.workDir, path)
			matches[relPath] = true
		}
	}

	// Process excludes
	for _, pattern := range s.excludes {
		paths, err := filepath.Glob(filepath.Join(s.workDir, pattern))
		if err != nil {
			continue // Ignore invalid exclude patterns
		}
		for _, path := range paths {
			relPath, _ := filepath.Rel(s.workDir, path)
			delete(matches, relPath)
		}
	}

	// Convert to slice
	result := make([]string, 0, len(matches))
	for path := range matches {
		info, err := os.Stat(filepath.Join(s.workDir, path))
		if err != nil || info.IsDir() {
			continue
		}
		result = append(result, path)
	}

	return result, nil
}
