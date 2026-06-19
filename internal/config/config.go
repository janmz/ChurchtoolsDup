package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const DefaultConfigName = "config.json"

// Config holds ChurchTools connection settings.
type Config struct {
	BaseURL          string           `json:"base_url"`
	LoginToken       string           `json:"login_token,omitempty"`
	Username         string           `json:"username,omitempty"`
	Password         string           `json:"password,omitempty"`
	DelayMS          int              `json:"delay_ms,omitempty"`
	CampusID         int              `json:"campus_id,omitempty"`
	PreJoinGroups    string           `json:"pre_join_groups,omitempty"`
	PermissionGroups PermissionGroups `json:"permission_groups,omitempty"`
	DuplicateRelType DuplicateRelType `json:"duplicate_relationship_type,omitempty"`
}

// DuplicateRelType optionally pins the ChurchTools relationship type for duplicates.
type DuplicateRelType struct {
	ID   int    `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// DefaultDuplicateRelationshipTypeID is used when duplicate_relationship_type.id is omitted.
const DefaultDuplicateRelationshipTypeID = 8

// DuplicateRelationshipTypeID returns the configured duplicate relationship type ID.
func (c Config) DuplicateRelationshipTypeID() int {
	if c.DuplicateRelType.ID > 0 {
		return c.DuplicateRelType.ID
	}
	return DefaultDuplicateRelationshipTypeID
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
const DefaultPreJoinGroups = "ChurchTools Admin,ChurchTools Verwaltung,Personen Administration,Personen verwalten"

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
	cfg := Config{DelayMS: 500}

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("config lesen (%s): %w", path, err)
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("config parsen: %w", err)
	}

	cfg.applyEnv()
	if err := cfg.Validate(); err != nil {
		return cfg, err
	}

	return cfg, nil
}

// LoadOrEmpty returns config from file when present, otherwise environment only.
func LoadOrEmpty(path string) (Config, error) {
	if path == "" {
		path = DefaultConfigName
	}

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		cfg := Config{DelayMS: 500}
		cfg.applyEnv()
		return cfg, cfg.Validate()
	}

	return Load(path)
}

// Save writes configuration to disk with restrictive permissions.
func Save(path string, cfg Config) error {
	if path == "" {
		path = DefaultConfigName
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && !errors.Is(err, os.ErrExist) {
		dir := filepath.Dir(path)
		if dir != "." {
			return fmt.Errorf("config-verzeichnis anlegen: %w", err)
		}
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("config serialisieren: %w", err)
	}

	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("config speichern: %w", err)
	}

	return nil
}

func (c *Config) applyEnv() {
	if v := strings.TrimSpace(os.Getenv("CT_BASE_URL")); v != "" {
		c.BaseURL = v
	}
	if v := strings.TrimSpace(os.Getenv("CT_LOGIN_TOKEN")); v != "" {
		c.LoginToken = v
	}
	if v := strings.TrimSpace(os.Getenv("CT_USERNAME")); v != "" {
		c.Username = v
	}
	if v := strings.TrimSpace(os.Getenv("CT_PASSWORD")); v != "" {
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

	hasToken := strings.TrimSpace(c.LoginToken) != ""
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
