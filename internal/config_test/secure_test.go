package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	config "github.com/janmz/churchtools-dup/internal/config"
)

func TestLoadEncryptsPlaintextSecretsOnDisk(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	plain := `{"base_url":"https://x.church.tools","login_token":"klartext-token","username":"u","password":" pass wort "}`
	if err := os.WriteFile(path, []byte(plain), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.LoginTokenPassword != "klartext-token" || cfg.Password != " pass wort " {
		t.Fatalf("decrypted values = %q / %q", cfg.LoginTokenPassword, cfg.Password)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, secret := range []string{"klartext-token", " pass wort "} {
		if strings.Contains(text, secret) {
			t.Fatalf("secret %q still in plaintext on disk:\n%s", secret, text)
		}
	}
	for _, key := range []string{`"login_token_secure"`, `"secure_password"`} {
		if !strings.Contains(text, key) {
			t.Fatalf("encrypted field %s missing:\n%s", key, text)
		}
	}

	// A second load must give the same decrypted values.
	again, err := config.Load(path)
	if err != nil || again.LoginTokenPassword != "klartext-token" {
		t.Fatalf("reload = %q, %v", again.LoginTokenPassword, err)
	}
}

func TestLoadNeverWritesEnvironmentSecrets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"base_url":"https://x.church.tools","login_token":"datei-token"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CT_LOGIN_TOKEN", "env-token")
	t.Setenv("CT_PASSWORD", "env-passwort")

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LoginTokenPassword != "env-token" {
		t.Fatalf("environment must override in memory, got %q", cfg.LoginTokenPassword)
	}

	reloaded, err := config.LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.LoginTokenPassword != "datei-token" || reloaded.Password != "" {
		t.Fatalf("file values changed by environment: token %q password %q", reloaded.LoginTokenPassword, reloaded.Password)
	}
}

func TestSaveEncryptsSecrets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := config.Default()
	cfg.BaseURL = "https://x.church.tools"
	cfg.LoginTokenPassword = "neues-token"
	if err := config.Save(path, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "neues-token") {
		t.Fatalf("token written in plaintext:\n%s", data)
	}
	loaded, err := config.Load(path)
	if err != nil || loaded.LoginTokenPassword != "neues-token" {
		t.Fatalf("Load after Save = %q, %v", loaded.LoginTokenPassword, err)
	}
}

func TestValidateRequiresHTTPS(t *testing.T) {
	cases := map[string]bool{
		"https://x.church.tools": true,
		"http://x.church.tools":  false,
		"http://localhost:8080":  true,
		"http://127.0.0.1:1234":  true,
		"ftp://x.church.tools":   false,
	}
	for base, ok := range cases {
		cfg := config.Config{BaseURL: base, LoginTokenPassword: "t"}
		if err := cfg.Validate(); (err == nil) != ok {
			t.Errorf("Validate(%q) err = %v, want ok=%v", base, err, ok)
		}
	}

	cfg := config.Config{BaseURL: "https://a-b.church.tools", MainURL: "http://a.church.tools", LoginTokenPassword: "t"}
	if err := cfg.Validate(); err == nil {
		t.Error("main_url with http must be rejected")
	}
}
