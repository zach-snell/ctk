package confluence

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// SearchUsersArgs are the parameters for searching Confluence users.
type SearchUsersArgs struct {
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty"`
}

// UserSearchResult is the CQL search response when searching for users.
type UserSearchResult struct {
	Results   []SearchResultItem `json:"results"`
	Start     int                `json:"start"`
	Limit     int                `json:"limit"`
	Size      int                `json:"size"`
	TotalSize int                `json:"totalSize"`
}

// SearchUsers searches for Confluence users by display name or email.
// Uses CQL: type=user AND (user.fullname~"query" OR user.email~"query")
func (c *Client) SearchUsers(args SearchUsersArgs) (*SearchResult, error) {
	if args.Query == "" {
		return nil, fmt.Errorf("query is required")
	}

	limit := args.Limit
	if limit == 0 {
		limit = 20
	}

	// CQL for user search
	cql := fmt.Sprintf("type=user AND user.fullname~%q", args.Query)

	return c.CQLSearch(CQLSearchArgs{
		CQL:   cql,
		Limit: limit,
	})
}

// GetCurrentUser returns the currently authenticated user.
func (c *Client) GetCurrentUser() (*User, error) {
	path := "/wiki/rest/api/user/current"
	data, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	// V1 API returns a different shape
	type v1User struct {
		AccountID   string `json:"accountId"`
		DisplayName string `json:"displayName"`
		Email       string `json:"email"`
		Type        string `json:"type"`
		ProfilePic  struct {
			Path string `json:"path"`
		} `json:"profilePicture"`
	}

	var raw v1User
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	return &User{
		AccountID:   raw.AccountID,
		DisplayName: raw.DisplayName,
		Email:       raw.Email,
		Type:        raw.Type,
	}, nil
}

// SearchUsersByQuery uses the Atlassian user search API to find users by query.
func (c *Client) SearchUsersByQuery(query string, limit int) ([]User, error) {
	if limit == 0 {
		limit = 20
	}

	path := fmt.Sprintf("/wiki/rest/api/search/user?cql=%s&limit=%d",
		url.QueryEscape(fmt.Sprintf("user.fullname~%q", query)), limit)

	data, err := c.Get(path)
	if err != nil {
		// Fall back to simple search if CQL user search is not available
		return nil, fmt.Errorf("user search failed: %w", err)
	}

	type userSearchResponse struct {
		Results []struct {
			User User `json:"user"`
		} `json:"results"`
	}

	var resp userSearchResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}

	users := make([]User, 0, len(resp.Results))
	for _, r := range resp.Results {
		users = append(users, r.User)
	}
	return users, nil
}
