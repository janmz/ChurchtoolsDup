package csvfile_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	churchtools "github.com/janmz/churchtools-dup/internal/churchtools"
	csvfile "github.com/janmz/churchtools-dup/internal/csvfile"
)

func TestReadDupAcceptsUTF8BOM(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dup.csv")

	entries := []csvfile.DupEntry{{
		DupID:    1,
		PersonID: 1537,
	}}
	if err := csvfile.WriteDup(path, entries); err != nil {
		t.Fatal(err)
	}

	loaded, err := csvfile.ReadDup(path)
	if err != nil {
		t.Fatalf("ReadDup: %v", err)
	}
	if len(loaded) != 1 || loaded[0].DupID != 1 || loaded[0].PersonID != 1537 {
		t.Fatalf("unexpected entries: %+v", loaded)
	}
}

func TestReadDupAcceptsBOMInHeaderOnly(t *testing.T) {
	csvText := "\ufeffDupID,ID,Vorname,Nachname,E-Mail,Straße,Stadt,Standort,Erstellungsdatum,Einladungsstatus\n1,1537,,,,,,,,\n"
	dir := t.TempDir()
	path := filepath.Join(dir, "dup.csv")
	if err := os.WriteFile(path, []byte(csvText), 0o600); err != nil {
		t.Fatal(err)
	}

	loaded, err := csvfile.ReadDup(path)
	if err != nil {
		t.Fatalf("ReadDup: %v", err)
	}
	if len(loaded) != 1 || loaded[0].PersonID != 1537 {
		t.Fatalf("unexpected entries: %+v", loaded)
	}
}

func TestReadDupAcceptsSemicolonDelimiter(t *testing.T) {
	header := "DupID;ID;Vorname;Nachname;E-Mail;Straße;Stadt;Standort;Erstellungsdatum;Einladungsstatus"
	csvText := header + "\n1;1537;Max;Muster;max@example.org;Hauptstr. 1;Mainz;Nord;20.05.2026;\n"
	dir := t.TempDir()
	path := filepath.Join(dir, "dup.csv")
	if err := os.WriteFile(path, []byte(csvText), 0o600); err != nil {
		t.Fatal(err)
	}

	loaded, err := csvfile.ReadDup(path)
	if err != nil {
		t.Fatalf("ReadDup: %v", err)
	}
	if len(loaded) != 1 || loaded[0].FirstName != "Max" || loaded[0].Street != "Hauptstr. 1" {
		t.Fatalf("unexpected entries: %+v", loaded)
	}
}

func TestReadDupAcceptsTabDelimiter(t *testing.T) {
	header := "DupID\tID\tVorname\tNachname\tE-Mail\tStraße\tStadt\tStandort\tErstellungsdatum\tEinladungsstatus"
	csvText := header + "\n1\t1537\tMax\tMuster\tmax@example.org\tHauptstr. 1\tMainz\tNord\t20.05.2026\t\n"
	dir := t.TempDir()
	path := filepath.Join(dir, "dup.csv")
	if err := os.WriteFile(path, []byte(csvText), 0o600); err != nil {
		t.Fatal(err)
	}

	loaded, err := csvfile.ReadDup(path)
	if err != nil {
		t.Fatalf("ReadDup: %v", err)
	}
	if len(loaded) != 1 || loaded[0].PersonID != 1537 {
		t.Fatalf("unexpected entries: %+v", loaded)
	}
}

func TestReadDupAcceptsQuotedCommaInField(t *testing.T) {
	csvText := "DupID,ID,Vorname,Nachname,E-Mail,Straße,Stadt,Standort,Erstellungsdatum,Einladungsstatus\n" +
		`1,1537,Max,Muster,max@example.org,"Hauptstr. 1, EG",Mainz,Nord,20.05.2026,` + "\n"
	dir := t.TempDir()
	path := filepath.Join(dir, "dup.csv")
	if err := os.WriteFile(path, []byte(csvText), 0o600); err != nil {
		t.Fatal(err)
	}

	loaded, err := csvfile.ReadDup(path)
	if err != nil {
		t.Fatalf("ReadDup: %v", err)
	}
	if len(loaded) != 1 || loaded[0].Street != "Hauptstr. 1, EG" {
		t.Fatalf("unexpected street: %+v", loaded)
	}
}

func TestWriteDupUsesCommaAndQuotesFieldsWithComma(t *testing.T) {
	var buf bytes.Buffer
	entry := csvfile.DupEntry{
		DupID:    1,
		PersonID: 1537,
		FirstName: "Max",
		Street:    "Hauptstr. 1, EG",
	}
	if err := csvfile.WriteDupTo(&buf, []csvfile.DupEntry{entry}); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if strings.Contains(output, ";") {
		t.Fatalf("export should use comma delimiter, got:\n%s", output)
	}
	if !strings.Contains(output, `"Hauptstr. 1, EG"`) {
		t.Fatalf("export should quote fields containing comma, got:\n%s", output)
	}
}

func TestWriteDupIncludesEmailCampusAndInvitationStatus(t *testing.T) {
	person := churchtools.Person{
		ID:               42,
		FirstName:        "Max",
		LastName:         "Muster",
		Email:            "max@example.org",
		Street:           "Hauptstr. 1",
		City:             "Mainz",
		CreatedAt:        "20.05.2026",
		InvitationStatus: "pending",
	}

	var buf bytes.Buffer
	entry := csvfile.DupEntryFromPerson(1, person, "Rhein-Main")
	if err := csvfile.WriteDupTo(&buf, []csvfile.DupEntry{entry}); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	for _, want := range []string{"E-Mail", "Standort", "Einladungsstatus", "max@example.org", "Rhein-Main", "20.05.2026", "Eingeladen"} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}
