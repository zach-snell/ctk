package confluence

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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
func (c *Client) DownloadAttachment(downloadURL string) (data []byte, mediaType string, err error) {
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

	data, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("reading attachment content: %w", err)
	}

	mediaType = resp.Header.Get("Content-Type")
	return data, mediaType, nil
}

// UploadAttachment uploads a file as an attachment to a page.
// filePath is a path on the MCP server's filesystem (stdio mode = local machine).
// Returns the created attachment metadata.
func (c *Client) UploadAttachment(pageID, filePath, comment string) (*Attachment, error) {
	if pageID == "" {
		return nil, fmt.Errorf("page_id is required")
	}
	if filePath == "" {
		return nil, fmt.Errorf("file_path is required")
	}

	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	filename := filepath.Base(filePath)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("creating form file: %w", err)
	}
	if _, err := part.Write(fileData); err != nil {
		return nil, fmt.Errorf("writing file data: %w", err)
	}

	if comment != "" {
		if err := writer.WriteField("comment", comment); err != nil {
			return nil, fmt.Errorf("writing comment field: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("closing multipart writer: %w", err)
	}

	apiPath := fmt.Sprintf("/wiki/api/v2/attachments?pageId=%s", url.QueryEscape(pageID))

	// Need custom request for multipart + X-Atlassian-Token header
	if err := c.rateLimiter.Wait(); err != nil {
		return nil, err
	}

	reqURL := c.baseURL + apiPath
	req, err := http.NewRequest(http.MethodPost, reqURL, &body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.SetBasicAuth(c.email, c.token)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Atlassian-Token", "nocheck")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing upload: %w", err)
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("upload failed (HTTP %d): %s", resp.StatusCode, string(respData))
	}

	// V2 API returns a single attachment object
	var att Attachment
	if err := json.Unmarshal(respData, &att); err != nil {
		return nil, fmt.Errorf("unmarshaling response: %w", err)
	}
	return &att, nil
}

// DeleteAttachment permanently deletes an attachment by ID.
func (c *Client) DeleteAttachment(attachmentID string) error {
	if attachmentID == "" {
		return fmt.Errorf("attachment_id is required")
	}
	path := fmt.Sprintf("/wiki/api/v2/attachments/%s", url.PathEscape(attachmentID))
	return c.Delete(path)
}
