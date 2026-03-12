package confluence

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// ListAttachmentsArgs are the parameters for listing page attachments.
type ListAttachmentsArgs struct {
	PageID string `json:"page_id"`
	Limit  int    `json:"limit,omitempty"`
	Cursor string `json:"cursor,omitempty"`
}

// ListAttachments returns attachments for a page.
func (c *Client) ListAttachments(args ListAttachmentsArgs) (*PagedResult[Attachment], error) {
	if args.PageID == "" {
		return nil, fmt.Errorf("page_id is required")
	}

	limit := args.Limit
	if limit == 0 {
		limit = 25
	}

	path := fmt.Sprintf("/wiki/api/v2/pages/%s/attachments?limit=%d",
		url.PathEscape(args.PageID), limit)

	if args.Cursor != "" {
		path += "&cursor=" + url.QueryEscape(args.Cursor)
	}

	return GetPaged[Attachment](c, path)
}

// GetAttachment returns a single attachment by ID.
func (c *Client) GetAttachment(attachmentID string) (*Attachment, error) {
	if attachmentID == "" {
		return nil, fmt.Errorf("attachment_id is required")
	}

	path := fmt.Sprintf("/wiki/api/v2/attachments/%s", url.PathEscape(attachmentID))
	return GetJSON[Attachment](c, path)
}

// DownloadAttachment downloads the content of an attachment.
// Returns the raw bytes, media type, and any error.
// The downloadURL should be a relative path from the attachment's downloadLink field.
func (c *Client) DownloadAttachment(downloadURL string) ([]byte, string, error) {
	if downloadURL == "" {
		return nil, "", fmt.Errorf("download URL is required")
	}

	resp, err := c.do(http.MethodGet, downloadURL, nil, "")
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("download failed with status %d: %s", resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("reading attachment content: %w", err)
	}

	mediaType := resp.Header.Get("Content-Type")
	return data, mediaType, nil
}
