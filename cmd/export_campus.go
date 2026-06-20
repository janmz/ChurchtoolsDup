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
	allCampuses bool,
) (exportCampusChoice, error) {
	if interactive {
		return promptExportCampus(client)
	}
	if allCampuses {
		return exportCampusChoice{all: true}, nil
	}

	campusFlag = strings.TrimSpace(campusFlag)
	if campusFlag != "" {
		return parseExportCampusFlag(client, campusFlag)
	}

	campusID, err := ensureCampusID(client, cfg)
	if err != nil {
		return exportCampusChoice{}, err
	}
	if campusID <= 0 {
		return exportCampusChoice{all: true}, nil
	}
	return exportCampusChoice{campusID: campusID}, nil
}

func parseExportCampusFlag(client *churchtools.Client, value string) (exportCampusChoice, error) {
	value = strings.TrimSpace(value)
	switch strings.ToLower(value) {
	case "all", "alle", "*":
		return exportCampusChoice{all: true}, nil
	}

	if id, err := strconv.Atoi(value); err == nil {
		if id <= 0 {
			return exportCampusChoice{}, fmt.Errorf("--campus muss positiv sein")
		}
		return exportCampusChoice{campusID: id}, nil
	}

	campuses, err := client.ListCampuses()
	if err != nil {
		return exportCampusChoice{}, err
	}
	id, err := matchCampusBySearch(campuses, value)
	if err != nil {
		return exportCampusChoice{}, err
	}
	return exportCampusChoice{campusID: id}, nil
}

func normalizeCampusSearch(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	for _, r := range value {
		if r >= 'a' && r <= 'z' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func matchCampusBySearch(campuses []churchtools.Campus, search string) (int, error) {
	needle := normalizeCampusSearch(search)
	if needle == "" {
		return 0, fmt.Errorf("--campus: leerer Suchstring")
	}

	var matches []churchtools.Campus
	for _, campus := range campuses {
		name := normalizeCampusSearch(campus.Name)
		if strings.Contains(name, needle) {
			matches = append(matches, campus)
		}
	}

	switch len(matches) {
	case 0:
		return 0, fmt.Errorf("--campus: kein Standort für %q gefunden", search)
	case 1:
		return matches[0].ID, nil
	default:
		names := make([]string, len(matches))
		for i, campus := range matches {
			names[i] = campus.Name
		}
		return 0, fmt.Errorf("--campus: mehrdeutig %q (%s)", search, strings.Join(names, ", "))
	}
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
