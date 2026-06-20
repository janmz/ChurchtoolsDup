package churchtools

import (
	"encoding/json"
	"testing"
)

func TestEnrichPersonFieldsCreatedAtAndCampus(t *testing.T) {
	raw := []byte(`{
		"id": 10005,
		"firstName": "Erika",
		"meta": {"createdDate": "2026-05-20T22:26:37Z"},
		"campus": {"id": 7, "name": "Campus Nord"}
	}`)

	person, err := decodePerson(raw)
	if err != nil {
		t.Fatal(err)
	}
	if person.CreatedAt != "20.05.2026" {
		t.Fatalf("CreatedAt = %q", person.CreatedAt)
	}
	if person.CampusID != 7 {
		t.Fatalf("CampusID = %d", person.CampusID)
	}
	if person.CampusName != "Campus Nord" {
		t.Fatalf("CampusName = %q", person.CampusName)
	}
}

func TestEnrichPersonFieldsNestedAddress(t *testing.T) {
	raw := []byte(`{
		"id": 2,
		"addresses": [{"street": "Lindenweg 4", "city": "Musterstadt"}]
	}`)

	person, err := decodePerson(raw)
	if err != nil {
		t.Fatal(err)
	}
	if person.Street != "Lindenweg 4" {
		t.Fatalf("Street = %q", person.Street)
	}
	if person.City != "Musterstadt" {
		t.Fatalf("City = %q", person.City)
	}
}

func TestMergePersonDetailsInvitationStatus(t *testing.T) {
	base := Person{ID: 1, FirstName: "Max", City: "Alt"}
	detail := Person{
		ID:               1,
		Email:            "max@example.org",
		InvitationStatus: "pending",
		CreatedAt:        "2020-01-01",
	}

	merged := MergePersonDetails(base, detail)
	if merged.Email != "max@example.org" {
		t.Fatalf("Email = %q", merged.Email)
	}
	if merged.CreatedAt != "2020-01-01" {
		t.Fatalf("CreatedAt = %q", merged.CreatedAt)
	}
	if merged.ExportStatusLabel() != "Eingeladen" {
		t.Fatalf("ExportStatusLabel = %q", merged.ExportStatusLabel())
	}
	if merged.City != "Alt" {
		t.Fatalf("City = %q", merged.City)
	}
}

func TestPrivacyPolicyAgreementArrayDecodingInDecodePerson(t *testing.T) {
	raw := json.RawMessage(`{
		"id": 3,
		"privacyPolicyAgreement": [{"date": "2024-05-01"}]
	}`)
	person, err := decodePerson(raw)
	if err != nil {
		t.Fatal(err)
	}
	if person.ExportStatusLabel() != "Registriert" {
		t.Fatalf("ExportStatusLabel = %q", person.ExportStatusLabel())
	}
}

func TestEnrichPersonFieldsSampleWithoutConsent(t *testing.T) {
	raw := []byte(`{
		"id": 10003,
		"firstName": "Max",
		"lastName": "Muster",
		"meta": {
			"createdDate": "2026-05-20T22:26:37Z",
			"modifiedDate": "2026-06-17T08:41:06Z"
		},
		"privacyPolicyAgreement": []
	}`)

	person, err := decodePerson(raw)
	if err != nil {
		t.Fatal(err)
	}
	if person.CreatedAt != "20.05.2026" {
		t.Fatalf("CreatedAt = %q", person.CreatedAt)
	}
	if person.ExportStatusLabel() != "NEU" {
		t.Fatalf("ExportStatusLabel = %q, want NEU for empty privacyPolicyAgreement", person.ExportStatusLabel())
	}
}

func TestEnrichPersonFieldsSampleMinimalRecord(t *testing.T) {
	raw := []byte(`{
		"id": 10006,
		"firstName": "Erika",
		"lastName": "Probe",
		"meta": {"createdDate": "2026-06-14T17:46:12Z"},
		"privacyPolicyAgreement": []
	}`)

	person, err := decodePerson(raw)
	if err != nil {
		t.Fatal(err)
	}
	if person.CreatedAt != "14.06.2026" {
		t.Fatalf("CreatedAt = %q", person.CreatedAt)
	}
}

func TestCampusLabel(t *testing.T) {
	person := Person{CampusID: 3}
	names := map[int]string{3: "Campus Süd"}
	if got := CampusLabel(person, names); got != "Campus Süd" {
		t.Fatalf("CampusLabel = %q", got)
	}
}
