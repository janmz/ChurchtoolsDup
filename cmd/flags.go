package cmd

import (
	"fmt"
	"strings"
)

// validatePathFlagValue rejects values that look like another CLI flag instead of
// a file path. "-" alone is allowed (stdout for export).
func validatePathFlagValue(flagName, value string) error {
	if value == "" || value == "-" {
		return nil
	}
	if strings.HasPrefix(value, "-") {
		return fmt.Errorf(
			"%s: %q sieht nach einer Option aus, nicht nach einem Dateinamen (fehlender Dateiname?)",
			flagName,
			value,
		)
	}
	return nil
}
