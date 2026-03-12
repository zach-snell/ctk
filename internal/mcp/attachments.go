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
	Action       string `json:"action" jsonschema:"Action to perform: 'list', 'download', 'upload', 'delete'" jsonschema_enum:"list,download,upload,delete"`
	PageID       string `json:"page_id,omitempty" jsonschema:"Page ID (required for list, upload)"`
	AttachmentID string `json:"attachment_id,omitempty" jsonschema:"Attachment ID (required for download, delete)"`
	FilePath     string `json:"file_path,omitempty" jsonschema:"Absolute path to the file to upload (required for upload). Note: paths refer to the MCP server's filesystem. In stdio mode this is the local machine."`
	Comment      string `json:"comment,omitempty" jsonschema:"Optional comment for the attachment (for upload)"`
	Limit        int    `json:"limit,omitempty" jsonschema:"Number of results per page (default 25)"`
	Cursor       string `json:"cursor,omitempty" jsonschema:"Pagination cursor for next page"`
}

const downloadDir = "/tmp/ctk-downloads"

// ManageAttachmentsHandler handles attachment operations, with optional write support.
func ManageAttachmentsHandler(c *confluence.Client, canWrite bool) func(context.Context, *mcp.CallToolRequest, ManageAttachmentsArgs) (*mcp.CallToolResult, any, error) {
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

			if err := os.WriteFile(savePath, data, 0o600); err != nil {
				return ToolResultError(fmt.Sprintf("failed to save attachment: %v", err)), nil, nil
			}

			result := map[string]interface{}{
				"path":       savePath,
				"filename":   filename,
				"media_type": mediaType,
				"size_bytes": len(data),
			}
			return ToolResultText(confluence.SafeMarshal(result)), nil, nil

		case "upload":
			if !canWrite {
				return ToolResultError("upload requires CTK_ENABLE_WRITES=true"), nil, nil
			}
			if args.PageID == "" || args.FilePath == "" {
				return ToolResultError("page_id and file_path are required for 'upload' action"), nil, nil
			}
			att, err := c.UploadAttachment(args.PageID, args.FilePath, args.Comment)
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to upload attachment: %v", err)), nil, nil
			}
			return ToolResultText(confluence.SafeMarshal(map[string]interface{}{
				"id":         att.ID,
				"title":      att.Title,
				"media_type": att.MediaType,
				"file_size":  att.FileSize,
				"status":     "uploaded",
			})), nil, nil

		case "delete":
			if !canWrite {
				return ToolResultError("delete requires CTK_ENABLE_WRITES=true"), nil, nil
			}
			if args.AttachmentID == "" {
				return ToolResultError("attachment_id is required for 'delete' action"), nil, nil
			}
			if err := c.DeleteAttachment(args.AttachmentID); err != nil {
				return ToolResultError(fmt.Sprintf("failed to delete attachment: %v", err)), nil, nil
			}
			return ToolResultText("Attachment deleted successfully"), nil, nil

		default:
			return ToolResultError(fmt.Sprintf("unknown action: %s. Valid actions: list, download, upload, delete", args.Action)), nil, nil
		}
	}
}
