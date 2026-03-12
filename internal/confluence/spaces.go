package confluence

import (
	"encoding/json"
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

// CreateSpaceRequest is the payload for creating a Confluence space.
// Uses the stable V1 API (POST /wiki/rest/api/space) for reliability.
type CreateSpaceRequest struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// createSpaceV1Body is the V1 API body format (wraps description in an object).
type createSpaceV1Body struct {
	Key         string             `json:"key"`
	Name        string             `json:"name"`
	Description *createSpaceV1Desc `json:"description,omitempty"`
}

type createSpaceV1Desc struct {
	Plain *createSpaceV1DescValue `json:"plain,omitempty"`
}

type createSpaceV1DescValue struct {
	Value          string `json:"value"`
	Representation string `json:"representation"`
}

// createSpaceV1Response is the V1 API response for creating a space.
type createSpaceV1Response struct {
	ID       int    `json:"id"`
	Key      string `json:"key"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Status   string `json:"status,omitempty"`
	Homepage *struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	} `json:"homepage,omitempty"`
}

// CreateSpace creates a new Confluence space.
// Uses the V1 API (POST /wiki/rest/api/space) which is stable,
// since the V2 spaces API create endpoint is experimental.
func (c *Client) CreateSpace(req CreateSpaceRequest) (*Space, error) {
	if req.Key == "" {
		return nil, fmt.Errorf("key is required")
	}
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}

	body := createSpaceV1Body{
		Key:  req.Key,
		Name: req.Name,
	}
	if req.Description != "" {
		body.Description = &createSpaceV1Desc{
			Plain: &createSpaceV1DescValue{
				Value:          req.Description,
				Representation: "plain",
			},
		}
	}

	data, err := c.Post("/wiki/rest/api/space", body)
	if err != nil {
		return nil, err
	}

	var v1resp createSpaceV1Response
	if err := json.Unmarshal(data, &v1resp); err != nil {
		return nil, fmt.Errorf("unmarshaling created space: %w", err)
	}

	// Convert V1 response to our V2-style Space type for consistency
	space := &Space{
		ID:     fmt.Sprintf("%d", v1resp.ID),
		Key:    v1resp.Key,
		Name:   v1resp.Name,
		Type:   v1resp.Type,
		Status: v1resp.Status,
	}
	if v1resp.Homepage != nil {
		space.HomepageID = v1resp.Homepage.ID
	}

	return space, nil
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
