package cmd

import (
	"testing"

	churchtools "github.com/janmz/churchtools-dup/internal/churchtools"
)

func TestParseExportCampusFlagAll(t *testing.T) {
	for _, value := range []string{"all", "ALL", "alle", "*"} {
		choice, err := parseExportCampusFlag(nil, value)
		if err != nil {
			t.Fatalf("parseExportCampusFlag(%q): %v", value, err)
		}
		if !choice.all || choice.campusID != 0 {
			t.Fatalf("choice = %+v, want all", choice)
		}
	}
}

func TestParseExportCampusFlagNumeric(t *testing.T) {
	choice, err := parseExportCampusFlag(nil, "42")
	if err != nil {
		t.Fatal(err)
	}
	if choice.all || choice.campusID != 42 {
		t.Fatalf("choice = %+v", choice)
	}
}

func TestNormalizeCampusSearch(t *testing.T) {
	if got := normalizeCampusSearch(" Rhein-Main "); got != "rheinmain" {
		t.Fatalf("got %q", got)
	}
}

func TestMatchCampusBySearch(t *testing.T) {
	campuses := []churchtools.Campus{
		{ID: 1, Name: "EMK Mitte"},
		{ID: 2, Name: "EMK Rhein-Main"},
		{ID: 3, Name: "EMK Nord"},
	}

	for _, search := range []string{"rhein", "Rhein-Main"} {
		id, err := matchCampusBySearch(campuses, search)
		if err != nil || id != 2 {
			t.Fatalf("search %q: id = %d, err = %v", search, id, err)
		}
	}

	if _, err := matchCampusBySearch(campuses, "emk"); err == nil {
		t.Fatal("expected ambiguous match")
	}

	if _, err := matchCampusBySearch(campuses, "sued"); err == nil {
		t.Fatal("expected no match")
	}
}

func TestParseExportCampusFlagInvalidNumeric(t *testing.T) {
	if _, err := parseExportCampusFlag(nil, "0"); err == nil {
		t.Fatal("expected error for zero campus id")
	}
}

func TestMatchCampusBySearchUmlautsAndDigits(t *testing.T) {
	campuses := []churchtools.Campus{
		{ID: 1, Name: "Standort 2"},
		{ID: 2, Name: "Standort 3"},
		{ID: 3, Name: "Süd"},
		{ID: 4, Name: "Sundern"},
		{ID: 5, Name: "Café Noël"},
	}

	cases := map[string]int{
		"standort 2": 1, // digits distinguish the campuses
		"Standort3":  2,
		"Süd":        3, // ü → ue, must not match "Sundern"
		"sued":       3,
		"cafe noel":  5, // accents removed
	}
	for search, want := range cases {
		id, err := matchCampusBySearch(campuses, search)
		if err != nil || id != want {
			t.Errorf("search %q: id = %d, err = %v, want %d", search, id, err, want)
		}
	}
}

func TestMatchCampusBySearchPrefersExactName(t *testing.T) {
	campuses := []churchtools.Campus{
		{ID: 1, Name: "Nord"},
		{ID: 2, Name: "Nordost"},
	}
	id, err := matchCampusBySearch(campuses, "nord")
	if err != nil || id != 1 {
		t.Fatalf("id = %d, err = %v, want exact match 1", id, err)
	}
}

func TestMaskEmail(t *testing.T) {
	cases := map[string]string{
		"info@gemeinde.de": "in***@gemeinde.de",
		"a@b.de":           "a***@b.de",
		"ölf@x.de":         "öl***@x.de",
		"kein-at":          "***",
	}
	for in, want := range cases {
		if got := maskEmail(in); got != want {
			t.Errorf("maskEmail(%q) = %q, want %q", in, got, want)
		}
	}
}
