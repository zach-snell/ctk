package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zach-snell/ctk/internal/confluence"
)

type ManageCommentsArgs struct {
	Action    string `json:"action" jsonschema:"Action to perform: 'list_footer', 'list_inline', 'get_replies', 'add_footer', 'reply'" jsonschema_enum:"list_footer,list_inline,get_replies,add_footer,reply"`
	PageID    string `json:"page_id,omitempty" jsonschema:"Page ID (required for list_footer, list_inline, add_footer)"`
	CommentID string `json:"comment_id,omitempty" jsonschema:"Comment ID (required for get_replies, reply)"`
	Body      string `json:"body,omitempty" jsonschema:"Comment body in Confluence storage format (required for add_footer, reply)"`
	Limit     int    `json:"limit,omitempty" jsonschema:"Number of results per page (default 25)"`
	Cursor    string `json:"cursor,omitempty" jsonschema:"Pagination cursor for next page"`
}

// ManageCommentsHandler handles comment operations, with optional write support.
func ManageCommentsHandler(c *confluence.Client, canWrite bool) func(context.Context, *mcp.CallToolRequest, ManageCommentsArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args ManageCommentsArgs) (*mcp.CallToolResult, any, error) {
		switch args.Action {
		case "list_footer":
			if args.PageID == "" {
				return ToolResultError("page_id is required for 'list_footer' action"), nil, nil
			}
			result, err := c.ListComments(confluence.ListCommentsArgs{
				PageID:     args.PageID,
				Limit:      args.Limit,
				Cursor:     args.Cursor,
				BodyFormat: "storage",
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to list footer comments: %v", err)), nil, nil
			}
			var flat []*confluence.FlattenedComment
			for i := range result.Results {
				flat = append(flat, confluence.FlattenComment(&result.Results[i]))
			}
			return ToolResultText(confluence.SafeMarshal(flat)), nil, nil

		case "list_inline":
			if args.PageID == "" {
				return ToolResultError("page_id is required for 'list_inline' action"), nil, nil
			}
			result, err := c.ListInlineComments(confluence.ListInlineCommentsArgs{
				PageID:     args.PageID,
				Limit:      args.Limit,
				Cursor:     args.Cursor,
				BodyFormat: "storage",
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to list inline comments: %v", err)), nil, nil
			}
			var flat []*confluence.FlattenedComment
			for i := range result.Results {
				flat = append(flat, confluence.FlattenInlineComment(&result.Results[i]))
			}
			return ToolResultText(confluence.SafeMarshal(flat)), nil, nil

		case "get_replies":
			if args.CommentID == "" {
				return ToolResultError("comment_id is required for 'get_replies' action"), nil, nil
			}
			result, err := c.GetInlineCommentReplies(args.CommentID, args.Limit, args.Cursor)
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to get comment replies: %v", err)), nil, nil
			}
			var flat []*confluence.FlattenedComment
			for i := range result.Results {
				flat = append(flat, confluence.FlattenInlineComment(&result.Results[i]))
			}
			return ToolResultText(confluence.SafeMarshal(flat)), nil, nil

		case "add_footer":
			if !canWrite {
				return ToolResultError("write operations are disabled. Set CTK_ENABLE_WRITES=true to enable."), nil, nil
			}
			if args.PageID == "" {
				return ToolResultError("page_id is required for 'add_footer' action"), nil, nil
			}
			if args.Body == "" {
				return ToolResultError("body is required for 'add_footer' action"), nil, nil
			}
			comment, err := c.AddFooterComment(args.PageID, args.Body)
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to add footer comment: %v", err)), nil, nil
			}
			return ToolResultText(confluence.SafeMarshal(confluence.FlattenComment(comment))), nil, nil

		case "reply":
			if !canWrite {
				return ToolResultError("write operations are disabled. Set CTK_ENABLE_WRITES=true to enable."), nil, nil
			}
			if args.CommentID == "" {
				return ToolResultError("comment_id is required for 'reply' action"), nil, nil
			}
			if args.Body == "" {
				return ToolResultError("body is required for 'reply' action"), nil, nil
			}
			comment, err := c.ReplyToComment(args.CommentID, args.Body)
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to reply to comment: %v", err)), nil, nil
			}
			return ToolResultText(confluence.SafeMarshal(confluence.FlattenComment(comment))), nil, nil

		default:
			return ToolResultError(fmt.Sprintf("unknown action: %s. Valid actions: list_footer, list_inline, get_replies, add_footer, reply", args.Action)), nil, nil
		}
	}
}
