package churchtools

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// ListPersonGroups returns groups the person belongs to.
func (c *Client) ListPersonGroups(personID int) ([]Group, error) {
	if personID <= 0 {
		return nil, fmt.Errorf("personen-id fehlt")
	}

	path := "/persons/" + strconv.Itoa(personID) + "/groups"
	items, err := c.fetchAPIPages(path, nil)
	if err != nil {
		return nil, fmt.Errorf("Personengruppen laden: %w", err)
	}

	groups := make([]Group, 0, len(items))
	seen := make(map[int]struct{}, len(items))
	for _, item := range items {
		group, ok := decodePersonGroup(item)
		if !ok || group.ID <= 0 {
			continue
		}
		if _, exists := seen[group.ID]; exists {
			continue
		}
		seen[group.ID] = struct{}{}
		groups = append(groups, group)
	}
	return groups, nil
}

func decodePersonGroup(item json.RawMessage) (Group, bool) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(item, &fields); err != nil {
		return Group{}, false
	}

	if raw, ok := fields["group"]; ok {
		if group, ok := decodeGroupReference(raw); ok {
			return group, true
		}
	}

	return decodeGroupReference(item)
}

func decodeGroupReference(raw json.RawMessage) (Group, bool) {
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return Group{}, false
	}

	id, ok := groupIDFromObject(obj)
	if !ok || id <= 0 {
		return Group{}, false
	}

	name := groupNameFromObject(obj)
	if name == "" {
		return Group{}, false
	}

	return Group{ID: id, Name: name}, true
}

func groupIDFromObject(obj map[string]any) (int, bool) {
	if id, ok := intFromAny(obj["id"]); ok && id > 0 {
		return id, true
	}
	if id, ok := intFromAny(obj["groupId"]); ok && id > 0 {
		return id, true
	}
	if apiURL, ok := obj["apiUrl"].(string); ok {
		if id, ok := groupIDFromAPIURL(apiURL); ok {
			return id, true
		}
	}
	if frontendURL, ok := obj["frontendUrl"].(string); ok {
		if id, ok := groupIDFromAPIURL(frontendURL); ok {
			return id, true
		}
	}
	return 0, false
}

func groupNameFromObject(obj map[string]any) string {
	for _, key := range []string{"name", "title"} {
		if value, ok := obj[key].(string); ok {
			if name := strings.TrimSpace(value); name != "" {
				return name
			}
		}
	}

	if information, ok := obj["information"].(map[string]any); ok {
		if value, ok := information["name"].(string); ok {
			if name := strings.TrimSpace(value); name != "" {
				return name
			}
		}
	}
	return ""
}

func groupIDFromAPIURL(apiURL string) (int, bool) {
	apiURL = strings.TrimSpace(apiURL)
	if apiURL == "" {
		return 0, false
	}

	apiURL = strings.TrimSuffix(apiURL, "/")
	parts := strings.Split(apiURL, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		part := strings.TrimSpace(parts[i])
		if part == "" || part == "groups" {
			continue
		}
		id, err := strconv.Atoi(part)
		if err == nil && id > 0 {
			return id, true
		}
	}
	return 0, false
}

// PlainGroupName strips HTML markup from a ChurchTools group name.
func PlainGroupName(name string) string {
	return plainGroupName(name)
}

// PersonIsInGroup reports whether a person belongs to a group by name.
func (c *Client) PersonIsInGroup(personID int, groupName string) (bool, error) {
	groupName = strings.TrimSpace(groupName)
	if personID <= 0 || groupName == "" {
		return false, fmt.Errorf("Personen-ID oder Gruppenname fehlt")
	}

	groups, err := c.ListPersonGroups(personID)
	if err != nil {
		return false, err
	}

	want := strings.ToLower(groupName)
	for _, group := range groups {
		plain := strings.ToLower(PlainGroupName(group.Name))
		if plain == want || strings.Contains(plain, want) {
			return true, nil
		}
	}
	return false, nil
}
