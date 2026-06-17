package churchtools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

var htmlTagPattern = regexp.MustCompile(`<[^>]*>`)

// FindGroupByNames returns the first matching group from the candidate names.
func (c *Client) FindGroupByNames(names []string) (Group, string, error) {
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		group, err := c.FindGroupByName(name)
		if err == nil {
			return group, name, nil
		}
	}
	return Group{}, "", fmt.Errorf("keine passende Gruppe gefunden")
}

// FindGroupByName returns a group whose name matches the requested permission group.
func (c *Client) FindGroupByName(name string) (Group, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Group{}, fmt.Errorf("Gruppenname fehlt")
	}

	groups, err := c.ListGroups(GroupListOptions{Query: name})
	if err != nil {
		return Group{}, err
	}

	var matches []Group
	for _, group := range groups {
		plain := plainGroupName(group.Name)
		if plain == name || strings.Contains(plain, name) {
			matches = append(matches, group)
		}
	}

	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return Group{}, fmt.Errorf("Gruppe %q nicht gefunden", name)
	default:
		return Group{}, fmt.Errorf("mehrere Gruppen passen zu %q", name)
	}
}

// RequestGroupMembership adds the person to a group or requests membership.
func (c *Client) RequestGroupMembership(groupID, personID int) (MembershipRequestResult, error) {
	path := fmt.Sprintf("/groups/%d/members/%d", groupID, personID)
	statusCode, body, err := c.putAPI(path, map[string]any{}, true)
	if err != nil {
		return MembershipRequestResult{}, err
	}

	switch statusCode {
	case http.StatusOK, http.StatusCreated, http.StatusNoContent:
		if waitingListFromBody(body) {
			return MembershipRequestResult{
				Status:  MembershipRequested,
				Message: "auf Warteliste gesetzt",
			}, nil
		}
		return MembershipRequestResult{Status: MembershipActive}, nil
	case http.StatusAccepted:
		return MembershipRequestResult{
			Status:  MembershipRequested,
			Message: "Mitgliedschaft beantragt",
		}, nil
	case http.StatusForbidden:
		return MembershipRequestResult{
			Status:  MembershipDenied,
			Message: "keine Berechtigung für Gruppenanmeldung oder Gruppe nicht offen",
		}, nil
	default:
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = fmt.Sprintf("http %d", statusCode)
		}
		return MembershipRequestResult{Status: MembershipDenied, Message: msg}, nil
	}
}

func plainGroupName(name string) string {
	plain := htmlTagPattern.ReplaceAllString(name, "")
	return strings.Join(strings.Fields(strings.TrimSpace(plain)), " ")
}

func waitingListFromBody(body []byte) bool {
	if len(body) == 0 {
		return false
	}
	var envelope struct {
		Data struct {
			WaitinglistPos *int `json:"waitinglistPos"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return false
	}
	return envelope.Data.WaitinglistPos != nil && *envelope.Data.WaitinglistPos > 0
}

func (c *Client) putAPI(path string, payload any, allowRelogin bool) (int, []byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, nil, err
	}

	req, err := http.NewRequest(http.MethodPut, c.apiURL(path), bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.applyAuth(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("api PUT %s: %w", path, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusUnauthorized {
		if err := c.reloginOnce(allowRelogin); err != nil {
			return resp.StatusCode, respBody, err
		}
		return c.putAPI(path, payload, false)
	}
	return resp.StatusCode, respBody, nil
}
