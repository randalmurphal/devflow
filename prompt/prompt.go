package prompt

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// embeddedPrompts holds default prompts embedded in the binary.
// To populate this, create a prompts/ directory with .txt files.
//
//go:embed prompts/*.txt
var embeddedPrompts embed.FS

// Loader loads and renders prompt templates.
type Loader struct {
	dirs    []string                      // Directories to search
	cache   map[string]*template.Template // Cached templates
	funcMap template.FuncMap              // Template functions
}

// NewLoader creates a prompt loader for the given project directory.
// It searches for prompts in the following order:
// 1. .devflow/prompts/ in project
// 2. prompts/ in project
// 3. Embedded prompts in devflow binary
func NewLoader(projectDir string) *Loader {
	return &Loader{
		dirs: []string{
			filepath.Join(projectDir, ".devflow", "prompts"),
			filepath.Join(projectDir, "prompts"),
		},
		cache:   make(map[string]*template.Template),
		funcMap: defaultPromptFuncMap(),
	}
}

// AddSearchDir adds a directory to search for prompts.
func (l *Loader) AddSearchDir(dir string) {
	l.dirs = append([]string{dir}, l.dirs...)
}

// AddFunc adds a custom template function.
func (l *Loader) AddFunc(name string, fn any) {
	l.funcMap[name] = fn
}

// Load loads a prompt by name without variable substitution.
func (l *Loader) Load(name string) (string, error) {
	return l.LoadWithVars(name, nil)
}

// LoadWithVars loads and renders a prompt with variable substitution.
func (l *Loader) LoadWithVars(name string, vars map[string]any) (string, error) {
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
func (l *Loader) Exists(name string) bool {
	_, err := l.loadRaw(name)
	return err == nil
}

// List returns all available prompt names.
func (l *Loader) List() ([]string, error) {
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
func (l *Loader) getTemplate(name string) (*template.Template, error) {
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
func (l *Loader) loadRaw(name string) (string, error) {
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
func (l *Loader) ClearCache() {
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
		"title":    cases.Title(language.English).String,
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

// Builder helps construct prompts programmatically.
type Builder struct {
	parts []string
}

// NewBuilder creates a new prompt builder.
func NewBuilder() *Builder {
	return &Builder{}
}

// Add adds text to the prompt.
func (b *Builder) Add(text string) *Builder {
	b.parts = append(b.parts, text)
	return b
}

// AddSection adds a markdown section with header.
func (b *Builder) AddSection(header, content string) *Builder {
	b.parts = append(b.parts, fmt.Sprintf("## %s\n\n%s", header, content))
	return b
}

// AddList adds a bulleted list.
func (b *Builder) AddList(header string, items []string) *Builder {
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
func (b *Builder) AddFile(path, content string) *Builder {
	b.parts = append(b.parts, fmt.Sprintf("<file path=%q>\n%s\n</file>", path, content))
	return b
}

// Build returns the constructed prompt.
func (b *Builder) Build() string {
	return strings.Join(b.parts, "\n\n")
}

// Clear resets the builder.
func (b *Builder) Clear() {
	b.parts = nil
}
