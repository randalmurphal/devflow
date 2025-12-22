package jira

import (
	"fmt"
	"regexp"
	"strings"
)

// WikiConverter handles conversion between Markdown and Jira Wiki Markup.
// This is used for Jira Server/Data Center API v2.
type WikiConverter struct{}

// NewWikiConverter creates a new Wiki Markup converter.
func NewWikiConverter() *WikiConverter {
	return &WikiConverter{}
}

// ToWiki converts Markdown to Jira Wiki Markup.
func (c *WikiConverter) ToWiki(markdown string) string {
	result := markdown

	// Code blocks FIRST (before inline code): ```lang\ncode\n``` -> {code:lang}\ncode\n{code}
	codeBlockRe := regexp.MustCompile("(?s)```(\\w*)\\n(.*?)\\n```")
	result = codeBlockRe.ReplaceAllStringFunc(result, func(s string) string {
		matches := codeBlockRe.FindStringSubmatch(s)
		if len(matches) == 3 {
			lang := matches[1]
			code := matches[2]
			if lang != "" {
				return "{code:" + lang + "}\n" + code + "\n{code}"
			}
			return "{code}\n" + code + "\n{code}"
		}
		return s
	})

	// Headers: # Header -> h1. Header
	for i := 6; i >= 1; i-- {
		prefix := strings.Repeat("#", i)
		re := regexp.MustCompile(`(?m)^` + prefix + ` (.+)$`)
		result = re.ReplaceAllString(result, fmt.Sprintf("h%d. $1", i))
	}

	// Bold: **text** -> *text*
	result = regexp.MustCompile(`\*\*([^*]+)\*\*`).ReplaceAllString(result, `*$1*`)

	// Strikethrough: ~~text~~ -> -text-
	result = regexp.MustCompile(`~~([^~]+)~~`).ReplaceAllString(result, `-$1-`)

	// Inline code: `code` -> {{code}}
	result = regexp.MustCompile("`([^`]+)`").ReplaceAllString(result, `{{$1}}`)

	// Blockquote: > text -> {quote}text{quote}
	lines := strings.Split(result, "\n")
	var quoteLines []string
	var outputLines []string
	inQuote := false

	for _, line := range lines {
		if strings.HasPrefix(line, "> ") {
			if !inQuote {
				outputLines = append(outputLines, "{quote}")
				inQuote = true
			}
			quoteLines = append(quoteLines, strings.TrimPrefix(line, "> "))
		} else {
			if inQuote {
				outputLines = append(outputLines, strings.Join(quoteLines, "\n"), "{quote}")
				quoteLines = nil
				inQuote = false
			}
			outputLines = append(outputLines, line)
		}
	}
	if inQuote {
		outputLines = append(outputLines, strings.Join(quoteLines, "\n"), "{quote}")
	}
	result = strings.Join(outputLines, "\n")

	// Links: [text](url) -> [text|url]
	result = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`).ReplaceAllString(result, `[$1|$2]`)

	// Bullet lists: - item -> * item
	result = regexp.MustCompile(`(?m)^- (.+)$`).ReplaceAllString(result, `* $1`)

	// Numbered lists: 1. item -> # item
	result = regexp.MustCompile(`(?m)^\d+\. (.+)$`).ReplaceAllString(result, `# $1`)

	// Horizontal rule: --- -> ----
	result = regexp.MustCompile(`(?m)^---+$`).ReplaceAllString(result, `----`)

	return result
}

// FromWiki converts Jira Wiki Markup to Markdown.
func (c *WikiConverter) FromWiki(wiki string) string {
	result := wiki

	// Code blocks FIRST: {code:lang}\ncode\n{code} -> ```lang\ncode\n```
	codeBlockRe := regexp.MustCompile(`(?s)\{code(?::(\w+))?\}(.*?)\{code\}`)
	result = codeBlockRe.ReplaceAllStringFunc(result, func(s string) string {
		matches := codeBlockRe.FindStringSubmatch(s)
		if len(matches) == 3 {
			lang := matches[1]
			code := strings.Trim(matches[2], "\n")
			return "```" + lang + "\n" + code + "\n```"
		}
		return s
	})

	// Blockquote BEFORE headers: {quote}text{quote} -> > text
	quoteRe := regexp.MustCompile(`(?s)\{quote\}(.*?)\{quote\}`)
	result = quoteRe.ReplaceAllStringFunc(result, func(s string) string {
		matches := quoteRe.FindStringSubmatch(s)
		if len(matches) == 2 {
			lines := strings.Split(strings.Trim(matches[1], "\n"), "\n")
			for i, line := range lines {
				lines[i] = "> " + line
			}
			return strings.Join(lines, "\n")
		}
		return s
	})

	// Numbered lists BEFORE headers: # item -> 1. item (so # doesn't get converted to heading)
	lines := strings.Split(result, "\n")
	counter := 0
	for i, line := range lines {
		if strings.HasPrefix(line, "# ") && !strings.HasPrefix(line, "# h") {
			counter++
			lines[i] = fmt.Sprintf("%d. %s", counter, strings.TrimPrefix(line, "# "))
		} else if strings.TrimSpace(line) == "" {
			counter = 0
		}
	}
	result = strings.Join(lines, "\n")

	// Headers: h1. Header -> # Header
	for i := 1; i <= 6; i++ {
		re := regexp.MustCompile(fmt.Sprintf(`(?m)^h%d\. (.+)$`, i))
		result = re.ReplaceAllString(result, strings.Repeat("#", i)+` $1`)
	}

	// Inline code BEFORE bold: {{code}} -> `code`
	result = regexp.MustCompile(`\{\{([^}]+)\}\}`).ReplaceAllString(result, "`$1`")

	// Bold: *text* -> **text** (but not bullet lists)
	// Process line by line to avoid matching bullet lists
	lines = strings.Split(result, "\n")
	boldRe := regexp.MustCompile(`\*([^*\n]+)\*`)
	for i, line := range lines {
		// Skip lines that start with bullet list marker
		if strings.HasPrefix(strings.TrimSpace(line), "* ") || strings.HasPrefix(strings.TrimSpace(line), "- ") {
			continue
		}
		lines[i] = boldRe.ReplaceAllString(line, `**$1**`)
	}
	result = strings.Join(lines, "\n")

	// Italic: _text_ -> *text*
	result = regexp.MustCompile(`_([^_]+)_`).ReplaceAllString(result, `*$1*`)

	// Strikethrough: -text- -> ~~text~~
	result = regexp.MustCompile(`-([^-\s][^-]*[^-\s])-`).ReplaceAllString(result, `~~$1~~`)

	// Links: [text|url] -> [text](url)
	result = regexp.MustCompile(`\[([^|\]]+)\|([^\]]+)\]`).ReplaceAllString(result, `[$1]($2)`)

	// Plain links: [url] -> [url](url)
	result = regexp.MustCompile(`\[([^\]|]+)\]`).ReplaceAllStringFunc(result, func(s string) string {
		url := strings.Trim(s, "[]")
		if strings.HasPrefix(url, "http") {
			return "[" + url + "](" + url + ")"
		}
		return s
	})

	// Bullet lists: * item -> - item (at start of line)
	result = regexp.MustCompile(`(?m)^\* (.+)$`).ReplaceAllString(result, `- $1`)

	// Horizontal rule: ---- -> ---
	result = regexp.MustCompile(`(?m)^----+$`).ReplaceAllString(result, `---`)

	return result
}

// RichTextConverter is an interface for converting rich text formats.
type RichTextConverter interface {
	// ToJira converts Markdown to the Jira format (ADF or Wiki Markup)
	ToJira(markdown string) (any, error)
	// FromJira converts from Jira format to Markdown
	FromJira(content any) (string, error)
}

// CloudConverter implements RichTextConverter for Jira Cloud (ADF).
type CloudConverter struct {
	adf *ADFConverter
}

// NewCloudConverter creates a converter for Jira Cloud.
func NewCloudConverter() *CloudConverter {
	return &CloudConverter{adf: NewADFConverter()}
}

// ToJira converts Markdown to ADF.
func (c *CloudConverter) ToJira(markdown string) (any, error) {
	return c.adf.ToADF(markdown)
}

// FromJira converts ADF to Markdown.
func (c *CloudConverter) FromJira(content any) (string, error) {
	return c.adf.FromADFAny(content)
}

// ServerConverter implements RichTextConverter for Jira Server (Wiki Markup).
type ServerConverter struct {
	wiki *WikiConverter
}

// NewServerConverter creates a converter for Jira Server/DC.
func NewServerConverter() *ServerConverter {
	return &ServerConverter{wiki: NewWikiConverter()}
}

// ToJira converts Markdown to Wiki Markup.
func (c *ServerConverter) ToJira(markdown string) (any, error) {
	return c.wiki.ToWiki(markdown), nil
}

// FromJira converts Wiki Markup to Markdown.
func (c *ServerConverter) FromJira(content any) (string, error) {
	if s, ok := content.(string); ok {
		return c.wiki.FromWiki(s), nil
	}
	return "", nil
}

// NewRichTextConverter creates the appropriate converter based on deployment type.
func NewRichTextConverter(deployment DeploymentType) RichTextConverter {
	if deployment == DeploymentCloud {
		return NewCloudConverter()
	}
	return NewServerConverter()
}

// WikiToMarkdown is a convenience function that converts Wiki Markup to Markdown.
func WikiToMarkdown(wiki string) string {
	return NewWikiConverter().FromWiki(wiki)
}

// MarkdownToWiki is a convenience function that converts Markdown to Wiki Markup.
func MarkdownToWiki(markdown string) string {
	return NewWikiConverter().ToWiki(markdown)
}
