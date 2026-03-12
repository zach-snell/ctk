package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zach-snell/ctk/internal/confluence"
)

type ManagePagesArgs struct {
	Action         string `json:"action" jsonschema:"Action to perform: 'get', 'get_by_title', 'list', 'create', 'update', 'delete', 'get_children', 'get_ancestors', 'list_versions', 'move', 'diff'" jsonschema_enum:"get,get_by_title,list,create,update,delete,get_children,get_ancestors,list_versions,move,diff"`
	PageID         string `json:"page_id,omitempty" jsonschema:"Page ID (required for get, update, delete, get_children, get_ancestors, move, diff)"`
	SpaceID        string `json:"space_id,omitempty" jsonschema:"Space ID (required for list, create, get_by_title)"`
	Title          string `json:"title,omitempty" jsonschema:"Page title (required for create, get_by_title; optional for update)"`
	Body           string `json:"body,omitempty" jsonschema:"Page body in Confluence storage format (XHTML) (for create, update)"`
	ParentID       string `json:"parent_id,omitempty" jsonschema:"Parent page ID (for create)"`
	Version        int    `json:"version,omitempty" jsonschema:"Page version number (required for update, move — must be current version + 1)"`
	Status         string `json:"status,omitempty" jsonschema:"Page status: 'current', 'draft' (for create, update, list filter)"`
	BodyFormat     string `json:"body_format,omitempty" jsonschema:"Body format to return: 'storage', 'atlas_doc_format', 'view'"`
	Limit          int    `json:"limit,omitempty" jsonschema:"Number of results per page (default 25)"`
	Cursor         string `json:"cursor,omitempty" jsonschema:"Pagination cursor for next page"`
	Sort           string `json:"sort,omitempty" jsonschema:"Sort order for list (e.g., '-modified-date', 'title')"`
	TargetSpaceID  string `json:"target_space_id,omitempty" jsonschema:"Target space ID (for move)"`
	TargetParentID string `json:"target_parent_id,omitempty" jsonschema:"Target parent page ID (for move)"`
	FromVersion    int    `json:"from_version,omitempty" jsonschema:"Starting version number (required for diff)"`
	ToVersion      int    `json:"to_version,omitempty" jsonschema:"Ending version number (required for diff)"`
}

// ManagePagesHandler handles page operations, with optional write support.
func ManagePagesHandler(c *confluence.Client, canWrite bool) func(context.Context, *mcp.CallToolRequest, ManagePagesArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args ManagePagesArgs) (*mcp.CallToolResult, any, error) {
		switch args.Action {
		case "get":
			if args.PageID == "" {
				return ToolResultError("page_id is required for 'get' action"), nil, nil
			}
			page, err := c.GetPage(confluence.GetPageArgs{
				PageID:      args.PageID,
				BodyFormat:  args.BodyFormat,
				IncludeBody: true,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to get page: %v", err)), nil, nil
			}
			return ToolResultText(confluence.SafeMarshal(confluence.FlattenPage(page))), nil, nil

		case "get_by_title":
			if args.SpaceID == "" {
				return ToolResultError("space_id is required for 'get_by_title' action"), nil, nil
			}
			if args.Title == "" {
				return ToolResultError("title is required for 'get_by_title' action"), nil, nil
			}
			page, err := c.GetPageByTitle(confluence.GetPageByTitleArgs{
				SpaceID:    args.SpaceID,
				Title:      args.Title,
				BodyFormat: args.BodyFormat,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to get page by title: %v", err)), nil, nil
			}
			return ToolResultText(confluence.SafeMarshal(confluence.FlattenPage(page))), nil, nil

		case "list":
			if args.SpaceID == "" {
				return ToolResultError("space_id is required for 'list' action"), nil, nil
			}
			result, err := c.ListPages(confluence.ListPagesArgs{
				SpaceID:    args.SpaceID,
				Limit:      args.Limit,
				Cursor:     args.Cursor,
				Title:      args.Title,
				Status:     args.Status,
				BodyFormat: args.BodyFormat,
				Sort:       args.Sort,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to list pages: %v", err)), nil, nil
			}
			var flat []*confluence.FlattenedPage
			for i := range result.Results {
				flat = append(flat, confluence.FlattenPage(&result.Results[i]))
			}
			return ToolResultText(confluence.SafeMarshal(flat)), nil, nil

		case "get_children":
			if args.PageID == "" {
				return ToolResultError("page_id is required for 'get_children' action"), nil, nil
			}
			result, err := c.GetChildren(confluence.GetChildrenArgs{
				PageID: args.PageID,
				Limit:  args.Limit,
				Cursor: args.Cursor,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to get children: %v", err)), nil, nil
			}
			var flat []*confluence.FlattenedPage
			for i := range result.Results {
				flat = append(flat, confluence.FlattenPage(&result.Results[i]))
			}
			return ToolResultText(confluence.SafeMarshal(flat)), nil, nil

		case "get_ancestors":
			if args.PageID == "" {
				return ToolResultError("page_id is required for 'get_ancestors' action"), nil, nil
			}
			result, err := c.GetAncestors(confluence.GetAncestorsArgs{
				PageID: args.PageID,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to get ancestors: %v", err)), nil, nil
			}
			var flat []*confluence.FlattenedPage
			for i := range result.Results {
				flat = append(flat, confluence.FlattenPage(&result.Results[i]))
			}
			return ToolResultText(confluence.SafeMarshal(flat)), nil, nil

		case "create":
			if !canWrite {
				return ToolResultError("write operations are disabled. Set CTK_ENABLE_WRITES=true to enable."), nil, nil
			}
			if args.SpaceID == "" || args.Title == "" {
				return ToolResultError("space_id and title are required for 'create' action"), nil, nil
			}
			page, err := c.CreatePage(confluence.CreatePageArgs{
				SpaceID:  args.SpaceID,
				Title:    args.Title,
				Body:     args.Body,
				ParentID: args.ParentID,
				Status:   args.Status,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to create page: %v", err)), nil, nil
			}
			return ToolResultText(confluence.SafeMarshal(confluence.FlattenPage(page))), nil, nil

		case "update":
			if !canWrite {
				return ToolResultError("write operations are disabled. Set CTK_ENABLE_WRITES=true to enable."), nil, nil
			}
			if args.PageID == "" || args.Title == "" {
				return ToolResultError("page_id and title are required for 'update' action"), nil, nil
			}
			if args.Version == 0 {
				return ToolResultError("version is required for 'update' action (must be current version + 1)"), nil, nil
			}
			page, err := c.UpdatePage(confluence.UpdatePageArgs{
				PageID:  args.PageID,
				Title:   args.Title,
				Body:    args.Body,
				Version: args.Version,
				Status:  args.Status,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to update page: %v", err)), nil, nil
			}
			return ToolResultText(confluence.SafeMarshal(confluence.FlattenPage(page))), nil, nil

		case "delete":
			if !canWrite {
				return ToolResultError("write operations are disabled. Set CTK_ENABLE_WRITES=true to enable."), nil, nil
			}
			if args.PageID == "" {
				return ToolResultError("page_id is required for 'delete' action"), nil, nil
			}
			err := c.DeletePage(confluence.DeletePageArgs{
				PageID: args.PageID,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to delete page: %v", err)), nil, nil
			}
			return ToolResultText("Page deleted successfully"), nil, nil

		case "list_versions":
			if args.PageID == "" {
				return ToolResultError("page_id is required for 'list_versions' action"), nil, nil
			}
			result, err := c.ListPageVersions(args.PageID, args.Limit, args.Cursor)
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to list page versions: %v", err)), nil, nil
			}
			var flat []*confluence.FlattenedPageVersion
			for i := range result.Results {
				flat = append(flat, confluence.FlattenPageVersion(&result.Results[i]))
			}
			return ToolResultText(confluence.SafeMarshal(flat)), nil, nil

		case "move":
			if !canWrite {
				return ToolResultError("write operations are disabled. Set CTK_ENABLE_WRITES=true to enable."), nil, nil
			}
			if args.PageID == "" {
				return ToolResultError("page_id is required for 'move' action"), nil, nil
			}
			if args.Version == 0 {
				return ToolResultError("version is required for 'move' action (must be current version + 1)"), nil, nil
			}
			if args.TargetSpaceID == "" && args.TargetParentID == "" {
				return ToolResultError("at least one of target_space_id or target_parent_id is required for 'move' action"), nil, nil
			}
			page, err := c.MovePage(confluence.MovePageArgs{
				PageID:         args.PageID,
				TargetSpaceID:  args.TargetSpaceID,
				TargetParentID: args.TargetParentID,
				Version:        args.Version,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to move page: %v", err)), nil, nil
			}
			return ToolResultText(confluence.SafeMarshal(confluence.FlattenPage(page))), nil, nil

		case "diff":
			if args.PageID == "" {
				return ToolResultError("page_id is required for 'diff' action"), nil, nil
			}
			if args.FromVersion == 0 || args.ToVersion == 0 {
				return ToolResultError("from_version and to_version are required for 'diff' action"), nil, nil
			}
			diff, err := c.DiffPageVersions(args.PageID, args.FromVersion, args.ToVersion)
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to diff page versions: %v", err)), nil, nil
			}
			return ToolResultText(confluence.SafeMarshal(diff)), nil, nil

		default:
			return ToolResultError(fmt.Sprintf("unknown action: %s. Valid actions: get, get_by_title, list, create, update, delete, get_children, get_ancestors, list_versions, move, diff", args.Action)), nil, nil
		}
	}
}
