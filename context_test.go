package devflow

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/randalmurphal/flowgraph/pkg/flowgraph/llm"
)

// Tests for context.go functionality not already covered in nodes_test.go or claude_test.go

func TestContextBuilder_WithLimits(t *testing.T) {
	cb := NewContextBuilder("/tmp/test")
	limits := ContextLimits{
		MaxFileSize:  50 * 1024,
		MaxTotalSize: 200 * 1024,
		MaxFileCount: 20,
	}

	result := cb.WithLimits(limits)

	if result != cb {
		t.Error("WithLimits should return same builder for chaining")
	}
	if cb.limits.MaxFileSize != 50*1024 {
		t.Errorf("MaxFileSize = %d, want %d", cb.limits.MaxFileSize, 50*1024)
	}
}

func TestContextBuilder_AddFile_Directory(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	cb := NewContextBuilder(tmpDir)
	err := cb.AddFile("subdir")
	if err == nil {
		t.Error("AddFile() should fail for directory")
	}
	if !strings.Contains(err.Error(), "is a directory") {
		t.Errorf("error should mention directory: %v", err)
	}
}

func TestContextBuilder_AddFile_NotExists(t *testing.T) {
	cb := NewContextBuilder("/tmp/nonexistent")
	err := cb.AddFile("missing.txt")
	if err == nil {
		t.Error("AddFile() should fail for nonexistent file")
	}
}

func TestContextBuilder_AddGlob(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	for _, name := range []string{"a.go", "b.go", "c.txt"} {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	cb := NewContextBuilder(tmpDir)
	err := cb.AddGlob("*.go")
	if err != nil {
		t.Fatalf("AddGlob() error = %v", err)
	}

	if cb.FileCount() != 2 {
		t.Errorf("FileCount() = %d, want 2 (.go files)", cb.FileCount())
	}
}

func TestContextBuilder_AddGlob_NoMatches(t *testing.T) {
	tmpDir := t.TempDir()

	cb := NewContextBuilder(tmpDir)
	err := cb.AddGlob("*.nonexistent")
	if err != nil {
		t.Fatalf("AddGlob() error = %v", err)
	}

	if cb.FileCount() != 0 {
		t.Errorf("FileCount() = %d, want 0 for no matches", cb.FileCount())
	}
}

func TestContextBuilder_AddContent(t *testing.T) {
	cb := NewContextBuilder("/tmp/test")
	cb.AddContent("virtual.txt", []byte("virtual content"))

	if cb.FileCount() != 1 {
		t.Errorf("FileCount() = %d, want 1", cb.FileCount())
	}

	ctx, err := cb.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if !strings.Contains(ctx, "virtual.txt") {
		t.Error("Build() output should contain virtual.txt")
	}
	if !strings.Contains(ctx, "virtual content") {
		t.Error("Build() output should contain virtual content")
	}
}

func TestContextBuilder_Build_TooManyFiles(t *testing.T) {
	cb := NewContextBuilder("/tmp/test")
	cb.limits.MaxFileCount = 2

	cb.AddContent("1.txt", []byte("one"))
	cb.AddContent("2.txt", []byte("two"))
	cb.AddContent("3.txt", []byte("three"))

	_, err := cb.Build()
	if err == nil {
		t.Error("Build() should fail when exceeding MaxFileCount")
	}
	if !strings.Contains(err.Error(), "3 files > max 2") {
		t.Errorf("error should mention file count: %v", err)
	}
}

func TestContextBuilder_Build_TotalSizeTooLarge(t *testing.T) {
	cb := NewContextBuilder("/tmp/test")
	cb.limits.MaxTotalSize = 10

	cb.AddContent("large.txt", []byte("this content is definitely more than 10 bytes"))

	_, err := cb.Build()
	if err == nil {
		t.Error("Build() should fail when exceeding MaxTotalSize")
	}
}

func TestContextBuilder_Build_LargeFile_Truncated(t *testing.T) {
	cb := NewContextBuilder("/tmp/test")
	cb.limits.MaxFileSize = 10
	cb.limits.MaxTotalSize = 1000

	cb.AddContent("large.txt", []byte("this content is definitely more than 10 bytes"))

	result, err := cb.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if !strings.Contains(result, "[... truncated ...]") {
		t.Error("Build() should truncate large files")
	}
}

func TestContextBuilder_Build_BinaryFile(t *testing.T) {
	cb := NewContextBuilder("/tmp/test")
	// Binary content with null byte
	cb.AddContent("binary.bin", []byte{0x89, 'P', 'N', 'G', 0x00, 0x01, 0x02})

	result, err := cb.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if !strings.Contains(result, "[Binary file:") {
		t.Error("Build() should mark binary files")
	}
	if !strings.Contains(result, "image/png") {
		t.Error("Build() should detect MIME type for PNG")
	}
}

func TestContextBuilder_TotalSize(t *testing.T) {
	cb := NewContextBuilder("/tmp/test")
	cb.AddContent("a.txt", []byte("hello"))
	cb.AddContent("b.txt", []byte("world"))

	size := cb.TotalSize()
	if size != 10 {
		t.Errorf("TotalSize() = %d, want 10", size)
	}
}

func TestContextBuilder_Clear(t *testing.T) {
	cb := NewContextBuilder("/tmp/test")
	cb.AddContent("a.txt", []byte("hello"))

	if cb.FileCount() != 1 {
		t.Fatal("precondition: FileCount should be 1")
	}

	cb.Clear()

	if cb.FileCount() != 0 {
		t.Errorf("FileCount() after Clear() = %d, want 0", cb.FileCount())
	}
}

func TestDefaultContextLimits(t *testing.T) {
	limits := DefaultContextLimits()

	if limits.MaxFileSize != 100*1024 {
		t.Errorf("MaxFileSize = %d, want %d", limits.MaxFileSize, 100*1024)
	}
	if limits.MaxTotalSize != 500*1024 {
		t.Errorf("MaxTotalSize = %d, want %d", limits.MaxTotalSize, 500*1024)
	}
	if limits.MaxFileCount != 50 {
		t.Errorf("MaxFileCount = %d, want %d", limits.MaxFileCount, 50)
	}
}

func TestFileSelector(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	files := []string{"a.go", "b.go", "a_test.go", "c.txt", "d.json"}
	for _, name := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("include only", func(t *testing.T) {
		fs := NewFileSelector(tmpDir)
		fs.Include("*.go")

		result, err := fs.Select()
		if err != nil {
			t.Fatalf("Select() error = %v", err)
		}

		if len(result) != 3 {
			t.Errorf("Select() returned %d files, want 3", len(result))
		}
	})

	t.Run("include and exclude", func(t *testing.T) {
		fs := NewFileSelector(tmpDir)
		fs.Include("*.go")
		fs.Exclude("*_test.go")

		result, err := fs.Select()
		if err != nil {
			t.Fatalf("Select() error = %v", err)
		}

		if len(result) != 2 {
			t.Errorf("Select() returned %d files, want 2", len(result))
		}

		for _, f := range result {
			if strings.Contains(f, "_test.go") {
				t.Error("Select() should exclude test files")
			}
		}
	})

	t.Run("multiple patterns", func(t *testing.T) {
		fs := NewFileSelector(tmpDir)
		fs.Include("*.go", "*.json")

		result, err := fs.Select()
		if err != nil {
			t.Fatalf("Select() error = %v", err)
		}

		if len(result) != 4 {
			t.Errorf("Select() returned %d files, want 4", len(result))
		}
	})

	t.Run("chaining", func(t *testing.T) {
		fs := NewFileSelector(tmpDir).
			Include("*.go").
			Exclude("*_test.go")

		result, err := fs.Select()
		if err != nil {
			t.Fatalf("Select() error = %v", err)
		}

		if len(result) != 2 {
			t.Errorf("Select() returned %d files, want 2", len(result))
		}
	})
}

// =============================================================================
// LLM Client Context Injection Tests
// =============================================================================

func TestWithLLMClient(t *testing.T) {
	ctx := context.Background()
	client := llm.NewMockClient("test response")

	ctx = WithLLMClient(ctx, client)
	got := LLMFromContext(ctx)

	if got != client {
		t.Error("LLMFromContext should return the same instance")
	}
}

func TestLLMFromContext_Missing(t *testing.T) {
	ctx := context.Background()
	got := LLMFromContext(ctx)

	if got != nil {
		t.Error("LLMFromContext should return nil when not set")
	}
}

func TestMustLLMFromContext_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustLLMFromContext should panic when not set")
		}
	}()

	ctx := context.Background()
	MustLLMFromContext(ctx)
}

func TestMustLLMFromContext_Success(t *testing.T) {
	ctx := context.Background()
	client := llm.NewMockClient("test")
	ctx = WithLLMClient(ctx, client)

	// Should not panic
	got := MustLLMFromContext(ctx)
	if got != client {
		t.Error("MustLLMFromContext should return the client")
	}
}

// =============================================================================
// Transcript Manager Context Injection Tests
// =============================================================================

func TestWithTranscriptManager(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	mgr, err := NewFileTranscriptStore(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	ctx = WithTranscriptManager(ctx, mgr)
	got := TranscriptManagerFromContext(ctx)

	if got != mgr {
		t.Error("TranscriptManagerFromContext should return the same instance")
	}
}

func TestTranscriptManagerFromContext_Missing(t *testing.T) {
	ctx := context.Background()
	got := TranscriptManagerFromContext(ctx)

	if got != nil {
		t.Error("TranscriptManagerFromContext should return nil when not set")
	}
}

func TestMustTranscriptManagerFromContext_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustTranscriptManagerFromContext should panic when not set")
		}
	}()

	ctx := context.Background()
	MustTranscriptManagerFromContext(ctx)
}

// =============================================================================
// Artifact Manager Context Injection Tests
// =============================================================================

func TestMustArtifactManagerFromContext_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustArtifactManagerFromContext should panic when not set")
		}
	}()

	ctx := context.Background()
	MustArtifactManagerFromContext(ctx)
}

func TestMustArtifactManagerFromContext_Success(t *testing.T) {
	ctx := context.Background()
	mgr := &ArtifactManager{}
	ctx = WithArtifactManager(ctx, mgr)

	// Should not panic
	got := MustArtifactManagerFromContext(ctx)
	if got != mgr {
		t.Error("MustArtifactManagerFromContext should return the manager")
	}
}

// =============================================================================
// Prompt Loader Context Injection Tests
// =============================================================================

func TestMustPromptLoaderFromContext_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustPromptLoaderFromContext should panic when not set")
		}
	}()

	ctx := context.Background()
	MustPromptLoaderFromContext(ctx)
}

func TestMustPromptLoaderFromContext_Success(t *testing.T) {
	ctx := context.Background()
	loader := &PromptLoader{}
	ctx = WithPromptLoader(ctx, loader)

	// Should not panic
	got := MustPromptLoaderFromContext(ctx)
	if got != loader {
		t.Error("MustPromptLoaderFromContext should return the loader")
	}
}

// =============================================================================
// Notifier Context Injection Tests
// =============================================================================

func TestWithNotifier(t *testing.T) {
	ctx := context.Background()
	notifier := &NopNotifier{}

	ctx = WithNotifier(ctx, notifier)
	got := NotifierFromContext(ctx)

	if got != notifier {
		t.Error("NotifierFromContext should return the same instance")
	}
}

func TestNotifierFromContext_Missing(t *testing.T) {
	ctx := context.Background()
	got := NotifierFromContext(ctx)

	if got != nil {
		t.Error("NotifierFromContext should return nil when not set")
	}
}

func TestMustNotifierFromContext_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustNotifierFromContext should panic when not set")
		}
	}()

	ctx := context.Background()
	MustNotifierFromContext(ctx)
}

func TestMustNotifierFromContext_Success(t *testing.T) {
	ctx := context.Background()
	notifier := &NopNotifier{}
	ctx = WithNotifier(ctx, notifier)

	// Should not panic
	got := MustNotifierFromContext(ctx)
	if got != notifier {
		t.Error("MustNotifierFromContext should return the notifier")
	}
}

// =============================================================================
// DevServices Tests
// =============================================================================

func TestDevServices_InjectAll_WithNotifier(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	transcripts, err := NewFileTranscriptStore(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	services := &DevServices{
		Git:         &GitContext{repoPath: "/tmp/repo"},
		LLM:         llm.NewMockClient("test"),
		Transcripts: transcripts,
		Artifacts:   &ArtifactManager{},
		Prompts:     &PromptLoader{},
		Notifier:    &NopNotifier{},
	}

	ctx = services.InjectAll(ctx)

	// Verify all services are injected
	if GitFromContext(ctx) == nil {
		t.Error("Git should be injected")
	}
	if LLMFromContext(ctx) == nil {
		t.Error("LLM should be injected")
	}
	if TranscriptManagerFromContext(ctx) == nil {
		t.Error("TranscriptManager should be injected")
	}
	if ArtifactManagerFromContext(ctx) == nil {
		t.Error("ArtifactManager should be injected")
	}
	if PromptLoaderFromContext(ctx) == nil {
		t.Error("PromptLoader should be injected")
	}
	if NotifierFromContext(ctx) == nil {
		t.Error("Notifier should be injected")
	}
}

func TestDevServices_InjectAll_PartialServices(t *testing.T) {
	ctx := context.Background()

	// Only inject some services
	services := &DevServices{
		Git: &GitContext{repoPath: "/tmp/repo"},
		LLM: llm.NewMockClient("test"),
	}

	ctx = services.InjectAll(ctx)

	// Verify only provided services are injected
	if GitFromContext(ctx) == nil {
		t.Error("Git should be injected")
	}
	if LLMFromContext(ctx) == nil {
		t.Error("LLM should be injected")
	}
	if TranscriptManagerFromContext(ctx) != nil {
		t.Error("TranscriptManager should not be injected when nil")
	}
	if ArtifactManagerFromContext(ctx) != nil {
		t.Error("ArtifactManager should not be injected when nil")
	}
	if NotifierFromContext(ctx) != nil {
		t.Error("Notifier should not be injected when nil")
	}
}

func TestDevServices_Empty(t *testing.T) {
	ctx := context.Background()
	services := &DevServices{}

	ctx = services.InjectAll(ctx)

	// Nothing should be injected
	if GitFromContext(ctx) != nil {
		t.Error("Git should not be injected when nil")
	}
	if LLMFromContext(ctx) != nil {
		t.Error("LLM should not be injected when nil")
	}
}

func TestNewDevServices(t *testing.T) {
	// Create a real git repo for testing
	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}

	services, err := NewDevServices(DevServicesConfig{
		RepoPath: tmpDir,
		BaseDir:  filepath.Join(tmpDir, ".devflow"),
	})
	if err != nil {
		t.Fatalf("NewDevServices: %v", err)
	}

	if services.Git == nil {
		t.Error("NewDevServices should set Git")
	}
	if services.LLM == nil {
		t.Error("NewDevServices should set LLM")
	}
	if services.Transcripts == nil {
		t.Error("NewDevServices should set Transcripts")
	}
	if services.Artifacts == nil {
		t.Error("NewDevServices should set Artifacts")
	}
	if services.Prompts == nil {
		t.Error("NewDevServices should set Prompts")
	}
}

func TestNewDevServices_NotGitRepo(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := NewDevServices(DevServicesConfig{
		RepoPath: tmpDir,
	})
	if err == nil {
		t.Error("NewDevServices should fail for non-git directory")
	}
}

// =============================================================================
// MIME Type Detection Tests
// =============================================================================

func TestDetectMimeType(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{
			name:     "PNG image",
			data:     []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A},
			expected: "image/png",
		},
		{
			name:     "JPEG image",
			data:     []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F'},
			expected: "image/jpeg",
		},
		{
			name:     "GIF image",
			data:     []byte("GIF89a..."),
			expected: "image/gif",
		},
		{
			name:     "GIF87a image",
			data:     []byte("GIF87a..."),
			expected: "image/gif",
		},
		{
			name:     "ZIP archive",
			data:     []byte("PK\x03\x04\x14\x00\x00\x00"),
			expected: "application/zip",
		},
		{
			name:     "PDF document",
			data:     []byte("%PDF-1.4\n"),
			expected: "application/pdf",
		},
		{
			name:     "ELF binary",
			data:     []byte{0x7F, 'E', 'L', 'F', 0x02, 0x01, 0x01, 0x00},
			expected: "application/x-elf",
		},
		{
			name:     "Windows executable",
			data:     []byte("MZ\x90\x00\x03\x00\x00\x00"),
			expected: "application/x-msdownload",
		},
		{
			name:     "Unknown binary",
			data:     []byte{0x00, 0x01, 0x02, 0x03, 0x04},
			expected: "application/octet-stream",
		},
		{
			name:     "Short data",
			data:     []byte{0x89, 'P'},
			expected: "application/octet-stream",
		},
		{
			name:     "Empty data",
			data:     []byte{},
			expected: "application/octet-stream",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectMimeType(tt.data)
			if got != tt.expected {
				t.Errorf("detectMimeType() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestIsBinary(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{
			name:     "text content",
			data:     []byte("Hello, World!\nThis is plain text."),
			expected: false,
		},
		{
			name:     "binary with null byte",
			data:     []byte("Hello\x00World"),
			expected: true,
		},
		{
			name:     "PNG header",
			data:     []byte{0x89, 'P', 'N', 'G', 0x00, 0x01, 0x02},
			expected: true,
		},
		{
			name:     "empty data",
			data:     []byte{},
			expected: false,
		},
		{
			name:     "long text without null",
			data:     []byte(strings.Repeat("Hello World ", 1000)),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isBinary(tt.data)
			if got != tt.expected {
				t.Errorf("isBinary() = %v, want %v", got, tt.expected)
			}
		})
	}
}
