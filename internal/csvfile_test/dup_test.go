package csvfile_test

import (
	"bytes"
	"strings"
	"testing"

	churchtools "github.com/janmz/churchtools-dup/internal/churchtools"
	csvfile "github.com/janmz/churchtools-dup/internal/csvfile"
)

func TestWriteDupIncludesEmailAndCampus(t *testing.T) {
	date := "2024-05-01"
	person := churchtools.Person{
		ID:        42,
		FirstName: "Max",
		LastName:  "Muster",
		Email:     "max@example.org",
		Street:    "Hauptstr. 1",
		City:      "Mainz",
		CreatedAt: "20.05.2026",
		PrivacyPolicyAgreement: &churchtools.PrivacyPolicyAgreement{
			Date: &date,
		},
	}

	var buf bytes.Buffer
	entry := csvfile.DupEntryFromPerson(1, person, "Rhein-Main")
	if err := csvfile.WriteDupTo(&buf, []csvfile.DupEntry{entry}); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	for _, want := range []string{"E-Mail", "Standort", "max@example.org", "Rhein-Main", "20.05.2026", "01.05.2024"} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}
