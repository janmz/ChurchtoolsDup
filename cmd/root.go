package cmd

import (
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

// Execute runs the root command.
func Execute(versionString string) error {
	rootCmd.Version = versionString
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "config.json", "Pfad zur Konfigurationsdatei")

	rootCmd.AddCommand(whoamiCmd)
	rootCmd.AddCommand(setupCmd)
}

func exitOnError(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
