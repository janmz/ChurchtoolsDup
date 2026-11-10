// Package fileacl restricts files with secrets or personal data (config.json,
// duplicate CSV) to the current user. Unix permission bits are not enforced
// on Windows, so there an explicit, non-inherited DACL is set instead.
package fileacl

// RestrictToOwner limits read and write access of path to the current user.
func RestrictToOwner(path string) error {
	return restrictToOwner(path)
}
