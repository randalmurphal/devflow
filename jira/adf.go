package jira

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ADFDocument represents an Atlassian Document Format document.
// This is used for rich text fields in Jira Cloud API v3.
type ADFDocument struct {
	Version int       `json:"version"` // Always 1
	Type    string    `json:"type"`    // Always "doc"
	Content []ADFNode `json:"content"`
}

// ADFNode represents a node in an ADF document.
type ADFNode struct {
	Type    string         `json:"type"`
	Content []ADFNode      `json:"content,omitempty"`
	Text    string         `json:"text,omitempty"`
	Marks   []ADFMark      `json:"marks,omitempty"`
	Attrs   map[string]any `json:"attrs,omitempty"`
}

// ADFMark represents formatting applied to text.
type ADFMark struct {
	Type  string         `json:"type"`
	Attrs map[string]any `json:"attrs,omitempty"`
}

// ADF node types
const (
	ADFNodeDoc         = "doc"
	ADFNodeParagraph   = "paragraph"
	ADFNodeText        = "text"
	ADFNodeHardBreak   = "hardBreak"
	ADFNodeHeading     = "heading"
	ADFNodeBulletList  = "bulletList"
	ADFNodeOrderedList = "orderedList"
	ADFNodeListItem    = "listItem"
	ADFNodeCodeBlock   = "codeBlock"
	ADFNodeBlockquote  = "blockquote"
	ADFNodeRule        = "rule"
	ADFNodeMention     = "mention"
	ADFNodeEmoji       = "emoji"
	ADFNodeInlineCard  = "inlineCard"
	ADFNodePanel       = "panel"
	ADFNodeTable       = "table"
	ADFNodeTableRow    = "tableRow"
	ADFNodeTableHeader = "tableHeader"
	ADFNodeTableCell   = "tableCell"
)

// ADF mark types
const (
	ADFMarkStrong    = "strong"
	ADFMarkEm        = "em"
	ADFMarkStrike    = "strike"
	ADFMarkCode      = "code"
	ADFMarkUnderline = "underline"
	ADFMarkLink      = "link"
	ADFMarkTextColor = "textColor"
	ADFMarkSubSup    = "subsup"
)

// NewADFDocument creates a new empty ADF document.
func NewADFDocument() *ADFDocument {
	return &ADFDocument{
		Version: 1,
		Type:    ADFNodeDoc,
		Content: []ADFNode{},
	}
}

// Validate validates the ADF document structure.
func (d *ADFDocument) Validate() error {
	if d.Version != 1 {
		return ErrADFVersionOnly
	}
	if d.Type != ADFNodeDoc {
		return ErrADFTypeInvalid
	}
	return nil
}

// AddParagraph adds a paragraph with text to the document.
func (d *ADFDocument) AddParagraph(text string) {
	node := ADFNode{
		Type: ADFNodeParagraph,
		Content: []ADFNode{
			{Type: ADFNodeText, Text: text},
		},
	}
	d.Content = append(d.Content, node)
}

// AddHeading adds a heading to the document.
func (d *ADFDocument) AddHeading(level int, text string) {
	if level < 1 {
		level = 1
	}
	if level > 6 {
		level = 6
	}
	node := ADFNode{
		Type: ADFNodeHeading,
		Attrs: map[string]any{
			"level": level,
		},
		Content: []ADFNode{
			{Type: ADFNodeText, Text: text},
		},
	}
	d.Content = append(d.Content, node)
}

// AddCodeBlock adds a code block to the document.
func (d *ADFDocument) AddCodeBlock(code, language string) {
	node := ADFNode{
		Type: ADFNodeCodeBlock,
		Attrs: map[string]any{
			"language": language,
		},
		Content: []ADFNode{
			{Type: ADFNodeText, Text: code},
		},
	}
	d.Content = append(d.Content, node)
}

// AddBulletList adds a bullet list to the document.
func (d *ADFDocument) AddBulletList(items []string) {
	listItems := make([]ADFNode, len(items))
	for i, item := range items {
		listItems[i] = ADFNode{
			Type: ADFNodeListItem,
			Content: []ADFNode{
				{
					Type: ADFNodeParagraph,
					Content: []ADFNode{
						{Type: ADFNodeText, Text: item},
					},
				},
			},
		}
	}
	node := ADFNode{
		Type:    ADFNodeBulletList,
		Content: listItems,
	}
	d.Content = append(d.Content, node)
}

// AddOrderedList adds an ordered list to the document.
func (d *ADFDocument) AddOrderedList(items []string) {
	listItems := make([]ADFNode, len(items))
	for i, item := range items {
		listItems[i] = ADFNode{
			Type: ADFNodeListItem,
			Content: []ADFNode{
				{
					Type: ADFNodeParagraph,
					Content: []ADFNode{
						{Type: ADFNodeText, Text: item},
					},
				},
			},
		}
	}
	node := ADFNode{
		Type:    ADFNodeOrderedList,
		Content: listItems,
	}
	d.Content = append(d.Content, node)
}

// AddBlockquote adds a blockquote to the document.
func (d *ADFDocument) AddBlockquote(text string) {
	node := ADFNode{
		Type: ADFNodeBlockquote,
		Content: []ADFNode{
			{
				Type: ADFNodeParagraph,
				Content: []ADFNode{
					{Type: ADFNodeText, Text: text},
				},
			},
		},
	}
	d.Content = append(d.Content, node)
}

// AddRule adds a horizontal rule to the document.
func (d *ADFDocument) AddRule() {
	node := ADFNode{Type: ADFNodeRule}
	d.Content = append(d.Content, node)
}

// TextWithMark creates a text node with formatting marks.
func TextWithMark(text, markType string, attrs map[string]any) ADFNode {
	mark := ADFMark{Type: markType}
	if attrs != nil {
		mark.Attrs = attrs
	}
	return ADFNode{
		Type:  ADFNodeText,
		Text:  text,
		Marks: []ADFMark{mark},
	}
}

// Bold creates bold text.
func Bold(text string) ADFNode {
	return TextWithMark(text, ADFMarkStrong, nil)
}

// Italic creates italic text.
func Italic(text string) ADFNode {
	return TextWithMark(text, ADFMarkEm, nil)
}

// Code creates inline code text.
func Code(text string) ADFNode {
	return TextWithMark(text, ADFMarkCode, nil)
}

// Link creates a linked text.
func Link(text, url string) ADFNode {
	return TextWithMark(text, ADFMarkLink, map[string]any{"href": url})
}

// Strikethrough creates strikethrough text.
func Strikethrough(text string) ADFNode {
	return TextWithMark(text, ADFMarkStrike, nil)
}

// ADFConverter handles conversion between Markdown and ADF.
type ADFConverter struct{}

// NewADFConverter creates a new ADF converter.
func NewADFConverter() *ADFConverter {
	return &ADFConverter{}
}

// ToADF converts Markdown text to an ADF document.
// This is a simplified converter that handles basic markdown.
func (c *ADFConverter) ToADF(markdown string) (*ADFDocument, error) {
	doc := NewADFDocument()

	lines := strings.Split(markdown, "\n")
	i := 0

	for i < len(lines) {
		line := lines[i]

		// Empty line
		if strings.TrimSpace(line) == "" {
			i++
			continue
		}

		// Heading
		if strings.HasPrefix(line, "#") {
			level := 0
			for _, ch := range line {
				if ch == '#' {
					level++
				} else {
					break
				}
			}
			if level > 0 && level <= 6 {
				text := strings.TrimSpace(strings.TrimLeft(line, "#"))
				doc.AddHeading(level, text)
				i++
				continue
			}
		}

		// Horizontal rule
		if line == "---" || line == "***" || line == "___" {
			doc.AddRule()
			i++
			continue
		}

		// Code block
		if strings.HasPrefix(line, "```") {
			language := strings.TrimPrefix(line, "```")
			var codeLines []string
			i++
			for i < len(lines) && !strings.HasPrefix(lines[i], "```") {
				codeLines = append(codeLines, lines[i])
				i++
			}
			i++ // Skip closing ```
			doc.AddCodeBlock(strings.Join(codeLines, "\n"), language)
			continue
		}

		// Blockquote
		if strings.HasPrefix(line, "> ") {
			text := strings.TrimPrefix(line, "> ")
			doc.AddBlockquote(text)
			i++
			continue
		}

		// Bullet list
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			var items []string
			for i < len(lines) && (strings.HasPrefix(lines[i], "- ") || strings.HasPrefix(lines[i], "* ")) {
				item := strings.TrimPrefix(strings.TrimPrefix(lines[i], "- "), "* ")
				items = append(items, item)
				i++
			}
			doc.AddBulletList(items)
			continue
		}

		// Ordered list
		if len(line) > 2 && line[0] >= '0' && line[0] <= '9' && strings.Contains(line[:3], ". ") {
			var items []string
			for i < len(lines) {
				l := lines[i]
				if len(l) > 2 && l[0] >= '0' && l[0] <= '9' && strings.Contains(l[:4], ". ") {
					idx := strings.Index(l, ". ")
					items = append(items, l[idx+2:])
					i++
				} else {
					break
				}
			}
			doc.AddOrderedList(items)
			continue
		}

		// Regular paragraph with inline formatting
		paragraph := c.parseInlineFormatting(line)
		doc.Content = append(doc.Content, paragraph)
		i++
	}

	return doc, nil
}

// parseInlineFormatting parses inline markdown formatting into ADF nodes.
func (c *ADFConverter) parseInlineFormatting(text string) ADFNode {
	// For simplicity, just return plain text
	// A full implementation would parse **bold**, *italic*, `code`, [links](url), etc.
	return ADFNode{
		Type: ADFNodeParagraph,
		Content: []ADFNode{
			{Type: ADFNodeText, Text: text},
		},
	}
}

// FromADF converts an ADF document to Markdown.
func (c *ADFConverter) FromADF(doc *ADFDocument) (string, error) {
	if err := doc.Validate(); err != nil {
		return "", err
	}

	var result strings.Builder
	for i := range doc.Content {
		c.nodeToMarkdown(&result, &doc.Content[i], 0)
	}

	return strings.TrimSpace(result.String()), nil
}

// FromADFAny converts an ADF document from any type (for JSON unmarshaling).
func (c *ADFConverter) FromADFAny(v any) (string, error) {
	if v == nil {
		return "", nil
	}

	// If it's already a string, return it
	if s, ok := v.(string); ok {
		return s, nil
	}

	// Try to marshal and unmarshal as ADFDocument
	jsonBytes, marshalErr := json.Marshal(v)
	if marshalErr != nil {
		return "", fmt.Errorf("marshal adf: %w", marshalErr)
	}

	var doc ADFDocument
	if unmarshalErr := json.Unmarshal(jsonBytes, &doc); unmarshalErr != nil {
		return "", fmt.Errorf("unmarshal adf: %w", unmarshalErr)
	}

	return c.FromADF(&doc)
}

func (c *ADFConverter) nodeToMarkdown(w *strings.Builder, node *ADFNode, depth int) {
	switch node.Type {
	case ADFNodeParagraph:
		c.inlineToMarkdown(w, node.Content)
		w.WriteString("\n\n")

	case ADFNodeHeading:
		level := 1
		if l, ok := node.Attrs["level"].(float64); ok {
			level = int(l)
		}
		w.WriteString(strings.Repeat("#", level))
		w.WriteString(" ")
		c.inlineToMarkdown(w, node.Content)
		w.WriteString("\n\n")

	case ADFNodeCodeBlock:
		lang := ""
		if l, ok := node.Attrs["language"].(string); ok {
			lang = l
		}
		w.WriteString("```")
		w.WriteString(lang)
		w.WriteString("\n")
		c.inlineToMarkdown(w, node.Content)
		w.WriteString("\n```\n\n")

	case ADFNodeBlockquote:
		for i := range node.Content {
			w.WriteString("> ")
			c.nodeToMarkdown(w, &node.Content[i], depth)
		}

	case ADFNodeBulletList:
		for _, item := range node.Content {
			w.WriteString("- ")
			for _, content := range item.Content {
				c.inlineToMarkdown(w, content.Content)
			}
			w.WriteString("\n")
		}
		w.WriteString("\n")

	case ADFNodeOrderedList:
		for i, item := range node.Content {
			fmt.Fprintf(w, "%d. ", i+1)
			for _, content := range item.Content {
				c.inlineToMarkdown(w, content.Content)
			}
			w.WriteString("\n")
		}
		w.WriteString("\n")

	case ADFNodeRule:
		w.WriteString("---\n\n")

	case ADFNodeText:
		c.textToMarkdown(w, node)

	default:
		// For unknown types, try to process content
		for i := range node.Content {
			c.nodeToMarkdown(w, &node.Content[i], depth+1)
		}
	}
}

func (c *ADFConverter) inlineToMarkdown(w *strings.Builder, nodes []ADFNode) {
	for i := range nodes {
		node := &nodes[i]
		switch node.Type {
		case ADFNodeText:
			c.textToMarkdown(w, node)
		case ADFNodeHardBreak:
			w.WriteString("\n")
		case ADFNodeMention:
			if id, ok := node.Attrs["id"].(string); ok {
				w.WriteString("@")
				w.WriteString(id)
			}
		case ADFNodeEmoji:
			if shortName, ok := node.Attrs["shortName"].(string); ok {
				w.WriteString(shortName)
			}
		case ADFNodeInlineCard:
			if url, ok := node.Attrs["url"].(string); ok {
				w.WriteString(url)
			}
		default:
			// Unknown inline, try to extract text
			c.inlineToMarkdown(w, node.Content)
		}
	}
}

func (c *ADFConverter) textToMarkdown(w *strings.Builder, node *ADFNode) {
	text := node.Text

	// Apply marks in reverse order for proper nesting
	prefix := ""
	suffix := ""

	for _, mark := range node.Marks {
		switch mark.Type {
		case ADFMarkStrong:
			prefix = "**" + prefix
			suffix += "**"
		case ADFMarkEm:
			prefix = "*" + prefix
			suffix += "*"
		case ADFMarkStrike:
			prefix = "~~" + prefix
			suffix += "~~"
		case ADFMarkCode:
			prefix = "`" + prefix
			suffix += "`"
		case ADFMarkLink:
			if href, ok := mark.Attrs["href"].(string); ok {
				prefix = "[" + prefix
				suffix = suffix + "](" + href + ")"
			}
		}
	}

	w.WriteString(prefix)
	w.WriteString(text)
	w.WriteString(suffix)
}

// MarkdownToADF is a convenience function that converts Markdown to ADF.
func MarkdownToADF(markdown string) (*ADFDocument, error) {
	return NewADFConverter().ToADF(markdown)
}

// ADFToMarkdown is a convenience function that converts ADF to Markdown.
func ADFToMarkdown(doc *ADFDocument) (string, error) {
	return NewADFConverter().FromADF(doc)
}
