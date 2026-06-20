package churchtools

import "testing"

func TestDecodePersonAcceptedSecurityCamelCase(t *testing.T) {
	person, err := decodePerson([]byte(`{
		"id": 42,
		"acceptedSecurity": "2024-01-15"
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if person.AcceptedSecurity == nil || *person.AcceptedSecurity != "2024-01-15" {
		t.Fatalf("AcceptedSecurity = %v", person.AcceptedSecurity)
	}
	if person.ExportStatusLabel() != "Registriert" {
		t.Fatalf("ExportStatusLabel = %q", person.ExportStatusLabel())
	}
}

func TestDecodePersonIsSystemUserAsInt(t *testing.T) {
	person, err := decodePerson([]byte(`{
		"id": 42,
		"firstName": "Max",
		"lastName": "Muster",
		"email": "max@example.org",
		"isSystemUser": 1
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if person.IsSystemUser == nil || !*person.IsSystemUser {
		t.Fatal("expected system user from numeric isSystemUser")
	}
}

func TestDecodePersonPrivacyPolicyAgreementArray(t *testing.T) {
	person, err := decodePerson([]byte(`{
		"id": 42,
		"privacyPolicyAgreement": [{"date": "2024-05-01"}]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if person.ExportStatusLabel() != "Registriert" {
		t.Fatalf("ExportStatusLabel = %q", person.ExportStatusLabel())
	}
}

func TestDecodePersonLastLogin(t *testing.T) {
	person, err := decodePerson([]byte(`{
		"id": 42,
		"lastLogin": "2024-01-15T10:00:00Z"
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if person.LastLogin == nil || *person.LastLogin == "" {
		t.Fatal("expected lastLogin to be decoded")
	}
}
