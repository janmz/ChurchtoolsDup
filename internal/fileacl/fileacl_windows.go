//go:build windows

package fileacl

import (
	"fmt"

	"golang.org/x/sys/windows"
)

// restrictToOwner replaces the file's DACL by a single entry granting full
// access to the current user and blocks inherited entries (e.g. "Users" or
// "Authenticated Users" from the parent directory).
func restrictToOwner(path string) error {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return fmt.Errorf("aktuellen Benutzer ermitteln: %w", err)
	}

	acl, err := windows.ACLFromEntries([]windows.EXPLICIT_ACCESS{{
		AccessPermissions: windows.GENERIC_ALL,
		AccessMode:        windows.SET_ACCESS,
		Inheritance:       windows.NO_INHERITANCE,
		Trustee: windows.TRUSTEE{
			TrusteeForm:  windows.TRUSTEE_IS_SID,
			TrusteeType:  windows.TRUSTEE_IS_USER,
			TrusteeValue: windows.TrusteeValueFromSID(user.User.Sid),
		},
	}}, nil)
	if err != nil {
		return fmt.Errorf("zugriffsliste erstellen: %w", err)
	}

	err = windows.SetNamedSecurityInfo(
		path,
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		nil, nil, acl, nil,
	)
	if err != nil {
		return fmt.Errorf("zugriffsrechte für %s setzen: %w", path, err)
	}
	return nil
}
