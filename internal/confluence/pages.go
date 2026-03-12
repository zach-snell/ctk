package confluence

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// GetPageArgs are the parameters for getting a single page.
type GetPageArgs struct {
	PageID      string `json:"page_id"`
	BodyFormat  string `json:"body_format,omitempty"` // "storage", "atlas_doc_format", "view"
	IncludeBody bool   `json:"include_body,omitempty"`
}

// GetPage returns a single page by ID.
func (c *Client) GetPage(args GetPageArgs) (*Page, error) {
	if args.PageID == "" {
		return nil, fmt.Errorf("page_id is required")
	}

	path := fmt.Sprintf("/wiki/api/v2/pages/%s", url.PathEscape(args.PageID))

	if args.IncludeBody {
		format := args.BodyFormat
		if format == "" {
			format = "storage"
		}
		path += "?body-format=" + url.QueryEscape(format)
	}

	return GetJSON[Page](c, path)
}

// GetPageByTitleArgs are the parameters for getting a page by title.
type GetPageByTitleArgs struct {
	SpaceID    string `json:"space_id"`
	Title      string `json:"title"`
	BodyFormat string `json:"body_format,omitempty"`
}

// GetPageByTitle returns a page matching the given title in a space.
func (c *Client) GetPageByTitle(args GetPageByTitleArgs) (*Page, error) {
	if args.SpaceID == "" {
		return nil, fmt.Errorf("space_id is required")
	}
	if args.Title == "" {
		return nil, fmt.Errorf("title is required")
	}

	path := fmt.Sprintf("/wiki/api/v2/pages?space-id=%s&title=%s&limit=1",
		url.QueryEscape(args.SpaceID),
		url.QueryEscape(args.Title),
	)

	if args.BodyFormat != "" {
		path += "&body-format=" + url.QueryEscape(args.BodyFormat)
	}

	result, err := GetPaged[Page](c, path)
	if err != nil {
		return nil, err
	}

	if len(result.Results) == 0 {
		return nil, fmt.Errorf("page with title %q not found in space %s", args.Title, args.SpaceID)
	}

	return &result.Results[0], nil
}

// ListPagesArgs are the parameters for listing pages in a space.
type ListPagesArgs struct {
	SpaceID    string `json:"space_id"`
	Limit      int    `json:"limit,omitempty"`
	Cursor     string `json:"cursor,omitempty"`
	Title      string `json:"title,omitempty"`
	Status     string `json:"status,omitempty"` // "current", "draft", "archived", "trashed"
	BodyFormat string `json:"body_format,omitempty"`
	Sort       string `json:"sort,omitempty"` // e.g., "-modified-date", "title"
}

// ListPages returns pages in a space.
func (c *Client) ListPages(args ListPagesArgs) (*PagedResult[Page], error) {
	if args.SpaceID == "" {
		return nil, fmt.Errorf("space_id is required")
	}

	limit := args.Limit
	if limit == 0 {
		limit = 25
	}

	path := fmt.Sprintf("/wiki/api/v2/spaces/%s/pages?limit=%d", url.PathEscape(args.SpaceID), limit)

	if args.Title != "" {
		path += "&title=" + url.QueryEscape(args.Title)
	}
	if args.Status != "" {
		path += "&status=" + url.QueryEscape(args.Status)
	}
	if args.BodyFormat != "" {
		path += "&body-format=" + url.QueryEscape(args.BodyFormat)
	}
	if args.Sort != "" {
		path += "&sort=" + url.QueryEscape(args.Sort)
	}
	if args.Cursor != "" {
		path += "&cursor=" + url.QueryEscape(args.Cursor)
	}

	return GetPaged[Page](c, path)
}

// GetChildrenArgs are the parameters for getting child pages.
type GetChildrenArgs struct {
	PageID string `json:"page_id"`
	Limit  int    `json:"limit,omitempty"`
	Cursor string `json:"cursor,omitempty"`
}

// GetChildren returns child pages of a given page.
func (c *Client) GetChildren(args GetChildrenArgs) (*PagedResult[Page], error) {
	if args.PageID == "" {
		return nil, fmt.Errorf("page_id is required")
	}

	limit := args.Limit
	if limit == 0 {
		limit = 25
	}

	path := fmt.Sprintf("/wiki/api/v2/pages/%s/children?limit=%d", url.PathEscape(args.PageID), limit)
	if args.Cursor != "" {
		path += "&cursor=" + url.QueryEscape(args.Cursor)
	}

	return GetPaged[Page](c, path)
}

// GetAncestorsArgs are the parameters for getting page ancestors.
type GetAncestorsArgs struct {
	PageID string `json:"page_id"`
}

// GetAncestors returns the ancestor pages of a given page.
func (c *Client) GetAncestors(args GetAncestorsArgs) (*PagedResult[Page], error) {
	if args.PageID == "" {
		return nil, fmt.Errorf("page_id is required")
	}

	return GetPaged[Page](c, fmt.Sprintf("/wiki/api/v2/pages/%s/ancestors", url.PathEscape(args.PageID)))
}

// CreatePageArgs are the parameters for creating a page.
type CreatePageArgs struct {
	SpaceID  string `json:"space_id"`
	Title    string `json:"title"`
	Body     string `json:"body"` // Storage format (XHTML)
	ParentID string `json:"parent_id,omitempty"`
	Status   string `json:"status,omitempty"` // "current" (default), "draft"
}

// CreatePage creates a new page.
func (c *Client) CreatePage(args CreatePageArgs) (*Page, error) {
	if args.SpaceID == "" {
		return nil, fmt.Errorf("space_id is required")
	}
	if args.Title == "" {
		return nil, fmt.Errorf("title is required")
	}

	status := args.Status
	if status == "" {
		status = "current"
	}

	req := CreatePageRequest{
		SpaceID:  args.SpaceID,
		Status:   status,
		Title:    args.Title,
		ParentID: args.ParentID,
		Body: &PageBody{
			Storage: &BodyRepresentation{
				Representation: "storage",
				Value:          args.Body,
			},
		},
	}

	data, err := c.Post("/wiki/api/v2/pages", req)
	if err != nil {
		return nil, err
	}

	var page Page
	if err := json.Unmarshal(data, &page); err != nil {
		return nil, fmt.Errorf("unmarshaling created page: %w", err)
	}

	return &page, nil
}

// UpdatePageArgs are the parameters for updating a page.
type UpdatePageArgs struct {
	PageID  string `json:"page_id"`
	Title   string `json:"title"`
	Body    string `json:"body,omitempty"` // Storage format (XHTML)
	Version int    `json:"version"`        // Must be current version + 1
	Status  string `json:"status,omitempty"`
}

// UpdatePage updates an existing page.
func (c *Client) UpdatePage(args UpdatePageArgs) (*Page, error) {
	if args.PageID == "" {
		return nil, fmt.Errorf("page_id is required")
	}
	if args.Title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if args.Version == 0 {
		return nil, fmt.Errorf("version is required (must be current version + 1)")
	}

	status := args.Status
	if status == "" {
		status = "current"
	}

	req := UpdatePageRequest{
		ID:     args.PageID,
		Status: status,
		Title:  args.Title,
		Version: &PageVersion{
			Number: args.Version,
		},
	}

	if args.Body != "" {
		req.Body = &PageBody{
			Storage: &BodyRepresentation{
				Representation: "storage",
				Value:          args.Body,
			},
		}
	}

	data, err := c.Put(fmt.Sprintf("/wiki/api/v2/pages/%s", url.PathEscape(args.PageID)), req)
	if err != nil {
		return nil, err
	}

	var page Page
	if err := json.Unmarshal(data, &page); err != nil {
		return nil, fmt.Errorf("unmarshaling updated page: %w", err)
	}

	return &page, nil
}

// ListPageVersions returns the version history of a page.
func (c *Client) ListPageVersions(pageID string, limit int, cursor string) (*PagedResult[PageVersionDetail], error) {
	if pageID == "" {
		return nil, fmt.Errorf("page_id is required")
	}

	if limit == 0 {
		limit = 25
	}

	path := fmt.Sprintf("/wiki/api/v2/pages/%s/versions?limit=%d", url.PathEscape(pageID), limit)
	if cursor != "" {
		path += "&cursor=" + url.QueryEscape(cursor)
	}

	return GetPaged[PageVersionDetail](c, path)
}

// DeletePageArgs are the parameters for deleting a page.
type DeletePageArgs struct {
	PageID string `json:"page_id"`
}

// DeletePage deletes a page by ID.
func (c *Client) DeletePage(args DeletePageArgs) error {
	if args.PageID == "" {
		return fmt.Errorf("page_id is required")
	}

	return c.Delete(fmt.Sprintf("/wiki/api/v2/pages/%s", url.PathEscape(args.PageID)))
}

// MovePageArgs are the parameters for moving a page to a new parent or space.
type MovePageArgs struct {
	PageID         string `json:"page_id"`
	TargetSpaceID  string `json:"target_space_id,omitempty"`
	TargetParentID string `json:"target_parent_id,omitempty"`
	Version        int    `json:"version"` // current version + 1
}

// MovePage moves a page to a new parent and/or space by updating it.
// In the Confluence V2 API, moving is done via PUT with new parentId/spaceId.
func (c *Client) MovePage(args MovePageArgs) (*Page, error) {
	if args.PageID == "" {
		return nil, fmt.Errorf("page_id is required")
	}
	if args.Version == 0 {
		return nil, fmt.Errorf("version is required (must be current version + 1)")
	}
	if args.TargetSpaceID == "" && args.TargetParentID == "" {
		return nil, fmt.Errorf("at least one of target_space_id or target_parent_id is required")
	}

	// Get the current page to preserve its title and status
	current, err := c.GetPage(GetPageArgs{PageID: args.PageID})
	if err != nil {
		return nil, fmt.Errorf("getting current page for move: %w", err)
	}

	req := MovePageRequest{
		ID:     args.PageID,
		Status: current.Status,
		Title:  current.Title,
		Version: &PageVersion{
			Number: args.Version,
		},
	}

	if args.TargetSpaceID != "" {
		req.SpaceID = args.TargetSpaceID
	}
	if args.TargetParentID != "" {
		req.ParentID = args.TargetParentID
	}

	path := fmt.Sprintf("/wiki/api/v2/pages/%s", url.PathEscape(args.PageID))

	data, err := c.Put(path, req)
	if err != nil {
		return nil, err
	}

	var page Page
	if err := json.Unmarshal(data, &page); err != nil {
		return nil, fmt.Errorf("unmarshaling moved page: %w", err)
	}

	return &page, nil
}

// GetPageVersionContent returns a page at a specific version number.
func (c *Client) GetPageVersionContent(pageID string, versionNumber int) (*Page, error) {
	if pageID == "" {
		return nil, fmt.Errorf("page_id is required")
	}
	if versionNumber <= 0 {
		return nil, fmt.Errorf("version number must be positive")
	}

	path := fmt.Sprintf("/wiki/api/v2/pages/%s?version=%d&body-format=storage",
		url.PathEscape(pageID), versionNumber)

	return GetJSON[Page](c, path)
}

// DiffPageVersions computes a text diff between two versions of a page.
// It fetches both versions, converts their storage format to markdown,
// and produces a unified-diff-style output.
func (c *Client) DiffPageVersions(pageID string, fromVersion, toVersion int) (*PageDiff, error) {
	if pageID == "" {
		return nil, fmt.Errorf("page_id is required")
	}
	if fromVersion <= 0 || toVersion <= 0 {
		return nil, fmt.Errorf("version numbers must be positive")
	}
	if fromVersion == toVersion {
		return nil, fmt.Errorf("from_version and to_version must be different")
	}

	fromPage, err := c.GetPageVersionContent(pageID, fromVersion)
	if err != nil {
		return nil, fmt.Errorf("getting version %d: %w", fromVersion, err)
	}

	toPage, err := c.GetPageVersionContent(pageID, toVersion)
	if err != nil {
		return nil, fmt.Errorf("getting version %d: %w", toVersion, err)
	}

	// Convert both to markdown
	var fromMD, toMD string
	if fromPage.Body != nil && fromPage.Body.Storage != nil {
		fromMD = StorageFormatToMarkdown(fromPage.Body.Storage.Value)
	}
	if toPage.Body != nil && toPage.Body.Storage != nil {
		toMD = StorageFormatToMarkdown(toPage.Body.Storage.Value)
	}

	fromLines := strings.Split(fromMD, "\n")
	toLines := strings.Split(toMD, "\n")

	diff := computeDiff(fromLines, toLines, fromVersion, toVersion)

	title := toPage.Title
	if title == "" {
		title = fromPage.Title
	}

	return &PageDiff{
		PageID:      pageID,
		Title:       title,
		FromVersion: fromVersion,
		ToVersion:   toVersion,
		Diff:        diff,
	}, nil
}

// computeDiff produces a unified-diff-style output comparing two slices of lines.
// Uses a simple longest common subsequence approach.
func computeDiff(fromLines, toLines []string, fromVer, toVer int) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("--- version %d\n", fromVer))
	sb.WriteString(fmt.Sprintf("+++ version %d\n", toVer))

	// Build LCS table
	m, n := len(fromLines), len(toLines)
	lcs := make([][]int, m+1)
	for i := range lcs {
		lcs[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			switch {
			case fromLines[i-1] == toLines[j-1]:
				lcs[i][j] = lcs[i-1][j-1] + 1
			case lcs[i-1][j] >= lcs[i][j-1]:
				lcs[i][j] = lcs[i-1][j]
			default:
				lcs[i][j] = lcs[i][j-1]
			}
		}
	}

	// Backtrack to produce diff
	type diffLine struct {
		op   byte // ' ', '-', '+'
		text string
	}
	var result []diffLine

	i, j := m, n
	for i > 0 || j > 0 {
		switch {
		case i > 0 && j > 0 && fromLines[i-1] == toLines[j-1]:
			result = append(result, diffLine{' ', fromLines[i-1]})
			i--
			j--
		case j > 0 && (i == 0 || lcs[i][j-1] >= lcs[i-1][j]):
			result = append(result, diffLine{'+', toLines[j-1]})
			j--
		case i > 0:
			result = append(result, diffLine{'-', fromLines[i-1]})
			i--
		}
	}

	// Reverse the result (we built it backwards)
	for left, right := 0, len(result)-1; left < right; left, right = left+1, right-1 {
		result[left], result[right] = result[right], result[left]
	}

	// Output only changed regions with context (3 lines)
	const contextLines = 3
	changed := make([]bool, len(result))
	for idx, dl := range result {
		if dl.op != ' ' {
			changed[idx] = true
		}
	}

	// Expand context around changes
	show := make([]bool, len(result))
	for idx := range result {
		if changed[idx] {
			for c := idx - contextLines; c <= idx+contextLines; c++ {
				if c >= 0 && c < len(result) {
					show[c] = true
				}
			}
		}
	}

	lastShown := -1
	for idx, dl := range result {
		if !show[idx] {
			continue
		}
		if lastShown >= 0 && idx > lastShown+1 {
			sb.WriteString("@@ ... @@\n")
		}
		sb.WriteByte(dl.op)
		sb.WriteString(dl.text)
		sb.WriteByte('\n')
		lastShown = idx
	}

	if sb.Len() == 0 {
		return "(no differences found)"
	}

	return sb.String()
}
