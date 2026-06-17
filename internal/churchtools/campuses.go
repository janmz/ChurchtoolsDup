package churchtools

import (
	"encoding/json"
	"fmt"
)

// Campus is a ChurchTools standort.
type Campus struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// ListCampuses returns all campuses the user may access.
func (c *Client) ListCampuses() ([]Campus, error) {
	items, err := c.fetchAPIList("/campuses", nil)
	if err != nil {
		return nil, fmt.Errorf("standorte laden: %w", err)
	}

	campuses := make([]Campus, 0, len(items))
	for _, item := range items {
		var campus Campus
		if err := json.Unmarshal(item, &campus); err != nil {
			return nil, fmt.Errorf("standort parsen: %w", err)
		}
		if campus.ID <= 0 {
			continue
		}
		campuses = append(campuses, campus)
	}
	return campuses, nil
}
