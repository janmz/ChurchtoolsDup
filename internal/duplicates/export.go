package duplicates

import (
	"fmt"

	churchtools "github.com/janmz/churchtools-dup/internal/churchtools"
)

// EnrichGroups loads full person details for all members of duplicate groups.
// Persons whose details cannot be loaded keep their list data; they are
// returned as failures so the caller can warn.
func EnrichGroups(client *churchtools.Client, groups []Group, opts churchtools.EnrichOptions) ([]Group, []churchtools.EnrichFailure, error) {
	persons := make([]churchtools.Person, 0)
	for _, group := range groups {
		persons = append(persons, group.Persons...)
	}

	enriched, failures, err := client.EnrichPersons(persons, opts)
	if err != nil {
		return nil, failures, fmt.Errorf("Personendetails laden: %w", err)
	}

	byID := make(map[int]churchtools.Person, len(enriched))
	for _, person := range enriched {
		byID[person.ID] = person
	}

	result := make([]Group, len(groups))
	for i, group := range groups {
		persons := make([]churchtools.Person, len(group.Persons))
		for j, person := range group.Persons {
			if detailed, ok := byID[person.ID]; ok {
				persons[j] = detailed
			} else {
				persons[j] = person
			}
		}
		result[i] = Group{DupID: group.DupID, Persons: persons}
	}
	return result, failures, nil
}
