package cmd

import (
	"fmt"
	"os"

	churchtools "github.com/janmz/churchtools-dup/internal/churchtools"
	config "github.com/janmz/churchtools-dup/internal/config"
)

func printPermissionNotes(notes []string) {
	for _, note := range notes {
		fmt.Fprintf(os.Stderr, "Berechtigung: %s\n", note)
	}
}

func ensureExportPermissions(client *churchtools.Client, cfg config.Config) error {
	notes, err := client.EnsurePermissions([]churchtools.PermissionRequirement{
		{
			Module:      churchtools.ModuleChurchDB,
			Permission:  churchtools.PermExportData,
			GroupNames:  cfg.ExportPersonsGroupNames(),
			Description: "Personen exportieren",
		},
	})
	if err != nil {
		return err
	}
	printPermissionNotes(notes)
	return nil
}

func ensureImportPermissions(client *churchtools.Client, cfg config.Config) error {
	notes, err := client.EnsurePermissions([]churchtools.PermissionRequirement{
		{
			Module:      churchtools.ModuleChurchDB,
			Permission:  churchtools.PermEditRelations,
			GroupNames:  cfg.EditPersonsGroupNames(),
			Description: "Beziehungen bearbeiten (edit relations)",
		},
	})
	if err != nil {
		return err
	}
	printPermissionNotes(notes)
	return nil
}
