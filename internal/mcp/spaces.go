package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zach-snell/ctk/internal/confluence"
)

type ManageSpacesArgs struct {
	Action  string `json:"action" jsonschema:"Action to perform: 'list', 'get', 'get_by_key'" jsonschema_enum:"list,get,get_by_key"`
	SpaceID string `json:"space_id,omitempty" jsonschema:"Space ID (required for 'get')"`
	Key     string `json:"key,omitempty" jsonschema:"Space key (required for 'get_by_key')"`
	Limit   int    `json:"limit,omitempty" jsonschema:"Number of results per page (default 25)"`
	Cursor  string `json:"cursor,omitempty" jsonschema:"Pagination cursor for next page"`
	Type    string `json:"type,omitempty" jsonschema:"Filter by type: 'global', 'personal'"`
	Status  string `json:"status,omitempty" jsonschema:"Filter by status: 'current', 'archived'"`
}

// ManageSpacesHandler handles list and get operations for spaces.
func ManageSpacesHandler(c *confluence.Client) func(context.Context, *mcp.CallToolRequest, ManageSpacesArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args ManageSpacesArgs) (*mcp.CallToolResult, any, error) {
		switch args.Action {
		case "list":
			result, err := c.ListSpaces(confluence.ListSpacesArgs{
				Limit:  args.Limit,
				Cursor: args.Cursor,
				Type:   args.Type,
				Status: args.Status,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to list spaces: %v", err)), nil, nil
			}
			var flat []confluence.FlattenedSpace
			for i := range result.Results {
				flat = append(flat, *confluence.FlattenSpace(&result.Results[i]))
			}
			return ToolResultText(confluence.SafeMarshal(flat)), nil, nil

		case "get":
			if args.SpaceID == "" {
				return ToolResultError("space_id is required for 'get' action"), nil, nil
			}
			space, err := c.GetSpace(confluence.GetSpaceArgs{SpaceID: args.SpaceID})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to get space: %v", err)), nil, nil
			}
			return ToolResultText(confluence.SafeMarshal(confluence.FlattenSpace(space))), nil, nil

		case "get_by_key":
			if args.Key == "" {
				return ToolResultError("key is required for 'get_by_key' action"), nil, nil
			}
			space, err := c.GetSpaceByKey(confluence.GetSpaceByKeyArgs{Key: args.Key})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to get space by key: %v", err)), nil, nil
			}
			return ToolResultText(confluence.SafeMarshal(confluence.FlattenSpace(space))), nil, nil

		default:
			return ToolResultError(fmt.Sprintf("unknown action: %s. Valid actions: list, get, get_by_key", args.Action)), nil, nil
		}
	}
}
