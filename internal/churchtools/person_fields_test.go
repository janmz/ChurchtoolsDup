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

func TestConsentDateFallbackAcceptedSecurity(t *testing.T) {
	security := "2021-08-01"
	person := Person{AcceptedSecurity: &security}
	if got := person.ConsentDate(); got != "01.08.2021" {
		t.Fatalf("ConsentDate = %q", got)
	}
}

func TestMergePersonDetails(t *testing.T) {
	date := "2024-05-01"
	base := Person{ID: 1, FirstName: "Max", City: "Alt"}
	detail := Person{
		ID:                     1,
		Email:                  "max@example.org",
		CreatedAt:              "2020-01-01",
		PrivacyPolicyAgreement: &PrivacyPolicyAgreement{Date: &date},
	}

	merged := MergePersonDetails(base, detail)
	if merged.Email != "max@example.org" {
		t.Fatalf("Email = %q", merged.Email)
	}
	if merged.CreatedAt != "2020-01-01" {
		t.Fatalf("CreatedAt = %q", merged.CreatedAt)
	}
	if merged.ConsentDate() != "01.05.2024" {
		t.Fatalf("ConsentDate = %q", merged.ConsentDate())
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
	if person.ConsentDate() != "01.05.2024" {
		t.Fatalf("ConsentDate = %q", person.ConsentDate())
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
	if person.ConsentDate() != "" {
		t.Fatalf("ConsentDate = %q, want empty for empty privacyPolicyAgreement", person.ConsentDate())
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
