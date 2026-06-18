package churchtools_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	churchtools "github.com/janmz/churchtools-dup/internal/churchtools"
)

func TestListRelationshipTypesUsesPersonEndpoint(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/whoami", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 1, "firstName": "A", "lastName": "B", "email": "a@example.org"},
		})
	})
	mux.HandleFunc("/api/csrftoken", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": "csrf"})
	})
	mux.HandleFunc("/api/person/relationshiptypes", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": 8, "name": "Dublette"},
			},
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := churchtools.NewClient(server.URL, "token", "", "")
	if err := client.Login(); err != nil {
		t.Fatal(err)
	}

	types, err := client.ListRelationshipTypes()
	if err != nil {
		t.Fatal(err)
	}
	if len(types) != 1 || types[0].ID != 8 || types[0].Name != "Dublette" {
		t.Fatalf("unexpected types: %+v", types)
	}

	relType, err := client.FindDuplicateRelationshipType(churchtools.DuplicateRelationshipOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if relType.ID != 8 {
		t.Fatalf("duplicate type id = %d, want default 8", relType.ID)
	}
}

func TestFindDuplicateRelationshipTypePrefersDefaultIDOverName(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/whoami", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 1, "firstName": "A", "lastName": "B", "email": "a@example.org"},
		})
	})
	mux.HandleFunc("/api/csrftoken", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": "csrf"})
	})
	mux.HandleFunc("/api/person/relationshiptypes", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": 8, "name": "Dublette"},
				{"id": 5, "name": "Duplikat"},
			},
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := churchtools.NewClient(server.URL, "token", "", "")
	if err := client.Login(); err != nil {
		t.Fatal(err)
	}

	relType, err := client.FindDuplicateRelationshipType(churchtools.DuplicateRelationshipOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if relType.ID != 8 {
		t.Fatalf("expected default id 8, got %d", relType.ID)
	}
}

func TestFindDuplicateRelationshipTypeByConfigID(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/whoami", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 1, "firstName": "A", "lastName": "B", "email": "a@example.org"},
		})
	})
	mux.HandleFunc("/api/csrftoken", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": "csrf"})
	})
	mux.HandleFunc("/api/person/relationshiptypes", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": 8, "name": "Dublette"},
				{"id": 5, "name": "Duplikat"},
			},
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := churchtools.NewClient(server.URL, "token", "", "")
	if err := client.Login(); err != nil {
		t.Fatal(err)
	}

	relType, err := client.FindDuplicateRelationshipType(churchtools.DuplicateRelationshipOptions{TypeID: 5})
	if err != nil {
		t.Fatal(err)
	}
	if relType.ID != 5 {
		t.Fatalf("expected configured id 5, got %d", relType.ID)
	}
}

func TestFindDuplicateRelationshipTypeFallsBackWithoutDefaultID(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/whoami", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 1, "firstName": "A", "lastName": "B", "email": "a@example.org"},
		})
	})
	mux.HandleFunc("/api/csrftoken", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": "csrf"})
	})
	mux.HandleFunc("/api/person/relationshiptypes", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": 5, "name": "Duplikat"},
			},
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := churchtools.NewClient(server.URL, "token", "", "")
	if err := client.Login(); err != nil {
		t.Fatal(err)
	}

	relType, err := client.FindDuplicateRelationshipType(churchtools.DuplicateRelationshipOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if relType.ID != 5 {
		t.Fatalf("expected fallback id 5, got %d", relType.ID)
	}
}
