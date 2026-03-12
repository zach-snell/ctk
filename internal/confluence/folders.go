package confluence

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// GetFolder returns a single folder by ID.
func (c *Client) GetFolder(folderID string) (*Folder, error) {
	if folderID == "" {
		return nil, fmt.Errorf("folder_id is required")
	}

	return GetJSON[Folder](c, fmt.Sprintf("/wiki/api/v2/folders/%s", url.PathEscape(folderID)))
}

// ListFolders returns folders in a space.
func (c *Client) ListFolders(spaceID string, limit int, cursor string) (*PagedResult[Folder], error) {
	if spaceID == "" {
		return nil, fmt.Errorf("space_id is required")
	}

	if limit == 0 {
		limit = 25
	}

	path := fmt.Sprintf("/wiki/api/v2/spaces/%s/folders?limit=%d", url.PathEscape(spaceID), limit)
	if cursor != "" {
		path += "&cursor=" + url.QueryEscape(cursor)
	}

	return GetPaged[Folder](c, path)
}

// GetFolderChildren returns child folders of a given folder.
func (c *Client) GetFolderChildren(folderID string, limit int, cursor string) (*PagedResult[Folder], error) {
	if folderID == "" {
		return nil, fmt.Errorf("folder_id is required")
	}

	if limit == 0 {
		limit = 25
	}

	path := fmt.Sprintf("/wiki/api/v2/folders/%s/children?limit=%d", url.PathEscape(folderID), limit)
	if cursor != "" {
		path += "&cursor=" + url.QueryEscape(cursor)
	}

	return GetPaged[Folder](c, path)
}

// CreateFolderArgs are the parameters for creating a folder.
type CreateFolderArgs struct {
	SpaceID  string `json:"space_id"`
	Title    string `json:"title"`
	ParentID string `json:"parent_id,omitempty"`
}

// CreateFolder creates a new folder.
func (c *Client) CreateFolder(args CreateFolderArgs) (*Folder, error) {
	if args.SpaceID == "" {
		return nil, fmt.Errorf("space_id is required")
	}
	if args.Title == "" {
		return nil, fmt.Errorf("title is required")
	}

	req := CreateFolderRequest(args)

	data, err := c.Post("/wiki/api/v2/folders", req)
	if err != nil {
		return nil, err
	}

	var folder Folder
	if err := json.Unmarshal(data, &folder); err != nil {
		return nil, fmt.Errorf("unmarshaling created folder: %w", err)
	}

	return &folder, nil
}

// UpdateFolderArgs are the parameters for updating a folder.
type UpdateFolderArgs struct {
	FolderID string `json:"folder_id"`
	Title    string `json:"title"`
	Version  int    `json:"version"` // Must be current version + 1
}

// UpdateFolder updates an existing folder.
func (c *Client) UpdateFolder(args UpdateFolderArgs) (*Folder, error) {
	if args.FolderID == "" {
		return nil, fmt.Errorf("folder_id is required")
	}
	if args.Title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if args.Version == 0 {
		return nil, fmt.Errorf("version is required (must be current version + 1)")
	}

	req := UpdateFolderRequest{
		ID:    args.FolderID,
		Title: args.Title,
		Version: &PageVersion{
			Number: args.Version,
		},
	}

	data, err := c.Put(fmt.Sprintf("/wiki/api/v2/folders/%s", url.PathEscape(args.FolderID)), req)
	if err != nil {
		return nil, err
	}

	var folder Folder
	if err := json.Unmarshal(data, &folder); err != nil {
		return nil, fmt.Errorf("unmarshaling updated folder: %w", err)
	}

	return &folder, nil
}

// DeleteFolder deletes a folder by ID.
func (c *Client) DeleteFolder(folderID string) error {
	if folderID == "" {
		return fmt.Errorf("folder_id is required")
	}

	return c.Delete(fmt.Sprintf("/wiki/api/v2/folders/%s", url.PathEscape(folderID)))
}
