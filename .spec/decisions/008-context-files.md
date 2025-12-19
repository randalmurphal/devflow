# ADR-008: Context File Handling

## Status

Accepted

## Context

When invoking Claude, we often need to provide file contents as context. We need to decide:

1. How to specify which files to include
2. How to handle large files or many files
3. How to format file content for Claude
4. How to handle binary files

## Decision

### 1. Context File Specification

Files are specified as paths relative to working directory:

```go
result, err := claude.Run(ctx, prompt,
    devflow.WithContext("main.go", "api/handler.go", "models/*.go"),
)
```

Glob patterns are supported for multiple files.

### 2. File Content Format

Files are formatted with clear delimiters:

```text
<file path="main.go">
package main

func main() {
    // ...
}
</file>

<file path="api/handler.go">
package api

func HandleRequest(w http.ResponseWriter, r *http.Request) {
    // ...
}
</file>
```

### 3. Size Limits

| Limit | Value | Behavior |
|-------|-------|----------|
| Single file max | 100KB | Truncate with warning |
| Total context max | 500KB | Error before sending |
| File count max | 50 | Error before sending |

These are configurable:

```go
devflow.WithContextLimits(devflow.ContextLimits{
    MaxFileSize:    100 * 1024, // 100KB
    MaxTotalSize:   500 * 1024, // 500KB
    MaxFileCount:   50,
})
```

### 4. Binary File Handling

Binary files are detected and excluded:

```go
// Binary detection
func isBinary(data []byte) bool {
    // Check for null bytes in first 8KB
    sample := data
    if len(sample) > 8192 {
        sample = sample[:8192]
    }
    return bytes.Contains(sample, []byte{0})
}
```

Binary files produce a placeholder:

```text
<file path="image.png">
[Binary file: 45KB, type: image/png]
</file>
```

### 5. Smart File Selection

For large codebases, provide helpers to select relevant files:

```go
// Select files based on patterns
files := devflow.SelectFiles(workDir,
    devflow.Include("*.go"),
    devflow.Include("*.md"),
    devflow.Exclude("*_test.go"),
    devflow.Exclude("vendor/**"),
)

// Select files related to a path
files := devflow.RelatedFiles(workDir, "api/handler.go",
    devflow.WithImports(true),    // Include imported packages
    devflow.WithTests(true),       // Include test files
    devflow.WithDepth(2),          // 2 levels of dependencies
)
```

## Alternatives Considered

### Alternative 1: Base64 Encoding

Encode file contents as base64.

**Rejected because:**
- Wastes tokens
- Claude can read plain text
- Harder to debug

### Alternative 2: File URLs

Pass file:// URLs to Claude.

**Rejected because:**
- Claude CLI may not support
- Security implications
- Less control over content

### Alternative 3: Concatenated Plain Text

Just concatenate files with minimal formatting.

**Rejected because:**
- Hard for Claude to parse
- No clear file boundaries
- Can't reference specific files

### Alternative 4: Inline in Prompt

Require users to inline file content in prompts.

**Rejected because:**
- Duplicates work
- Error-prone
- Can't automate

## Consequences

### Positive

- **Clear structure**: XML-like tags are parseable
- **Flexible selection**: Globs and smart selection
- **Safe limits**: Won't blow up context window
- **Binary handling**: Gracefully handles non-text

### Negative

- **Token overhead**: Tags add tokens
- **Selection complexity**: Users must think about what to include
- **Truncation**: Large files may lose important content

### Mitigations

1. **Smart truncation**: Keep imports/function signatures
2. **Relevance scoring**: Prioritize most relevant content
3. **Summary mode**: For very large files, generate summary

## Code Example

```go
package devflow

import (
    "bytes"
    "fmt"
    "io/fs"
    "os"
    "path/filepath"
)

// ContextLimits configures file context limits
type ContextLimits struct {
    MaxFileSize  int64 // Max size per file
    MaxTotalSize int64 // Max total size
    MaxFileCount int   // Max number of files
}

// DefaultContextLimits returns sensible defaults
func DefaultContextLimits() ContextLimits {
    return ContextLimits{
        MaxFileSize:  100 * 1024,  // 100KB
        MaxTotalSize: 500 * 1024,  // 500KB
        MaxFileCount: 50,
    }
}

// ContextBuilder builds file context for Claude
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

// NewContextBuilder creates a context builder
func NewContextBuilder(workDir string) *ContextBuilder {
    return &ContextBuilder{
        workDir: workDir,
        limits:  DefaultContextLimits(),
    }
}

// WithLimits sets custom limits
func (b *ContextBuilder) WithLimits(limits ContextLimits) *ContextBuilder {
    b.limits = limits
    return b
}

// AddFile adds a file to context
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

// AddGlob adds files matching a glob pattern
func (b *ContextBuilder) AddGlob(pattern string) error {
    matches, err := filepath.Glob(filepath.Join(b.workDir, pattern))
    if err != nil {
        return fmt.Errorf("glob %s: %w", pattern, err)
    }

    for _, match := range matches {
        relPath, _ := filepath.Rel(b.workDir, match)
        info, _ := os.Stat(match)
        if !info.IsDir() {
            if err := b.AddFile(relPath); err != nil {
                return err
            }
        }
    }

    return nil
}

// Build generates the formatted context string
func (b *ContextBuilder) Build() (string, error) {
    // Check file count
    if len(b.files) > b.limits.MaxFileCount {
        return "", fmt.Errorf("too many files: %d > %d", len(b.files), b.limits.MaxFileCount)
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
            return "", fmt.Errorf("total context too large: %d > %d", totalSize, b.limits.MaxTotalSize)
        }

        // Format file
        fmt.Fprintf(&buf, "<file path=%q>\n", f.path)
        buf.Write(content)
        if !bytes.HasSuffix(content, []byte("\n")) {
            buf.WriteByte('\n')
        }
        buf.WriteString("</file>\n\n")
    }

    return buf.String(), nil
}

// isBinary detects if content is binary
func isBinary(data []byte) bool {
    sample := data
    if len(sample) > 8192 {
        sample = sample[:8192]
    }
    return bytes.Contains(sample, []byte{0})
}

// detectMimeType detects MIME type from content
func detectMimeType(data []byte) string {
    // Simple detection based on magic bytes
    if len(data) >= 4 {
        switch {
        case bytes.HasPrefix(data, []byte{0x89, 'P', 'N', 'G'}):
            return "image/png"
        case bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF}):
            return "image/jpeg"
        case bytes.HasPrefix(data, []byte{'G', 'I', 'F'}):
            return "image/gif"
        case bytes.HasPrefix(data, []byte{'P', 'K'}):
            return "application/zip"
        }
    }
    return "application/octet-stream"
}
```

### Usage

```go
// Simple usage with WithContext option
result, err := claude.Run(ctx, prompt,
    devflow.WithContext("main.go", "api/*.go"),
)

// Advanced usage with builder
builder := devflow.NewContextBuilder(workDir).
    WithLimits(devflow.ContextLimits{
        MaxFileSize:  200 * 1024,
        MaxTotalSize: 1024 * 1024,
        MaxFileCount: 100,
    })

builder.AddFile("main.go")
builder.AddGlob("api/*.go")
builder.AddGlob("models/*.go")

context, err := builder.Build()
if err != nil {
    return err
}

result, err := claude.Run(ctx, prompt + "\n\n## Context Files\n\n" + context)
```

### Smart File Selection

```go
// Find files related to a path
func RelatedFiles(workDir, targetFile string, opts ...RelatedOption) ([]string, error) {
    cfg := &relatedConfig{
        includeImports: true,
        includeTests:   false,
        maxDepth:       2,
    }
    for _, opt := range opts {
        opt(cfg)
    }

    files := make(map[string]bool)
    files[targetFile] = true

    // Parse imports from Go files
    if cfg.includeImports && strings.HasSuffix(targetFile, ".go") {
        imports, _ := parseGoImports(filepath.Join(workDir, targetFile))
        for _, imp := range imports {
            // Find local package files
            localFiles, _ := filepath.Glob(filepath.Join(workDir, imp, "*.go"))
            for _, f := range localFiles {
                rel, _ := filepath.Rel(workDir, f)
                files[rel] = true
            }
        }
    }

    // Include test files
    if cfg.includeTests {
        base := strings.TrimSuffix(targetFile, ".go")
        testFile := base + "_test.go"
        if _, err := os.Stat(filepath.Join(workDir, testFile)); err == nil {
            files[testFile] = true
        }
    }

    result := make([]string, 0, len(files))
    for f := range files {
        result = append(result, f)
    }
    return result, nil
}
```

## References

- [Claude Context Window](https://docs.anthropic.com/en/docs/models-overview)
- ADR-006: Claude CLI Wrapper
- ADR-007: Prompt Management
