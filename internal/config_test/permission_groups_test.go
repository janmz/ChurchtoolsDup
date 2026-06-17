package config_test

import (
	"testing"

	config "github.com/janmz/churchtools-dup/internal/config"
)

func TestEditPersonsGroupNamesDefaults(t *testing.T) {
	cfg := config.Config{}
	names := cfg.EditPersonsGroupNames()
	if len(names) != 2 {
		t.Fatalf("expected 2 defaults, got %d", len(names))
	}
	if names[0] != "Personen Administration" || names[1] != "Personen bearbeiten" {
		t.Fatalf("unexpected defaults: %v", names)
	}
}

func TestEditPersonsGroupNamesConfigured(t *testing.T) {
	cfg := config.Config{
		PermissionGroups: config.PermissionGroups{
			EditPersons: "Eigene Gruppe",
		},
	}
	names := cfg.EditPersonsGroupNames()
	if len(names) != 1 || names[0] != "Eigene Gruppe" {
		t.Fatalf("unexpected names: %v", names)
	}
}
