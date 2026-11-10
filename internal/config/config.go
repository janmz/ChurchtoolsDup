package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/janmz/churchtools-dup/internal/fileacl"
	"github.com/janmz/churchtools-dup/internal/iolimit"
)

const DefaultConfigName = "config.json"

// Config holds ChurchTools connection settings.
//
// Secrets are stored with github.com/janmz/sconfig: a plaintext value in
// "login_token" or "password" is encrypted (machine-bound) into
// "login_token_secure" / "secure_password" on the first load and replaced by
// a marker. In memory the fields hold the decrypted values.
type Config struct {
	BaseURL string `json:"base_url"`
	// MainURL is the main (central) instance for a sub-instance login. It is
	// only used when set explicitly; no host is derived automatically.
	MainURL                  string           `json:"main_url,omitempty"`
	LoginTokenPassword       string           `json:"login_token,omitempty"`
	LoginTokenSecurePassword string           `json:"login_token_secure,omitempty"`
	Username                 string           `json:"username,omitempty"`
	Password                 string           `json:"password,omitempty"`
	SecurePassword           string           `json:"secure_password,omitempty"`
	DelayMS                  int              `json:"delay_ms,omitempty"`
	CampusID                 int              `json:"campus_id,omitempty"`
	PreJoinGroups            string           `json:"pre_join_groups,omitempty"`
	PermissionGroups         PermissionGroups `json:"permission_groups,omitempty"`
	DuplicateRelType         DuplicateRelType `json:"duplicate_relationship_type,omitempty"`
}

// DuplicateRelType optionally pins the ChurchTools relationship type for duplicates.
type DuplicateRelType struct {
	ID   int    `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// Central defaults. Change them only here.
const (
	// DefaultDuplicateRelationshipTypeID is tried when neither
	// duplicate_relationship_type.id nor .name is configured.
	DefaultDuplicateRelationshipTypeID = 8
	// DefaultDuplicateGroupName is the group primaries are added to on import.
	DefaultDuplicateGroupName = "Duplikate"
	// DefaultDelayMS is the pause between API calls in milliseconds.
	DefaultDelayMS = 500
)

// Default returns a configuration with all defaults applied.
func Default() Config {
	return Config{DelayMS: DefaultDelayMS}
}

// DuplicateRelationshipTypeID returns the explicitly configured duplicate
// relationship type ID, or 0 when none is set (then the name, the default ID
// and a keyword search are tried in that order).
func (c Config) DuplicateRelationshipTypeID() int {
	if c.DuplicateRelType.ID > 0 {
		return c.DuplicateRelType.ID
	}
	return 0
}

// PermissionGroups names ChurchTools groups used to request missing rights.
type PermissionGroups struct {
	EditPersons   string `json:"edit_persons,omitempty"`
	ExportPersons string `json:"export_persons,omitempty"`
}

var defaultEditPersonsGroups = []string{
	"Personen Administration",
	"Personen bearbeiten",
}

var defaultExportPersonsGroups = []string{
	"Personen exportieren",
}

// DefaultPreJoinGroups is the comma-separated list of groups joined before export/import.
const DefaultPreJoinGroups = "ChurchTools Verwaltung,Gruppen Administration,ChurchTools Admin,Personen Administration,Personen verwalten"

// ParseCommaSeparatedNames splits a comma-separated list and trims empty entries.
func ParseCommaSeparatedNames(raw string) []string {
	parts := strings.Split(raw, ",")
	names := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		names = append(names, part)
	}
	return names
}

// PreJoinGroupNames returns groups to join in order before export/import.
// Set pre_join_groups to "-" or "none" to disable.
func (c Config) PreJoinGroupNames() []string {
	raw := strings.TrimSpace(c.PreJoinGroups)
	if raw == "-" || strings.EqualFold(raw, "none") {
		return nil
	}
	if raw == "" {
		return ParseCommaSeparatedNames(DefaultPreJoinGroups)
	}
	return ParseCommaSeparatedNames(raw)
}

// EditPersonsGroupNames returns candidate groups for write/admin permissions.
func (c Config) EditPersonsGroupNames() []string {
	if name := strings.TrimSpace(c.PermissionGroups.EditPersons); name != "" {
		return []string{name}
	}
	return append([]string(nil), defaultEditPersonsGroups...)
}

// EditPersonsGroupName returns the primary group for write access.
func (c Config) EditPersonsGroupName() string {
	names := c.EditPersonsGroupNames()
	if len(names) == 0 {
		return ""
	}
	return names[0]
}

// ExportPersonsGroupNames returns candidate groups for export permission.
func (c Config) ExportPersonsGroupNames() []string {
	if name := strings.TrimSpace(c.PermissionGroups.ExportPersons); name != "" {
		return []string{name}
	}
	return append([]string(nil), defaultExportPersonsGroups...)
}

// ExportPersonsGroupName returns the primary group for export permission.
func (c Config) ExportPersonsGroupName() string {
	names := c.ExportPersonsGroupNames()
	if len(names) == 0 {
		return ""
	}
	return names[0]
}

// Load reads configuration from a JSON file and applies environment overrides.
func Load(path string) (Config, error) {
	cfg, err := LoadFile(path)
	if err != nil {
		return cfg, err
	}

	cfg.applyEnv()
	if err := cfg.Validate(); err != nil {
		return cfg, err
	}

	return cfg, nil
}

// LoadFile reads only the JSON file (defaults applied, no environment
// overrides, no validation). Plaintext secrets in the file are encrypted on
// disk by sconfig; the returned struct holds the decrypted values. Use it
// when the file is to be modified and saved again, so values from the
// environment do not end up on disk.
func LoadFile(path string) (Config, error) {
	cfg := Default()
	// Same limit and error text as for CSV files, before sconfig parses it.
	if _, err := iolimit.ReadFile(path, iolimit.MaxConfigBytes); err != nil {
		return cfg, fmt.Errorf("config lesen (%s): %w", path, err)
	}
	if err := loadSecure(&cfg, path); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// LoadOrEmpty returns config from file when present, otherwise environment only.
func LoadOrEmpty(path string) (Config, error) {
	if path == "" {
		path = DefaultConfigName
	}

	if !Exists(path) {
		cfg := Default()
		cfg.applyEnv()
		return cfg, cfg.Validate()
	}

	return Load(path)
}

// Exists reports whether a config file exists at path.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return !errors.Is(err, os.ErrNotExist)
}

// SaveCampusID stores campus_id in the config file without touching any other
// key. Unknown keys are preserved and environment values are never written.
// A missing file is created.
func SaveCampusID(path string, campusID int) error {
	if path == "" {
		path = DefaultConfigName
	}

	raw := map[string]any{}
	if Exists(path) {
		data, err := iolimit.ReadFile(path, iolimit.MaxConfigBytes)
		if err != nil {
			return fmt.Errorf("config lesen (%s): %w", path, err)
		}
		if err := json.Unmarshal(data, &raw); err != nil {
			return fmt.Errorf("config parsen: %w", err)
		}
	}
	raw["campus_id"] = campusID
	return writeJSON(path, raw)
}

// Save writes the configuration with encrypted secrets and restricts the
// file to the current user.
func Save(path string, cfg Config) error {
	if path == "" {
		path = DefaultConfigName
	}
	return saveSecure(&cfg, path)
}

// writeJSON writes value as indented JSON with restrictive permissions.
func writeJSON(path string, value any) error {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("config-verzeichnis anlegen: %w", err)
		}
	}

	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("config serialisieren: %w", err)
	}

	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("config speichern: %w", err)
	}

	return fileacl.RestrictToOwner(path)
}

func (c *Config) applyEnv() {
	if v := strings.TrimSpace(os.Getenv("CT_BASE_URL")); v != "" {
		c.BaseURL = v
	}
	if v := strings.TrimSpace(os.Getenv("CT_MAIN_URL")); v != "" {
		c.MainURL = v
	}
	if v := strings.TrimSpace(os.Getenv("CT_LOGIN_TOKEN")); v != "" {
		c.LoginTokenPassword = v
	}
	if v := strings.TrimSpace(os.Getenv("CT_USERNAME")); v != "" {
		c.Username = v
	}
	// The password is taken verbatim: spaces may be part of it.
	if v := os.Getenv("CT_PASSWORD"); strings.TrimSpace(v) != "" {
		c.Password = v
	}
	if v := strings.TrimSpace(os.Getenv("CT_PRE_JOIN_GROUPS")); v != "" {
		c.PreJoinGroups = v
	}
}

// Validate checks required fields.
func (c *Config) Validate() error {
	if strings.TrimSpace(c.BaseURL) == "" {
		return errors.New("base_url fehlt (config oder CT_BASE_URL)")
	}

	c.BaseURL = NormalizeBaseURL(c.BaseURL)
	if err := validateInstanceURL("base_url", c.BaseURL); err != nil {
		return err
	}
	if strings.TrimSpace(c.MainURL) != "" {
		c.MainURL = NormalizeBaseURL(c.MainURL)
		if err := validateInstanceURL("main_url", c.MainURL); err != nil {
			return err
		}
	}

	hasToken := strings.TrimSpace(c.LoginTokenPassword) != ""
	hasPassword := strings.TrimSpace(c.Username) != "" && strings.TrimSpace(c.Password) != ""
	if !hasToken && !hasPassword {
		return errors.New("login_token oder username/password erforderlich")
	}

	return nil
}

// NormalizeBaseURL removes trailing slashes and /api suffixes. When raw is only
// an instance name (no scheme), https://{name}.church.tools is assumed.
func NormalizeBaseURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if !strings.Contains(raw, "://") {
		if url, err := BaseURLFromInstanceName(raw); err == nil {
			return url
		}
	}
	url := raw
	url = strings.TrimSuffix(url, "/")
	url = strings.TrimSuffix(url, "/api")
	return url
}
