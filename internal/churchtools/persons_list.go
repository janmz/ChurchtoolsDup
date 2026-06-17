package churchtools

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

const defaultPersonPageSize = 100

// PersonListOptions filters persons loaded from ChurchTools.
type PersonListOptions struct {
	IDs      []int
	GroupID  int
	CampusID int
	StatusID int
}

// ListPersons returns persons for export or batch processing.
func (c *Client) ListPersons(opts PersonListOptions) ([]Person, error) {
	if opts.GroupID > 0 {
		ids, err := c.groupMemberPersonIDs(opts.GroupID)
		if err != nil {
			return nil, err
		}
		if len(ids) == 0 {
			return nil, nil
		}
		opts.IDs = ids
	}

	if len(opts.IDs) > 0 {
		return c.listPersonsByIDs(opts)
	}

	return c.listAllPersons(opts)
}

func (c *Client) buildPersonQuery(opts PersonListOptions, limit int) url.Values {
	query := url.Values{}
	query.Set("limit", strconv.Itoa(limit))
	if opts.CampusID > 0 {
		query.Add("campus_ids[]", strconv.Itoa(opts.CampusID))
	}
	if opts.StatusID > 0 {
		query.Add("status_ids[]", strconv.Itoa(opts.StatusID))
	}
	return query
}

func (c *Client) listAllPersons(opts PersonListOptions) ([]Person, error) {
	query := c.buildPersonQuery(opts, defaultPersonPageSize)

	items, err := c.fetchAPIPages("/persons", query)
	if err != nil {
		return nil, err
	}
	return decodePersonList(items)
}

func (c *Client) listPersonsByIDs(opts PersonListOptions) ([]Person, error) {
	const chunkSize = 50
	persons := make([]Person, 0, len(opts.IDs))

	for start := 0; start < len(opts.IDs); start += chunkSize {
		end := start + chunkSize
		if end > len(opts.IDs) {
			end = len(opts.IDs)
		}
		chunk := opts.IDs[start:end]

		query := c.buildPersonQuery(opts, len(chunk))
		for _, id := range chunk {
			query.Add("ids[]", strconv.Itoa(id))
		}

		items, err := c.fetchAPIPages("/persons", query)
		if err != nil {
			return nil, err
		}

		chunkPersons, err := decodePersonList(items)
		if err != nil {
			return nil, err
		}
		persons = append(persons, chunkPersons...)
	}

	return sortPersonsByIDs(persons, opts.IDs), nil
}

func (c *Client) groupMemberPersonIDs(groupID int) ([]int, error) {
	path := "/groups/" + strconv.Itoa(groupID) + "/members"
	items, err := c.fetchAPIPages(path, nil)
	if err != nil {
		return nil, fmt.Errorf("gruppenmitglieder laden: %w", err)
	}

	seen := make(map[int]struct{}, len(items))
	ids := make([]int, 0, len(items))
	for _, item := range items {
		id, ok := personIDFromMember(item)
		if !ok || id <= 0 {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids, nil
}

func decodePersonList(items []json.RawMessage) ([]Person, error) {
	persons := make([]Person, 0, len(items))
	for _, item := range items {
		person, err := decodePerson(item)
		if err != nil {
			return nil, fmt.Errorf("person parsen: %w", err)
		}
		if person.ID <= 0 {
			continue
		}
		persons = append(persons, person)
	}
	return persons, nil
}

func personIDFromMember(raw json.RawMessage) (int, bool) {
	var member map[string]any
	if err := json.Unmarshal(raw, &member); err != nil {
		return 0, false
	}

	if id, ok := intFromAny(member["personId"]); ok {
		return id, true
	}
	if person, ok := member["person"].(map[string]any); ok {
		if id, ok := intFromAny(person["id"]); ok {
			return id, true
		}
	}
	return 0, false
}

func intFromAny(value any) (int, bool) {
	switch v := value.(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	case json.Number:
		n, err := v.Int64()
		if err != nil {
			return 0, false
		}
		return int(n), true
	default:
		return 0, false
	}
}

func sortPersonsByIDs(persons []Person, ids []int) []Person {
	byID := make(map[int]Person, len(persons))
	for _, person := range persons {
		byID[person.ID] = person
	}

	ordered := make([]Person, 0, len(ids))
	for _, id := range ids {
		if person, ok := byID[id]; ok {
			ordered = append(ordered, person)
		}
	}
	return ordered
}
