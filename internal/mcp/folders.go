package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zach-snell/ctk/internal/confluence"
)

type ManageFoldersArgs struct {
	Action   string `json:"action" jsonschema:"Action to perform: 'list', 'get', 'get_children', 'create', 'update', 'delete'" jsonschema_enum:"list,get,get_children,create,update,delete"`
	FolderID string `json:"folder_id,omitempty" jsonschema:"Folder ID (required for get, get_children, update, delete)"`
	SpaceID  string `json:"space_id,omitempty" jsonschema:"Space ID (required for list, create)"`
	Title    string `json:"title,omitempty" jsonschema:"Folder title (required for create, update)"`
	ParentID string `json:"parent_id,omitempty" jsonschema:"Parent folder ID (optional for create)"`
	Version  int    `json:"version,omitempty" jsonschema:"Folder version number (required for update — must be current version + 1)"`
	Limit    int    `json:"limit,omitempty" jsonschema:"Number of results per page (default 25)"`
	Cursor   string `json:"cursor,omitempty" jsonschema:"Pagination cursor for next page"`
}

// ManageFoldersHandler handles folder operations, with optional write support.
func ManageFoldersHandler(c *confluence.Client, canWrite bool) func(context.Context, *mcp.CallToolRequest, ManageFoldersArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args ManageFoldersArgs) (*mcp.CallToolResult, any, error) {
		switch args.Action {
		case "list":
			if args.SpaceID == "" {
				return ToolResultError("space_id is required for 'list' action"), nil, nil
			}
			result, err := c.ListFolders(args.SpaceID, args.Limit, args.Cursor)
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to list folders: %v", err)), nil, nil
			}
			var flat []*confluence.FlattenedFolder
			for i := range result.Results {
				flat = append(flat, confluence.FlattenFolder(&result.Results[i]))
			}
			return ToolResultText(confluence.SafeMarshal(flat)), nil, nil

		case "get":
			if args.FolderID == "" {
				return ToolResultError("folder_id is required for 'get' action"), nil, nil
			}
			folder, err := c.GetFolder(args.FolderID)
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to get folder: %v", err)), nil, nil
			}
			return ToolResultText(confluence.SafeMarshal(confluence.FlattenFolder(folder))), nil, nil

		case "get_children":
			if args.FolderID == "" {
				return ToolResultError("folder_id is required for 'get_children' action"), nil, nil
			}
			result, err := c.GetFolderChildren(args.FolderID, args.Limit, args.Cursor)
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to get folder children: %v", err)), nil, nil
			}
			var flat []*confluence.FlattenedFolder
			for i := range result.Results {
				flat = append(flat, confluence.FlattenFolder(&result.Results[i]))
			}
			return ToolResultText(confluence.SafeMarshal(flat)), nil, nil

		case "create":
			if !canWrite {
				return ToolResultError("write operations are disabled. Set CTK_ENABLE_WRITES=true to enable."), nil, nil
			}
			if args.SpaceID == "" || args.Title == "" {
				return ToolResultError("space_id and title are required for 'create' action"), nil, nil
			}
			folder, err := c.CreateFolder(confluence.CreateFolderArgs{
				SpaceID:  args.SpaceID,
				Title:    args.Title,
				ParentID: args.ParentID,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to create folder: %v", err)), nil, nil
			}
			return ToolResultText(confluence.SafeMarshal(confluence.FlattenFolder(folder))), nil, nil

		case "update":
			if !canWrite {
				return ToolResultError("write operations are disabled. Set CTK_ENABLE_WRITES=true to enable."), nil, nil
			}
			if args.FolderID == "" || args.Title == "" {
				return ToolResultError("folder_id and title are required for 'update' action"), nil, nil
			}
			if args.Version == 0 {
				return ToolResultError("version is required for 'update' action (must be current version + 1)"), nil, nil
			}
			folder, err := c.UpdateFolder(confluence.UpdateFolderArgs{
				FolderID: args.FolderID,
				Title:    args.Title,
				Version:  args.Version,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to update folder: %v", err)), nil, nil
			}
			return ToolResultText(confluence.SafeMarshal(confluence.FlattenFolder(folder))), nil, nil

		case "delete":
			if !canWrite {
				return ToolResultError("write operations are disabled. Set CTK_ENABLE_WRITES=true to enable."), nil, nil
			}
			if args.FolderID == "" {
				return ToolResultError("folder_id is required for 'delete' action"), nil, nil
			}
			err := c.DeleteFolder(args.FolderID)
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to delete folder: %v", err)), nil, nil
			}
			return ToolResultText("Folder deleted successfully"), nil, nil

		default:
			return ToolResultError(fmt.Sprintf("unknown action: %s. Valid actions: list, get, get_children, create, update, delete", args.Action)), nil, nil
		}
	}
}
