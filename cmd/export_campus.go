package cmd

import (
	"fmt"
	"strconv"
	"strings"

	churchtools "github.com/janmz/churchtools-dup/internal/churchtools"
	config "github.com/janmz/churchtools-dup/internal/config"
)

const campusMenuAll = -1

type exportCampusChoice struct {
	all      bool
	campusID int
}

func resolveExportCampus(
	client *churchtools.Client,
	cfg *config.Config,
	interactive bool,
	campusFlag string,
) (exportCampusChoice, error) {
	campusFlag = strings.TrimSpace(campusFlag)
	if campusFlag != "" {
		return parseExportCampusFlag(campusFlag)
	}

	if interactive {
		return promptExportCampus(client)
	}

	campusID, err := ensureCampusID(client, cfg)
	if err != nil {
		return exportCampusChoice{}, err
	}
	if campusID <= 0 {
		return exportCampusChoice{}, fmt.Errorf("Kein Standort gewählt – --campus-id setzen oder --interactive nutzen")
	}
	return exportCampusChoice{campusID: campusID}, nil
}

func parseExportCampusFlag(value string) (exportCampusChoice, error) {
	value = strings.TrimSpace(value)
	switch strings.ToLower(value) {
	case "all", "alle", "*":
		return exportCampusChoice{all: true}, nil
	}

	id, err := strconv.Atoi(value)
	if err != nil {
		return exportCampusChoice{}, fmt.Errorf("--campus-id: Zahl oder \"all\" erwartet, erhalten %q", value)
	}
	if id <= 0 {
		return exportCampusChoice{}, fmt.Errorf("--campus-id muss positiv sein")
	}
	return exportCampusChoice{campusID: id}, nil
}

func promptExportCampus(client *churchtools.Client) (exportCampusChoice, error) {
	campuses, err := client.ListCampuses()
	if err != nil {
		return exportCampusChoice{}, err
	}

	items := make([]menuItem, 0, len(campuses)+1)
	items = append(items, menuItem{id: campusMenuAll, name: "Alle Standorte"})
	for _, campus := range campuses {
		items = append(items, menuItem{id: campus.ID, name: campus.Name})
	}

	selectedID, err := promptMenu("Standort für Dubletten-Suche", items, false)
	if err != nil {
		return exportCampusChoice{}, err
	}
	if selectedID == campusMenuAll {
		return exportCampusChoice{all: true}, nil
	}
	return exportCampusChoice{campusID: selectedID}, nil
}

func describeExportScope(choice exportCampusChoice, campusName string) string {
	if choice.all {
		return "alle Standorte"
	}
	return describeCampus(choice.campusID, campusName)
}
