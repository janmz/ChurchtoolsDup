package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// flagUsageError triggers the same Error + Usage output as unknown CLI flags.
type flagUsageError struct {
	err error
	cmd *cobra.Command
}

func (e *flagUsageError) Error() string {
	return e.err.Error()
}

// checkPathFlag validates a path flag and wraps errors for unified CLI output.
func checkPathFlag(cmd *cobra.Command, flagName, value string) error {
	if err := validatePathFlagValue(flagName, value); err != nil {
		return &flagUsageError{err: err, cmd: cmd}
	}
	return nil
}

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
