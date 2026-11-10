package fileacl

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRestrictToOwnerKeepsFileUsable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret.json")
	if err := os.WriteFile(path, []byte(`{"a":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RestrictToOwner(path); err != nil {
		t.Fatalf("RestrictToOwner: %v", err)
	}
	// The owner must still be able to read and rewrite the file.
	if _, err := os.ReadFile(path); err != nil {
		t.Fatalf("read after restrict: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"a":2}`), 0o600); err != nil {
		t.Fatalf("write after restrict: %v", err)
	}
}
