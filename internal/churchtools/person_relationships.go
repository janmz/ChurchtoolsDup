package churchtools

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// PersonRelationship is a link from one person to a relative.
type PersonRelationship struct {
	ID                 int
	RelationshipTypeID int
	RelatedPersonID    int
	Name               string
}

// ListPersonRelationships returns relationships visible for a person.
func (c *Client) ListPersonRelationships(personID int) ([]PersonRelationship, error) {
	if personID <= 0 {
		return nil, fmt.Errorf("Personen-ID fehlt")
	}

	path := "/persons/" + strconv.Itoa(personID) + "/relationships"
	items, err := c.fetchAPIPages(path, nil)
	if err != nil {
		return nil, fmt.Errorf("Beziehungen laden: %w", err)
	}

	rels := make([]PersonRelationship, 0, len(items))
	for _, item := range items {
		rel, ok := decodePersonRelationship(item)
		if !ok {
			continue
		}
		rels = append(rels, rel)
	}
	return rels, nil
}

// DuplicateRelationshipExists reports whether two persons are already linked as duplicates.
func (c *Client) DuplicateRelationshipExists(primaryID, otherID int, relType RelationshipType) (bool, error) {
	if primaryID <= 0 || otherID <= 0 || primaryID == otherID {
		return false, fmt.Errorf("ungültige Personen-IDs für Duplikat-Prüfung")
	}

	found, err := c.hasRelationshipTo(primaryID, otherID, relType.ID)
	if err != nil || found {
		return found, err
	}
	return c.hasRelationshipTo(otherID, primaryID, relType.ID)
}

func (c *Client) hasRelationshipTo(personID, relatedID, relTypeID int) (bool, error) {
	rels, err := c.ListPersonRelationships(personID)
	if err != nil {
		return false, err
	}
	for _, rel := range rels {
		if rel.RelatedPersonID != relatedID {
			continue
		}
		if relationshipTypeMatches(rel, relTypeID) {
			return true, nil
		}
	}
	return false, nil
}

func relationshipTypeMatches(rel PersonRelationship, relTypeID int) bool {
	if relTypeID <= 0 {
		return true
	}
	if rel.RelationshipTypeID == relTypeID {
		return true
	}
	if rel.RelationshipTypeID == 0 && isDuplicateRelationshipLabel(rel.Name) {
		return true
	}
	return false
}

func isDuplicateRelationshipLabel(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	for _, keyword := range duplicateRelationshipKeywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}

func decodePersonRelationship(item json.RawMessage) (PersonRelationship, bool) {
	var fields map[string]any
	if err := json.Unmarshal(item, &fields); err != nil {
		return PersonRelationship{}, false
	}

	id, _ := intFromAny(fields["id"])
	relTypeID := relationshipTypeIDFromFields(fields)

	relatedID := relatedPersonIDFromFields(fields)
	if relatedID <= 0 {
		return PersonRelationship{}, false
	}

	name, _ := fields["relationshipName"].(string)
	if name == "" {
		name, _ = fields["degreeOfRelationship"].(string)
	}

	return PersonRelationship{
		ID:                 id,
		RelationshipTypeID: relTypeID,
		RelatedPersonID:    relatedID,
		Name:               strings.TrimSpace(name),
	}, true
}

func relationshipTypeIDFromFields(fields map[string]any) int {
	for _, key := range []string{
		"relationshipTypeId",
		"relationTypeId",
		"relationship_type_id",
		"relation_type_id",
	} {
		if id, ok := intFromAny(fields[key]); ok && id > 0 {
			return id
		}
	}
	if nested, ok := fields["relationshipType"].(map[string]any); ok {
		if id, ok := intFromAny(nested["id"]); ok && id > 0 {
			return id
		}
	}
	return 0
}

func relatedPersonIDFromFields(fields map[string]any) int {
	for _, key := range []string{"personId", "relatedPersonId", "relativePersonId"} {
		if id, ok := intFromAny(fields[key]); ok && id > 0 {
			return id
		}
	}

	if relative, ok := fields["relative"].(map[string]any); ok {
		if id, ok := personIDFromObject(relative); ok {
			return id
		}
		if id, ok := personIDFromDomainIdentifier(relative["domainIdentifier"]); ok {
			return id
		}
	}
	if person, ok := fields["person"].(map[string]any); ok {
		if id, ok := personIDFromObject(person); ok {
			return id
		}
	}
	return 0
}

func personIDFromDomainIdentifier(value any) (int, bool) {
	text, ok := value.(string)
	if !ok {
		return 0, false
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, false
	}
	if id, err := strconv.Atoi(text); err == nil && id > 0 {
		return id, true
	}
	for _, part := range strings.Split(text, "-") {
		part = strings.TrimSpace(part)
		if id, err := strconv.Atoi(part); err == nil && id > 0 {
			return id, true
		}
	}
	return 0, false
}

func personIDFromObject(obj map[string]any) (int, bool) {
	if id, ok := intFromAny(obj["id"]); ok && id > 0 {
		return id, true
	}
	if id, ok := intFromAny(obj["personId"]); ok && id > 0 {
		return id, true
	}
	if apiURL, ok := obj["apiUrl"].(string); ok {
		if id, ok := personIDFromAPIURL(apiURL); ok {
			return id, true
		}
	}
	if frontendURL, ok := obj["frontendUrl"].(string); ok {
		if id, ok := personIDFromAPIURL(frontendURL); ok {
			return id, true
		}
	}
	return 0, false
}

func personIDFromAPIURL(apiURL string) (int, bool) {
	apiURL = strings.TrimSpace(apiURL)
	if apiURL == "" {
		return 0, false
	}

	apiURL = strings.TrimSuffix(apiURL, "/")
	parts := strings.Split(apiURL, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		part := strings.TrimSpace(parts[i])
		if part == "" || part == "persons" || part == "person" {
			continue
		}
		id, err := strconv.Atoi(part)
		if err == nil && id > 0 {
			return id, true
		}
	}
	return 0, false
}
