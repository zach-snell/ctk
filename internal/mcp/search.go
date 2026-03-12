package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zach-snell/ctk/internal/confluence"
)

type ManageSearchArgs struct {
	Action     string `json:"action" jsonschema:"Action to perform: 'cql', 'quick'" jsonschema_enum:"cql,quick"`
	CQL        string `json:"cql,omitempty" jsonschema:"CQL query string (required for 'cql' action). Common CQL patterns: 'type=page AND space=DEV AND title~\"architecture\"', 'text~\"search term\" AND type=page', 'label=\"my-label\" AND space=TEAM', 'ancestor=12345 AND type=page', 'creator=currentUser() ORDER BY lastModified DESC', 'lastModified >= \"2024-01-01\" AND type=page', 'type=blogpost AND space=ENG'"`
	Query      string `json:"query,omitempty" jsonschema:"Text to search for (required for 'quick' action)"`
	Limit      int    `json:"limit,omitempty" jsonschema:"Number of results (default 25)"`
	Start      int    `json:"start,omitempty" jsonschema:"Starting offset for pagination"`
	IncludeArc bool   `json:"include_archived_spaces,omitempty" jsonschema:"Include archived spaces in results"`
}

// ManageSearchHandler handles CQL and quick text search operations.
func ManageSearchHandler(c *confluence.Client) func(context.Context, *mcp.CallToolRequest, ManageSearchArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args ManageSearchArgs) (*mcp.CallToolResult, any, error) {
		switch args.Action {
		case "cql":
			if args.CQL == "" {
				return ToolResultError("cql is required for 'cql' action"), nil, nil
			}
			result, err := c.CQLSearch(confluence.CQLSearchArgs{
				CQL:        args.CQL,
				Limit:      args.Limit,
				Start:      args.Start,
				IncludeArc: args.IncludeArc,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to search: %v", err)), nil, nil
			}
			flat := confluence.FlattenSearchResults(result)
			return ToolResultText(confluence.SafeMarshal(struct {
				Results   []confluence.FlattenedSearchResult `json:"results"`
				TotalSize int                                `json:"total_size"`
				Start     int                                `json:"start"`
				Limit     int                                `json:"limit"`
			}{
				Results:   flat,
				TotalSize: result.TotalSize,
				Start:     result.Start,
				Limit:     result.Limit,
			})), nil, nil

		case "quick":
			if args.Query == "" {
				return ToolResultError("query is required for 'quick' action"), nil, nil
			}
			result, err := c.QuickSearch(confluence.QuickSearchArgs{
				Query: args.Query,
				Limit: args.Limit,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to search: %v", err)), nil, nil
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
			return ToolResultError(fmt.Sprintf("unknown action: %s. Valid actions: cql, quick", args.Action)), nil, nil
		}
	}
}
