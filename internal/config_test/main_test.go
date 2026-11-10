package config_test

import (
	"os"
	"testing"

	config "github.com/janmz/churchtools-dup/internal/config"
)

// TestMain lets sconfig accept config files in the temp directory used by
// t.TempDir() and fixes the machine identity used for encryption.
func TestMain(m *testing.M) {
	config.SetTestEnvironment(os.TempDir())
	os.Exit(m.Run())
}
