package duplicates

import (
	"fmt"

	churchtools "github.com/janmz/churchtools-dup/internal/churchtools"
)

// EnrichGroups loads full person details for all members of duplicate groups.
func EnrichGroups(client *churchtools.Client, groups []Group) ([]Group, error) {
	persons := make([]churchtools.Person, 0)
	for _, group := range groups {
		persons = append(persons, group.Persons...)
	}

	enriched, err := client.EnrichPersons(persons)
	if err != nil {
		return nil, fmt.Errorf("Personendetails laden: %w", err)
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
	return result, nil
}
