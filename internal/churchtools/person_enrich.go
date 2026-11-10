package churchtools

import (
	"fmt"
	"strings"
	"time"
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
	if strings.TrimSpace(detail.InvitationStatus) != "" {
		result.InvitationStatus = detail.InvitationStatus
	}
	if detail.IsSystemUser != nil {
		result.IsSystemUser = detail.IsSystemUser
	}
	if detail.IsAllowedToLogin != nil {
		result.IsAllowedToLogin = detail.IsAllowedToLogin
	}
	if strings.TrimSpace(detail.CMSUserID) != "" {
		result.CMSUserID = detail.CMSUserID
	}
	if hasStringValue(detail.LastLogin) {
		result.LastLogin = detail.LastLogin
	}
	if detail.PrivacyPolicyAgreement != nil && hasStringValue(detail.PrivacyPolicyAgreement.Date) {
		result.PrivacyPolicyAgreement = detail.PrivacyPolicyAgreement
	}
	if hasStringValue(detail.AcceptedSecurity) {
		result.AcceptedSecurity = detail.AcceptedSecurity
	}

	return result
}

// EnrichOptions controls EnrichPersons.
type EnrichOptions struct {
	// Delay is the pause between two detail requests.
	Delay time.Duration
	// Progress, when set, is called after each person with done and total.
	Progress func(done, total int)
}

// EnrichFailure records a person whose details could not be loaded.
type EnrichFailure struct {
	PersonID int
	Err      error
}

// EnrichPersons loads full person records and merges them into the given
// slice. A failing detail request does not abort the run: the list data of
// that person is kept and the failure is returned for a warning. Only when
// every request fails (e.g. lost session) an error is returned.
func (c *Client) EnrichPersons(persons []Person, opts EnrichOptions) ([]Person, []EnrichFailure, error) {
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
	var failures []EnrichFailure
	for i, id := range ids {
		if i > 0 && opts.Delay > 0 {
			sleep(opts.Delay)
		}
		detail, err := c.PersonByID(id)
		if err != nil {
			failures = append(failures, EnrichFailure{PersonID: id, Err: err})
		} else {
			details[id] = detail
		}
		if opts.Progress != nil {
			opts.Progress(i+1, len(ids))
		}
	}
	if len(ids) > 0 && len(failures) == len(ids) {
		return nil, failures, fmt.Errorf("keine Personendetails ladbar: %w", failures[0].Err)
	}

	enriched := make([]Person, len(persons))
	for i, person := range persons {
		if detail, ok := details[person.ID]; ok {
			enriched[i] = MergePersonDetails(person, detail)
		} else {
			enriched[i] = person
		}
	}
	return enriched, failures, nil
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
