package confluence

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// ListLabelsArgs are the parameters for listing page labels.
type ListLabelsArgs struct {
	PageID string `json:"page_id"`
	Limit  int    `json:"limit,omitempty"`
	Cursor string `json:"cursor,omitempty"`
}

// ListLabels returns labels for a page.
func (c *Client) ListLabels(args ListLabelsArgs) (*PagedResult[Label], error) {
	if args.PageID == "" {
		return nil, fmt.Errorf("page_id is required")
	}

	limit := args.Limit
	if limit == 0 {
		limit = 25
	}

	path := fmt.Sprintf("/wiki/api/v2/pages/%s/labels?limit=%d",
		url.PathEscape(args.PageID), limit)

	if args.Cursor != "" {
		path += "&cursor=" + url.QueryEscape(args.Cursor)
	}

	return GetPaged[Label](c, path)
}

// AddLabelArgs are the parameters for adding a label to a page.
type AddLabelArgs struct {
	PageID string `json:"page_id"`
	Name   string `json:"name"`
}

// AddLabel adds a label to a page.
func (c *Client) AddLabel(args AddLabelArgs) (*Label, error) {
	if args.PageID == "" {
		return nil, fmt.Errorf("page_id is required")
	}
	if args.Name == "" {
		return nil, fmt.Errorf("name is required")
	}

	// V1 API for adding labels (V2 doesn't have a direct add label endpoint yet on some instances)
	body := AddLabelsRequest{
		{Prefix: "global", Name: args.Name},
	}

	data, err := c.Post(
		fmt.Sprintf("/wiki/rest/api/content/%s/label", url.PathEscape(args.PageID)),
		body,
	)
	if err != nil {
		return nil, err
	}

	// V1 returns a label list result
	var result struct {
		Results []Label `json:"results"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshaling label response: %w", err)
	}

	if len(result.Results) > 0 {
		return &result.Results[len(result.Results)-1], nil
	}

	return &Label{Name: args.Name, Prefix: "global"}, nil
}

// RemoveLabelArgs are the parameters for removing a label from a page.
type RemoveLabelArgs struct {
	PageID string `json:"page_id"`
	Label  string `json:"label"` // label name
}

// RemoveLabel removes a label from a page.
func (c *Client) RemoveLabel(args RemoveLabelArgs) error {
	if args.PageID == "" {
		return fmt.Errorf("page_id is required")
	}
	if args.Label == "" {
		return fmt.Errorf("label is required")
	}

	// V1 API for removing labels
	return c.Delete(fmt.Sprintf("/wiki/rest/api/content/%s/label/%s",
		url.PathEscape(args.PageID),
		url.PathEscape(args.Label),
	))
}
