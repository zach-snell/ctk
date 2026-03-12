package confluence

import (
	"fmt"
	"net/url"
)

// CQLSearchArgs are the parameters for CQL search (V1 API).
type CQLSearchArgs struct {
	CQL        string `json:"cql"`
	Limit      int    `json:"limit,omitempty"`
	Start      int    `json:"start,omitempty"`
	IncludeArc bool   `json:"include_archived_spaces,omitempty"`
}

// CQLSearch performs a CQL search using the V1 API.
func (c *Client) CQLSearch(args CQLSearchArgs) (*SearchResult, error) {
	if args.CQL == "" {
		return nil, fmt.Errorf("cql is required")
	}

	limit := args.Limit
	if limit == 0 {
		limit = 25
	}

	path := fmt.Sprintf("/wiki/rest/api/search?cql=%s&limit=%d&start=%d",
		url.QueryEscape(args.CQL),
		limit,
		args.Start,
	)
	if args.IncludeArc {
		path += "&includeArchivedSpaces=true"
	}

	return GetJSON[SearchResult](c, path)
}

// QuickSearchArgs are the parameters for quick text search.
type QuickSearchArgs struct {
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty"`
}

// QuickSearch performs a quick text search by wrapping the query in a CQL siteSearch clause.
func (c *Client) QuickSearch(args QuickSearchArgs) (*SearchResult, error) {
	if args.Query == "" {
		return nil, fmt.Errorf("query is required")
	}

	limit := args.Limit
	if limit == 0 {
		limit = 25
	}

	// Use CQL's siteSearch for quick text search
	cql := fmt.Sprintf("siteSearch ~ %q", args.Query)

	return c.CQLSearch(CQLSearchArgs{
		CQL:   cql,
		Limit: limit,
	})
}
