package confluence

import (
	"fmt"
	"net/url"
)

// ListSpacesArgs are the parameters for listing spaces.
type ListSpacesArgs struct {
	Limit  int    `json:"limit,omitempty"`
	Cursor string `json:"cursor,omitempty"`
	Type   string `json:"type,omitempty"`   // e.g., "global", "personal"
	Status string `json:"status,omitempty"` // e.g., "current", "archived"
}

// ListSpaces returns all spaces the authenticated user has access to.
func (c *Client) ListSpaces(args ListSpacesArgs) (*PagedResult[Space], error) {
	limit := args.Limit
	if limit == 0 {
		limit = 25
	}

	path := fmt.Sprintf("/wiki/api/v2/spaces?limit=%d", limit)
	if args.Type != "" {
		path += "&type=" + url.QueryEscape(args.Type)
	}
	if args.Status != "" {
		path += "&status=" + url.QueryEscape(args.Status)
	}
	if args.Cursor != "" {
		path += "&cursor=" + url.QueryEscape(args.Cursor)
	}

	return GetPaged[Space](c, path)
}

// GetSpaceArgs are the parameters for getting a single space.
type GetSpaceArgs struct {
	SpaceID string `json:"space_id"`
}

// GetSpace returns details for a single space by ID.
func (c *Client) GetSpace(args GetSpaceArgs) (*Space, error) {
	if args.SpaceID == "" {
		return nil, fmt.Errorf("space_id is required")
	}
	return GetJSON[Space](c, fmt.Sprintf("/wiki/api/v2/spaces/%s", url.PathEscape(args.SpaceID)))
}

// GetSpaceByKeyArgs are the parameters for getting a space by key.
type GetSpaceByKeyArgs struct {
	Key string `json:"key"`
}

// GetSpaceByKey returns a space matching the given key.
func (c *Client) GetSpaceByKey(args GetSpaceByKeyArgs) (*Space, error) {
	if args.Key == "" {
		return nil, fmt.Errorf("key is required")
	}

	result, err := GetPaged[Space](c, fmt.Sprintf("/wiki/api/v2/spaces?keys=%s&limit=1", url.QueryEscape(args.Key)))
	if err != nil {
		return nil, err
	}

	if len(result.Results) == 0 {
		return nil, fmt.Errorf("space with key %q not found", args.Key)
	}

	return &result.Results[0], nil
}
