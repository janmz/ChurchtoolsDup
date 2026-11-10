package churchtools

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// ListAllPersons loads the complete person list without campus filter.
func (c *Client) ListAllPersons() ([]Person, error) {
	query := url.Values{}
	query.Set("limit", strconv.Itoa(defaultPageSize))

	items, err := c.fetchAPIPages("/persons", query)
	if err != nil {
		return nil, err
	}
	return decodePersonList(items)
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

// intFromAny converts a decoded JSON number (or numeric string) to int.
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
	case string:
		n, err := strconv.Atoi(v)
		if err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}
