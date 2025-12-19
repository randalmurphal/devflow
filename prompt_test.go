package devflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewPromptLoader(t *testing.T) {
	loader := NewPromptLoader("/tmp/project")

	if len(loader.dirs) != 2 {
		t.Errorf("expected 2 search dirs, got %d", len(loader.dirs))
	}
	if loader.cache == nil {
		t.Error("cache should be initialized")
	}
	if loader.funcMap == nil {
		t.Error("funcMap should be initialized")
	}
}

func TestPromptLoader_LoadEmbedded(t *testing.T) {
	loader := NewPromptLoader("/nonexistent")

	// Should load from embedded prompts
	content, err := loader.Load("generate-spec")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if content == "" {
		t.Error("expected non-empty content")
	}
	if !strings.Contains(content, "technical specification") {
		t.Error("content should contain 'technical specification'")
	}
}

func TestPromptLoader_LoadFromDir(t *testing.T) {
	// Create temp directory with custom prompt
	dir := t.TempDir()
	promptsDir := filepath.Join(dir, ".devflow", "prompts")
	os.MkdirAll(promptsDir, 0755)
	os.WriteFile(filepath.Join(promptsDir, "custom.txt"), []byte("Custom prompt content"), 0644)

	loader := NewPromptLoader(dir)

	content, err := loader.Load("custom")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if content != "Custom prompt content" {
		t.Errorf("content = %q, want 'Custom prompt content'", content)
	}
}

func TestPromptLoader_LoadWithVars(t *testing.T) {
	// Create temp directory with template prompt
	dir := t.TempDir()
	promptsDir := filepath.Join(dir, "prompts")
	os.MkdirAll(promptsDir, 0755)
	os.WriteFile(filepath.Join(promptsDir, "template.txt"),
		[]byte("Hello {{.Name}}! You are {{.Role}}."), 0644)

	loader := NewPromptLoader(dir)

	content, err := loader.LoadWithVars("template", map[string]any{
		"Name": "Alice",
		"Role": "a developer",
	})
	if err != nil {
		t.Fatalf("LoadWithVars: %v", err)
	}

	want := "Hello Alice! You are a developer."
	if content != want {
		t.Errorf("content = %q, want %q", content, want)
	}
}

func TestPromptLoader_Exists(t *testing.T) {
	loader := NewPromptLoader("/nonexistent")

	if !loader.Exists("generate-spec") {
		t.Error("generate-spec should exist (embedded)")
	}
	if loader.Exists("nonexistent-prompt") {
		t.Error("nonexistent-prompt should not exist")
	}
}

func TestPromptLoader_List(t *testing.T) {
	loader := NewPromptLoader("/nonexistent")

	prompts, err := loader.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(prompts) == 0 {
		t.Error("expected at least one prompt")
	}

	// Check that embedded prompts are listed
	found := false
	for _, p := range prompts {
		if p == "generate-spec" {
			found = true
			break
		}
	}
	if !found {
		t.Error("generate-spec should be in list")
	}
}

func TestPromptLoader_ClearCache(t *testing.T) {
	loader := NewPromptLoader("/nonexistent")

	// Load something to populate cache
	loader.Load("generate-spec")
	if len(loader.cache) == 0 {
		t.Error("cache should have entry")
	}

	loader.ClearCache()
	if len(loader.cache) != 0 {
		t.Error("cache should be empty after clear")
	}
}

func TestPromptLoader_AddSearchDir(t *testing.T) {
	loader := NewPromptLoader("/project")
	initialDirs := len(loader.dirs)

	loader.AddSearchDir("/custom/prompts")

	if len(loader.dirs) != initialDirs+1 {
		t.Error("should have added search dir")
	}
	if loader.dirs[0] != "/custom/prompts" {
		t.Error("custom dir should be first in search order")
	}
}

func TestPromptLoader_NotFound(t *testing.T) {
	loader := NewPromptLoader("/nonexistent")

	_, err := loader.Load("definitely-not-a-real-prompt")
	if err == nil {
		t.Error("expected error for non-existent prompt")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found': %v", err)
	}
}

func TestPromptLoader_TemplateError(t *testing.T) {
	dir := t.TempDir()
	promptsDir := filepath.Join(dir, "prompts")
	os.MkdirAll(promptsDir, 0755)
	os.WriteFile(filepath.Join(promptsDir, "bad.txt"),
		[]byte("{{.Missing}"), 0644) // Invalid template syntax

	loader := NewPromptLoader(dir)

	_, err := loader.Load("bad")
	if err == nil {
		t.Error("expected error for invalid template")
	}
}

func TestPromptFunctions(t *testing.T) {
	dir := t.TempDir()
	promptsDir := filepath.Join(dir, "prompts")
	os.MkdirAll(promptsDir, 0755)

	tests := []struct {
		name     string
		template string
		vars     map[string]any
		want     string
	}{
		{
			name:     "join",
			template: `{{join .Items ", "}}`,
			vars:     map[string]any{"Items": []string{"a", "b", "c"}},
			want:     "a, b, c",
		},
		{
			name:     "upper",
			template: `{{upper .Text}}`,
			vars:     map[string]any{"Text": "hello"},
			want:     "HELLO",
		},
		{
			name:     "lower",
			template: `{{lower .Text}}`,
			vars:     map[string]any{"Text": "HELLO"},
			want:     "hello",
		},
		{
			name:     "trim",
			template: `{{trim .Text}}`,
			vars:     map[string]any{"Text": "  hello  "},
			want:     "hello",
		},
		{
			name:     "indent",
			template: `{{indent 4 .Text}}`,
			vars:     map[string]any{"Text": "line1\nline2"},
			want:     "    line1\n    line2",
		},
		{
			name:     "default",
			template: `{{default "fallback" .Value}}`,
			vars:     map[string]any{"Value": ""},
			want:     "fallback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			promptFile := filepath.Join(promptsDir, tt.name+".txt")
			os.WriteFile(promptFile, []byte(tt.template), 0644)

			loader := NewPromptLoader(dir)
			got, err := loader.LoadWithVars(tt.name, tt.vars)
			if err != nil {
				t.Fatalf("LoadWithVars: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPromptBuilder(t *testing.T) {
	t.Run("basic build", func(t *testing.T) {
		b := NewPromptBuilder()
		b.Add("Hello World")
		b.Add("More text")

		got := b.Build()
		if !strings.Contains(got, "Hello World") {
			t.Error("should contain 'Hello World'")
		}
		if !strings.Contains(got, "More text") {
			t.Error("should contain 'More text'")
		}
	})

	t.Run("add section", func(t *testing.T) {
		b := NewPromptBuilder()
		b.AddSection("Header", "Content here")

		got := b.Build()
		if !strings.Contains(got, "## Header") {
			t.Error("should contain section header")
		}
		if !strings.Contains(got, "Content here") {
			t.Error("should contain content")
		}
	})

	t.Run("add list", func(t *testing.T) {
		b := NewPromptBuilder()
		b.AddList("Items", []string{"one", "two", "three"})

		got := b.Build()
		if !strings.Contains(got, "## Items") {
			t.Error("should contain list header")
		}
		if !strings.Contains(got, "- one") {
			t.Error("should contain list items")
		}
	})

	t.Run("add file", func(t *testing.T) {
		b := NewPromptBuilder()
		b.AddFile("test.go", "package main")

		got := b.Build()
		if !strings.Contains(got, `<file path="test.go">`) {
			t.Error("should contain file tag with path")
		}
		if !strings.Contains(got, "package main") {
			t.Error("should contain file content")
		}
	})

	t.Run("clear", func(t *testing.T) {
		b := NewPromptBuilder()
		b.Add("text")
		b.Clear()

		if b.Build() != "" {
			t.Error("should be empty after clear")
		}
	})
}

func TestIndentString(t *testing.T) {
	tests := []struct {
		indent int
		input  string
		want   string
	}{
		{2, "hello", "  hello"},
		{4, "line1\nline2", "    line1\n    line2"},
		{2, "", ""},
		{2, "a\n\nb", "  a\n\n  b"}, // Empty lines stay empty
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := indentString(tt.indent, tt.input)
			if got != tt.want {
				t.Errorf("indentString(%d, %q) = %q, want %q",
					tt.indent, tt.input, got, tt.want)
			}
		})
	}
}

func TestDefaultValue(t *testing.T) {
	tests := []struct {
		name       string
		defaultVal any
		value      any
		want       any
	}{
		{"nil value", "default", nil, "default"},
		{"empty string", "default", "", "default"},
		{"has value", "default", "actual", "actual"},
		{"zero int", "default", 0, 0}, // Non-string values kept
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := defaultValue(tt.defaultVal, tt.value)
			if got != tt.want {
				t.Errorf("defaultValue(%v, %v) = %v, want %v",
					tt.defaultVal, tt.value, got, tt.want)
			}
		})
	}
}

func TestQuoteString(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", `"hello"`},
		{`hello "world"`, `"hello \"world\""`},
		{"line\nbreak", `"line\nbreak"`},
		{"", `""`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := quoteString(tt.input)
			if got != tt.want {
				t.Errorf("quoteString(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestPromptLoader_AddFunc(t *testing.T) {
	loader := NewPromptLoader("/nonexistent")

	// Add a custom function
	loader.AddFunc("uppercase", func(s string) string {
		return strings.ToUpper(s)
	})

	// Create a temp dir with a template that uses our custom function
	dir := t.TempDir()
	promptsDir := filepath.Join(dir, "prompts")
	os.MkdirAll(promptsDir, 0755)
	os.WriteFile(filepath.Join(promptsDir, "custom-func.txt"),
		[]byte("{{uppercase .Name}}"), 0644)

	// Need to add the search dir
	loader.AddSearchDir(promptsDir)

	content, err := loader.LoadWithVars("custom-func", map[string]any{
		"Name": "hello",
	})
	if err != nil {
		t.Fatalf("LoadWithVars: %v", err)
	}

	if content != "HELLO" {
		t.Errorf("content = %q, want 'HELLO'", content)
	}
}
