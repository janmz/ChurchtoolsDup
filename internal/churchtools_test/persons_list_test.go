package churchtools_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	churchtools "github.com/janmz/churchtools-dup/internal/churchtools"
)

func TestListPersonsPaginated(t *testing.T) {
	page := 0
	mux := http.NewServeMux()

	mux.HandleFunc("/api/whoami", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 1, "firstName": "Admin", "lastName": "User", "email": "admin@example.org"},
		})
	})
	mux.HandleFunc("/api/csrftoken", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": "csrf"})
	})
	mux.HandleFunc("/api/persons", func(w http.ResponseWriter, r *http.Request) {
		page++
		if page == 1 {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{{
					"id": 1, "firstName": "Max", "lastName": "A", "email": "a@example.org",
				}},
				"meta": map[string]any{
					"pagination": map[string]any{"current": 1, "lastPage": 2},
				},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{
				"id": 2, "firstName": "Erika", "lastName": "B", "email": "b@example.org",
			}},
			"meta": map[string]any{
				"pagination": map[string]any{"current": 2, "lastPage": 2},
			},
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := churchtools.NewClient(server.URL, "token", "", "")
	if err := client.Login(); err != nil {
		t.Fatal(err)
	}

	persons, err := client.ListPersons(churchtools.PersonListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(persons) != 2 {
		t.Fatalf("len = %d", len(persons))
	}
}

func TestListPersonsByGroup(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/whoami", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 1, "firstName": "Admin", "lastName": "User", "email": "admin@example.org"},
		})
	})
	mux.HandleFunc("/api/csrftoken", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": "csrf"})
	})
	mux.HandleFunc("/api/groups/5/members", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"personId": 10},
				{"personId": 20},
			},
		})
	})
	mux.HandleFunc("/api/persons", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": 10, "firstName": "Max", "lastName": "Muster", "email": "max@example.org"},
				{"id": 20, "firstName": "Erika", "lastName": "Beispiel", "email": "erika@example.org"},
			},
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := churchtools.NewClient(server.URL, "token", "", "")
	if err := client.Login(); err != nil {
		t.Fatal(err)
	}

	persons, err := client.ListPersons(churchtools.PersonListOptions{GroupID: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(persons) != 2 || persons[0].ID != 10 {
		t.Fatalf("unexpected persons: %+v", persons)
	}
}
