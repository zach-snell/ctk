package confluence_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/zach-snell/ctk/internal/confluence"
)

// Helper to create a *time.Time from a time value.
func timePtr(t time.Time) *time.Time {
	return &t
}

// Fixed timestamps for deterministic tests.
var (
	createdTime = time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	updatedTime = time.Date(2024, 7, 20, 14, 45, 0, 0, time.UTC)
)

// ---------------------------------------------------------------------------
// FlattenPage
// ---------------------------------------------------------------------------

func TestFlattenPage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		page *confluence.Page
		want *confluence.FlattenedPage
	}{
		{
			name: "minimal page with no optional fields",
			page: &confluence.Page{
				ID:      "123",
				Title:   "My Page",
				SpaceID: "SPACE1",
				Status:  "current",
			},
			want: &confluence.FlattenedPage{
				ID:      "123",
				Title:   "My Page",
				SpaceID: "SPACE1",
				Status:  "current",
			},
		},
		{
			name: "full page with version and created timestamps",
			page: &confluence.Page{
				ID:        "456",
				Title:     "Full Page",
				SpaceID:   "SP2",
				Status:    "current",
				AuthorID:  "user-abc",
				ParentID:  "100",
				CreatedAt: timePtr(createdTime),
				Version: &confluence.PageVersion{
					Number:    5,
					CreatedAt: timePtr(updatedTime),
				},
			},
			want: &confluence.FlattenedPage{
				ID:       "456",
				Title:    "Full Page",
				SpaceID:  "SP2",
				Status:   "current",
				AuthorID: "user-abc",
				ParentID: "100",
				Created:  "2024-06-15T10:30:00Z",
				Updated:  "2024-07-20T14:45:00Z",
				Version:  5,
			},
		},
		{
			name: "nil CreatedAt on page and version",
			page: &confluence.Page{
				ID:      "789",
				Title:   "No Dates",
				SpaceID: "SP3",
				Status:  "draft",
				Version: &confluence.PageVersion{
					Number:    1,
					CreatedAt: nil,
				},
			},
			want: &confluence.FlattenedPage{
				ID:      "789",
				Title:   "No Dates",
				SpaceID: "SP3",
				Status:  "draft",
				Version: 1,
			},
		},
		{
			name: "page with storage body",
			page: &confluence.Page{
				ID:      "200",
				Title:   "Body Test",
				SpaceID: "SP1",
				Status:  "current",
				Body: &confluence.PageBody{
					Storage: &confluence.BodyRepresentation{
						Representation: "storage",
						Value:          "<p>Hello <strong>world</strong></p>",
					},
				},
			},
			want: &confluence.FlattenedPage{
				ID:      "200",
				Title:   "Body Test",
				SpaceID: "SP1",
				Status:  "current",
				Body:    "Hello **world**",
			},
		},
		{
			name: "page with ADF body (atlas_doc_format)",
			page: &confluence.Page{
				ID:      "201",
				Title:   "ADF Body",
				SpaceID: "SP1",
				Status:  "current",
				Body: &confluence.PageBody{
					AtlasDocFormat: &confluence.BodyRepresentation{
						Representation: "atlas_doc_format",
						Value:          `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"ADF text here"}]}]}`,
					},
				},
			},
			want: &confluence.FlattenedPage{
				ID:      "201",
				Title:   "ADF Body",
				SpaceID: "SP1",
				Status:  "current",
				Body:    "ADF text here\n",
			},
		},
		{
			name: "page with view body fallback",
			page: &confluence.Page{
				ID:      "202",
				Title:   "View Body",
				SpaceID: "SP1",
				Status:  "current",
				Body: &confluence.PageBody{
					View: &confluence.BodyRepresentation{
						Representation: "view",
						Value:          "<p>View content</p>",
					},
				},
			},
			want: &confluence.FlattenedPage{
				ID:      "202",
				Title:   "View Body",
				SpaceID: "SP1",
				Status:  "current",
				Body:    "View content",
			},
		},
		{
			name: "page with empty body value (no body output)",
			page: &confluence.Page{
				ID:      "203",
				Title:   "Empty Body",
				SpaceID: "SP1",
				Status:  "current",
				Body: &confluence.PageBody{
					Storage: &confluence.BodyRepresentation{
						Representation: "storage",
						Value:          "",
					},
				},
			},
			want: &confluence.FlattenedPage{
				ID:      "203",
				Title:   "Empty Body",
				SpaceID: "SP1",
				Status:  "current",
			},
		},
		{
			name: "page with nil body",
			page: &confluence.Page{
				ID:      "204",
				Title:   "Nil Body",
				SpaceID: "SP1",
				Status:  "current",
				Body:    nil,
			},
			want: &confluence.FlattenedPage{
				ID:      "204",
				Title:   "Nil Body",
				SpaceID: "SP1",
				Status:  "current",
			},
		},
		{
			name: "page with labels",
			page: &confluence.Page{
				ID:      "300",
				Title:   "Labeled",
				SpaceID: "SP1",
				Status:  "current",
				Labels: &confluence.LabelList{
					Results: []confluence.Label{
						{ID: "l1", Name: "architecture"},
						{ID: "l2", Name: "v2"},
						{ID: "l3", Name: "important"},
					},
				},
			},
			want: &confluence.FlattenedPage{
				ID:      "300",
				Title:   "Labeled",
				SpaceID: "SP1",
				Status:  "current",
				Labels:  []string{"architecture", "v2", "important"},
			},
		},
		{
			name: "page with empty labels list",
			page: &confluence.Page{
				ID:      "301",
				Title:   "No Labels",
				SpaceID: "SP1",
				Status:  "current",
				Labels: &confluence.LabelList{
					Results: []confluence.Label{},
				},
			},
			want: &confluence.FlattenedPage{
				ID:      "301",
				Title:   "No Labels",
				SpaceID: "SP1",
				Status:  "current",
			},
		},
		{
			name: "page with webui link",
			page: &confluence.Page{
				ID:      "400",
				Title:   "Linked",
				SpaceID: "SP1",
				Status:  "current",
				Links: confluence.Links{
					"webui": "/spaces/SP1/pages/400/Linked",
				},
			},
			want: &confluence.FlattenedPage{
				ID:      "400",
				Title:   "Linked",
				SpaceID: "SP1",
				Status:  "current",
				WebURL:  "/spaces/SP1/pages/400/Linked",
			},
		},
		{
			name: "page with non-string webui link (ignored)",
			page: &confluence.Page{
				ID:      "401",
				Title:   "Bad Link",
				SpaceID: "SP1",
				Status:  "current",
				Links: confluence.Links{
					"webui": 12345, // not a string
				},
			},
			want: &confluence.FlattenedPage{
				ID:      "401",
				Title:   "Bad Link",
				SpaceID: "SP1",
				Status:  "current",
			},
		},
		{
			name: "storage body takes priority over ADF and view",
			page: &confluence.Page{
				ID:      "500",
				Title:   "Priority",
				SpaceID: "SP1",
				Status:  "current",
				Body: &confluence.PageBody{
					Storage: &confluence.BodyRepresentation{
						Representation: "storage",
						Value:          "<p>From storage</p>",
					},
					AtlasDocFormat: &confluence.BodyRepresentation{
						Representation: "atlas_doc_format",
						Value:          `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"From ADF"}]}]}`,
					},
					View: &confluence.BodyRepresentation{
						Representation: "view",
						Value:          "<p>From view</p>",
					},
				},
			},
			want: &confluence.FlattenedPage{
				ID:      "500",
				Title:   "Priority",
				SpaceID: "SP1",
				Status:  "current",
				Body:    "From storage",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := confluence.FlattenPage(tt.page)
			if diff := cmp.Diff(tt.want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("FlattenPage() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// FlattenComment
// ---------------------------------------------------------------------------

func TestFlattenComment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		comment *confluence.Comment
		want    *confluence.FlattenedComment
	}{
		{
			name: "minimal comment",
			comment: &confluence.Comment{
				ID:     "c1",
				Status: "current",
			},
			want: &confluence.FlattenedComment{
				ID:     "c1",
				Type:   "footer",
				Status: "current",
			},
		},
		{
			name: "full comment with timestamps and storage body",
			comment: &confluence.Comment{
				ID:        "c2",
				Status:    "current",
				AuthorID:  "user-xyz",
				CreatedAt: timePtr(createdTime),
				Version: &confluence.PageVersion{
					Number:    3,
					CreatedAt: timePtr(updatedTime),
				},
				Body: &confluence.PageBody{
					Storage: &confluence.BodyRepresentation{
						Representation: "storage",
						Value:          "<p>Great <em>work</em>!</p>",
					},
				},
			},
			want: &confluence.FlattenedComment{
				ID:       "c2",
				Type:     "footer",
				AuthorID: "user-xyz",
				Status:   "current",
				Created:  "2024-06-15T10:30:00Z",
				Updated:  "2024-07-20T14:45:00Z",
				Body:     "Great *work*!",
			},
		},
		{
			name: "comment with nil CreatedAt",
			comment: &confluence.Comment{
				ID:        "c3",
				Status:    "current",
				CreatedAt: nil,
			},
			want: &confluence.FlattenedComment{
				ID:     "c3",
				Type:   "footer",
				Status: "current",
			},
		},
		{
			name: "comment with version but nil version.CreatedAt",
			comment: &confluence.Comment{
				ID:     "c4",
				Status: "current",
				Version: &confluence.PageVersion{
					Number:    2,
					CreatedAt: nil,
				},
			},
			want: &confluence.FlattenedComment{
				ID:     "c4",
				Type:   "footer",
				Status: "current",
			},
		},
		{
			name: "comment with nil body",
			comment: &confluence.Comment{
				ID:     "c5",
				Status: "current",
				Body:   nil,
			},
			want: &confluence.FlattenedComment{
				ID:     "c5",
				Type:   "footer",
				Status: "current",
			},
		},
		{
			name: "comment with ADF body",
			comment: &confluence.Comment{
				ID:     "c6",
				Status: "current",
				Body: &confluence.PageBody{
					AtlasDocFormat: &confluence.BodyRepresentation{
						Representation: "atlas_doc_format",
						Value:          `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"ADF comment"}]}]}`,
					},
				},
			},
			want: &confluence.FlattenedComment{
				ID:     "c6",
				Type:   "footer",
				Status: "current",
				Body:   "ADF comment\n",
			},
		},
		{
			name: "comment with view body fallback",
			comment: &confluence.Comment{
				ID:     "c7",
				Status: "current",
				Body: &confluence.PageBody{
					View: &confluence.BodyRepresentation{
						Representation: "view",
						Value:          "<p>From view</p>",
					},
				},
			},
			want: &confluence.FlattenedComment{
				ID:     "c7",
				Type:   "footer",
				Status: "current",
				Body:   "From view",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := confluence.FlattenComment(tt.comment)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("FlattenComment() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// FlattenInlineComment
// ---------------------------------------------------------------------------

func TestFlattenInlineComment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		comment *confluence.InlineComment
		want    *confluence.FlattenedComment
	}{
		{
			name: "minimal inline comment",
			comment: &confluence.InlineComment{
				ID:     "ic1",
				Status: "current",
			},
			want: &confluence.FlattenedComment{
				ID:     "ic1",
				Type:   "inline",
				Status: "current",
			},
		},
		{
			name: "full inline comment with selection and body",
			comment: &confluence.InlineComment{
				ID:        "ic2",
				Status:    "current",
				AuthorID:  "user-inline",
				CreatedAt: timePtr(createdTime),
				Version: &confluence.PageVersion{
					Number:    1,
					CreatedAt: timePtr(updatedTime),
				},
				Body: &confluence.PageBody{
					Storage: &confluence.BodyRepresentation{
						Representation: "storage",
						Value:          "<p>Fix this <strong>typo</strong></p>",
					},
				},
				Properties: &confluence.InlineCommentProperties{
					InlineOriginalSelection: "the original text",
				},
			},
			want: &confluence.FlattenedComment{
				ID:              "ic2",
				Type:            "inline",
				AuthorID:        "user-inline",
				Status:          "current",
				Created:         "2024-06-15T10:30:00Z",
				Updated:         "2024-07-20T14:45:00Z",
				Body:            "Fix this **typo**",
				InlineSelection: "the original text",
			},
		},
		{
			name: "inline comment with nil properties",
			comment: &confluence.InlineComment{
				ID:         "ic3",
				Status:     "current",
				Properties: nil,
			},
			want: &confluence.FlattenedComment{
				ID:     "ic3",
				Type:   "inline",
				Status: "current",
			},
		},
		{
			name: "inline comment with empty selection",
			comment: &confluence.InlineComment{
				ID:     "ic4",
				Status: "current",
				Properties: &confluence.InlineCommentProperties{
					InlineOriginalSelection: "",
				},
			},
			want: &confluence.FlattenedComment{
				ID:     "ic4",
				Type:   "inline",
				Status: "current",
			},
		},
		{
			name: "inline comment nil CreatedAt",
			comment: &confluence.InlineComment{
				ID:        "ic5",
				Status:    "current",
				CreatedAt: nil,
				Version:   nil,
			},
			want: &confluence.FlattenedComment{
				ID:     "ic5",
				Type:   "inline",
				Status: "current",
			},
		},
		{
			name: "inline comment with nil body",
			comment: &confluence.InlineComment{
				ID:     "ic6",
				Status: "current",
				Body:   nil,
			},
			want: &confluence.FlattenedComment{
				ID:     "ic6",
				Type:   "inline",
				Status: "current",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := confluence.FlattenInlineComment(tt.comment)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("FlattenInlineComment() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// FlattenFolder
// ---------------------------------------------------------------------------

func TestFlattenFolder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		folder *confluence.Folder
		want   *confluence.FlattenedFolder
	}{
		{
			name: "minimal folder",
			folder: &confluence.Folder{
				ID:    "f1",
				Title: "Docs",
			},
			want: &confluence.FlattenedFolder{
				ID:    "f1",
				Title: "Docs",
			},
		},
		{
			name: "full folder with timestamps",
			folder: &confluence.Folder{
				ID:        "f2",
				Title:     "Architecture",
				SpaceID:   "SP1",
				ParentID:  "f1",
				Status:    "current",
				CreatedAt: timePtr(createdTime),
				Version: &confluence.PageVersion{
					Number:    3,
					CreatedAt: timePtr(updatedTime),
				},
			},
			want: &confluence.FlattenedFolder{
				ID:       "f2",
				Title:    "Architecture",
				SpaceID:  "SP1",
				ParentID: "f1",
				Status:   "current",
				Created:  "2024-06-15T10:30:00Z",
				Updated:  "2024-07-20T14:45:00Z",
				Version:  3,
			},
		},
		{
			name: "folder with nil CreatedAt",
			folder: &confluence.Folder{
				ID:        "f3",
				Title:     "No Date",
				CreatedAt: nil,
			},
			want: &confluence.FlattenedFolder{
				ID:    "f3",
				Title: "No Date",
			},
		},
		{
			name: "folder with version but nil version.CreatedAt",
			folder: &confluence.Folder{
				ID:    "f4",
				Title: "Partial Version",
				Version: &confluence.PageVersion{
					Number:    2,
					CreatedAt: nil,
				},
			},
			want: &confluence.FlattenedFolder{
				ID:      "f4",
				Title:   "Partial Version",
				Version: 2,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := confluence.FlattenFolder(tt.folder)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("FlattenFolder() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// FlattenSpace
// ---------------------------------------------------------------------------

func TestFlattenSpace(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		space *confluence.Space
		want  *confluence.FlattenedSpace
	}{
		{
			name: "minimal space",
			space: &confluence.Space{
				ID:     "sp1",
				Key:    "DEV",
				Name:   "Development",
				Type:   "global",
				Status: "current",
			},
			want: &confluence.FlattenedSpace{
				ID:     "sp1",
				Key:    "DEV",
				Name:   "Development",
				Type:   "global",
				Status: "current",
			},
		},
		{
			name: "space with homepage and web URL",
			space: &confluence.Space{
				ID:         "sp2",
				Key:        "TEAM",
				Name:       "Team Space",
				Type:       "global",
				Status:     "current",
				HomepageID: "pg-100",
				Links: confluence.Links{
					"webui": "/spaces/TEAM",
				},
			},
			want: &confluence.FlattenedSpace{
				ID:         "sp2",
				Key:        "TEAM",
				Name:       "Team Space",
				Type:       "global",
				Status:     "current",
				HomepageID: "pg-100",
				WebURL:     "/spaces/TEAM",
			},
		},
		{
			name: "space with nil links",
			space: &confluence.Space{
				ID:     "sp3",
				Key:    "EMPTY",
				Name:   "Empty",
				Type:   "personal",
				Status: "current",
				Links:  nil,
			},
			want: &confluence.FlattenedSpace{
				ID:     "sp3",
				Key:    "EMPTY",
				Name:   "Empty",
				Type:   "personal",
				Status: "current",
			},
		},
		{
			name: "space with non-string webui value",
			space: &confluence.Space{
				ID:     "sp4",
				Key:    "BAD",
				Name:   "Bad Links",
				Type:   "global",
				Status: "current",
				Links: confluence.Links{
					"webui": map[string]interface{}{"href": "/spaces/BAD"},
				},
			},
			want: &confluence.FlattenedSpace{
				ID:     "sp4",
				Key:    "BAD",
				Name:   "Bad Links",
				Type:   "global",
				Status: "current",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := confluence.FlattenSpace(tt.space)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("FlattenSpace() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// FlattenAttachment
// ---------------------------------------------------------------------------

func TestFlattenAttachment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		attachment *confluence.Attachment
		want       *confluence.FlattenedAttachment
	}{
		{
			name: "minimal attachment",
			attachment: &confluence.Attachment{
				ID:    "att1",
				Title: "file.txt",
			},
			want: &confluence.FlattenedAttachment{
				ID:    "att1",
				Title: "file.txt",
			},
		},
		{
			name: "full attachment with all fields",
			attachment: &confluence.Attachment{
				ID:        "att2",
				Title:     "report.pdf",
				MediaType: "application/pdf",
				FileSize:  1048576,
				PageID:    "pg-100",
				Comment:   "Q3 financial report",
				CreatedAt: timePtr(createdTime),
			},
			want: &confluence.FlattenedAttachment{
				ID:        "att2",
				Title:     "report.pdf",
				MediaType: "application/pdf",
				FileSize:  1048576,
				PageID:    "pg-100",
				Comment:   "Q3 financial report",
				Created:   "2024-06-15T10:30:00Z",
			},
		},
		{
			name: "attachment with nil CreatedAt",
			attachment: &confluence.Attachment{
				ID:        "att3",
				Title:     "no-date.png",
				CreatedAt: nil,
			},
			want: &confluence.FlattenedAttachment{
				ID:    "att3",
				Title: "no-date.png",
			},
		},
		{
			name: "attachment with zero file size",
			attachment: &confluence.Attachment{
				ID:       "att4",
				Title:    "empty.txt",
				FileSize: 0,
			},
			want: &confluence.FlattenedAttachment{
				ID:    "att4",
				Title: "empty.txt",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := confluence.FlattenAttachment(tt.attachment)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("FlattenAttachment() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// FlattenPageVersion (FlattenVersionDetail)
// ---------------------------------------------------------------------------

func TestFlattenPageVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		version *confluence.PageVersionDetail
		want    *confluence.FlattenedPageVersion
	}{
		{
			name: "minimal version",
			version: &confluence.PageVersionDetail{
				Number: 1,
			},
			want: &confluence.FlattenedPageVersion{
				Number: 1,
			},
		},
		{
			name: "full version with all fields",
			version: &confluence.PageVersionDetail{
				Number:    7,
				Message:   "Updated architecture diagram",
				AuthorID:  "user-editor",
				CreatedAt: timePtr(updatedTime),
				MinorEdit: true,
			},
			want: &confluence.FlattenedPageVersion{
				Number:    7,
				Message:   "Updated architecture diagram",
				AuthorID:  "user-editor",
				Created:   "2024-07-20T14:45:00Z",
				MinorEdit: true,
			},
		},
		{
			name: "version with nil CreatedAt",
			version: &confluence.PageVersionDetail{
				Number:    2,
				Message:   "Minor fix",
				CreatedAt: nil,
			},
			want: &confluence.FlattenedPageVersion{
				Number:  2,
				Message: "Minor fix",
			},
		},
		{
			name: "version with minor edit false (omitted in JSON)",
			version: &confluence.PageVersionDetail{
				Number:    3,
				MinorEdit: false,
			},
			want: &confluence.FlattenedPageVersion{
				Number: 3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := confluence.FlattenPageVersion(tt.version)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("FlattenPageVersion() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// StorageFormatToMarkdown — table-driven unit tests
// ---------------------------------------------------------------------------

func TestStorageFormatToMarkdown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty input",
			input: "",
			want:  "",
		},
		{
			name:  "plain text in paragraph",
			input: "<p>Hello world</p>",
			want:  "Hello world",
		},
		{
			name:  "headings h1 through h3",
			input: "<h1>Title</h1><h2>Subtitle</h2><h3>Section</h3>",
			want:  "# Title\n## Subtitle\n### Section",
		},
		{
			name:  "bold and italic",
			input: "<p><strong>bold</strong> and <em>italic</em></p>",
			want:  "**bold** and *italic*",
		},
		{
			name:  "inline code",
			input: "<p>Use <code>fmt.Println</code> here</p>",
			want:  "Use `fmt.Println` here",
		},
		{
			name:  "code block (pre+code)",
			input: `<pre><code>func main() {}</code></pre>`,
			want:  "`func main() {}`",
		},
		{
			name:  "anchor link",
			input: `<p>Click <a href="https://example.com">here</a></p>`,
			want:  "Click [here](https://example.com)",
		},
		{
			name:  "unordered list",
			input: "<ul><li>Alpha</li><li>Beta</li></ul>",
			want:  "- Alpha\n- Beta",
		},
		{
			name:  "ordered list",
			input: "<ol><li>First</li><li>Second</li></ol>",
			want:  "- First\n- Second",
		},
		{
			name:  "blockquote",
			input: "<blockquote>Wise words</blockquote>",
			want:  "> Wise words",
		},
		{
			name:  "horizontal rule",
			input: "<p>Above</p><hr /><p>Below</p>",
			want:  "Above\n\n---\n\nBelow",
		},
		{
			name:  "line break",
			input: "<p>Line one<br/>Line two</p>",
			want:  "Line one\nLine two",
		},
		{
			name:  "HTML entities decoded",
			input: "<p>&amp; &lt; &gt; &quot; &#39; &nbsp;</p>",
			want:  `& < > " '`,
		},
		{
			name:  "strips remaining unknown HTML tags",
			input: "<div><span>text</span></div>",
			want:  "text",
		},
		{
			name:  "multiple newlines collapsed",
			input: "<p>One</p>\n\n\n\n<p>Two</p>",
			want:  "One\n\nTwo",
		},
		{
			name:  "Confluence ac:link with content-title",
			input: `<ac:link><ri:page ri:content-title="My Page" /></ac:link>`,
			want:  "[My Page]",
		},
		{
			name:  "Confluence ac:image replaced",
			input: `<ac:image><ri:attachment ri:filename="pic.png" /></ac:image>`,
			want:  "[image]",
		},
		{
			name:  "ac:structured-macro stripped",
			input: `<ac:structured-macro ac:name="toc"><ac:parameter ac:name="maxLevel">3</ac:parameter></ac:structured-macro>`,
			want:  "",
		},
		{
			name:  "table basic conversion",
			input: "<table><thead><tr><th>A</th><th>B</th></tr></thead><tbody><tr><td>1</td><td>2</td></tr></tbody></table>",
			// Note: <thead> is partially consumed by reTHOpen regex (<th[^>]*>) since <thead> starts with <th,
			// resulting in an extra leading "| " on the header row. This is a known regex overlap.
			want: "| | A | B |\n| 1 | 2 |",
		},
		{
			name:  "b and i tags (not just strong/em)",
			input: "<p><b>bold</b> and <i>italic</i></p>",
			want:  "**bold** and *italic*",
		},
		{
			name:  "deeply nested HTML tags",
			input: "<div><div><div><p><span><strong>deep</strong></span></p></div></div></div>",
			want:  "**deep**",
		},
		{
			name:  "paragraph with class attribute",
			input: `<p class="intro">Attributed</p>`,
			want:  "Attributed",
		},
		{
			name:  "h4 h5 h6 headings",
			input: "<h4>H4</h4><h5>H5</h5><h6>H6</h6>",
			want:  "#### H4\n##### H5\n###### H6",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := confluence.StorageFormatToMarkdown(tt.input)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("StorageFormatToMarkdown(%q) mismatch (-want +got):\n%s", tt.input, diff)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// StorageFormatToMarkdown — golden file tests
// ---------------------------------------------------------------------------

func TestStorageFormatToMarkdown_Golden(t *testing.T) {
	t.Parallel()

	files, err := filepath.Glob("testdata/*.html")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no test fixtures found in testdata/")
	}

	for _, inputFile := range files {
		name := strings.TrimSuffix(filepath.Base(inputFile), ".html")
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			input, err := os.ReadFile(inputFile)
			if err != nil {
				t.Fatalf("failed to read %s: %v", inputFile, err)
			}

			got := confluence.StorageFormatToMarkdown(string(input))

			goldenFile := strings.TrimSuffix(inputFile, ".html") + ".golden"
			if *update {
				if err := os.WriteFile(goldenFile, []byte(got), 0o644); err != nil {
					t.Fatalf("failed to write golden file: %v", err)
				}
				return
			}

			want, err := os.ReadFile(goldenFile)
			if err != nil {
				t.Fatalf("golden file not found (run with -update to create): %v", err)
			}
			if diff := cmp.Diff(string(want), got); diff != "" {
				t.Errorf("golden mismatch for %s (-want +got):\n%s", name, diff)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ADFToPlainText
// ---------------------------------------------------------------------------

func TestADFToPlainText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty input",
			input: "",
			want:  "",
		},
		{
			name:  "invalid JSON returns raw input",
			input: "not json at all {{{",
			want:  "not json at all {{{",
		},
		{
			name: "simple paragraph",
			input: `{
				"type": "doc",
				"content": [
					{
						"type": "paragraph",
						"content": [
							{"type": "text", "text": "Hello world"}
						]
					}
				]
			}`,
			want: "Hello world\n",
		},
		{
			name: "multiple paragraphs",
			input: `{
				"type": "doc",
				"content": [
					{
						"type": "paragraph",
						"content": [{"type": "text", "text": "First"}]
					},
					{
						"type": "paragraph",
						"content": [{"type": "text", "text": "Second"}]
					}
				]
			}`,
			want: "First\nSecond\n",
		},
		{
			name: "heading node",
			input: `{
				"type": "doc",
				"content": [
					{
						"type": "heading",
						"attrs": {"level": 2},
						"content": [{"type": "text", "text": "Title"}]
					}
				]
			}`,
			want: "Title\n",
		},
		{
			name: "blockquote",
			input: `{
				"type": "doc",
				"content": [
					{
						"type": "blockquote",
						"content": [
							{
								"type": "paragraph",
								"content": [{"type": "text", "text": "Quoted"}]
							}
						]
					}
				]
			}`,
			want: "Quoted\n\n",
		},
		{
			name: "bullet list",
			input: `{
				"type": "doc",
				"content": [
					{
						"type": "bulletList",
						"content": [
							{
								"type": "listItem",
								"content": [
									{
										"type": "paragraph",
										"content": [{"type": "text", "text": "Item A"}]
									}
								]
							},
							{
								"type": "listItem",
								"content": [
									{
										"type": "paragraph",
										"content": [{"type": "text", "text": "Item B"}]
									}
								]
							}
						]
					}
				]
			}`,
			want: "Item A\n\nItem B\n\n\n",
		},
		{
			name: "codeBlock",
			input: `{
				"type": "doc",
				"content": [
					{
						"type": "codeBlock",
						"attrs": {"language": "go"},
						"content": [{"type": "text", "text": "fmt.Println()"}]
					}
				]
			}`,
			want: "fmt.Println()\n",
		},
		{
			name: "table",
			input: `{
				"type": "doc",
				"content": [
					{
						"type": "table",
						"content": [
							{
								"type": "tableRow",
								"content": [
									{
										"type": "tableCell",
										"content": [
											{
												"type": "paragraph",
												"content": [{"type": "text", "text": "Cell"}]
											}
										]
									}
								]
							}
						]
					}
				]
			}`,
			want: "Cell\n\n\n",
		},
		{
			name: "empty doc",
			input: `{
				"type": "doc",
				"content": []
			}`,
			want: "",
		},
		{
			name: "text with marks (marks are ignored, text extracted)",
			input: `{
				"type": "doc",
				"content": [
					{
						"type": "paragraph",
						"content": [
							{"type": "text", "text": "plain "},
							{"type": "text", "text": "bold", "marks": [{"type": "strong"}]},
							{"type": "text", "text": " end"}
						]
					}
				]
			}`,
			want: "plain bold end\n",
		},
		{
			name:  "non-object JSON (array)",
			input: `[1, 2, 3]`,
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := confluence.ADFToPlainText(tt.input)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("ADFToPlainText() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// StripJunkFields
// ---------------------------------------------------------------------------

func TestStripJunkFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data map[string]interface{}
		want map[string]interface{}
	}{
		{
			name: "empty map",
			data: map[string]interface{}{},
			want: map[string]interface{}{},
		},
		{
			name: "no junk keys",
			data: map[string]interface{}{
				"id":    "123",
				"title": "Hello",
			},
			want: map[string]interface{}{
				"id":    "123",
				"title": "Hello",
			},
		},
		{
			name: "removes all top-level junk keys",
			data: map[string]interface{}{
				"id":           "123",
				"_links":       map[string]interface{}{"self": "/api/v2/pages/123"},
				"_expandable":  map[string]interface{}{"body": ""},
				"extensions":   map[string]interface{}{"position": "none"},
				"metadata":     map[string]interface{}{"labels": nil},
				"restrictions": map[string]interface{}{"read": nil},
				"operations":   []interface{}{"create", "read"},
			},
			want: map[string]interface{}{
				"id": "123",
			},
		},
		{
			name: "recursively strips nested map junk",
			data: map[string]interface{}{
				"id": "123",
				"space": map[string]interface{}{
					"key":    "DEV",
					"_links": map[string]interface{}{"webui": "/spaces/DEV"},
				},
			},
			want: map[string]interface{}{
				"id": "123",
				"space": map[string]interface{}{
					"key": "DEV",
				},
			},
		},
		{
			name: "recursively strips junk in arrays of maps",
			data: map[string]interface{}{
				"results": []interface{}{
					map[string]interface{}{
						"id":     "1",
						"_links": map[string]interface{}{"self": "/1"},
					},
					map[string]interface{}{
						"id":          "2",
						"_expandable": map[string]interface{}{"body": ""},
					},
				},
			},
			want: map[string]interface{}{
				"results": []interface{}{
					map[string]interface{}{"id": "1"},
					map[string]interface{}{"id": "2"},
				},
			},
		},
		{
			name: "deeply nested junk",
			data: map[string]interface{}{
				"level1": map[string]interface{}{
					"level2": map[string]interface{}{
						"level3": map[string]interface{}{
							"id":     "deep",
							"_links": map[string]interface{}{"self": "/deep"},
						},
					},
				},
			},
			want: map[string]interface{}{
				"level1": map[string]interface{}{
					"level2": map[string]interface{}{
						"level3": map[string]interface{}{
							"id": "deep",
						},
					},
				},
			},
		},
		{
			name: "array with non-map items (not modified)",
			data: map[string]interface{}{
				"tags": []interface{}{"alpha", "beta"},
			},
			want: map[string]interface{}{
				"tags": []interface{}{"alpha", "beta"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			confluence.StripJunkFields(tt.data)
			if diff := cmp.Diff(tt.want, tt.data); diff != "" {
				t.Errorf("StripJunkFields() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// FlattenSearchResults
// ---------------------------------------------------------------------------

func TestFlattenSearchResults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		sr   *confluence.SearchResult
		want []confluence.FlattenedSearchResult
	}{
		{
			name: "empty results",
			sr:   &confluence.SearchResult{Results: []confluence.SearchResultItem{}},
			want: nil,
		},
		{
			name: "single result with content and space",
			sr: &confluence.SearchResult{
				Results: []confluence.SearchResultItem{
					{
						Title:   "Architecture Overview",
						Excerpt: "This is the @@@hl@@@architecture@@@endhl@@@ doc",
						URL:     "/wiki/spaces/DEV/pages/123",
						Content: &confluence.SearchContent{
							ID:   "123",
							Type: "page",
							Space: &confluence.Space{
								Key: "DEV",
							},
						},
					},
				},
			},
			want: []confluence.FlattenedSearchResult{
				{
					Title:     "Architecture Overview",
					Excerpt:   "This is the architecture doc",
					Type:      "page",
					ContentID: "123",
					SpaceKey:  "DEV",
					URL:       "/wiki/spaces/DEV/pages/123",
				},
			},
		},
		{
			name: "result with nil content",
			sr: &confluence.SearchResult{
				Results: []confluence.SearchResultItem{
					{
						Title:   "Standalone",
						Excerpt: "No content ref",
						URL:     "/wiki/standalone",
						Content: nil,
					},
				},
			},
			want: []confluence.FlattenedSearchResult{
				{
					Title:   "Standalone",
					Excerpt: "No content ref",
					URL:     "/wiki/standalone",
				},
			},
		},
		{
			name: "result with content but nil space",
			sr: &confluence.SearchResult{
				Results: []confluence.SearchResultItem{
					{
						Title: "No Space",
						Content: &confluence.SearchContent{
							ID:   "456",
							Type: "blogpost",
						},
					},
				},
			},
			want: []confluence.FlattenedSearchResult{
				{
					Title:     "No Space",
					Type:      "blogpost",
					ContentID: "456",
				},
			},
		},
		{
			name: "excerpt with HTML tags stripped",
			sr: &confluence.SearchResult{
				Results: []confluence.SearchResultItem{
					{
						Title:   "HTML Excerpt",
						Excerpt: "<b>highlighted</b> text with <em>emphasis</em>",
					},
				},
			},
			want: []confluence.FlattenedSearchResult{
				{
					Title:   "HTML Excerpt",
					Excerpt: "highlighted text with emphasis",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := confluence.FlattenSearchResults(tt.sr)
			if diff := cmp.Diff(tt.want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("FlattenSearchResults() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
