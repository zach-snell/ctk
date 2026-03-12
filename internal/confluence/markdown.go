package confluence

import (
	"regexp"
	"strings"
)

// MarkdownToStorage converts markdown text to Confluence storage format (XHTML).
// This is a zero-dependency, best-effort converter that handles common markdown
// constructs. It is the inverse of StorageFormatToMarkdown in flattener.go.
func MarkdownToStorage(markdown string) string {
	if markdown == "" {
		return ""
	}

	// Normalize literal \n sequences (e.g. from JSON) into real newlines.
	s := strings.ReplaceAll(markdown, `\n`, "\n")

	// Normalize line endings.
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")

	lines := strings.Split(s, "\n")

	var out strings.Builder
	i := 0

	for i < len(lines) {
		line := lines[i]

		// --- Fenced code blocks (``` or ~~~) ---
		if mdFencedCodeOpen.MatchString(line) {
			lang := mdFencedCodeOpen.FindStringSubmatch(line)[1]
			i++
			var code strings.Builder
			for i < len(lines) && !mdFencedCodeClose.MatchString(lines[i]) {
				if code.Len() > 0 {
					code.WriteString("\n")
				}
				code.WriteString(lines[i])
				i++
			}
			if i < len(lines) {
				i++ // skip closing fence
			}
			if lang != "" {
				out.WriteString(`<ac:structured-macro ac:name="code">`)
				out.WriteString(`<ac:parameter ac:name="language">`)
				out.WriteString(escapeXML(lang))
				out.WriteString(`</ac:parameter>`)
				out.WriteString(`<ac:plain-text-body><![CDATA[`)
				out.WriteString(code.String())
				out.WriteString(`]]></ac:plain-text-body>`)
				out.WriteString(`</ac:structured-macro>`)
			} else {
				out.WriteString("<pre><code>")
				out.WriteString(escapeXML(code.String()))
				out.WriteString("</code></pre>")
			}
			continue
		}

		// --- Horizontal rule (---, ***, ___) ---
		if mdHorizontalRule.MatchString(line) {
			out.WriteString("<hr />")
			i++
			continue
		}

		// --- Table block ---
		if mdTableRow.MatchString(line) {
			out.WriteString("<table><tbody>")
			isFirstRow := true
			for i < len(lines) && mdTableRow.MatchString(lines[i]) {
				row := lines[i]
				i++
				// Skip separator rows (|---|---|)
				if mdTableSep.MatchString(row) {
					continue
				}
				cells := parseTableCells(row)
				out.WriteString("<tr>")
				tag := "td"
				if isFirstRow {
					tag = "th"
					isFirstRow = false
				}
				for _, cell := range cells {
					out.WriteString("<")
					out.WriteString(tag)
					out.WriteString(">")
					out.WriteString(convertInline(strings.TrimSpace(cell)))
					out.WriteString("</")
					out.WriteString(tag)
					out.WriteString(">")
				}
				out.WriteString("</tr>")
			}
			out.WriteString("</tbody></table>")
			continue
		}

		// --- Headings (# to ######) ---
		if m := mdHeading.FindStringSubmatch(line); m != nil {
			level := len(m[1])
			text := m[2]
			out.WriteString("<h")
			out.WriteByte(byte('0' + level))
			out.WriteString(">")
			out.WriteString(convertInline(text))
			out.WriteString("</h")
			out.WriteByte(byte('0' + level))
			out.WriteString(">")
			i++
			continue
		}

		// --- Blockquote (> text) ---
		if mdBlockquote.MatchString(line) {
			var bqLines []string
			for i < len(lines) && mdBlockquote.MatchString(lines[i]) {
				bqLines = append(bqLines, mdBlockquote.FindStringSubmatch(lines[i])[1])
				i++
			}
			out.WriteString("<blockquote><p>")
			out.WriteString(convertInline(strings.Join(bqLines, " ")))
			out.WriteString("</p></blockquote>")
			continue
		}

		// --- Unordered list (- item or * item) ---
		if mdUnorderedList.MatchString(line) {
			out.WriteString("<ul>")
			for i < len(lines) && mdUnorderedList.MatchString(lines[i]) {
				text := mdUnorderedList.FindStringSubmatch(lines[i])[1]
				out.WriteString("<li>")
				out.WriteString(convertInline(text))
				out.WriteString("</li>")
				i++
			}
			out.WriteString("</ul>")
			continue
		}

		// --- Ordered list (1. item) ---
		if mdOrderedList.MatchString(line) {
			out.WriteString("<ol>")
			for i < len(lines) && mdOrderedList.MatchString(lines[i]) {
				text := mdOrderedList.FindStringSubmatch(lines[i])[1]
				out.WriteString("<li>")
				out.WriteString(convertInline(text))
				out.WriteString("</li>")
				i++
			}
			out.WriteString("</ol>")
			continue
		}

		// --- Empty line (paragraph separator) ---
		if strings.TrimSpace(line) == "" {
			i++
			continue
		}

		// --- Default: paragraph ---
		// Collect consecutive non-blank, non-special lines into one paragraph.
		var paraLines []string
		for i < len(lines) {
			l := lines[i]
			trimmed := strings.TrimSpace(l)
			if trimmed == "" ||
				mdHeading.MatchString(l) ||
				mdFencedCodeOpen.MatchString(l) ||
				mdHorizontalRule.MatchString(l) ||
				mdBlockquote.MatchString(l) ||
				mdUnorderedList.MatchString(l) ||
				mdOrderedList.MatchString(l) ||
				mdTableRow.MatchString(l) {
				break
			}
			paraLines = append(paraLines, trimmed)
			i++
		}
		if len(paraLines) > 0 {
			out.WriteString("<p>")
			out.WriteString(convertInline(strings.Join(paraLines, " ")))
			out.WriteString("</p>")
		}
	}

	return out.String()
}

// convertInline processes inline markdown formatting within a line of text.
func convertInline(text string) string {
	if text == "" {
		return ""
	}

	// Bold: **text** or __text__
	text = mdInlineBold.ReplaceAllString(text, "<strong>$1</strong>")
	// Italic: *text* or _text_ (but not inside words for underscore)
	text = mdInlineItalic.ReplaceAllString(text, "<em>$1</em>")
	// Strikethrough: ~~text~~
	text = mdInlineStrike.ReplaceAllString(text, "<del>$1</del>")
	// Inline code: `text`
	text = mdInlineCode.ReplaceAllString(text, "<code>$1</code>")
	// Links: [text](url)
	text = mdInlineLink.ReplaceAllString(text, `<a href="$2">$1</a>`)
	// Bare URLs (http:// or https://) not already inside an href or tag
	text = mdBareURL.ReplaceAllStringFunc(text, func(match string) string {
		return `<a href="` + match + `">` + match + `</a>`
	})

	return text
}

// parseTableCells splits a markdown table row into cells.
// Input: "| a | b | c |"  →  ["a", "b", "c"]
func parseTableCells(row string) []string {
	row = strings.TrimSpace(row)
	// Strip leading/trailing pipes.
	row = strings.TrimPrefix(row, "|")
	row = strings.TrimSuffix(row, "|")
	parts := strings.Split(row, "|")
	var cells []string
	for _, p := range parts {
		cells = append(cells, strings.TrimSpace(p))
	}
	return cells
}

// escapeXML escapes special XML characters for use in XHTML content.
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}

// Pre-compiled regexps for markdown parsing.
var (
	// Block-level patterns.
	mdHeading         = regexp.MustCompile(`^(#{1,6})\s+(.+)$`)
	mdFencedCodeOpen  = regexp.MustCompile("^```+(\\w*)\\s*$")
	mdFencedCodeClose = regexp.MustCompile("^```+\\s*$")
	mdHorizontalRule  = regexp.MustCompile(`^(?:---+|\*\*\*+|___+)\s*$`)
	mdBlockquote      = regexp.MustCompile(`^>\s?(.*)$`)
	mdUnorderedList   = regexp.MustCompile(`^\s*[-*+]\s+(.+)$`)
	mdOrderedList     = regexp.MustCompile(`^\s*\d+\.\s+(.+)$`)
	mdTableRow        = regexp.MustCompile(`^\|.+\|$`)
	mdTableSep        = regexp.MustCompile(`^\|[\s:]*-+[\s:]*(?:\|[\s:]*-+[\s:]*)*\|$`)

	// Inline patterns — order matters: bold before italic, code before links.
	mdInlineBold   = regexp.MustCompile(`\*\*(.+?)\*\*`)
	mdInlineItalic = regexp.MustCompile(`(?:^|[^*])\*([^*]+?)\*(?:[^*]|$)`)
	mdInlineStrike = regexp.MustCompile(`~~(.+?)~~`)
	mdInlineCode   = regexp.MustCompile("`([^`]+)`")
	mdInlineLink   = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	// Match bare URLs not already inside a tag attribute (preceded by whitespace or start).
	mdBareURL = regexp.MustCompile(`(?:^|[\s(])(https?://[^\s<>\])"]+)`)
)
