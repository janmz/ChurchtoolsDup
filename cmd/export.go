package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	churchtools "github.com/janmz/churchtools-dup/internal/churchtools"
	config "github.com/janmz/churchtools-dup/internal/config"
	csvfile "github.com/janmz/churchtools-dup/internal/csvfile"
	duplicates "github.com/janmz/churchtools-dup/internal/duplicates"

	"github.com/spf13/cobra"
)

// largeGroupWarning is the group size from which the export asks for a check.
const largeGroupWarning = 2 * duplicates.MaxSharedEmailPersons

var (
	exportOutput          string
	exportCampusFlag      string
	exportInteractive     bool
	exportAllCampuses     bool
	exportSkipPermRequest bool
	exportSkipPreJoin     bool
	exportForce           bool
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Dubletten für einen Standort als CSV exportieren",
	Long: `Lädt alle Personen aus ChurchTools und sucht Dubletten für den gewählten
Standort. Treffer können auch ohne Standort oder mit anderem Standort zugeordnet
sein. Die CSV enthält DupID, ID, Vorname, Nachname, E-Mail, Straße, Stadt,
Standort, Erstellungsdatum und Einladungsstatus (NEU, Eingeladen, Registriert).

Eine vorhandene Ziel-Datei wird nur mit --force überschrieben.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDupExport(cmd)
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)

	exportCmd.Flags().StringVarP(&exportOutput, "output", "o", "duplikate.csv", "Ziel-Datei (- für stdout)")
	exportCmd.Flags().StringVar(&exportCampusFlag, "campus", "", "Standort-ID, eindeutiger Namens-Teilstring oder \"all\"")
	exportCmd.Flags().BoolVarP(&exportInteractive, "interactive", "i", false, "Standort interaktiv auswählen")
	exportCmd.Flags().BoolVar(&exportAllCampuses, "all-campuses", false, "Keinen Standort-Filter anwenden (Alias für --campus all)")
	exportCmd.Flags().BoolVar(&exportSkipPermRequest, "skip-permission-request", false, "Keine Gruppenmitgliedschaft für fehlende Berechtigungen beantragen")
	exportCmd.Flags().BoolVar(&exportSkipPreJoin, "skip-pre-join-groups", false, "Keine Vorab-Gruppen vor dem Export beitreten")
	exportCmd.Flags().BoolVar(&exportForce, "force", false, "Vorhandene Ziel-Datei überschreiben")
}

func runDupExport(cmd *cobra.Command) error {
	if exportOutput == "" {
		return fmt.Errorf("--output ist erforderlich")
	}
	if err := checkPathFlag(cmd, "--output", exportOutput); err != nil {
		return err
	}
	// Check before the (long) export so an edited CSV is not overwritten.
	if exportOutput != "-" && !exportForce {
		if _, err := os.Stat(exportOutput); err == nil {
			return fmt.Errorf("%s existiert bereits – anderen Dateinamen wählen oder --force zum Überschreiben", exportOutput)
		}
	}

	cfg, err := config.LoadOrEmpty(configPath)
	if err != nil {
		return err
	}

	client, err := connectChurchTools(cfg)
	if err != nil {
		return err
	}
	// Leave groups joined only for this run, also when the run fails.
	defer leaveTemporaryGroups(client)

	if !exportSkipPreJoin {
		if err := ensurePreJoinGroups(client, cfg); err != nil {
			return err
		}
	}

	if !exportSkipPermRequest {
		if err := ensureExportPermissions(client, cfg); err != nil {
			return err
		}
	}

	campusChoice, err := resolveExportCampus(client, &cfg, exportInteractive, exportCampusFlag, exportAllCampuses)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Lade Gesamtbestand…\n")
	allPersons, err := client.ListAllPersons()
	if err != nil {
		return err
	}
	if len(allPersons) == 0 {
		return fmt.Errorf("keine Personen gefunden")
	}

	var groups []duplicates.Group
	var report duplicates.MatchReport
	if campusChoice.all {
		groups, report = duplicates.FindAllGroups(allPersons)
	} else {
		groups, report = duplicates.FindGroups(campusChoice.campusID, allPersons)
	}
	printMatchReport(report)
	if len(groups) == 0 {
		if campusChoice.all {
			return fmt.Errorf("keine Dubletten im Gesamtbestand gefunden")
		}
		return fmt.Errorf("keine Dubletten für Standort-ID %d gefunden", campusChoice.campusID)
	}

	campuses, err := client.ListCampuses()
	if err != nil {
		return err
	}
	campusNames := churchtools.CampusNamesByID(campuses)

	fmt.Fprintf(os.Stderr, "Reichere Dubletten mit Personendetails an …\n")
	groups, failures, err := duplicates.EnrichGroups(client, groups, churchtools.EnrichOptions{
		Delay:    time.Duration(cfg.DelayMS) * time.Millisecond,
		Progress: printProgress,
	})
	if err != nil {
		return err
	}
	for _, failure := range failures {
		fmt.Fprintf(os.Stderr, "Warnung: Details für Person %d nicht ladbar (Listendaten verwendet): %v\n", failure.PersonID, failure.Err)
	}
	warnLargeGroups(groups)

	entries := duplicates.GroupsToEntries(groups, campusNames)
	scopeLabel := describeExportScope(campusChoice, "")
	if !campusChoice.all {
		scopeLabel = describeExportScope(campusChoice, campusName(campuses, campusChoice.campusID))
	}

	if exportOutput == "-" {
		if err := csvfile.WriteDupTo(os.Stdout, entries); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "%d Zeilen (%d Dubletten) für %s exportiert\n",
			len(entries), len(groups), scopeLabel)
		return nil
	}

	if err := csvfile.WriteDup(exportOutput, entries); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "%d Zeilen (%d Dubletten) nach %s exportiert (%s)\n",
		len(entries), len(groups), exportOutput, scopeLabel)
	printDeletionHint(exportOutput)
	return nil
}

// printProgress shows enrichment progress on stderr every 25 persons.
func printProgress(done, total int) {
	if done == total || done%25 == 0 {
		fmt.Fprintf(os.Stderr, "  %d/%d Personen geladen\n", done, total)
	}
}

// printMatchReport lists e-mail addresses that were not used for matching.
func printMatchReport(report duplicates.MatchReport) {
	for _, shared := range report.SharedEmails {
		fmt.Fprintf(os.Stderr,
			"Hinweis: E-Mail %q wird von %d Personen genutzt (> %d) und wurde nicht zur Dubletten-Erkennung verwendet.\n",
			maskEmail(shared.Email), shared.Count, duplicates.MaxSharedEmailPersons)
	}
}

// maskEmail shortens the local part for console output (terminal history and
// logs should not keep full addresses): "info@gemeinde.de" → "in***@gemeinde.de".
func maskEmail(email string) string {
	at := strings.LastIndex(email, "@")
	if at < 0 {
		return "***"
	}
	local := []rune(email[:at])
	keep := 2
	if len(local) < keep {
		keep = len(local)
	}
	return string(local[:keep]) + "***" + email[at:]
}

// printDeletionHint reminds the user that the CSV holds personal data
// (GDPR storage limitation).
func printDeletionHint(path string) {
	fmt.Fprintf(os.Stderr,
		"Datenschutz: %s enthält personenbezogene Daten. Datei nach Abschluss von Prüfung und Import löschen.\n", path)
}

// warnLargeGroups asks for a manual check of unusually large DupIDs.
func warnLargeGroups(groups []duplicates.Group) {
	for _, group := range groups {
		if len(group.Persons) >= largeGroupWarning {
			fmt.Fprintf(os.Stderr, "Warnung: DupID %d enthält %d Personen – bitte vor dem Import prüfen.\n",
				group.DupID, len(group.Persons))
		}
	}
}

func describeCampus(campusID int, name string) string {
	if name != "" {
		return fmt.Sprintf("Standort %q, ID %d", name, campusID)
	}
	return fmt.Sprintf("Standort-ID %d", campusID)
}
