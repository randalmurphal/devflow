package devflow

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestNewClaudeCLI_NotFound(t *testing.T) {
	_, err := NewClaudeCLI(ClaudeConfig{
		BinaryPath: "/nonexistent/binary",
	})
	if err != ErrClaudeNotFound {
		t.Errorf("err = %v, want ErrClaudeNotFound", err)
	}
}

func TestNewClaudeCLI_Defaults(t *testing.T) {
	// Skip if claude not installed
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude CLI not installed")
	}

	cli, err := NewClaudeCLI(ClaudeConfig{})
	if err != nil {
		t.Fatalf("NewClaudeCLI: %v", err)
	}

	if cli.BinaryPath() != "claude" {
		t.Errorf("BinaryPath = %q, want %q", cli.BinaryPath(), "claude")
	}
	if cli.DefaultTimeout() != 5*time.Minute {
		t.Errorf("DefaultTimeout = %v, want 5m", cli.DefaultTimeout())
	}
	if cli.DefaultMaxTurns() != 10 {
		t.Errorf("DefaultMaxTurns = %d, want 10", cli.DefaultMaxTurns())
	}
}

func TestNewClaudeCLI_CustomConfig(t *testing.T) {
	// Skip if claude not installed
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude CLI not installed")
	}

	cli, err := NewClaudeCLI(ClaudeConfig{
		Model:    "claude-3-haiku",
		Timeout:  10 * time.Minute,
		MaxTurns: 20,
	})
	if err != nil {
		t.Fatalf("NewClaudeCLI: %v", err)
	}

	if cli.DefaultModel() != "claude-3-haiku" {
		t.Errorf("DefaultModel = %q, want %q", cli.DefaultModel(), "claude-3-haiku")
	}
	if cli.DefaultTimeout() != 10*time.Minute {
		t.Errorf("DefaultTimeout = %v, want 10m", cli.DefaultTimeout())
	}
	if cli.DefaultMaxTurns() != 20 {
		t.Errorf("DefaultMaxTurns = %d, want 20", cli.DefaultMaxTurns())
	}
}

func TestClaudeCLI_BuildArgs(t *testing.T) {
	cli := &ClaudeCLI{
		binaryPath: "claude",
		timeout:    5 * time.Minute,
		maxTurns:   10,
	}

	tests := []struct {
		name   string
		cfg    *runConfig
		prompt string
		want   []string
	}{
		{
			name:   "basic prompt",
			cfg:    &runConfig{},
			prompt: "Hello",
			want:   []string{"--print", "--output-format", "json", "-p", "Hello"},
		},
		{
			name: "with model",
			cfg: &runConfig{
				model: "claude-3-opus",
			},
			prompt: "Hello",
			want:   []string{"--print", "--output-format", "json", "--model", "claude-3-opus", "-p", "Hello"},
		},
		{
			name: "with system prompt",
			cfg: &runConfig{
				systemPrompt: "You are a helpful assistant",
			},
			prompt: "Hello",
			want:   []string{"--print", "--output-format", "json", "--system-prompt", "You are a helpful assistant", "-p", "Hello"},
		},
		{
			name: "with max turns",
			cfg: &runConfig{
				maxTurns: 5,
			},
			prompt: "Hello",
			want:   []string{"--print", "--output-format", "json", "--max-turns", "5", "-p", "Hello"},
		},
		{
			name: "with session",
			cfg: &runConfig{
				sessionID: "abc123",
			},
			prompt: "Hello",
			want:   []string{"--print", "--output-format", "json", "--resume", "abc123", "-p", "Hello"},
		},
		{
			name: "with allowed tools",
			cfg: &runConfig{
				allowedTools: []string{"Read", "Write"},
			},
			prompt: "Hello",
			want:   []string{"--print", "--output-format", "json", "--allowedTools", "Read", "--allowedTools", "Write", "-p", "Hello"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cli.buildArgs(tt.cfg, tt.prompt)
			if len(got) != len(tt.want) {
				t.Errorf("buildArgs() returned %d args, want %d\ngot:  %v\nwant: %v",
					len(got), len(tt.want), got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("buildArgs()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestParseClaudeOutput(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    *RunResult
		wantErr bool
	}{
		{
			name:  "valid json",
			input: []byte(`{"result": "Hello!", "tokens_in": 100, "tokens_out": 50, "session_id": "abc"}`),
			want: &RunResult{
				Output:    "Hello!",
				TokensIn:  100,
				TokensOut: 50,
				SessionID: "abc",
			},
		},
		{
			name:  "alternative field names",
			input: []byte(`{"result": "Hello!", "input_tokens": 100, "output_tokens": 50}`),
			want: &RunResult{
				Output:    "Hello!",
				TokensIn:  100,
				TokensOut: 50,
			},
		},
		{
			name:  "with cost",
			input: []byte(`{"result": "Hello!", "tokens_in": 100, "tokens_out": 50, "cost": 0.05}`),
			want: &RunResult{
				Output:    "Hello!",
				TokensIn:  100,
				TokensOut: 50,
				Cost:      0.05,
			},
		},
		{
			name:  "json embedded in output",
			input: []byte(`Some text before {"result": "Hello!", "tokens_in": 100, "tokens_out": 50} and after`),
			want: &RunResult{
				Output:    "Hello!",
				TokensIn:  100,
				TokensOut: 50,
			},
		},
		{
			name:    "invalid json",
			input:   []byte(`not json at all`),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseClaudeOutput(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("parseClaudeOutput() expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseClaudeOutput() error = %v", err)
			}
			if got.Output != tt.want.Output {
				t.Errorf("Output = %q, want %q", got.Output, tt.want.Output)
			}
			if got.TokensIn != tt.want.TokensIn {
				t.Errorf("TokensIn = %d, want %d", got.TokensIn, tt.want.TokensIn)
			}
			if got.TokensOut != tt.want.TokensOut {
				t.Errorf("TokensOut = %d, want %d", got.TokensOut, tt.want.TokensOut)
			}
			if got.SessionID != tt.want.SessionID {
				t.Errorf("SessionID = %q, want %q", got.SessionID, tt.want.SessionID)
			}
		})
	}
}

func TestRunOptions(t *testing.T) {
	cfg := &runConfig{}

	// Test WithSystemPrompt
	WithSystemPrompt("You are helpful")(cfg)
	if cfg.systemPrompt != "You are helpful" {
		t.Errorf("systemPrompt = %q, want %q", cfg.systemPrompt, "You are helpful")
	}

	// Test WithContext
	WithContext("file1.go", "file2.go")(cfg)
	if len(cfg.contextFiles) != 2 {
		t.Errorf("contextFiles = %v, want 2 files", cfg.contextFiles)
	}

	// Test WithWorkDir
	WithWorkDir("/tmp/test")(cfg)
	if cfg.workDir != "/tmp/test" {
		t.Errorf("workDir = %q, want %q", cfg.workDir, "/tmp/test")
	}

	// Test WithMaxTurns
	WithMaxTurns(5)(cfg)
	if cfg.maxTurns != 5 {
		t.Errorf("maxTurns = %d, want 5", cfg.maxTurns)
	}

	// Test WithClaudeTimeout
	WithClaudeTimeout(10 * time.Second)(cfg)
	if cfg.timeout != 10*time.Second {
		t.Errorf("timeout = %v, want 10s", cfg.timeout)
	}

	// Test WithModel
	WithModel("claude-3-opus")(cfg)
	if cfg.model != "claude-3-opus" {
		t.Errorf("model = %q, want %q", cfg.model, "claude-3-opus")
	}

	// Test WithSession
	WithSession("session123")(cfg)
	if cfg.sessionID != "session123" {
		t.Errorf("sessionID = %q, want %q", cfg.sessionID, "session123")
	}

	// Test WithAllowedTools
	WithAllowedTools("Read", "Write")(cfg)
	if len(cfg.allowedTools) != 2 {
		t.Errorf("allowedTools = %v, want 2 tools", cfg.allowedTools)
	}

	// Test WithDisallowedTools
	WithDisallowedTools("Bash")(cfg)
	if len(cfg.disallowedTools) != 1 {
		t.Errorf("disallowedTools = %v, want 1 tool", cfg.disallowedTools)
	}
}

// Integration test - only runs if claude is installed
func TestClaudeCLI_Run_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude CLI not installed")
	}

	cli, err := NewClaudeCLI(ClaudeConfig{
		Timeout:  30 * time.Second,
		MaxTurns: 1,
	})
	if err != nil {
		t.Fatalf("NewClaudeCLI: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := cli.Run(ctx, "Say exactly 'Hello World' with no other text",
		WithMaxTurns(1),
	)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if result.Output == "" {
		t.Error("expected non-empty output")
	}
}

// Test context timeout handling
func TestClaudeCLI_Run_Timeout(t *testing.T) {
	if _, err := exec.LookPath("sleep"); err != nil {
		t.Skip("sleep command not available")
	}

	// Create a mock that will timeout
	cli := &ClaudeCLI{
		binaryPath: "sleep",
		timeout:    100 * time.Millisecond,
		maxTurns:   10,
	}

	ctx := context.Background()
	_, err := cli.Run(ctx, "10", // sleep for 10 seconds
		WithClaudeTimeout(100*time.Millisecond),
	)

	// Should get timeout error
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestContextBuilder(t *testing.T) {
	// Create temp directory with test files
	dir := t.TempDir()

	// Create test files
	os.WriteFile(filepath.Join(dir, "test1.txt"), []byte("content1"), 0644)
	os.WriteFile(filepath.Join(dir, "test2.txt"), []byte("content2"), 0644)
	os.WriteFile(filepath.Join(dir, "sub", "test3.txt"), []byte("content3"), 0755)
	os.MkdirAll(filepath.Join(dir, "sub"), 0755)
	os.WriteFile(filepath.Join(dir, "sub", "test3.txt"), []byte("content3"), 0644)

	t.Run("add single file", func(t *testing.T) {
		b := NewContextBuilder(dir)
		if err := b.AddFile("test1.txt"); err != nil {
			t.Fatalf("AddFile: %v", err)
		}
		if b.FileCount() != 1 {
			t.Errorf("FileCount = %d, want 1", b.FileCount())
		}
	})

	t.Run("add glob", func(t *testing.T) {
		b := NewContextBuilder(dir)
		if err := b.AddGlob("*.txt"); err != nil {
			t.Fatalf("AddGlob: %v", err)
		}
		if b.FileCount() != 2 {
			t.Errorf("FileCount = %d, want 2", b.FileCount())
		}
	})

	t.Run("build context", func(t *testing.T) {
		b := NewContextBuilder(dir)
		b.AddFile("test1.txt")

		content, err := b.Build()
		if err != nil {
			t.Fatalf("Build: %v", err)
		}

		if content == "" {
			t.Error("expected non-empty content")
		}
		if !containsAll(content, "<file path=", "test1.txt", "content1", "</file>") {
			t.Errorf("content missing expected parts:\n%s", content)
		}
	})

	t.Run("file count limit", func(t *testing.T) {
		b := NewContextBuilder(dir)
		b.WithLimits(ContextLimits{MaxFileCount: 1, MaxFileSize: 100000, MaxTotalSize: 100000})
		b.AddFile("test1.txt")
		b.AddFile("test2.txt")

		_, err := b.Build()
		if err == nil {
			t.Error("expected error for exceeding file count")
		}
	})
}

func TestContextBuilder_Binary(t *testing.T) {
	dir := t.TempDir()

	// Create binary file
	binaryData := []byte{0x89, 'P', 'N', 'G', 0, 0, 0, 0}
	os.WriteFile(filepath.Join(dir, "image.png"), binaryData, 0644)

	b := NewContextBuilder(dir)
	b.AddFile("image.png")

	content, err := b.Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if !containsAll(content, "Binary file", "image/png") {
		t.Errorf("binary file not handled correctly:\n%s", content)
	}
}

func TestIsBinary(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{"text", []byte("Hello World"), false},
		{"binary with null", []byte("Hello\x00World"), true},
		{"png with null", []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0x00}, true},
		{"empty", []byte{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isBinary(tt.data)
			if got != tt.want {
				t.Errorf("isBinary() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetectMimeType(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{"png", []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}, "image/png"},
		{"jpeg", []byte{0xFF, 0xD8, 0xFF, 0xE0}, "image/jpeg"},
		{"gif", []byte("GIF89a"), "image/gif"},
		{"zip", []byte("PK\x03\x04"), "application/zip"},
		{"pdf", []byte("%PDF-1.4"), "application/pdf"},
		{"unknown", []byte{0x00, 0x01, 0x02, 0x03}, "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectMimeType(tt.data)
			if got != tt.want {
				t.Errorf("detectMimeType() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Helper function
func containsAll(s string, substrings ...string) bool {
	for _, sub := range substrings {
		if !contains(s, sub) {
			return false
		}
	}
	return true
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsAt(s, sub))
}

func containsAt(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
