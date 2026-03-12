package confluence

import "time"

// Space represents a Confluence space.
type Space struct {
	ID          string     `json:"id"`
	Key         string     `json:"key"`
	Name        string     `json:"name"`
	Type        string     `json:"type"`
	Status      string     `json:"status"`
	Description string     `json:"description,omitempty"`
	HomepageID  string     `json:"homepageId,omitempty"`
	CreatedAt   *time.Time `json:"createdAt,omitempty"`
	AuthorID    string     `json:"authorId,omitempty"`
	Links       Links      `json:"_links,omitempty"`
}

// Page represents a Confluence page (V2 API).
type Page struct {
	ID         string       `json:"id"`
	Status     string       `json:"status"`
	Title      string       `json:"title"`
	SpaceID    string       `json:"spaceId,omitempty"`
	ParentID   string       `json:"parentId,omitempty"`
	ParentType string       `json:"parentType,omitempty"`
	AuthorID   string       `json:"authorId,omitempty"`
	CreatedAt  *time.Time   `json:"createdAt,omitempty"`
	Version    *PageVersion `json:"version,omitempty"`
	Body       *PageBody    `json:"body,omitempty"`
	Labels     *LabelList   `json:"labels,omitempty"`
	Links      Links        `json:"_links,omitempty"`
}

// PageVersion represents version info for a page.
type PageVersion struct {
	Number    int        `json:"number"`
	Message   string     `json:"message,omitempty"`
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	AuthorID  string     `json:"authorId,omitempty"`
}

// PageBody holds the body representations.
type PageBody struct {
	Storage        *BodyRepresentation `json:"storage,omitempty"`
	AtlasDocFormat *BodyRepresentation `json:"atlas_doc_format,omitempty"`
	View           *BodyRepresentation `json:"view,omitempty"`
}

// BodyRepresentation is a single body format (storage, atlas_doc_format, view).
type BodyRepresentation struct {
	Representation string `json:"representation"`
	Value          string `json:"value"`
}

// Label represents a Confluence label.
type Label struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Prefix string `json:"prefix,omitempty"`
}

// LabelList is a paginated list of labels.
type LabelList struct {
	Results []Label `json:"results"`
	Links   Links   `json:"_links,omitempty"`
}

// Comment represents a page comment.
type Comment struct {
	ID        string       `json:"id"`
	Status    string       `json:"status"`
	Title     string       `json:"title,omitempty"`
	Body      *PageBody    `json:"body,omitempty"`
	Version   *PageVersion `json:"version,omitempty"`
	AuthorID  string       `json:"authorId,omitempty"`
	CreatedAt *time.Time   `json:"createdAt,omitempty"`
	PageID    string       `json:"pageId,omitempty"`
	Links     Links        `json:"_links,omitempty"`
}

// Attachment represents a page attachment.
type Attachment struct {
	ID          string     `json:"id"`
	Status      string     `json:"status"`
	Title       string     `json:"title"`
	MediaType   string     `json:"mediaType,omitempty"`
	FileSize    int64      `json:"fileSize,omitempty"`
	Comment     string     `json:"comment,omitempty"`
	PageID      string     `json:"pageId,omitempty"`
	Version     int        `json:"version,omitempty"`
	CreatedAt   *time.Time `json:"createdAt,omitempty"`
	DownloadURL string     `json:"downloadLink,omitempty"`
	Links       Links      `json:"_links,omitempty"`
}

// User represents an Atlassian user.
type User struct {
	AccountID   string `json:"accountId"`
	DisplayName string `json:"displayName,omitempty"`
	Email       string `json:"email,omitempty"`
	Type        string `json:"type,omitempty"`
}

// SearchResult is the V1 CQL search response.
type SearchResult struct {
	Results        []SearchResultItem `json:"results"`
	Start          int                `json:"start"`
	Limit          int                `json:"limit"`
	Size           int                `json:"size"`
	TotalSize      int                `json:"totalSize"`
	CQLQuery       string             `json:"cqlQuery,omitempty"`
	SearchDuration int                `json:"searchDuration,omitempty"`
	Links          Links              `json:"_links,omitempty"`
}

// SearchResultItem is a single item in a search result.
type SearchResultItem struct {
	Content               *SearchContent   `json:"content,omitempty"`
	Title                 string           `json:"title"`
	Excerpt               string           `json:"excerpt"`
	URL                   string           `json:"url"`
	ResultGlobalContainer *GlobalContainer `json:"resultGlobalContainer,omitempty"`
	EntityType            string           `json:"entityType,omitempty"`
	LastModified          string           `json:"lastModified,omitempty"`
}

// GlobalContainer represents the space/container context for a search result.
type GlobalContainer struct {
	Title      string `json:"title"`
	DisplayURL string `json:"displayUrl"`
}

// SearchContent is the content inside a search result item.
type SearchContent struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Status string `json:"status"`
	Title  string `json:"title"`
	Space  *Space `json:"space,omitempty"`
	Links  Links  `json:"_links,omitempty"`
}

// PagedResult is the V2 cursor-based paginated response.
type PagedResult[T any] struct {
	Results []T   `json:"results"`
	Links   Links `json:"_links,omitempty"`
}

// Links is a map of link objects returned by Confluence.
type Links map[string]interface{}

// CreatePageRequest is the body for creating a page via V2 API.
type CreatePageRequest struct {
	SpaceID  string    `json:"spaceId"`
	Status   string    `json:"status,omitempty"`
	Title    string    `json:"title"`
	ParentID string    `json:"parentId,omitempty"`
	Body     *PageBody `json:"body"`
}

// UpdatePageRequest is the body for updating a page via V2 API.
type UpdatePageRequest struct {
	ID      string       `json:"id"`
	Status  string       `json:"status"`
	Title   string       `json:"title"`
	Body    *PageBody    `json:"body,omitempty"`
	Version *PageVersion `json:"version"`
}

// AddLabelsRequest is the body for adding labels to a page.
type AddLabelsRequest []AddLabelEntry

// AddLabelEntry is a single label to add.
type AddLabelEntry struct {
	Prefix string `json:"prefix"`
	Name   string `json:"name"`
}

// Folder represents a Confluence folder (V2 API).
type Folder struct {
	ID        string       `json:"id"`
	Title     string       `json:"title"`
	SpaceID   string       `json:"spaceId,omitempty"`
	ParentID  string       `json:"parentId,omitempty"`
	Status    string       `json:"status,omitempty"`
	CreatedAt *time.Time   `json:"createdAt,omitempty"`
	Version   *PageVersion `json:"version,omitempty"`
	Links     Links        `json:"_links,omitempty"`
}

// CreateFolderRequest is the body for creating a folder via V2 API.
type CreateFolderRequest struct {
	SpaceID  string `json:"spaceId"`
	Title    string `json:"title"`
	ParentID string `json:"parentId,omitempty"`
}

// UpdateFolderRequest is the body for updating a folder via V2 API.
type UpdateFolderRequest struct {
	ID      string       `json:"id"`
	Title   string       `json:"title"`
	Version *PageVersion `json:"version"`
}

// PageVersionDetail represents a single version entry in a page's version history.
type PageVersionDetail struct {
	Number    int        `json:"number"`
	Message   string     `json:"message,omitempty"`
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	AuthorID  string     `json:"authorId,omitempty"`
	MinorEdit bool       `json:"minorEdit,omitempty"`
}

// InlineComment represents an inline comment on a Confluence page.
type InlineComment struct {
	ID         string                   `json:"id"`
	Status     string                   `json:"status"`
	Title      string                   `json:"title,omitempty"`
	Body       *PageBody                `json:"body,omitempty"`
	Version    *PageVersion             `json:"version,omitempty"`
	AuthorID   string                   `json:"authorId,omitempty"`
	CreatedAt  *time.Time               `json:"createdAt,omitempty"`
	PageID     string                   `json:"pageId,omitempty"`
	Properties *InlineCommentProperties `json:"properties,omitempty"`
	Links      Links                    `json:"_links,omitempty"`
}

// InlineCommentProperties holds inline-comment-specific metadata.
type InlineCommentProperties struct {
	InlineMarkerRef         string `json:"inlineMarkerRef,omitempty"`
	InlineOriginalSelection string `json:"inlineOriginalSelection,omitempty"`
}

// CreateCommentRequest is the body for creating a footer comment or reply.
type CreateCommentRequest struct {
	PageID string    `json:"pageId,omitempty"`
	Body   *PageBody `json:"body"`
}

// MovePageRequest is the body for moving a page (updating its parent/space).
type MovePageRequest struct {
	ID       string       `json:"id"`
	Status   string       `json:"status"`
	Title    string       `json:"title"`
	SpaceID  string       `json:"spaceId,omitempty"`
	ParentID string       `json:"parentId,omitempty"`
	Version  *PageVersion `json:"version"`
}

// PageDiff represents the diff between two versions of a page.
type PageDiff struct {
	PageID      string `json:"page_id"`
	Title       string `json:"title"`
	FromVersion int    `json:"from_version"`
	ToVersion   int    `json:"to_version"`
	Diff        string `json:"diff"`
}

// APIError is a Confluence API error response.
type APIError struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
	Data       struct {
		Authorized bool     `json:"authorized"`
		Valid      bool     `json:"valid"`
		Errors     []string `json:"errors"`
	} `json:"data,omitempty"`
}
