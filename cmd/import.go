package cmd

import (
	"fmt"
	"time"

	churchtools "github.com/janmz/churchtools-dup/internal/churchtools"
	config "github.com/janmz/churchtools-dup/internal/config"
	csvfile "github.com/janmz/churchtools-dup/internal/csvfile"
	duplicates "github.com/janmz/churchtools-dup/internal/duplicates"

	"github.com/spf13/cobra"
)

var (
	importCSVPath           string
	importDryRun            bool
	importDelayMS           int
	importSkipPermRequest   bool
	importSkipGroupAdd      bool
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Bearbeitete Dubletten-CSV importieren und zur Zusammenführung vormerken",
	Long: `Liest eine Dubletten-CSV und verknüpft die verbleibenden Personen pro DupID
über das Beziehungsmanagement als Duplikate. Der erste Eintrag jeder DupID wird
zusätzlich in die Gruppe "Duplikate" aufgenommen.`,
	Run: func(cmd *cobra.Command, args []string) {
		exitOnError(runDupImport())
	},
}

func init() {
	rootCmd.AddCommand(importCmd)

	importCmd.Flags().StringVarP(&importCSVPath, "csv", "f", "", "Pfad zur Dubletten-CSV (Pflicht)")
	importCmd.Flags().BoolVar(&importDryRun, "dry-run", false, "Prüfen/simulieren ohne Änderungen in ChurchTools")
	importCmd.Flags().IntVar(&importDelayMS, "delay-ms", 0, "Pause zwischen API-Aufrufen (0 = config.delay_ms)")
	importCmd.Flags().BoolVar(&importSkipPermRequest, "skip-permission-request", false, "Keine Gruppenmitgliedschaft für fehlende Berechtigungen beantragen")
	importCmd.Flags().BoolVar(&importSkipGroupAdd, "skip-group-add", false, "Personen nicht in die Gruppe Duplikate aufnehmen")
	_ = importCmd.MarkFlagRequired("csv")
}

func runDupImport() error {
	if importCSVPath == "" {
		return fmt.Errorf("--csv ist erforderlich")
	}

	cfg, err := config.LoadOrEmpty(configPath)
	if err != nil {
		return err
	}

	entries, err := csvfile.ReadDup(importCSVPath)
	if err != nil {
		return err
	}

	client, err := connectChurchTools(cfg)
	if err != nil {
		return err
	}

	if !importSkipPermRequest {
		if err := ensureImportPermissions(client, cfg); err != nil {
			return err
		}
	}

	relType, err := client.FindDuplicateRelationshipType(churchtools.DuplicateRelationshipOptions{
		TypeID:   cfg.DuplicateRelationshipTypeID(),
		TypeName: cfg.DuplicateRelType.Name,
	})
	if err != nil {
		return fmt.Errorf("Beziehungstyp: %w", err)
	}

	delay := time.Duration(cfg.DelayMS) * time.Millisecond
	if importDelayMS > 0 {
		delay = time.Duration(importDelayMS) * time.Millisecond
	}

	groups := csvfile.GroupDupEntries(entries)
	runner := duplicates.ImportRunner{
		Client:       client,
		RelType:      relType,
		GroupName:    churchtools.DuplicateGroupName,
		SkipGroupAdd: importSkipGroupAdd,
	}

	if importDryRun {
		fmt.Printf("Dry-Run für %d DupID-Gruppen (Beziehungstyp %q, ID %d) …\n",
			len(groups), relType.Name, relType.ID)
	} else {
		fmt.Printf("Importiere %d DupID-Gruppen …\n", len(groups))
	}

	results, err := runner.Run(groups, duplicates.ImportOptions{
		DryRun: importDryRun,
		Delay:  delay,
	})
	if err != nil {
		return err
	}

	duplicates.PrintImportSummary(results)

	for _, result := range results {
		if !result.Success && !duplicates.IsSkippedImportResult(result) {
			return fmt.Errorf("mindestens eine DupID-Gruppe ist fehlgeschlagen")
		}
	}
	return nil
}
