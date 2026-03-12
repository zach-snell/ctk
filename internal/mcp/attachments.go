package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zach-snell/ctk/internal/confluence"
)

type ManageAttachmentsArgs struct {
	Action       string `json:"action" jsonschema:"Action to perform: 'list', 'download'" jsonschema_enum:"list,download"`
	PageID       string `json:"page_id,omitempty" jsonschema:"Page ID (required for list)"`
	AttachmentID string `json:"attachment_id,omitempty" jsonschema:"Attachment ID (required for download)"`
	Limit        int    `json:"limit,omitempty" jsonschema:"Number of results per page (default 25)"`
	Cursor       string `json:"cursor,omitempty" jsonschema:"Pagination cursor for next page"`
}

const downloadDir = "/tmp/ctk-downloads"

// ManageAttachmentsHandler handles attachment operations.
func ManageAttachmentsHandler(c *confluence.Client) func(context.Context, *mcp.CallToolRequest, ManageAttachmentsArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args ManageAttachmentsArgs) (*mcp.CallToolResult, any, error) {
		switch args.Action {
		case "list":
			if args.PageID == "" {
				return ToolResultError("page_id is required for 'list' action"), nil, nil
			}
			result, err := c.ListAttachments(confluence.ListAttachmentsArgs{
				PageID: args.PageID,
				Limit:  args.Limit,
				Cursor: args.Cursor,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to list attachments: %v", err)), nil, nil
			}
			var flat []*confluence.FlattenedAttachment
			for i := range result.Results {
				flat = append(flat, confluence.FlattenAttachment(&result.Results[i]))
			}
			return ToolResultText(confluence.SafeMarshal(flat)), nil, nil

		case "download":
			if args.AttachmentID == "" {
				return ToolResultError("attachment_id is required for 'download' action"), nil, nil
			}

			// First get attachment metadata to find the download URL
			att, err := c.GetAttachment(args.AttachmentID)
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to get attachment metadata: %v", err)), nil, nil
			}

			dlURL := att.DownloadURL
			if dlURL == "" {
				// Try _links.download as fallback
				if att.Links != nil {
					if dl, ok := att.Links["download"]; ok {
						if s, ok := dl.(string); ok {
							dlURL = s
						}
					}
				}
			}

			if dlURL == "" {
				return ToolResultError("no download URL found for this attachment"), nil, nil
			}

			data, mediaType, err := c.DownloadAttachment(dlURL)
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to download attachment: %v", err)), nil, nil
			}

			// Save to download directory
			if err := os.MkdirAll(downloadDir, 0o755); err != nil {
				return ToolResultError(fmt.Sprintf("failed to create download directory: %v", err)), nil, nil
			}

			filename := att.Title
			if filename == "" {
				filename = fmt.Sprintf("attachment-%s", args.AttachmentID)
			}
			savePath := filepath.Join(downloadDir, filename)

			if err := os.WriteFile(savePath, data, 0o644); err != nil {
				return ToolResultError(fmt.Sprintf("failed to save attachment: %v", err)), nil, nil
			}

			result := map[string]interface{}{
				"path":       savePath,
				"filename":   filename,
				"media_type": mediaType,
				"size_bytes": len(data),
			}
			return ToolResultText(confluence.SafeMarshal(result)), nil, nil

		default:
			return ToolResultError(fmt.Sprintf("unknown action: %s. Valid actions: list, download", args.Action)), nil, nil
		}
	}
}
