package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zach-snell/ctk/internal/confluence"
)

type ManageUsersArgs struct {
	Action string `json:"action" jsonschema:"Action to perform: 'get_current', 'search'" jsonschema_enum:"get_current,search"`
	Query  string `json:"query,omitempty" jsonschema:"Search query — display name, email, etc. (for 'search')"`
	Limit  int    `json:"limit,omitempty" jsonschema:"Maximum results to return (for 'search', default 20)"`
}

// ManageUsersHandler handles Confluence user operations.
func ManageUsersHandler(c *confluence.Client) func(context.Context, *mcp.CallToolRequest, ManageUsersArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args ManageUsersArgs) (*mcp.CallToolResult, any, error) {
		switch args.Action {
		case "get_current":
			user, err := c.GetCurrentUser()
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to get current user: %v", err)), nil, nil
			}
			return ToolResultText(confluence.SafeMarshal(user)), nil, nil

		case "search":
			if args.Query == "" {
				return ToolResultError("query is required for 'search' action"), nil, nil
			}
			// Use CQL-based search
			result, err := c.SearchUsers(confluence.SearchUsersArgs{
				Query: args.Query,
				Limit: args.Limit,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to search users: %v", err)), nil, nil
			}
			flat := confluence.FlattenSearchResults(result)
			return ToolResultText(confluence.SafeMarshal(struct {
				Results   []confluence.FlattenedSearchResult `json:"results"`
				TotalSize int                                `json:"total_size"`
			}{
				Results:   flat,
				TotalSize: result.TotalSize,
			})), nil, nil

		default:
			return ToolResultError(fmt.Sprintf("unknown action: %s. Valid actions: get_current, search", args.Action)), nil, nil
		}
	}
}
