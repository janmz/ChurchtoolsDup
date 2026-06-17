package churchtools

import (
	"fmt"
	"strings"
)

const (
	ModuleChurchCore = "churchcore"
	ModuleChurchDB   = "churchdb"

	PermExportData        = "export data"
	PermWriteAccess      = "write access"
	PermAdministerPersons = "administer persons"
	PermEditRelations    = "edit relations"
)

// HasModulePermission checks a global permission flag from /permissions/global.
func HasModulePermission(perms map[string]any, module, name string) bool {
	mod, ok := perms[module].(map[string]any)
	if !ok {
		return false
	}
	value, ok := mod[name]
	if !ok {
		return false
	}
	switch v := value.(type) {
	case bool:
		return v
	case []any:
		return len(v) > 0
	case float64:
		return v != 0
	case int:
		return v != 0
	default:
		return false
	}
}

// HasModulePermission loads global permissions and checks one flag.
func (c *Client) HasModulePermission(module, name string) (bool, error) {
	perms, err := c.GlobalPermissions()
	if err != nil {
		return false, err
	}
	return HasModulePermission(perms, module, name), nil
}

// PermissionRequirement describes a missing permission and its grant group.
type PermissionRequirement struct {
	Module       string
	Permission   string
	GroupNames   []string
	Description  string
}

// MembershipRequestStatus describes the outcome of a group membership request.
type MembershipRequestStatus string

const (
	MembershipActive    MembershipRequestStatus = "active"
	MembershipRequested MembershipRequestStatus = "requested"
	MembershipDenied    MembershipRequestStatus = "denied"
)

// MembershipRequestResult is returned by RequestGroupMembership.
type MembershipRequestResult struct {
	Status  MembershipRequestStatus
	Message string
}

// EnsurePermissions requests group membership when required permissions are missing.
func (c *Client) EnsurePermissions(reqs []PermissionRequirement) ([]string, error) {
	perms, err := c.GlobalPermissions()
	if err != nil {
		return nil, fmt.Errorf("Berechtigungen laden: %w", err)
	}

	personID := c.PersonID()
	if personID <= 0 {
		user, err := c.WhoAmI()
		if err != nil {
			return nil, err
		}
		personID = user.ID
	}

	var notes []string
	requested := false
	activeMembership := make(map[string]bool)

	for _, req := range reqs {
		if HasModulePermission(perms, req.Module, req.Permission) {
			continue
		}

		group, groupName, err := c.FindGroupByNames(req.GroupNames)
		if err != nil {
			notes = append(notes, fmt.Sprintf(
				"%s fehlt; keine passende Gruppe gefunden (%s)",
				req.Description,
				formatGroupNames(req.GroupNames),
			))
			continue
		}

		result, err := c.RequestGroupMembership(group.ID, personID)
		if err != nil {
			notes = append(notes, fmt.Sprintf(
				"%s fehlt; Anfrage für Gruppe %q fehlgeschlagen: %v",
				req.Description,
				groupName,
				err,
			))
			continue
		}
		requested = true

		switch result.Status {
		case MembershipActive:
			activeMembership[req.Permission] = true
			notes = append(notes, fmt.Sprintf(
				"%s: Mitgliedschaft in %q aktiv (Berechtigung ggf. erst nach erneutem Login)",
				req.Description,
				groupName,
			))
		case MembershipRequested:
			notes = append(notes, fmt.Sprintf(
				"%s fehlt; Mitgliedschaft in %q beantragt (Freigabe durch Administrator nötig)",
				req.Description,
				groupName,
			))
		default:
			msg := result.Message
			if msg == "" {
				msg = "Anfrage abgelehnt"
			}
			notes = append(notes, fmt.Sprintf(
				"%s fehlt; Mitgliedschaft in %q nicht möglich: %s",
				req.Description,
				groupName,
				msg,
			))
		}
	}

	if !requested {
		return notes, nil
	}

	perms, err = c.GlobalPermissions()
	if err != nil {
		return notes, err
	}

	for _, req := range reqs {
		if HasModulePermission(perms, req.Module, req.Permission) {
			continue
		}
		if activeMembership[req.Permission] {
			continue
		}
		notes = append(notes, fmt.Sprintf(
			"%s weiterhin nicht aktiv (evtl. wartet Freigabe)",
			req.Description,
		))
	}

	return notes, nil
}

func formatGroupNames(names []string) string {
	return strings.Join(names, " / ")
}
