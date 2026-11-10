//go:build windows

package fileacl

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

func TestRestrictToOwnerLeavesSingleACE(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret.json")
	if err := os.WriteFile(path, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := RestrictToOwner(path); err != nil {
		t.Fatal(err)
	}

	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		t.Fatal(err)
	}
	if dacl == nil || dacl.AceCount != 1 {
		t.Fatalf("DACL has %d entries, want exactly 1 (current user)", dacl.AceCount)
	}
	control, _, err := sd.Control()
	if err != nil {
		t.Fatal(err)
	}
	if control&windows.SE_DACL_PROTECTED == 0 {
		t.Fatal("DACL must be protected against inherited entries")
	}
}
