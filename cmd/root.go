package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var configPath string

var rootCmd = &cobra.Command{
	Use:   "Churchtools-Dup",
	Short: "ChurchTools-Dubletten finden und zur Zusammenführung vormerken",
	Long: `ChurchTools-Dup sucht für einen Standort Dubletten im Gesamtbestand,
exportiert sie als CSV und kann bearbeitete Listen wieder importieren.

Nutze 'setup' für Ersteinrichtung von URL, Login-Token und Berechtigungsprüfung.`,
	Version: "undefined",
}

// printedCLIError marks errors whose message and usage were already written.
type printedCLIError struct {
	err error
}

func (e *printedCLIError) Error() string {
	return e.err.Error()
}

// Execute runs the root command.
func Execute(versionString string) error {
	rootCmd.Version = versionString
	configureCLIErrors(rootCmd)

	err := rootCmd.Execute()
	if err != nil {
		printCLIError(err)
	}
	return err
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "config.json", "Pfad zur Konfigurationsdatei")

	rootCmd.AddCommand(whoamiCmd)
	rootCmd.AddCommand(setupCmd)
}

func configureCLIErrors(cmd *cobra.Command) {
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetFlagErrorFunc(reportFlagError)
}

func reportFlagError(c *cobra.Command, err error) error {
	fmt.Fprintf(c.ErrOrStderr(), "Error: %s\n", err)
	_ = c.Usage()
	return &printedCLIError{err: err}
}

func printCLIError(err error) {
	var printed *printedCLIError
	if errors.As(err, &printed) {
		return
	}

	var usage *flagUsageError
	if errors.As(err, &usage) {
		reportFlagError(usage.cmd, usage.err)
		return
	}

	fmt.Fprintln(os.Stderr, err)
}

func exitOnError(err error) {
	if err != nil {
		printCLIError(err)
		os.Exit(1)
	}
}
