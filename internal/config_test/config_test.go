package config_test

import (
	"os"
	"path/filepath"
	"testing"

	config "github.com/janmz/churchtools-dup/internal/config"
)

func TestNormalizeBaseURL(t *testing.T) {
	tests := map[string]string{
		"https://demo.church.tools/":     "https://demo.church.tools",
		"https://demo.church.tools/api/": "https://demo.church.tools",
		"  https://demo.church.tools  ":  "https://demo.church.tools",
		"demo":                           "https://demo.church.tools",
		"Gemeinde-Unterstadt":            "https://gemeinde-unterstadt.church.tools",
		"demo.church.tools":              "https://demo.church.tools",
	}

	for input, want := range tests {
		if got := config.NormalizeBaseURL(input); got != want {
			t.Fatalf("NormalizeBaseURL(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestBaseURLFromInstanceName(t *testing.T) {
	url, err := config.BaseURLFromInstanceName("gemeinde-unterstadt")
	if err != nil {
		t.Fatal(err)
	}
	if url != "https://gemeinde-unterstadt.church.tools" {
		t.Fatalf("url = %q", url)
	}

	if _, err := config.BaseURLFromInstanceName("bad/name"); err == nil {
		t.Fatal("expected error for slash in name")
	}
}

func TestLoadAndSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg := config.Config{
		BaseURL:    "https://demo.church.tools",
		LoginToken: "secret-token",
		DelayMS:    250,
	}

	if err := config.Save(path, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.BaseURL != cfg.BaseURL || loaded.LoginToken != cfg.LoginToken || loaded.DelayMS != cfg.DelayMS {
		t.Fatalf("loaded config mismatch: %+v", loaded)
	}
}

func TestLoadCampusID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg := config.Config{
		BaseURL:    "https://demo.church.tools",
		LoginToken: "secret-token",
		CampusID:   42,
	}

	if err := config.Save(path, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.CampusID != 42 {
		t.Fatalf("campus_id = %d, want 42", loaded.CampusID)
	}
}

func TestValidateRequiresAuth(t *testing.T) {
	cfg := config.Config{BaseURL: "demo"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error without credentials")
	}
	if cfg.BaseURL != "https://demo.church.tools" {
		t.Fatalf("base url = %q", cfg.BaseURL)
	}
}

func TestLoadAppliesEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	if err := os.WriteFile(path, []byte(`{"base_url":"https://old.example"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("CT_BASE_URL", "https://demo.church.tools")
	t.Setenv("CT_LOGIN_TOKEN", "from-env")

	loaded, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.BaseURL != "https://demo.church.tools" {
		t.Fatalf("base url = %q", loaded.BaseURL)
	}
	if loaded.LoginToken != "from-env" {
		t.Fatalf("token = %q", loaded.LoginToken)
	}
}

func TestLoadAppliesEnvInstanceName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	if err := os.WriteFile(path, []byte(`{"login_token":"x"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("CT_BASE_URL", "gemeinde-unterstadt")
	t.Setenv("CT_LOGIN_TOKEN", "from-env")

	loaded, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.BaseURL != "https://gemeinde-unterstadt.church.tools" {
		t.Fatalf("base url = %q", loaded.BaseURL)
	}
}

func TestDuplicateRelationshipTypeIDDefaultsToEight(t *testing.T) {
	cfg := config.Config{}
	if got := cfg.DuplicateRelationshipTypeID(); got != 8 {
		t.Fatalf("default id = %d, want 8", got)
	}

	cfg = config.Config{DuplicateRelType: config.DuplicateRelType{ID: 5}}
	if got := cfg.DuplicateRelationshipTypeID(); got != 5 {
		t.Fatalf("configured id = %d, want 5", got)
	}
}
