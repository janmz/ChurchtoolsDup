package churchtools

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// Group is a ChurchTools group summary.
type Group struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// GroupListOptions filters groups loaded from ChurchTools.
type GroupListOptions struct {
	CampusID int
	Query    string
}

// ListGroups returns groups optionally filtered by campus.
func (c *Client) ListGroups(opts GroupListOptions) ([]Group, error) {
	query := url.Values{}
	query.Set("limit", strconv.Itoa(defaultPersonPageSize))
	if opts.CampusID > 0 {
		query.Add("campus_ids[]", strconv.Itoa(opts.CampusID))
	}
	if opts.Query != "" {
		query.Set("query", opts.Query)
	}

	items, err := c.fetchAPIPages("/groups", query)
	if err != nil {
		return nil, fmt.Errorf("Gruppen laden: %w", err)
	}

	groups := make([]Group, 0, len(items))
	for _, item := range items {
		var group Group
		if err := json.Unmarshal(item, &group); err != nil {
			return nil, fmt.Errorf("Gruppe parsen: %w", err)
		}
		if group.ID <= 0 {
			continue
		}
		groups = append(groups, group)
	}
	return groups, nil
}
