package devflow

import (
	"bytes"
	"context"
	"fmt"
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
		MaxFileSize:  100 * 1024,  // 100KB per file
		MaxTotalSize: 500 * 1024,  // 500KB total
		MaxFileCount: 50,          // 50 files max
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
			continue
		}

		info, err := os.Stat(match)
		if err != nil {
			continue
		}

		if !info.IsDir() {
			if err := b.AddFile(relPath); err != nil {
				// Skip files that can't be read
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

// =============================================================================
// Context Injection Helpers
// =============================================================================
// These helpers allow devflow services to be injected into context.Context
// for use by flowgraph nodes.

// serviceContextKey is a private type for context keys to avoid collisions
type serviceContextKey string

// Context keys for devflow services
const (
	gitServiceKey        serviceContextKey = "devflow.git"
	claudeServiceKey     serviceContextKey = "devflow.claude"
	transcriptServiceKey serviceContextKey = "devflow.transcripts"
	artifactServiceKey   serviceContextKey = "devflow.artifacts"
	promptServiceKey     serviceContextKey = "devflow.prompts"
)

// WithGitContext adds a GitContext to the context
func WithGitContext(ctx context.Context, git *GitContext) context.Context {
	return context.WithValue(ctx, gitServiceKey, git)
}

// GitFromContext extracts GitContext from context
func GitFromContext(ctx context.Context) *GitContext {
	if git, ok := ctx.Value(gitServiceKey).(*GitContext); ok {
		return git
	}
	return nil
}

// MustGitFromContext extracts GitContext or panics
func MustGitFromContext(ctx context.Context) *GitContext {
	git := GitFromContext(ctx)
	if git == nil {
		panic("devflow: GitContext not found in context")
	}
	return git
}

// WithClaudeCLI adds a ClaudeCLI to the context
func WithClaudeCLI(ctx context.Context, claude *ClaudeCLI) context.Context {
	return context.WithValue(ctx, claudeServiceKey, claude)
}

// ClaudeFromContext extracts ClaudeCLI from context
func ClaudeFromContext(ctx context.Context) *ClaudeCLI {
	if claude, ok := ctx.Value(claudeServiceKey).(*ClaudeCLI); ok {
		return claude
	}
	return nil
}

// MustClaudeFromContext extracts ClaudeCLI or panics
func MustClaudeFromContext(ctx context.Context) *ClaudeCLI {
	claude := ClaudeFromContext(ctx)
	if claude == nil {
		panic("devflow: ClaudeCLI not found in context")
	}
	return claude
}

// WithTranscriptManager adds a TranscriptManager to the context
func WithTranscriptManager(ctx context.Context, mgr TranscriptManager) context.Context {
	return context.WithValue(ctx, transcriptServiceKey, mgr)
}

// TranscriptManagerFromContext extracts TranscriptManager from context
func TranscriptManagerFromContext(ctx context.Context) TranscriptManager {
	if mgr, ok := ctx.Value(transcriptServiceKey).(TranscriptManager); ok {
		return mgr
	}
	return nil
}

// MustTranscriptManagerFromContext extracts TranscriptManager or panics
func MustTranscriptManagerFromContext(ctx context.Context) TranscriptManager {
	mgr := TranscriptManagerFromContext(ctx)
	if mgr == nil {
		panic("devflow: TranscriptManager not found in context")
	}
	return mgr
}

// WithArtifactManager adds an ArtifactManager to the context
func WithArtifactManager(ctx context.Context, mgr *ArtifactManager) context.Context {
	return context.WithValue(ctx, artifactServiceKey, mgr)
}

// ArtifactManagerFromContext extracts ArtifactManager from context
func ArtifactManagerFromContext(ctx context.Context) *ArtifactManager {
	if mgr, ok := ctx.Value(artifactServiceKey).(*ArtifactManager); ok {
		return mgr
	}
	return nil
}

// MustArtifactManagerFromContext extracts ArtifactManager or panics
func MustArtifactManagerFromContext(ctx context.Context) *ArtifactManager {
	mgr := ArtifactManagerFromContext(ctx)
	if mgr == nil {
		panic("devflow: ArtifactManager not found in context")
	}
	return mgr
}

// WithPromptLoader adds a PromptLoader to the context
func WithPromptLoader(ctx context.Context, loader *PromptLoader) context.Context {
	return context.WithValue(ctx, promptServiceKey, loader)
}

// PromptLoaderFromContext extracts PromptLoader from context
func PromptLoaderFromContext(ctx context.Context) *PromptLoader {
	if loader, ok := ctx.Value(promptServiceKey).(*PromptLoader); ok {
		return loader
	}
	return nil
}

// MustPromptLoaderFromContext extracts PromptLoader or panics
func MustPromptLoaderFromContext(ctx context.Context) *PromptLoader {
	loader := PromptLoaderFromContext(ctx)
	if loader == nil {
		panic("devflow: PromptLoader not found in context")
	}
	return loader
}

// DevServices wraps all devflow services for convenient initialization
type DevServices struct {
	Git         *GitContext
	Claude      *ClaudeCLI
	Transcripts TranscriptManager
	Artifacts   *ArtifactManager
	Prompts     *PromptLoader
}

// InjectAll adds all configured services to the context
func (d *DevServices) InjectAll(ctx context.Context) context.Context {
	if d.Git != nil {
		ctx = WithGitContext(ctx, d.Git)
	}
	if d.Claude != nil {
		ctx = WithClaudeCLI(ctx, d.Claude)
	}
	if d.Transcripts != nil {
		ctx = WithTranscriptManager(ctx, d.Transcripts)
	}
	if d.Artifacts != nil {
		ctx = WithArtifactManager(ctx, d.Artifacts)
	}
	if d.Prompts != nil {
		ctx = WithPromptLoader(ctx, d.Prompts)
	}
	return ctx
}

// DevServicesConfig configures NewDevServices
type DevServicesConfig struct {
	RepoPath   string       // Path to git repository (required)
	BaseDir    string       // Base directory for storage (default: ".devflow")
	Claude     ClaudeConfig // Claude CLI config
	PromptDir  string       // Directory for prompt templates (default: ".devflow/prompts")
}

// NewDevServices creates DevServices with common defaults
func NewDevServices(cfg DevServicesConfig) (*DevServices, error) {
	ds := &DevServices{}

	// Create GitContext
	git, err := NewGitContext(cfg.RepoPath)
	if err != nil {
		return nil, err
	}
	ds.Git = git

	// Create ClaudeCLI with defaults
	claudeCfg := cfg.Claude
	if claudeCfg.Model == "" {
		claudeCfg.Model = "claude-sonnet-4-20250514"
	}
	claude, err := NewClaudeCLI(claudeCfg)
	if err != nil {
		return nil, err
	}
	ds.Claude = claude

	// Create base directory for storage
	baseDir := cfg.BaseDir
	if baseDir == "" {
		baseDir = ".devflow"
	}

	// Create TranscriptManager
	transcripts, err := NewFileTranscriptStore(baseDir)
	if err != nil {
		return nil, err
	}
	ds.Transcripts = transcripts

	// Create ArtifactManager
	ds.Artifacts = NewArtifactManager(ArtifactConfig{
		BaseDir: baseDir,
	})

	// Create PromptLoader
	promptDir := cfg.PromptDir
	if promptDir == "" {
		promptDir = ".devflow/prompts"
	}
	ds.Prompts = NewPromptLoader(promptDir)

	return ds, nil
}
