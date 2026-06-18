package duplicates_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	churchtools "github.com/janmz/churchtools-dup/internal/churchtools"
	csvfile "github.com/janmz/churchtools-dup/internal/csvfile"
	"github.com/janmz/churchtools-dup/internal/duplicates"
)

func TestImportRunnerDryRunDetectsExistingRelationship(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/whoami", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 1, "firstName": "A", "lastName": "B", "email": "a@example.org"},
		})
	})
	mux.HandleFunc("/api/csrftoken", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": "csrf"})
	})
	mux.HandleFunc("/api/persons/10005/relationships", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{
					"relationshipTypeId": 5,
					"relative": map[string]any{
						"apiUrl": "/api/persons/10006",
					},
				},
			},
		})
	})
	mux.HandleFunc("/api/persons/10005/groups", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": 9, "name": "Duplikate"},
			},
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := churchtools.NewClient(server.URL, "token", "", "")
	if err := client.Login(); err != nil {
		t.Fatal(err)
	}

	runner := duplicates.ImportRunner{
		Client:  client,
		RelType: churchtools.RelationshipType{ID: 5, Name: "Duplikat"},
	}
	groups := [][]csvfile.DupEntry{{
		{DupID: 3, PersonID: 10005},
		{DupID: 3, PersonID: 10006},
	}}

	results, err := runner.Run(groups, duplicates.ImportOptions{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || !results[0].Success {
		t.Fatalf("unexpected results: %+v", results)
	}
	if results[0].Linked != 0 || results[0].AlreadyLinked != 1 {
		t.Fatalf("counts = linked %d already %d", results[0].Linked, results[0].AlreadyLinked)
	}
	if !strings.Contains(results[0].Message, "bereits mit 10006 verknüpft") {
		t.Fatalf("message = %q", results[0].Message)
	}
	if !strings.Contains(results[0].Message, "bereits in Gruppe \"Duplikate\"") {
		t.Fatalf("message = %q", results[0].Message)
	}
	if duplicates.ImportResultStatus(results[0]) != "skipped" {
		t.Fatalf("expected skipped status, got %q", duplicates.ImportResultStatus(results[0]))
	}
}

func TestImportRunnerSkipsExistingRelationship(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/whoami", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 1, "firstName": "A", "lastName": "B", "email": "a@example.org"},
		})
	})
	mux.HandleFunc("/api/csrftoken", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": "csrf"})
	})
	mux.HandleFunc("/api/persons/10005/relationships", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			t.Fatal("unexpected POST for existing relationship")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{
					"relationshipTypeId": 5,
					"relative": map[string]any{
						"apiUrl": "/api/persons/10006",
					},
				},
			},
		})
	})
	mux.HandleFunc("/api/groups", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{"id": 9, "name": "Duplikate"}},
		})
	})
	mux.HandleFunc("/api/groups/9/members/10005", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := churchtools.NewClient(server.URL, "token", "", "")
	if err := client.Login(); err != nil {
		t.Fatal(err)
	}

	runner := duplicates.ImportRunner{
		Client:  client,
		RelType: churchtools.RelationshipType{ID: 5, Name: "Duplikat"},
	}
	groups := [][]csvfile.DupEntry{{
		{DupID: 3, PersonID: 10005},
		{DupID: 3, PersonID: 10006},
	}}

	results, err := runner.Run(groups, duplicates.ImportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || !results[0].Success {
		t.Fatalf("unexpected results: %+v", results)
	}
	if results[0].Message != "Person 10005: alle 1 Duplikat-Beziehung(en) bereits vorhanden" {
		t.Fatalf("message = %q", results[0].Message)
	}
	if results[0].Linked != 0 || results[0].AlreadyLinked != 1 {
		t.Fatalf("counts = linked %d already %d", results[0].Linked, results[0].AlreadyLinked)
	}
	if duplicates.ImportResultStatus(results[0]) != "skipped" {
		t.Fatalf("expected skipped status, got %q", duplicates.ImportResultStatus(results[0]))
	}
}

func TestPrintImportSummaryCountsExistingAsSkipped(t *testing.T) {
	results := []duplicates.ImportResult{
		{DupID: 2, Success: true, Linked: 1, AlreadyLinked: 0, Message: "Person 10003 mit 1 Dublette(n) verknüpft"},
		{DupID: 3, Success: true, Linked: 0, AlreadyLinked: 1, Message: "Person 10005: alle 1 Duplikat-Beziehung(en) bereits vorhanden"},
	}

	if duplicates.ImportResultStatus(results[0]) != "ok" {
		t.Fatalf("dup 2 status = %q", duplicates.ImportResultStatus(results[0]))
	}
	if duplicates.ImportResultStatus(results[1]) != "skipped" {
		t.Fatalf("dup 3 status = %q", duplicates.ImportResultStatus(results[1]))
	}
}
