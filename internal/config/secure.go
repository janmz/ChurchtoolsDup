package config

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/janmz/churchtools-dup/internal/fileacl"
	"github.com/janmz/sconfig/v3"
)

// hardwareIDFunc replaces the machine identity for sconfig (tests only).
var hardwareIDFunc func() (uint64, error)

// SetTestEnvironment prepares sconfig for tests: config files below homeDir
// are accepted (sconfig only allows paths under the executable directory,
// the working directory or the home directory) and a fixed machine identity
// makes encryption independent of the test host. Not for production use.
func SetTestEnvironment(homeDir string) {
	sconfig.ResetForTest()
	sconfig.SetHomeDirForTest(homeDir)
	hardwareIDFunc = func() (uint64, error) { return 0x5eed, nil }
}

// loadSecure reads path with sconfig: plaintext secrets are encrypted in the
// file, the struct receives decrypted values. The file is then restricted to
// the current user, because sconfig may have rewritten it.
func loadSecure(cfg *Config, path string) error {
	if err := loadSecureNoACL(cfg, path); err != nil {
		return err
	}
	return fileacl.RestrictToOwner(path)
}

// saveSecure writes cfg with encrypted secrets and restricts the file.
func saveSecure(cfg *Config, path string) error {
	// sconfig.UpdateConfig requires an initialised key state: load once
	// (into a scratch value; a missing file is not created by this).
	scratch := Default()
	if err := loadSecureNoACL(&scratch, path); err != nil {
		return err
	}
	if err := sconfig.UpdateConfig(cfg, path); err != nil {
		return fmt.Errorf("config speichern: %w", err)
	}
	return fileacl.RestrictToOwner(path)
}

func loadSecureNoACL(cfg *Config, path string) error {
	var err error
	if hardwareIDFunc != nil {
		err = sconfig.LoadConfig(cfg, 0, path, false, false, hardwareIDFunc)
	} else {
		err = sconfig.LoadConfig(cfg, 0, path, false, false)
	}
	if err != nil {
		return fmt.Errorf("config lesen (%s): %w", path, err)
	}
	return nil
}

// validateInstanceURL accepts only https URLs; plain http is allowed for
// localhost (tests, local proxies) because credentials never leave the PC.
func validateInstanceURL(field, raw string) error {
	if raw == "" {
		return nil
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return fmt.Errorf("%s ungültig: %q", field, raw)
	}
	switch strings.ToLower(parsed.Scheme) {
	case "https":
		return nil
	case "http":
		host := strings.ToLower(parsed.Hostname())
		if host == "localhost" || host == "127.0.0.1" || host == "::1" {
			return nil
		}
		return fmt.Errorf("%s muss https verwenden (http nur für localhost): %q", field, raw)
	default:
		return fmt.Errorf("%s: nicht unterstütztes Schema %q", field, parsed.Scheme)
	}
}
