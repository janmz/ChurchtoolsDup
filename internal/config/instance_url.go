package config

import (
	"errors"
	"fmt"
	"strings"
)

// BaseURLFromInstanceName builds https://{name}.church.tools from an instance
// name. Also accepts names with a .church.tools suffix or a full base URL.
func BaseURLFromInstanceName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("instanzname fehlt")
	}

	if strings.Contains(name, "://") {
		return NormalizeBaseURL(name), nil
	}

	name = strings.TrimPrefix(name, "https://")
	name = strings.TrimPrefix(name, "http://")
	name = strings.TrimSuffix(name, "/")
	name = strings.TrimSuffix(name, ".church.tools")

	if name == "" {
		return "", errors.New("instanzname fehlt")
	}
	if strings.ContainsAny(name, "/ \t") {
		return "", fmt.Errorf("ungültiger instanzname: %q", name)
	}
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' {
			continue
		}
		return "", fmt.Errorf("ungültiger instanzname: %q", name)
	}

	return "https://" + strings.ToLower(name) + ".church.tools", nil
}
