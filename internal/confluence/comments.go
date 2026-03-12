package confluence

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// ListCommentsArgs are the parameters for listing page comments.
type ListCommentsArgs struct {
	PageID     string `json:"page_id"`
	Limit      int    `json:"limit,omitempty"`
	Cursor     string `json:"cursor,omitempty"`
	BodyFormat string `json:"body_format,omitempty"`
}

// ListComments returns footer comments on a page.
func (c *Client) ListComments(args ListCommentsArgs) (*PagedResult[Comment], error) {
	if args.PageID == "" {
		return nil, fmt.Errorf("page_id is required")
	}

	limit := args.Limit
	if limit == 0 {
		limit = 25
	}

	path := fmt.Sprintf("/wiki/api/v2/pages/%s/footer-comments?limit=%d",
		url.PathEscape(args.PageID), limit)

	if args.BodyFormat != "" {
		path += "&body-format=" + url.QueryEscape(args.BodyFormat)
	}
	if args.Cursor != "" {
		path += "&cursor=" + url.QueryEscape(args.Cursor)
	}

	return GetPaged[Comment](c, path)
}

// ListInlineCommentsArgs are the parameters for listing inline comments on a page.
type ListInlineCommentsArgs struct {
	PageID     string `json:"page_id"`
	Limit      int    `json:"limit,omitempty"`
	Cursor     string `json:"cursor,omitempty"`
	BodyFormat string `json:"body_format,omitempty"`
}

// ListInlineComments returns inline comments on a page.
func (c *Client) ListInlineComments(args ListInlineCommentsArgs) (*PagedResult[InlineComment], error) {
	if args.PageID == "" {
		return nil, fmt.Errorf("page_id is required")
	}

	limit := args.Limit
	if limit == 0 {
		limit = 25
	}

	path := fmt.Sprintf("/wiki/api/v2/pages/%s/inline-comments?limit=%d",
		url.PathEscape(args.PageID), limit)

	if args.BodyFormat != "" {
		path += "&body-format=" + url.QueryEscape(args.BodyFormat)
	}
	if args.Cursor != "" {
		path += "&cursor=" + url.QueryEscape(args.Cursor)
	}

	return GetPaged[InlineComment](c, path)
}

// AddFooterComment creates a new footer comment on a page.
func (c *Client) AddFooterComment(pageID string, body string) (*Comment, error) {
	if pageID == "" {
		return nil, fmt.Errorf("page_id is required")
	}
	if body == "" {
		return nil, fmt.Errorf("body is required")
	}

	req := CreateCommentRequest{
		Body: &PageBody{
			Storage: &BodyRepresentation{
				Representation: "storage",
				Value:          body,
			},
		},
	}

	path := fmt.Sprintf("/wiki/api/v2/pages/%s/footer-comments", url.PathEscape(pageID))
	data, err := c.Post(path, req)
	if err != nil {
		return nil, err
	}

	var comment Comment
	if err := json.Unmarshal(data, &comment); err != nil {
		return nil, fmt.Errorf("unmarshaling created comment: %w", err)
	}

	return &comment, nil
}

// ReplyToComment creates a reply to an existing footer comment.
func (c *Client) ReplyToComment(commentID string, body string) (*Comment, error) {
	if commentID == "" {
		return nil, fmt.Errorf("comment_id is required")
	}
	if body == "" {
		return nil, fmt.Errorf("body is required")
	}

	req := CreateCommentRequest{
		Body: &PageBody{
			Storage: &BodyRepresentation{
				Representation: "storage",
				Value:          body,
			},
		},
	}

	path := fmt.Sprintf("/wiki/api/v2/footer-comments/%s/children", url.PathEscape(commentID))
	data, err := c.Post(path, req)
	if err != nil {
		return nil, err
	}

	var comment Comment
	if err := json.Unmarshal(data, &comment); err != nil {
		return nil, fmt.Errorf("unmarshaling reply comment: %w", err)
	}

	return &comment, nil
}

// GetInlineCommentReplies returns child (reply) comments of an inline comment.
func (c *Client) GetInlineCommentReplies(commentID string, limit int, cursor string) (*PagedResult[InlineComment], error) {
	if commentID == "" {
		return nil, fmt.Errorf("comment_id is required")
	}

	if limit == 0 {
		limit = 25
	}

	path := fmt.Sprintf("/wiki/api/v2/inline-comments/%s/children?limit=%d",
		url.PathEscape(commentID), limit)

	if cursor != "" {
		path += "&cursor=" + url.QueryEscape(cursor)
	}

	return GetPaged[InlineComment](c, path)
}
