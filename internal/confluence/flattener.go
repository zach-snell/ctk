package confluence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const maxResponseChars = 40000
const logDir = "/tmp/ctk-logs"

// FlattenedPage is a clean, token-efficient representation of a Confluence page.
type FlattenedPage struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	SpaceID  string   `json:"space_id"`
	Status   string   `json:"status"`
	AuthorID string   `json:"author_id,omitempty"`
	Created  string   `json:"created,omitempty"`
	Updated  string   `json:"updated,omitempty"`
	Version  int      `json:"version,omitempty"`
	Body     string   `json:"body,omitempty"`
	Labels   []string `json:"labels,omitempty"`
	ParentID string   `json:"parent_id,omitempty"`
	WebURL   string   `json:"web_url,omitempty"`
}

// FlattenedSpace is a clean representation of a Confluence space.
type FlattenedSpace struct {
	ID         string `json:"id"`
	Key        string `json:"key"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Status     string `json:"status"`
	HomepageID string `json:"homepage_id,omitempty"`
	WebURL     string `json:"web_url,omitempty"`
}

// FlattenedSearchResult is a clean representation of a search result item.
type FlattenedSearchResult struct {
	Title     string `json:"title"`
	Excerpt   string `json:"excerpt"`
	Type      string `json:"type,omitempty"`
	ContentID string `json:"content_id,omitempty"`
	SpaceKey  string `json:"space_key,omitempty"`
	URL       string `json:"url,omitempty"`
}

// FlattenPage converts a raw Page into a FlattenedPage.
func FlattenPage(p *Page) *FlattenedPage {
	fp := &FlattenedPage{
		ID:       p.ID,
		Title:    p.Title,
		SpaceID:  p.SpaceID,
		Status:   p.Status,
		AuthorID: p.AuthorID,
		ParentID: p.ParentID,
	}

	if !p.CreatedAt.IsZero() {
		fp.Created = p.CreatedAt.Format(time.RFC3339)
	}

	if p.Version != nil {
		fp.Version = p.Version.Number
		if !p.Version.CreatedAt.IsZero() {
			fp.Updated = p.Version.CreatedAt.Format(time.RFC3339)
		}
	}

	// Convert body from storage format to markdown
	if p.Body != nil {
		switch {
		case p.Body.Storage != nil && p.Body.Storage.Value != "":
			fp.Body = StorageFormatToMarkdown(p.Body.Storage.Value)
		case p.Body.AtlasDocFormat != nil && p.Body.AtlasDocFormat.Value != "":
			fp.Body = ADFToPlainText(p.Body.AtlasDocFormat.Value)
		case p.Body.View != nil && p.Body.View.Value != "":
			fp.Body = StorageFormatToMarkdown(p.Body.View.Value)
		}
	}

	// Extract labels
	if p.Labels != nil {
		for _, l := range p.Labels.Results {
			fp.Labels = append(fp.Labels, l.Name)
		}
	}

	// Extract web URL from links
	if p.Links != nil {
		if webui, ok := p.Links["webui"]; ok {
			if s, ok := webui.(string); ok {
				fp.WebURL = s
			}
		}
	}

	return fp
}

// FlattenSpace converts a raw Space into a FlattenedSpace.
func FlattenSpace(s *Space) *FlattenedSpace {
	fs := &FlattenedSpace{
		ID:         s.ID,
		Key:        s.Key,
		Name:       s.Name,
		Type:       s.Type,
		Status:     s.Status,
		HomepageID: s.HomepageID,
	}

	if s.Links != nil {
		if webui, ok := s.Links["webui"]; ok {
			if str, ok := webui.(string); ok {
				fs.WebURL = str
			}
		}
	}

	return fs
}

// FlattenSearchResults converts raw search results into flattened results.
func FlattenSearchResults(sr *SearchResult) []FlattenedSearchResult {
	var results []FlattenedSearchResult
	for _, item := range sr.Results {
		fsr := FlattenedSearchResult{
			Title:   item.Title,
			Excerpt: cleanExcerpt(item.Excerpt),
			URL:     item.URL,
		}
		if item.Content != nil {
			fsr.ContentID = item.Content.ID
			fsr.Type = item.Content.Type
			if item.Content.Space != nil {
				fsr.SpaceKey = item.Content.Space.Key
			}
		}
		results = append(results, fsr)
	}
	return results
}

// FlattenedFolder is a clean, token-efficient representation of a Confluence folder.
type FlattenedFolder struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	SpaceID  string `json:"space_id,omitempty"`
	ParentID string `json:"parent_id,omitempty"`
	Status   string `json:"status,omitempty"`
	Created  string `json:"created,omitempty"`
	Updated  string `json:"updated,omitempty"`
	Version  int    `json:"version,omitempty"`
}

// FlattenFolder converts a raw Folder into a FlattenedFolder.
func FlattenFolder(f *Folder) *FlattenedFolder {
	ff := &FlattenedFolder{
		ID:       f.ID,
		Title:    f.Title,
		SpaceID:  f.SpaceID,
		ParentID: f.ParentID,
		Status:   f.Status,
	}

	if !f.CreatedAt.IsZero() {
		ff.Created = f.CreatedAt.Format(time.RFC3339)
	}

	if f.Version != nil {
		ff.Version = f.Version.Number
		if !f.Version.CreatedAt.IsZero() {
			ff.Updated = f.Version.CreatedAt.Format(time.RFC3339)
		}
	}

	return ff
}

// FlattenedComment is a clean representation of a Confluence comment (footer or inline).
type FlattenedComment struct {
	ID              string `json:"id"`
	Type            string `json:"type"` // "footer" or "inline"
	AuthorID        string `json:"author_id,omitempty"`
	Body            string `json:"body,omitempty"` // Converted to markdown
	Created         string `json:"created,omitempty"`
	Updated         string `json:"updated,omitempty"`
	Status          string `json:"status,omitempty"`
	InlineSelection string `json:"inline_selection,omitempty"` // What text was highlighted
}

// FlattenComment converts a footer Comment into a FlattenedComment.
func FlattenComment(c *Comment) *FlattenedComment {
	fc := &FlattenedComment{
		ID:       c.ID,
		Type:     "footer",
		AuthorID: c.AuthorID,
		Status:   c.Status,
	}

	if !c.CreatedAt.IsZero() {
		fc.Created = c.CreatedAt.Format(time.RFC3339)
	}

	if c.Version != nil && !c.Version.CreatedAt.IsZero() {
		fc.Updated = c.Version.CreatedAt.Format(time.RFC3339)
	}

	if c.Body != nil {
		switch {
		case c.Body.Storage != nil && c.Body.Storage.Value != "":
			fc.Body = StorageFormatToMarkdown(c.Body.Storage.Value)
		case c.Body.AtlasDocFormat != nil && c.Body.AtlasDocFormat.Value != "":
			fc.Body = ADFToPlainText(c.Body.AtlasDocFormat.Value)
		case c.Body.View != nil && c.Body.View.Value != "":
			fc.Body = StorageFormatToMarkdown(c.Body.View.Value)
		}
	}

	return fc
}

// FlattenInlineComment converts an InlineComment into a FlattenedComment.
func FlattenInlineComment(c *InlineComment) *FlattenedComment {
	fc := &FlattenedComment{
		ID:       c.ID,
		Type:     "inline",
		AuthorID: c.AuthorID,
		Status:   c.Status,
	}

	if !c.CreatedAt.IsZero() {
		fc.Created = c.CreatedAt.Format(time.RFC3339)
	}

	if c.Version != nil && !c.Version.CreatedAt.IsZero() {
		fc.Updated = c.Version.CreatedAt.Format(time.RFC3339)
	}

	if c.Body != nil {
		switch {
		case c.Body.Storage != nil && c.Body.Storage.Value != "":
			fc.Body = StorageFormatToMarkdown(c.Body.Storage.Value)
		case c.Body.AtlasDocFormat != nil && c.Body.AtlasDocFormat.Value != "":
			fc.Body = ADFToPlainText(c.Body.AtlasDocFormat.Value)
		case c.Body.View != nil && c.Body.View.Value != "":
			fc.Body = StorageFormatToMarkdown(c.Body.View.Value)
		}
	}

	if c.Properties != nil && c.Properties.InlineOriginalSelection != "" {
		fc.InlineSelection = c.Properties.InlineOriginalSelection
	}

	return fc
}

// FlattenedPageVersion is a clean representation of a page version history entry.
type FlattenedPageVersion struct {
	Number    int    `json:"number"`
	Message   string `json:"message,omitempty"`
	AuthorID  string `json:"author_id,omitempty"`
	Created   string `json:"created,omitempty"`
	MinorEdit bool   `json:"minor_edit,omitempty"`
}

// FlattenPageVersion converts a PageVersionDetail into a FlattenedPageVersion.
func FlattenPageVersion(v *PageVersionDetail) *FlattenedPageVersion {
	fv := &FlattenedPageVersion{
		Number:    v.Number,
		Message:   v.Message,
		AuthorID:  v.AuthorID,
		MinorEdit: v.MinorEdit,
	}

	if !v.CreatedAt.IsZero() {
		fv.Created = v.CreatedAt.Format(time.RFC3339)
	}

	return fv
}

// FlattenedAttachment is a clean, token-efficient representation of a Confluence attachment.
type FlattenedAttachment struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	MediaType string `json:"media_type,omitempty"`
	FileSize  int64  `json:"file_size,omitempty"`
	PageID    string `json:"page_id,omitempty"`
	Created   string `json:"created,omitempty"`
	Comment   string `json:"comment,omitempty"`
}

// FlattenAttachment converts a raw Attachment into a FlattenedAttachment.
func FlattenAttachment(a *Attachment) *FlattenedAttachment {
	fa := &FlattenedAttachment{
		ID:        a.ID,
		Title:     a.Title,
		MediaType: a.MediaType,
		FileSize:  a.FileSize,
		PageID:    a.PageID,
		Comment:   a.Comment,
	}

	if !a.CreatedAt.IsZero() {
		fa.Created = a.CreatedAt.Format(time.RFC3339)
	}

	return fa
}

// StripJunkFields removes token-heavy fields from a generic map recursively.
func StripJunkFields(data map[string]interface{}) {
	junkKeys := []string{"_links", "_expandable", "extensions", "metadata", "restrictions", "operations"}
	for _, key := range junkKeys {
		delete(data, key)
	}
	for _, v := range data {
		switch val := v.(type) {
		case map[string]interface{}:
			StripJunkFields(val)
		case []interface{}:
			for _, item := range val {
				if m, ok := item.(map[string]interface{}); ok {
					StripJunkFields(m)
				}
			}
		}
	}
}

// SafeMarshal marshals data to JSON, and if > maxResponseChars, truncates at a
// newline boundary and dumps the full response to a file.
func SafeMarshal(data interface{}) string {
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": "failed to marshal response: %v"}`, err)
	}

	s := string(out)
	if len(s) <= maxResponseChars {
		return s
	}

	originalLen := len(s)

	// Dump full response to file
	_ = os.MkdirAll(logDir, 0o755)
	filename := fmt.Sprintf("ctk-response-%d.json", time.Now().UnixMilli())
	fpath := filepath.Join(logDir, filename)
	_ = os.WriteFile(fpath, out, 0o600)

	// Truncate at a newline boundary for cleaner output
	truncated := s[:maxResponseChars]
	if idx := strings.LastIndex(truncated, "\n"); idx > 0 {
		truncated = truncated[:idx]
	}

	truncPct := (maxResponseChars * 100) / originalLen
	guidance := fmt.Sprintf(`

---
## Response Truncated
This response was truncated to ~%dk chars (%d%% of original %dk chars).

**To get the full data:**
- Full response saved to: %s
- Try narrowing your CQL query
- Use pagination with smaller limit values`,
		maxResponseChars/1000, truncPct, originalLen/1000, fpath)

	return truncated + guidance
}

// StorageFormatToMarkdown converts Confluence storage format (XHTML) to markdown.
// This is a best-effort conversion for LLM consumption, not a perfect HTML-to-MD converter.
func StorageFormatToMarkdown(html string) string {
	if html == "" {
		return ""
	}

	s := html

	// Remove Confluence-specific XML namespaces and macros
	s = reACMacro.ReplaceAllString(s, "")
	s = reACParam.ReplaceAllString(s, "")
	s = reACLink.ReplaceAllStringFunc(s, extractACLinkTitle)
	s = reACImage.ReplaceAllString(s, "[image]")
	s = reRIAttachment.ReplaceAllString(s, "")
	s = reRIPage.ReplaceAllString(s, "")

	// Headers
	for i := 6; i >= 1; i-- {
		prefix := strings.Repeat("#", i)
		openTag := fmt.Sprintf("<h%d[^>]*>", i)
		closeTag := fmt.Sprintf("</h%d>", i)
		s = regexp.MustCompile(openTag).ReplaceAllString(s, prefix+" ")
		s = strings.ReplaceAll(s, closeTag, "\n")
	}

	// Paragraphs
	s = rePOpen.ReplaceAllString(s, "\n")
	s = strings.ReplaceAll(s, "</p>", "\n")

	// Line breaks
	s = reBR.ReplaceAllString(s, "\n")

	// Bold and italic
	s = reStrong.ReplaceAllString(s, "**$1**")
	s = reEm.ReplaceAllString(s, "*$1*")

	// Code blocks
	s = rePreCode.ReplaceAllString(s, "\n```\n$1\n```\n")
	s = reCode.ReplaceAllString(s, "`$1`")

	// Links
	s = reAnchor.ReplaceAllString(s, "[$2]($1)")

	// Unordered lists
	s = reULOpen.ReplaceAllString(s, "\n")
	s = strings.ReplaceAll(s, "</ul>", "\n")
	s = reOLOpen.ReplaceAllString(s, "\n")
	s = strings.ReplaceAll(s, "</ol>", "\n")
	s = reLIOpen.ReplaceAllString(s, "- ")
	s = strings.ReplaceAll(s, "</li>", "\n")

	// Tables - basic conversion
	s = reTROpen.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "</tr>", "|\n")
	s = reTHOpen.ReplaceAllString(s, "| ")
	s = strings.ReplaceAll(s, "</th>", " ")
	s = reTDOpen.ReplaceAllString(s, "| ")
	s = strings.ReplaceAll(s, "</td>", " ")
	s = reTableOpen.ReplaceAllString(s, "\n")
	s = strings.ReplaceAll(s, "</table>", "\n")
	s = reTBody.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "</tbody>", "")
	s = reTHead.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "</thead>", "")

	// Blockquotes
	s = reBQOpen.ReplaceAllString(s, "> ")
	s = strings.ReplaceAll(s, "</blockquote>", "\n")

	// Horizontal rules
	s = reHR.ReplaceAllString(s, "\n---\n")

	// Strip any remaining HTML tags
	s = reAnyTag.ReplaceAllString(s, "")

	// Decode common HTML entities
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", `"`)
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.ReplaceAll(s, "&nbsp;", " ")

	// Clean up excessive whitespace
	s = reMultiNewline.ReplaceAllString(s, "\n\n")
	s = strings.TrimSpace(s)

	return s
}

// ADFToPlainText converts Atlassian Document Format (ADF) JSON to plain text.
func ADFToPlainText(adfJSON string) string {
	if adfJSON == "" {
		return ""
	}

	var doc interface{}
	if err := json.Unmarshal([]byte(adfJSON), &doc); err != nil {
		return adfJSON // Return raw if can't parse
	}

	return extractADFText(doc)
}

func extractADFText(node interface{}) string {
	switch v := node.(type) {
	case map[string]interface{}:
		var sb strings.Builder

		// Extract text from text nodes
		if nodeType, ok := v["type"].(string); ok && nodeType == "text" {
			if text, ok := v["text"].(string); ok {
				sb.WriteString(text)
			}
		}

		// Recurse into content array
		if content, ok := v["content"].([]interface{}); ok {
			for _, child := range content {
				sb.WriteString(extractADFText(child))
			}
		}

		// Add newlines for block-level elements
		if nodeType, ok := v["type"].(string); ok {
			switch nodeType {
			case "paragraph", "heading", "blockquote", "codeBlock",
				"bulletList", "orderedList", "listItem", "table", "tableRow":
				sb.WriteString("\n")
			}
		}

		return sb.String()

	case []interface{}:
		var sb strings.Builder
		for _, item := range v {
			sb.WriteString(extractADFText(item))
		}
		return sb.String()
	}

	return ""
}

func extractACLinkTitle(match string) string {
	// Try to extract ri:content-title from ac:link
	re := regexp.MustCompile(`ri:content-title="([^"]*)"`)
	m := re.FindStringSubmatch(match)
	if len(m) > 1 {
		return "[" + m[1] + "]"
	}
	return ""
}

func cleanExcerpt(excerpt string) string {
	// Remove HTML highlight tags from search excerpts
	excerpt = strings.ReplaceAll(excerpt, "@@@hl@@@", "")
	excerpt = strings.ReplaceAll(excerpt, "@@@endhl@@@", "")
	excerpt = reAnyTag.ReplaceAllString(excerpt, "")
	return strings.TrimSpace(excerpt)
}

// Pre-compiled regexps for storage format conversion.
var (
	reACMacro      = regexp.MustCompile(`<ac:structured-macro[^>]*>.*?</ac:structured-macro>`)
	reACParam      = regexp.MustCompile(`<ac:parameter[^>]*>.*?</ac:parameter>`)
	reACLink       = regexp.MustCompile(`<ac:link[^>]*>.*?</ac:link>`)
	reACImage      = regexp.MustCompile(`<ac:image[^>]*>.*?</ac:image>`)
	reRIAttachment = regexp.MustCompile(`<ri:attachment[^>]*/??>`)
	reRIPage       = regexp.MustCompile(`<ri:page[^>]*/??>`)
	rePOpen        = regexp.MustCompile(`<p[^>]*>`)
	reBR           = regexp.MustCompile(`<br\s*/?>`)
	reStrong       = regexp.MustCompile(`<(?:strong|b)[^>]*>(.*?)</(?:strong|b)>`)
	reEm           = regexp.MustCompile(`<(?:em|i)[^>]*>(.*?)</(?:em|i)>`)
	rePreCode      = regexp.MustCompile(`<pre[^>]*><code[^>]*>(.*?)</code></pre>`)
	reCode         = regexp.MustCompile(`<code[^>]*>(.*?)</code>`)
	reAnchor       = regexp.MustCompile(`<a[^>]*href="([^"]*)"[^>]*>(.*?)</a>`)
	reULOpen       = regexp.MustCompile(`<ul[^>]*>`)
	reOLOpen       = regexp.MustCompile(`<ol[^>]*>`)
	reLIOpen       = regexp.MustCompile(`<li[^>]*>`)
	reTROpen       = regexp.MustCompile(`<tr[^>]*>`)
	reTHOpen       = regexp.MustCompile(`<th[^>]*>`)
	reTDOpen       = regexp.MustCompile(`<td[^>]*>`)
	reTableOpen    = regexp.MustCompile(`<table[^>]*>`)
	reTBody        = regexp.MustCompile(`<tbody[^>]*>`)
	reTHead        = regexp.MustCompile(`<thead[^>]*>`)
	reBQOpen       = regexp.MustCompile(`<blockquote[^>]*>`)
	reHR           = regexp.MustCompile(`<hr\s*/?>`)
	reAnyTag       = regexp.MustCompile(`<[^>]+>`)
	reMultiNewline = regexp.MustCompile(`\n{3,}`)
)
