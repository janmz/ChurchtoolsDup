package churchtools

import (
	"fmt"
	"strings"
)

// PreJoinGroupResult describes one group in the pre-join sequence.
type PreJoinGroupResult struct {
	GroupName string
	Status    MembershipRequestStatus
	Message   string
	Skipped   bool
}

// EnsurePreJoinGroups joins the authenticated person to each group in order when
// not already a member. After a successful join the session is refreshed so
// follow-up groups that depend on earlier memberships can be requested.
func (c *Client) EnsurePreJoinGroups(groupNames []string) ([]PreJoinGroupResult, error) {
	personID := c.PersonID()
	if personID <= 0 {
		user, err := c.WhoAmI()
		if err != nil {
			return nil, err
		}
		personID = user.ID
	}

	results := make([]PreJoinGroupResult, 0, len(groupNames))
	for _, name := range groupNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		inGroup, err := c.PersonIsInGroup(personID, name)
		if err != nil {
			return results, fmt.Errorf("Gruppe %q prüfen: %w", name, err)
		}
		if inGroup {
			results = append(results, PreJoinGroupResult{
				GroupName: name,
				Status:    MembershipActive,
				Message:   "bereits Mitglied",
				Skipped:   true,
			})
			continue
		}

		group, err := c.FindGroupByName(name)
		if err != nil {
			return results, fmt.Errorf("Gruppe %q finden: %w", name, err)
		}

		membership, err := c.RequestGroupMembership(group.ID, personID)
		if err != nil {
			return results, fmt.Errorf("Gruppe %q beitreten: %w", name, err)
		}

		result := PreJoinGroupResult{
			GroupName: name,
			Status:    membership.Status,
			Message:   membership.Message,
		}
		results = append(results, result)

		if membership.Status == MembershipActive {
			if err := c.refreshSession(); err != nil {
				result.Message = strings.TrimSpace(result.Message + "; Sitzung nach Beitritt nicht erneuert: " + err.Error())
				results[len(results)-1] = result
			}
		}
	}

	return results, nil
}

func (c *Client) refreshSession() error {
	if strings.TrimSpace(c.loginToken) != "" {
		return c.relogin()
	}
	return c.Login()
}
