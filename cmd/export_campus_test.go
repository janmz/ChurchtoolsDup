package cmd

import "testing"

func TestParseExportCampusFlagAll(t *testing.T) {
	for _, value := range []string{"all", "ALL", "alle", "*"} {
		choice, err := parseExportCampusFlag(value)
		if err != nil {
			t.Fatalf("parseExportCampusFlag(%q): %v", value, err)
		}
		if !choice.all || choice.campusID != 0 {
			t.Fatalf("choice = %+v, want all", choice)
		}
	}
}

func TestParseExportCampusFlagNumeric(t *testing.T) {
	choice, err := parseExportCampusFlag("42")
	if err != nil {
		t.Fatal(err)
	}
	if choice.all || choice.campusID != 42 {
		t.Fatalf("choice = %+v", choice)
	}
}

func TestParseExportCampusFlagInvalid(t *testing.T) {
	if _, err := parseExportCampusFlag("abc"); err == nil {
		t.Fatal("expected error for invalid campus id")
	}
	if _, err := parseExportCampusFlag("0"); err == nil {
		t.Fatal("expected error for zero campus id")
	}
}
