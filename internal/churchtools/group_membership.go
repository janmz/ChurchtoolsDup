package churchtools

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

var htmlTagPattern = regexp.MustCompile(`<[^>]*>`)

// numberBadgePattern matches markup that only holds a number, such as the
// sort badge in "<span>1</span> Personen bearbeiten".
var numberBadgePattern = regexp.MustCompile(`<([a-zA-Z]+)[^>]*>\s*[0-9]+\s*</([a-zA-Z]+)>`)

// ErrGroupNotFound is returned when ListGroups has no matching group name.
var ErrGroupNotFound = errors.New("Gruppe nicht gefunden")

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

	matches := matchGroupsByName(groups, name)
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return Group{}, fmt.Errorf("%w: %q", ErrGroupNotFound, name)
	default:
		names := make([]string, len(matches))
		for i, group := range matches {
			names[i] = fmt.Sprintf("%q (ID %d)", plainGroupName(group.Name), group.ID)
		}
		return Group{}, fmt.Errorf("mehrere Gruppen passen zu %q: %s", name, strings.Join(names, ", "))
	}
}

// groupNameKey normalises a group name for comparison: number badges and
// HTML stripped, whitespace collapsed, case-insensitive.
func groupNameKey(name string) string {
	return strings.ToLower(plainGroupName(numberBadgePattern.ReplaceAllString(name, "")))
}

// matchGroupsByName returns the groups whose name equals name
// (case-insensitive, HTML stripped). Substrings never match: the tool joins
// these groups to gain rights, and a group such as "Kurs ChurchTools Admin"
// created by anyone must not be mistaken for "ChurchTools Admin".
func matchGroupsByName(groups []Group, name string) []Group {
	want := groupNameKey(name)
	var exact []Group
	for _, group := range groups {
		if groupNameKey(group.Name) == want {
			exact = append(exact, group)
		}
	}
	return exact
}

// PublicSignupConfirm decides whether personID may be signed up for groupID
// through the public group signup form (used when a direct membership request
// is forbidden). Without a confirm function no public signup is attempted.
type PublicSignupConfirm func(groupID, personID int) bool

// SetPublicSignupConfirm sets the confirmation used before a public signup.
func (c *Client) SetPublicSignupConfirm(confirm PublicSignupConfirm) {
	c.confirmPublicSignup = confirm
}

// groupMembershipStatus returns the status of an existing membership
// ("active", "requested", "waiting", …) or "" when none is visible.
func (c *Client) groupMembershipStatus(groupID, personID int) (string, bool, error) {
	path := fmt.Sprintf("/groups/%d/members/%d", groupID, personID)
	status, body, err := c.callAPI(http.MethodGet, path, nil, nil)
	if err != nil {
		return "", false, err
	}
	switch status {
	case http.StatusOK:
		var envelope struct {
			Data struct {
				GroupMemberStatus string `json:"groupMemberStatus"`
			} `json:"data"`
		}
		_ = json.Unmarshal(body, &envelope)
		return strings.ToLower(strings.TrimSpace(envelope.Data.GroupMemberStatus)), true, nil
	case http.StatusNotFound:
		return "", true, nil
	default:
		// e.g. 403: membership not visible to us – caller decides.
		return "", false, nil
	}
}

// RequestGroupMembership adds the person to a group or requests membership.
// An existing membership is left untouched (a PUT would clear its comment and
// end date), and the PUT uses only_add so it never updates one.
//
// Memberships newly created for the logged-in user are remembered so that
// LeaveJoinedGroups can undo them after the run.
func (c *Client) RequestGroupMembership(groupID, personID int) (MembershipRequestResult, error) {
	result, err := c.requestGroupMembership(groupID, personID)
	if err == nil && !result.AlreadyMember && personID == c.personID &&
		(result.Status == MembershipActive || result.Status == MembershipRequested) {
		c.joinedGroups = append(c.joinedGroups, groupID)
	}
	return result, err
}

// LeaveResult reports leaving one temporarily joined group.
type LeaveResult struct {
	GroupID int
	Err     error
}

// LeaveJoinedGroups removes the logged-in user from all groups this client
// joined (in reverse order, so rights granted by an earlier group are still
// available while leaving a later one). Groups the user belonged to before
// are never touched.
func (c *Client) LeaveJoinedGroups() []LeaveResult {
	results := make([]LeaveResult, 0, len(c.joinedGroups))
	for i := len(c.joinedGroups) - 1; i >= 0; i-- {
		groupID := c.joinedGroups[i]
		path := fmt.Sprintf("/groups/%d/members/%d", groupID, c.personID)
		status, body, err := c.callAPI(http.MethodDelete, path, nil, nil)
		if err == nil && status != http.StatusOK && status != http.StatusNoContent && status != http.StatusNotFound {
			err = newAPIError(status, apiBodyMessage(body, "Austritt fehlgeschlagen"), body)
		}
		results = append(results, LeaveResult{GroupID: groupID, Err: err})
	}
	c.joinedGroups = nil
	return results
}

func (c *Client) requestGroupMembership(groupID, personID int) (MembershipRequestResult, error) {
	existing, known, err := c.groupMembershipStatus(groupID, personID)
	if err != nil {
		return MembershipRequestResult{}, err
	}
	if known && existing != "" {
		switch existing {
		case "requested", "waiting":
			return MembershipRequestResult{Status: MembershipRequested, Message: "Mitgliedschaft bereits beantragt", AlreadyMember: true}, nil
		case "to_delete":
			return MembershipRequestResult{Status: MembershipDenied, Message: "Mitgliedschaft ist zum Löschen markiert", AlreadyMember: true}, nil
		default:
			return MembershipRequestResult{Status: MembershipActive, Message: "Bereits Mitglied", AlreadyMember: true}, nil
		}
	}

	path := fmt.Sprintf("/groups/%d/members/%d", groupID, personID)
	statusCode, body, err := c.putAPI(path, map[string]any{"only_add": true})
	if err != nil {
		return MembershipRequestResult{}, err
	}

	switch statusCode {
	case http.StatusOK, http.StatusCreated, http.StatusNoContent:
		if waitingListFromBody(body) {
			return MembershipRequestResult{
				Status:  MembershipRequested,
				Message: "Auf Warteliste gesetzt",
			}, nil
		}
		return MembershipRequestResult{Status: MembershipActive}, nil
	case http.StatusAccepted:
		return MembershipRequestResult{
			Status:  MembershipRequested,
			Message: "Mitgliedschaft beantragt",
		}, nil
	case http.StatusForbidden:
		if c.confirmPublicSignup == nil || !c.confirmPublicSignup(groupID, personID) {
			return MembershipRequestResult{
				Status:  MembershipDenied,
				Message: apiBodyMessage(body, "Beitritt nicht erlaubt (öffentliche Anmeldung nicht bestätigt)"),
			}, nil
		}
		return c.requestGroupMembershipViaPublicSignup(groupID, personID)
	default:
		return MembershipRequestResult{
			Status:  MembershipDenied,
			Message: apiBodyMessage(body, fmt.Sprintf("http %d", statusCode)),
		}, nil
	}
}

// plainGroupName removes HTML tags for display (number badges keep their
// digits, e.g. "1 Personen bearbeiten").
func plainGroupName(name string) string {
	plain := htmlTagPattern.ReplaceAllString(name, "")
	return strings.Join(strings.Fields(strings.TrimSpace(plain)), " ")
}

func (c *Client) requestGroupMembershipViaPublicSignup(groupID, personID int) (MembershipRequestResult, error) {
	formPath := fmt.Sprintf("/publicgroups/%d/form", groupID)
	formBody, err := c.getAPI(formPath, nil)
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) {
			return MembershipRequestResult{
				Status:  MembershipDenied,
				Message: publicSignupDeniedMessage(apiErr),
			}, nil
		}
		return MembershipRequestResult{}, err
	}

	var formEnvelope struct {
		Data struct {
			Token string `json:"token"`
			Group struct {
				AutoAccept bool `json:"autoAccept"`
			} `json:"group"`
		} `json:"data"`
	}
	if err := json.Unmarshal(formBody, &formEnvelope); err != nil {
		return MembershipRequestResult{}, fmt.Errorf("publicgroups form parsen: %w", err)
	}

	token := strings.TrimSpace(formEnvelope.Data.Token)
	if token == "" {
		return MembershipRequestResult{
			Status:  MembershipDenied,
			Message: "Öffentliche Gruppenanmeldung: kein Token",
		}, nil
	}

	tokenPath := fmt.Sprintf("/publicgroups/%d/token", groupID)
	tokenStatus, tokenBody, err := c.postAPI(tokenPath, map[string]any{
		"personId": personID,
		"clicked":  []int{personID},
	})
	if err != nil {
		return MembershipRequestResult{}, err
	}
	if tokenStatus < http.StatusOK || tokenStatus >= http.StatusMultipleChoices {
		return MembershipRequestResult{
			Status:  MembershipDenied,
			Message: apiBodyMessage(tokenBody, "Person für Gruppenanmeldung nicht auswählbar"),
		}, nil
	}
	if updated := tokenFromPublicGroupBody(tokenBody); updated != "" {
		token = updated
	}

	signupPath := fmt.Sprintf("/publicgroups/%d/signup", groupID)
	signupStatus, signupBody, err := c.postAPI(signupPath, map[string]any{
		"token": token,
		"forms": []map[string]any{
			{"personId": personID, "form": []any{}},
		},
	})
	if err != nil {
		return MembershipRequestResult{}, err
	}

	return membershipResultFromPublicSignup(signupStatus, signupBody, formEnvelope.Data.Group.AutoAccept), nil
}

func membershipResultFromPublicSignup(statusCode int, body []byte, autoAccept bool) MembershipRequestResult {
	switch statusCode {
	case http.StatusOK, http.StatusCreated, http.StatusNoContent:
		if waitingListFromBody(body) {
			return MembershipRequestResult{
				Status:  MembershipRequested,
				Message: "Auf Warteliste gesetzt",
			}
		}
		if autoAccept {
			return MembershipRequestResult{
				Status:  MembershipActive,
				Message: "Mitgliedschaft angenommen (Gruppenanmeldung)",
			}
		}
		return MembershipRequestResult{
			Status:  MembershipRequested,
			Message: "Mitgliedschaft beantragt",
		}
	case http.StatusAccepted:
		return MembershipRequestResult{
			Status:  MembershipRequested,
			Message: "Mitgliedschaft beantragt",
		}
	default:
		return MembershipRequestResult{
			Status:  MembershipDenied,
			Message: apiBodyMessage(body, "Öffentliche Gruppenanmeldung fehlgeschlagen"),
		}
	}
}

func tokenFromPublicGroupBody(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	var envelope struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return ""
	}
	return strings.TrimSpace(envelope.Data.Token)
}

func publicSignupDeniedMessage(err *APIError) string {
	if msg := strings.TrimSpace(err.Message); msg != "" && msg != "daten konnten nicht geladen werden" {
		return msg
	}
	if msg := apiBodyMessage([]byte(err.Body), ""); msg != "" {
		return msg
	}
	return fmt.Sprintf("Öffentliche Gruppenanmeldung nicht verfügbar (HTTP %d)", err.StatusCode)
}

func apiBodyMessage(body []byte, fallback string) string {
	if len(body) == 0 {
		return fallback
	}
	var envelope struct {
		Message string `json:"message"`
		Data    struct {
			Message string `json:"message"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		if msg := cleanMessage(string(body)); msg != "" {
			return msg
		}
		return fallback
	}
	if msg := cleanMessage(envelope.Message); msg != "" {
		return msg
	}
	if msg := cleanMessage(envelope.Data.Message); msg != "" {
		return msg
	}
	return fallback
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
