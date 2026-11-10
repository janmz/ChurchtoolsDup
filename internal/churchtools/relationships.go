package churchtools

import (
	"encoding/json"
	"fmt"
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
	// TypeID is an explicitly configured type ID; it must exist.
	TypeID int
	// TypeName is an explicitly configured type name (case-insensitive).
	TypeName string
	// DefaultTypeID is tried when neither ID nor name is configured; it is
	// only used when its labels look like a duplicate type.
	DefaultTypeID int
}

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

// relationshipTypePaths lists the current endpoint first, then legacy variants.
var relationshipTypePaths = []string{"/person/relationshiptypes", "/relationshiptypes", "/relationshipTypes", "/relationship-types"}

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
	details, err := c.listRelationshipTypeDetails()
	if err != nil {
		return nil, err
	}
	result := make([]RelationshipType, len(details))
	for i, relType := range details {
		result[i] = relType.relationshipType()
	}
	return result, nil
}

func (c *Client) listRelationshipTypeDetails() ([]relationshipTypeDetail, error) {
	for _, path := range relationshipTypePaths {
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

// FindDuplicateRelationshipType returns the relationship type used for
// duplicates. Order: configured ID, configured name, default ID (only when it
// looks like a duplicate type), keyword search over all types.
func (c *Client) FindDuplicateRelationshipType(opts DuplicateRelationshipOptions) (RelationshipType, error) {
	types, err := c.listRelationshipTypeDetails()
	if err != nil {
		return RelationshipType{}, err
	}

	if opts.TypeID > 0 {
		if relType, ok := relationshipTypeByID(types, opts.TypeID); ok {
			return relType.relationshipType(), nil
		}
		return RelationshipType{}, fmt.Errorf("Beziehungstyp mit ID %d nicht gefunden", opts.TypeID)
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

	if opts.DefaultTypeID > 0 {
		if relType, ok := relationshipTypeByID(types, opts.DefaultTypeID); ok && duplicateRelationshipScore(relType) > 0 {
			return relType.relationshipType(), nil
		}
	}

	return pickDuplicateRelationshipType(types)
}

func relationshipTypeByID(types []relationshipTypeDetail, id int) (relationshipTypeDetail, bool) {
	for _, relType := range types {
		if relType.ID == id {
			return relType, true
		}
	}
	return relationshipTypeDetail{}, false
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

	switch len(matches) {
	case 0:
		return RelationshipType{}, fmt.Errorf("Beziehungstyp für Duplikate nicht gefunden (duplicate_relationship_type in config.json setzen)")
	case 1:
		return matches[0].relationshipType(), nil
	default:
		return tieBreakDuplicateRelationship(matches).relationshipType(), nil
	}
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

// tieBreakDuplicateRelationship prefers the canonical names, then the lowest ID.
func tieBreakDuplicateRelationship(matches []relationshipTypeDetail) relationshipTypeDetail {
	for _, preferred := range duplicateRelationshipNamePriority {
		for _, relType := range matches {
			if strings.ToLower(strings.TrimSpace(relType.Name)) == preferred {
				return relType
			}
		}
	}

	best := matches[0]
	for _, relType := range matches[1:] {
		if relType.ID < best.ID {
			best = relType
		}
	}
	return best
}

func formatRelationshipTypeList(types []relationshipTypeDetail) string {
	parts := make([]string, 0, len(types))
	for _, relType := range types {
		parts = append(parts, fmt.Sprintf("%q (ID %d)", relType.displayName(), relType.ID))
	}
	return strings.Join(parts, ", ")
}

// LinkAsDuplicate creates a duplicate relationship from primaryID to otherID
// (POST /persons/{id}/relationships). It does not check for an existing
// relationship; use DuplicateRelationshipExists before calling it. The legacy
// AJAX API is only used when the REST endpoint does not exist (HTTP 404).
func (c *Client) LinkAsDuplicate(primaryID, otherID int, relType RelationshipType) error {
	if primaryID <= 0 || otherID <= 0 || primaryID == otherID {
		return fmt.Errorf("ungültige Personen-IDs für Duplikat-Verknüpfung")
	}

	path := fmt.Sprintf("/persons/%d/relationships", primaryID)
	status, body, err := c.postAPI(path, map[string]any{
		"otherPersonId":      otherID,
		"relationshipTypeId": relType.ID,
	})
	if err != nil {
		return err
	}
	switch status {
	case http.StatusCreated, http.StatusOK, http.StatusNoContent:
		return nil
	case http.StatusNotFound:
		return c.linkAsDuplicateLegacy(primaryID, otherID, relType.ID)
	default:
		return newAPIError(status, "Duplikat-Beziehung konnte nicht angelegt werden: "+apiBodyMessage(body, ""), body)
	}
}

// legacyNotConfirmedMessage is reported when the legacy API did not confirm success.
const legacyNotConfirmedMessage = "Legacy-API hat das Anlegen der Beziehung nicht bestätigt"

func (c *Client) linkAsDuplicateLegacy(primaryID, otherID, relTypeID int) error {
	if strings.TrimSpace(c.csrfToken) == "" {
		token, err := c.fetchCSRFToken()
		if err != nil {
			return err
		}
		c.csrfToken = token
	}

	payload, err := json.Marshal(map[string]any{
		"func":     "add_rel",
		"id":       strconv.Itoa(primaryID),
		"child_id": strconv.Itoa(otherID),
		"rel_id":   strconv.Itoa(relTypeID),
	})
	if err != nil {
		return err
	}

	reqURL := strings.TrimSuffix(c.baseURL, "/") + "/?q=churchdb/ajax"
	status, respBody, err := c.doRequest(http.MethodPost, reqURL, payload, true)
	if err != nil {
		return fmt.Errorf("Legacy-Beziehung anlegen: %w", err)
	}
	if status != http.StatusOK {
		return newAPIError(status, "Legacy-Duplikat-Beziehung fehlgeschlagen", respBody)
	}

	// Only an explicit {"status":"success"} counts; an HTML login page or any
	// other non-JSON answer with HTTP 200 must not be reported as linked.
	var envelope struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(respBody, &envelope); err != nil || !strings.EqualFold(envelope.Status, "success") {
		return newAPIError(status, legacyNotConfirmedMessage, respBody)
	}
	return nil
}
