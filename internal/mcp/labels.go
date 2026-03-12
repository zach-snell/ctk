package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zach-snell/ctk/internal/confluence"
)

type ManageLabelsArgs struct {
	Action string `json:"action" jsonschema:"Action to perform: 'list', 'add', 'remove'" jsonschema_enum:"list,add,remove"`
	PageID string `json:"page_id" jsonschema:"Page ID (required for all actions)"`
	Label  string `json:"label,omitempty" jsonschema:"Label name (required for 'add' and 'remove')"`
	Limit  int    `json:"limit,omitempty" jsonschema:"Number of results per page (default 25)"`
	Cursor string `json:"cursor,omitempty" jsonschema:"Pagination cursor for next page"`
}

// ManageLabelsHandler handles label operations on pages.
func ManageLabelsHandler(c *confluence.Client, canWrite bool) func(context.Context, *mcp.CallToolRequest, ManageLabelsArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args ManageLabelsArgs) (*mcp.CallToolResult, any, error) {
		if args.PageID == "" {
			return ToolResultError("page_id is required for all label operations"), nil, nil
		}

		switch args.Action {
		case "list":
			result, err := c.ListLabels(confluence.ListLabelsArgs{
				PageID: args.PageID,
				Limit:  args.Limit,
				Cursor: args.Cursor,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to list labels: %v", err)), nil, nil
			}
			var names []string
			for _, l := range result.Results {
				names = append(names, l.Name)
			}
			return ToolResultText(confluence.SafeMarshal(names)), nil, nil

		case "add":
			if !canWrite {
				return ToolResultError("write operations are disabled. Set CTK_ENABLE_WRITES=true to enable."), nil, nil
			}
			if args.Label == "" {
				return ToolResultError("label is required for 'add' action"), nil, nil
			}
			label, err := c.AddLabel(confluence.AddLabelArgs{
				PageID: args.PageID,
				Name:   args.Label,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to add label: %v", err)), nil, nil
			}
			return ToolResultText(fmt.Sprintf("Label '%s' added to page %s", label.Name, args.PageID)), nil, nil

		case "remove":
			if !canWrite {
				return ToolResultError("write operations are disabled. Set CTK_ENABLE_WRITES=true to enable."), nil, nil
			}
			if args.Label == "" {
				return ToolResultError("label is required for 'remove' action"), nil, nil
			}
			err := c.RemoveLabel(confluence.RemoveLabelArgs{
				PageID: args.PageID,
				Label:  args.Label,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to remove label: %v", err)), nil, nil
			}
			return ToolResultText(fmt.Sprintf("Label '%s' removed from page %s", args.Label, args.PageID)), nil, nil

		default:
			return ToolResultError(fmt.Sprintf("unknown action: %s. Valid actions: list, add, remove", args.Action)), nil, nil
		}
	}
}
