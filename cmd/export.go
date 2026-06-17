package cmd

import (
	"fmt"
	"os"

	churchtools "github.com/janmz/churchtools-dup/internal/churchtools"
	config "github.com/janmz/churchtools-dup/internal/config"
	csvfile "github.com/janmz/churchtools-dup/internal/csvfile"
	duplicates "github.com/janmz/churchtools-dup/internal/duplicates"

	"github.com/spf13/cobra"
)

var (
	exportOutput          string
	exportCampus          int
	exportInteractive     bool
	exportSkipPermRequest bool
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Dubletten für einen Standort als CSV exportieren",
	Long: `Lädt alle Personen aus ChurchTools und sucht Dubletten für den gewählten
Standort. Treffer können auch ohne Standort oder mit anderem Standort zugeordnet
sein. Die CSV enthält DupID, ID, Vorname, Nachname, E-Mail, Straße, Stadt,
Standort, Erstellungsdatum und Einwilligungsdatum.`,
	Run: func(cmd *cobra.Command, args []string) {
		exitOnError(runDupExport())
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)

	exportCmd.Flags().StringVarP(&exportOutput, "output", "o", "duplikate.csv", "Ziel-Datei (- für stdout)")
	exportCmd.Flags().IntVar(&exportCampus, "campus-id", 0, "Standort-ID für die Dubletten-Suche")
	exportCmd.Flags().BoolVarP(&exportInteractive, "interactive", "i", false, "Standort interaktiv auswählen")
	exportCmd.Flags().BoolVar(&exportSkipPermRequest, "skip-permission-request", false, "Keine Gruppenmitgliedschaft für fehlende Berechtigungen beantragen")
}

func runDupExport() error {
	if exportOutput == "" {
		return fmt.Errorf("--output ist erforderlich")
	}

	cfg, err := config.LoadOrEmpty(configPath)
	if err != nil {
		return err
	}

	client, err := connectChurchTools(cfg)
	if err != nil {
		return err
	}

	if !exportSkipPermRequest {
		if err := ensureExportPermissions(client, cfg); err != nil {
			return err
		}
	}

	campusID := exportCampus
	if exportInteractive || campusID <= 0 {
		campusID, err = ensureCampusID(client, &cfg)
		if err != nil {
			return err
		}
	}
	if campusID <= 0 {
		return fmt.Errorf("Kein Standort gewählt – --campus-id setzen oder --interactive nutzen")
	}

	fmt.Fprintf(os.Stderr, "Lade Gesamtbestand…\n")
	allPersons, err := client.ListAllPersons()
	if err != nil {
		return err
	}
	if len(allPersons) == 0 {
		return fmt.Errorf("Keine Personen gefunden")
	}

	groups := duplicates.FindGroups(campusID, allPersons)
	if len(groups) == 0 {
		return fmt.Errorf("keine Dubletten für Standort-ID %d gefunden", campusID)
	}

	campuses, err := client.ListCampuses()
	if err != nil {
		return err
	}
	campusNames := churchtools.CampusNamesByID(campuses)

	fmt.Fprintf(os.Stderr, "Reichere Dubletten mit Personendetails an …\n")
	groups, err = duplicates.EnrichGroups(client, groups)
	if err != nil {
		return err
	}

	entries := duplicates.GroupsToEntries(groups, campusNames)
	campusName := campusDisplayName(client, campusID)

	if exportOutput == "-" {
		if err := csvfile.WriteDupTo(os.Stdout, entries); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "%d Zeilen (%d Dubletten) für %s exportiert\n",
			len(entries), len(groups), describeCampus(campusID, campusName))
		return nil
	}

	if err := csvfile.WriteDup(exportOutput, entries); err != nil {
		return err
	}

	fmt.Printf("%d Zeilen (%d Dubletten) nach %s exportiert (%s)\n",
		len(entries), len(groups), exportOutput, describeCampus(campusID, campusName))
	return nil
}

func describeCampus(campusID int, name string) string {
	if name != "" {
		return fmt.Sprintf("Standort %q, ID %d", name, campusID)
	}
	return fmt.Sprintf("Standort-ID %d", campusID)
}
