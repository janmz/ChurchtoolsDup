package churchtools

import (
	"strings"
)

// MergePersonDetails overlays non-empty detail fields onto a list person record.
func MergePersonDetails(base, detail Person) Person {
	result := base

	if strings.TrimSpace(detail.FirstName) != "" {
		result.FirstName = detail.FirstName
	}
	if strings.TrimSpace(detail.LastName) != "" {
		result.LastName = detail.LastName
	}
	if strings.TrimSpace(detail.Email) != "" {
		result.Email = detail.Email
	}
	if len(detail.Emails) > 0 {
		result.Emails = detail.Emails
	}
	if detail.CampusID > 0 {
		result.CampusID = detail.CampusID
	}
	if strings.TrimSpace(detail.CampusName) != "" {
		result.CampusName = detail.CampusName
	}
	if strings.TrimSpace(detail.Street) != "" {
		result.Street = detail.Street
	}
	if strings.TrimSpace(detail.City) != "" {
		result.City = detail.City
	}
	if strings.TrimSpace(detail.CreatedAt) != "" {
		result.CreatedAt = detail.CreatedAt
	}
	if detail.PrivacyPolicyAgreement != nil && hasStringValue(detail.PrivacyPolicyAgreement.Date) {
		result.PrivacyPolicyAgreement = detail.PrivacyPolicyAgreement
	}
	if hasStringValue(detail.AcceptedSecurity) {
		result.AcceptedSecurity = detail.AcceptedSecurity
	}

	return result
}

// EnrichPersons loads full person records and merges them into the given slice.
func (c *Client) EnrichPersons(persons []Person) ([]Person, error) {
	seen := make(map[int]struct{}, len(persons))
	ids := make([]int, 0, len(persons))
	for _, person := range persons {
		if person.ID <= 0 {
			continue
		}
		if _, ok := seen[person.ID]; ok {
			continue
		}
		seen[person.ID] = struct{}{}
		ids = append(ids, person.ID)
	}

	details := make(map[int]Person, len(ids))
	for _, id := range ids {
		detail, err := c.PersonByID(id)
		if err != nil {
			return nil, err
		}
		details[id] = detail
	}

	enriched := make([]Person, len(persons))
	for i, person := range persons {
		if detail, ok := details[person.ID]; ok {
			enriched[i] = MergePersonDetails(person, detail)
		} else {
			enriched[i] = person
		}
	}
	return enriched, nil
}

// CampusNamesByID builds a lookup table for standort names.
func CampusNamesByID(campuses []Campus) map[int]string {
	names := make(map[int]string, len(campuses))
	for _, campus := range campuses {
		if campus.ID > 0 {
			names[campus.ID] = campus.Name
		}
	}
	return names
}

// CampusLabel returns the standort name for a person.
func CampusLabel(person Person, campusNames map[int]string) string {
	if strings.TrimSpace(person.CampusName) != "" {
		return strings.TrimSpace(person.CampusName)
	}
	if person.CampusID <= 0 {
		return ""
	}
	return strings.TrimSpace(campusNames[person.CampusID])
}
