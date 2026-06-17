package cmd

import (
	"fmt"
	"os"

	churchtools "github.com/janmz/churchtools-dup/internal/churchtools"
	config "github.com/janmz/churchtools-dup/internal/config"
)

func applyDefaultCampus(client *churchtools.Client, cfg *config.Config, opts *churchtools.PersonListOptions) error {
	if opts.CampusID > 0 {
		return nil
	}

	campusID, err := ensureCampusID(client, cfg)
	if err != nil {
		return err
	}
	if campusID <= 0 {
		return nil
	}

	opts.CampusID = campusID
	name := campusDisplayName(client, campusID)
	if name != "" {
		fmt.Fprintf(os.Stderr, "Standort automatisch: %s (ID %d)\n", name, campusID)
	} else {
		fmt.Fprintf(os.Stderr, "Standort automatisch: ID %d\n", campusID)
	}
	return nil
}

func ensureCampusID(client *churchtools.Client, cfg *config.Config) (int, error) {
	campusID, err := client.CurrentUserCampusID()
	if err != nil {
		return 0, err
	}
	if campusID > 0 {
		return campusID, nil
	}
	if cfg.CampusID > 0 {
		return cfg.CampusID, nil
	}

	campuses, err := client.ListCampuses()
	if err != nil {
		return 0, err
	}

	campusItems := make([]menuItem, len(campuses))
	for i, campus := range campuses {
		campusItems[i] = menuItem{id: campus.ID, name: campus.Name}
	}

	selectedID, err := promptMenu(
		"Kein Standort am Benutzer hinterlegt – bitte Standort auswählen (wird in config gespeichert)",
		campusItems,
		false,
	)
	if err != nil {
		return 0, err
	}

	cfg.CampusID = selectedID
	path := configPath
	if path == "" {
		path = config.DefaultConfigName
	}
	if err := config.Save(path, *cfg); err != nil {
		return 0, err
	}

	name := campusName(campuses, selectedID)
	fmt.Fprintf(os.Stderr, "Standort %q (ID %d) in %s gespeichert.\n", name, selectedID, path)
	return selectedID, nil
}

func campusDisplayName(client *churchtools.Client, campusID int) string {
	campuses, err := client.ListCampuses()
	if err != nil {
		return ""
	}
	return campusName(campuses, campusID)
}

func campusName(campuses []churchtools.Campus, campusID int) string {
	for _, campus := range campuses {
		if campus.ID == campusID {
			return campus.Name
		}
	}
	return ""
}
