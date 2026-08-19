package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func captureExportCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()

	exportOutput = ""
	exportCampusFlag = ""
	exportInteractive = false
	exportAllCampuses = false
	exportSkipPermRequest = false
	exportSkipPreJoin = false

	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(append([]string{"export"}, args...))

	configureCLIErrors(rootCmd)
	err := rootCmd.Execute()
	if err != nil {
		printCLIError(err)
	}
	return buf.String(), err
}

func TestPathFlagErrorMatchesUnknownFlagFormat(t *testing.T) {
	pathOut, pathErr := captureExportCLI(t, "-o", "-i")
	if pathErr == nil {
		t.Fatal("expected path flag error")
	}

	flagOut, flagErr := captureExportCLI(t, "--invalid")
	if flagErr == nil {
		t.Fatal("expected unknown flag error")
	}

	for _, out := range []string{pathOut, flagOut} {
		if !strings.Contains(out, "Error:") {
			t.Fatalf("output missing Error prefix: %q", out)
		}
		if !strings.Contains(out, "Usage:") {
			t.Fatalf("output missing Usage section: %q", out)
		}
		if !strings.Contains(out, "Flags:") {
			t.Fatalf("output missing Flags section: %q", out)
		}
		if !strings.Contains(out, "Global Flags:") {
			t.Fatalf("output missing Global Flags section: %q", out)
		}
	}
}

func TestHelpAsPathFlagShowsFullUsage(t *testing.T) {
	out, err := captureExportCLI(t, "-o", "--help")
	if err == nil {
		t.Fatal("expected path flag error")
	}
	if !strings.Contains(out, `--output: "--help" sieht nach einer Option aus`) {
		t.Fatalf("unexpected error text: %q", out)
	}
	if !strings.Contains(out, "Usage:") || !strings.Contains(out, "Flags:") {
		t.Fatalf("expected full usage output: %q", out)
	}
}
