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

func ensurePreJoinGroups(client *churchtools.Client, cfg config.Config) error {
	names := cfg.PreJoinGroupNames()
	if len(names) == 0 {
		return nil
	}

	results, err := client.EnsurePreJoinGroups(names)
	if err != nil {
		return err
	}

	for _, result := range results {
		switch {
		case result.Skipped:
			fmt.Fprintf(os.Stderr, "Vorab-Gruppe %q: Bereits Mitglied\n", result.GroupName)
		case result.Status == churchtools.MembershipActive:
			msg := result.Message
			if msg == "" {
				msg = "Mitgliedschaft aktiv"
			}
			fmt.Fprintf(os.Stderr, "Vorab-Gruppe %q: %s\n", result.GroupName, msg)
		case result.Status == churchtools.MembershipRequested:
			fmt.Fprintf(os.Stderr, "Vorab-Gruppe %q: beantragt", result.GroupName)
			if result.Message != "" {
				fmt.Fprintf(os.Stderr, " (%s)", result.Message)
			}
			fmt.Fprintln(os.Stderr)
		default:
			msg := result.Message
			if msg == "" {
				msg = "Beitritt nicht möglich"
			}
			fmt.Fprintf(os.Stderr, "Vorab-Gruppe %q: %s\n", result.GroupName, msg)
		}
	}
	return nil
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
