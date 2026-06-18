package churchtools_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	churchtools "github.com/janmz/churchtools-dup/internal/churchtools"
)

func TestDuplicateRelationshipExists(t *testing.T) {
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
					"id":                 99,
					"relationshipTypeId": 5,
					"relationshipName":   "Duplikat",
					"relative": map[string]any{
						"apiUrl": "/api/persons/10006",
					},
				},
			},
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := churchtools.NewClient(server.URL, "token", "", "")
	if err := client.Login(); err != nil {
		t.Fatal(err)
	}

	relType := churchtools.RelationshipType{ID: 5, Name: "Duplikat"}
	exists, err := client.DuplicateRelationshipExists(10005, 10006, relType)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("expected existing duplicate relationship")
	}
}

func TestDuplicateRelationshipExistsNestedType(t *testing.T) {
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
					"relationshipType": map[string]any{"id": 8},
					"relationshipName": "Duplikat",
					"relative": map[string]any{
						"id": 10006,
					},
				},
			},
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := churchtools.NewClient(server.URL, "token", "", "")
	if err := client.Login(); err != nil {
		t.Fatal(err)
	}

	exists, err := client.DuplicateRelationshipExists(10005, 10006, churchtools.RelationshipType{ID: 8, Name: "Duplikat"})
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("expected existing duplicate relationship")
	}
}

func TestDuplicateRelationshipExistsByNameWhenTypeMissing(t *testing.T) {
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
					"relationshipName": "Duplikat",
					"relative": map[string]any{
						"domainIdentifier": "10006",
					},
				},
			},
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := churchtools.NewClient(server.URL, "token", "", "")
	if err := client.Login(); err != nil {
		t.Fatal(err)
	}

	exists, err := client.DuplicateRelationshipExists(10005, 10006, churchtools.RelationshipType{ID: 8, Name: "Duplikat"})
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("expected existing duplicate relationship")
	}
}

func TestLinkAsDuplicateSkipsExistingRelationship(t *testing.T) {
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
			t.Fatal("POST should not be called when relationship already exists")
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

	server := httptest.NewServer(mux)
	defer server.Close()

	client := churchtools.NewClient(server.URL, "token", "", "")
	if err := client.Login(); err != nil {
		t.Fatal(err)
	}

	_, err := client.LinkAsDuplicate(10005, 10006, churchtools.RelationshipType{ID: 5, Name: "Duplikat"})
	if err != nil {
		t.Fatalf("LinkAsDuplicate: %v", err)
	}
}
