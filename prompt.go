package devflow

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// embeddedPrompts holds default prompts embedded in the binary.
// To populate this, create a prompts/ directory with .txt files.
//
//go:embed prompts/*.txt
var embeddedPrompts embed.FS

// PromptLoader loads and renders prompt templates.
type PromptLoader struct {
	dirs    []string                      // Directories to search
	cache   map[string]*template.Template // Cached templates
	funcMap template.FuncMap              // Template functions
}

// NewPromptLoader creates a prompt loader for the given project directory.
// It searches for prompts in the following order:
// 1. .devflow/prompts/ in project
// 2. prompts/ in project
// 3. Embedded prompts in devflow binary
func NewPromptLoader(projectDir string) *PromptLoader {
	return &PromptLoader{
		dirs: []string{
			filepath.Join(projectDir, ".devflow", "prompts"),
			filepath.Join(projectDir, "prompts"),
		},
		cache:   make(map[string]*template.Template),
		funcMap: defaultPromptFuncMap(),
	}
}

// AddSearchDir adds a directory to search for prompts.
func (l *PromptLoader) AddSearchDir(dir string) {
	l.dirs = append([]string{dir}, l.dirs...)
}

// AddFunc adds a custom template function.
func (l *PromptLoader) AddFunc(name string, fn any) {
	l.funcMap[name] = fn
}

// Load loads a prompt by name without variable substitution.
func (l *PromptLoader) Load(name string) (string, error) {
	return l.LoadWithVars(name, nil)
}

// LoadWithVars loads and renders a prompt with variable substitution.
func (l *PromptLoader) LoadWithVars(name string, vars map[string]any) (string, error) {
	tmpl, err := l.getTemplate(name)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("render prompt %s: %w", name, err)
	}

	return buf.String(), nil
}

// Exists checks if a prompt exists.
func (l *PromptLoader) Exists(name string) bool {
	_, err := l.loadRaw(name)
	return err == nil
}

// List returns all available prompt names.
func (l *PromptLoader) List() ([]string, error) {
	prompts := make(map[string]bool)

	// Search directories
	for _, dir := range l.dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".txt") {
				name := strings.TrimSuffix(entry.Name(), ".txt")
				prompts[name] = true
			}
		}
	}

	// Search embedded
	entries, err := embeddedPrompts.ReadDir("prompts")
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".txt") {
				name := strings.TrimSuffix(entry.Name(), ".txt")
				prompts[name] = true
			}
		}
	}

	result := make([]string, 0, len(prompts))
	for name := range prompts {
		result = append(result, name)
	}
	return result, nil
}

// getTemplate loads and caches a template.
func (l *PromptLoader) getTemplate(name string) (*template.Template, error) {
	if tmpl, ok := l.cache[name]; ok {
		return tmpl, nil
	}

	content, err := l.loadRaw(name)
	if err != nil {
		return nil, err
	}

	tmpl, err := template.New(name).Funcs(l.funcMap).Parse(content)
	if err != nil {
		return nil, fmt.Errorf("parse prompt template %s: %w", name, err)
	}

	l.cache[name] = tmpl
	return tmpl, nil
}

// loadRaw loads raw prompt content without parsing.
func (l *PromptLoader) loadRaw(name string) (string, error) {
	filename := name + ".txt"

	// Search directories
	for _, dir := range l.dirs {
		path := filepath.Join(dir, filename)
		data, err := os.ReadFile(path)
		if err == nil {
			return string(data), nil
		}
	}

	// Fall back to embedded
	data, err := embeddedPrompts.ReadFile("prompts/" + filename)
	if err != nil {
		return "", fmt.Errorf("prompt not found: %s", name)
	}

	return string(data), nil
}

// ClearCache clears the template cache.
func (l *PromptLoader) ClearCache() {
	l.cache = make(map[string]*template.Template)
}

// defaultPromptFuncMap returns default template functions.
func defaultPromptFuncMap() template.FuncMap {
	return template.FuncMap{
		"join":     strings.Join,
		"split":    strings.Split,
		"trim":     strings.TrimSpace,
		"upper":    strings.ToUpper,
		"lower":    strings.ToLower,
		"title":    strings.Title,
		"contains": strings.Contains,
		"replace":  strings.ReplaceAll,
		"indent":   indentString,
		"default":  defaultValue,
		"quote":    quoteString,
	}
}

// indentString indents all lines of a string.
func indentString(indent int, s string) string {
	if s == "" {
		return s
	}
	prefix := strings.Repeat(" ", indent)
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = prefix + line
		}
	}
	return strings.Join(lines, "\n")
}

// defaultValue returns the default if value is empty.
func defaultValue(defaultVal, value any) any {
	if value == nil {
		return defaultVal
	}
	if s, ok := value.(string); ok && s == "" {
		return defaultVal
	}
	return value
}

// quoteString quotes a string for safe inclusion.
func quoteString(s string) string {
	return fmt.Sprintf("%q", s)
}

// PromptBuilder helps construct prompts programmatically.
type PromptBuilder struct {
	parts []string
}

// NewPromptBuilder creates a new prompt builder.
func NewPromptBuilder() *PromptBuilder {
	return &PromptBuilder{}
}

// Add adds text to the prompt.
func (b *PromptBuilder) Add(text string) *PromptBuilder {
	b.parts = append(b.parts, text)
	return b
}

// AddSection adds a markdown section with header.
func (b *PromptBuilder) AddSection(header, content string) *PromptBuilder {
	b.parts = append(b.parts, fmt.Sprintf("## %s\n\n%s", header, content))
	return b
}

// AddList adds a bulleted list.
func (b *PromptBuilder) AddList(header string, items []string) *PromptBuilder {
	var buf strings.Builder
	if header != "" {
		buf.WriteString("## ")
		buf.WriteString(header)
		buf.WriteString("\n\n")
	}
	for _, item := range items {
		buf.WriteString("- ")
		buf.WriteString(item)
		buf.WriteString("\n")
	}
	b.parts = append(b.parts, buf.String())
	return b
}

// AddFile adds file content with XML-style tags.
func (b *PromptBuilder) AddFile(path, content string) *PromptBuilder {
	b.parts = append(b.parts, fmt.Sprintf("<file path=%q>\n%s\n</file>", path, content))
	return b
}

// Build returns the constructed prompt.
func (b *PromptBuilder) Build() string {
	return strings.Join(b.parts, "\n\n")
}

// Clear resets the builder.
func (b *PromptBuilder) Clear() {
	b.parts = nil
}
