package churchtools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// RelationshipType describes a ChurchTools person relationship type.
type RelationshipType struct {
	ID   int
	Name string
}

// DuplicateRelationshipOptions configures duplicate relationship type resolution.
type DuplicateRelationshipOptions struct {
	TypeID   int
	TypeName string
}

const duplicateGroupName = "Duplikate"

var duplicateRelationshipKeywords = []string{
	"duplikat",
	"duplicate",
	"dublette",
}

var duplicateRelationshipNamePriority = []string{
	"duplikat",
	"dublette",
	"duplicate",
}

type relationshipTypeDetail struct {
	ID          int
	Name        string
	DegreeNameA string
	DegreeNameB string
	ExportTitle string
}

func (t relationshipTypeDetail) relationshipType() RelationshipType {
	return RelationshipType{ID: t.ID, Name: t.displayName()}
}

func (t relationshipTypeDetail) displayName() string {
	if name := strings.TrimSpace(t.Name); name != "" {
		return name
	}
	if name := strings.TrimSpace(t.ExportTitle); name != "" {
		return name
	}
	return fmt.Sprintf("ID %d", t.ID)
}

func (t relationshipTypeDetail) labels() []string {
	return []string{t.Name, t.DegreeNameA, t.DegreeNameB, t.ExportTitle}
}

// ListRelationshipTypes returns relationship types from the ChurchTools API.
func (c *Client) ListRelationshipTypes() ([]RelationshipType, error) {
	for _, path := range []string{"/person/relationshiptypes", "/relationshiptypes", "/relationshipTypes", "/relationship-types"} {
		items, err := c.fetchAPIPages(path, nil)
		if err != nil {
			continue
		}
		types, err := decodeRelationshipTypeDetails(items)
		if err != nil {
			return nil, err
		}
		if len(types) > 0 {
			result := make([]RelationshipType, len(types))
			for i, relType := range types {
				result[i] = relType.relationshipType()
			}
			return result, nil
		}
	}
	return nil, fmt.Errorf("Beziehungstypen konnten nicht geladen werden")
}

func decodeRelationshipTypeDetails(items []json.RawMessage) ([]relationshipTypeDetail, error) {
	types := make([]relationshipTypeDetail, 0, len(items))
	for _, item := range items {
		var fields map[string]any
		if err := json.Unmarshal(item, &fields); err != nil {
			return nil, err
		}
		id, ok := intFromAny(fields["id"])
		if !ok || id <= 0 {
			continue
		}
		types = append(types, relationshipTypeDetail{
			ID:          id,
			Name:        relationshipFieldString(fields, "name"),
			DegreeNameA: relationshipFieldString(fields, "degreeNameA"),
			DegreeNameB: relationshipFieldString(fields, "degreeNameB"),
			ExportTitle: relationshipFieldString(fields, "exportTitle"),
		})
	}
	return types, nil
}

func relationshipFieldString(fields map[string]any, key string) string {
	if value, ok := fields[key].(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}

func decodeRelationshipTypes(items []json.RawMessage) ([]RelationshipType, error) {
	details, err := decodeRelationshipTypeDetails(items)
	if err != nil {
		return nil, err
	}
	types := make([]RelationshipType, 0, len(details))
	for _, detail := range details {
		types = append(types, detail.relationshipType())
	}
	return types, nil
}

func (c *Client) listRelationshipTypeDetails() ([]relationshipTypeDetail, error) {
	for _, path := range []string{"/person/relationshiptypes", "/relationshiptypes", "/relationshipTypes", "/relationship-types"} {
		items, err := c.fetchAPIPages(path, nil)
		if err != nil {
			continue
		}
		types, err := decodeRelationshipTypeDetails(items)
		if err != nil {
			return nil, err
		}
		if len(types) > 0 {
			return types, nil
		}
	}
	return nil, fmt.Errorf("Beziehungstypen konnten nicht geladen werden")
}

// FindDuplicateRelationshipType returns the relationship type used for duplicates.
func (c *Client) FindDuplicateRelationshipType(opts DuplicateRelationshipOptions) (RelationshipType, error) {
	types, err := c.listRelationshipTypeDetails()
	if err != nil {
		return RelationshipType{}, err
	}

	if opts.TypeID > 0 {
		for _, relType := range types {
			if relType.ID == opts.TypeID {
				return relType.relationshipType(), nil
			}
		}
		return RelationshipType{}, fmt.Errorf(
			"Beziehungstyp mit ID %d nicht gefunden",
			opts.TypeID,
		)
	}

	if name := strings.TrimSpace(opts.TypeName); name != "" {
		matches := matchRelationshipTypesByName(types, name)
		switch len(matches) {
		case 1:
			return matches[0].relationshipType(), nil
		case 0:
			return RelationshipType{}, fmt.Errorf("Beziehungstyp %q nicht gefunden", name)
		default:
			return RelationshipType{}, fmt.Errorf(
				"mehrere Beziehungstypen passen zu %q: %s",
				name,
				formatRelationshipTypeList(matches),
			)
		}
	}

	return pickDuplicateRelationshipType(types)
}

func matchRelationshipTypesByName(types []relationshipTypeDetail, name string) []relationshipTypeDetail {
	needle := strings.ToLower(strings.TrimSpace(name))
	var matches []relationshipTypeDetail
	for _, relType := range types {
		if strings.ToLower(relType.displayName()) == needle {
			matches = append(matches, relType)
		}
	}
	return matches
}

func pickDuplicateRelationshipType(types []relationshipTypeDetail) (RelationshipType, error) {
	bestScore := 0
	var matches []relationshipTypeDetail

	for _, relType := range types {
		score := duplicateRelationshipScore(relType)
		if score <= 0 {
			continue
		}
		if score > bestScore {
			bestScore = score
			matches = []relationshipTypeDetail{relType}
			continue
		}
		if score == bestScore {
			matches = append(matches, relType)
		}
	}

	if len(matches) == 0 {
		return RelationshipType{}, fmt.Errorf("Beziehungstyp für Duplikate nicht gefunden")
	}
	if len(matches) == 1 {
		return matches[0].relationshipType(), nil
	}

	if chosen, ok := tieBreakDuplicateRelationship(matches); ok {
		return chosen.relationshipType(), nil
	}

	return RelationshipType{}, fmt.Errorf(
		"mehrere Beziehungstypen für Duplikate gefunden: %s (duplicate_relationship_type.id in config.json setzen)",
		formatRelationshipTypeList(matches),
	)
}

func duplicateRelationshipScore(relType relationshipTypeDetail) int {
	best := 0
	for _, label := range relType.labels() {
		lower := strings.ToLower(strings.TrimSpace(label))
		if lower == "" {
			continue
		}
		for _, exact := range duplicateRelationshipNamePriority {
			if lower == exact {
				return 100
			}
		}
		for _, keyword := range duplicateRelationshipKeywords {
			if strings.Contains(lower, keyword) {
				score := 50 + len(keyword) - len(lower)
				if score > best {
					best = score
				}
			}
		}
	}
	return best
}

func tieBreakDuplicateRelationship(matches []relationshipTypeDetail) (relationshipTypeDetail, bool) {
	for _, preferred := range duplicateRelationshipNamePriority {
		for _, relType := range matches {
			if strings.ToLower(strings.TrimSpace(relType.Name)) == preferred {
				return relType, true
			}
		}
	}

	best := matches[0]
	for _, relType := range matches[1:] {
		if relType.ID < best.ID {
			best = relType
		}
	}
	return best, true
}

func formatRelationshipTypeList(types []relationshipTypeDetail) string {
	parts := make([]string, 0, len(types))
	for _, relType := range types {
		parts = append(parts, fmt.Sprintf("%q (ID %d)", relType.displayName(), relType.ID))
	}
	return strings.Join(parts, ", ")
}

// LinkAsDuplicate connects two persons as duplicates via relationship management.
func (c *Client) LinkAsDuplicate(primaryID, otherID int, relType RelationshipType) error {
	if primaryID <= 0 || otherID <= 0 || primaryID == otherID {
		return fmt.Errorf("ungültige Personen-IDs für Duplikat-Verknüpfung")
	}

	exists, err := c.DuplicateRelationshipExists(primaryID, otherID, relType)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	payloads := []map[string]any{
		{"personId": otherID, "relationshipTypeId": relType.ID},
		{"relatedPersonId": otherID, "relationshipTypeId": relType.ID},
		{"personId": otherID, "relationTypeId": relType.ID},
	}

	path := fmt.Sprintf("/persons/%d/relationships", primaryID)
	for _, payload := range payloads {
		status, body, err := c.postAPI(path, payload, true)
		if err != nil {
			return err
		}
		if status == http.StatusOK || status == http.StatusCreated || status == http.StatusNoContent {
			return nil
		}
		if status != http.StatusNotFound && status != http.StatusBadRequest {
			return &APIError{
				StatusCode: status,
				Message:    "Duplikat-Beziehung konnte nicht angelegt werden",
				Body:       string(body),
			}
		}
	}

	if err := c.linkAsDuplicateLegacy(primaryID, otherID, relType.ID); err != nil {
		return err
	}
	return nil
}

func (c *Client) linkAsDuplicateLegacy(primaryID, otherID, relTypeID int) error {
	if strings.TrimSpace(c.csrfToken) == "" {
		token, err := c.fetchCSRFToken()
		if err != nil {
			return err
		}
		c.csrfToken = token
	}

	payload := map[string]any{
		"func":     "add_rel",
		"id":       strconv.Itoa(primaryID),
		"child_id": strconv.Itoa(otherID),
		"rel_id":   strconv.Itoa(relTypeID),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	reqURL := strings.TrimSuffix(c.baseURL, "/") + "/?q=churchdb/ajax"
	req, err := http.NewRequest(http.MethodPost, reqURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", c.csrfToken)
	c.applyAuth(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("Legacy-Beziehung anlegen: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    "Legacy-Duplikat-Beziehung fehlgeschlagen",
			Body:       string(respBody),
		}
	}

	var envelope struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(respBody, &envelope); err == nil {
		if strings.EqualFold(envelope.Status, "error") {
			return &APIError{
				StatusCode: resp.StatusCode,
				Message:    "Legacy-Duplikat-Beziehung abgelehnt",
				Body:       string(respBody),
			}
		}
	}
	return nil
}

// EnsurePersonInGroup adds a person to a group by name (e.g. "Duplikate").
func (c *Client) EnsurePersonInGroup(groupName string, personID int) (MembershipRequestResult, error) {
	group, err := c.FindGroupByName(groupName)
	if err != nil {
		return MembershipRequestResult{}, err
	}
	return c.RequestGroupMembership(group.ID, personID)
}

// DuplicateGroupName is the default group for marked duplicate primaries.
const DuplicateGroupName = duplicateGroupName

// ListAllPersons loads the complete person list without campus filter.
func (c *Client) ListAllPersons() ([]Person, error) {
	return c.ListPersons(PersonListOptions{})
}
