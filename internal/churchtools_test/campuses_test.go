package churchtools_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	churchtools "github.com/janmz/churchtools-dup/internal/churchtools"
)

func TestListCampuses(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/whoami", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 1, "firstName": "Admin", "lastName": "User", "email": "admin@example.org"},
		})
	})
	mux.HandleFunc("/api/csrftoken", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": "csrf"})
	})
	mux.HandleFunc("/api/campuses", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": 1, "name": "Zentrum"},
				{"id": 2, "name": "Nord"},
			},
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := churchtools.NewClient(server.URL, "token", "", "")
	if err := client.Login(); err != nil {
		t.Fatal(err)
	}

	campuses, err := client.ListCampuses()
	if err != nil {
		t.Fatal(err)
	}
	if len(campuses) != 2 || campuses[0].Name != "Zentrum" {
		t.Fatalf("unexpected campuses: %+v", campuses)
	}
}

func TestListPersonsByCampus(t *testing.T) {
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
		if got := r.URL.Query().Get("campus_ids[]"); got != "3" {
			t.Fatalf("campus_ids[] = %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": 10, "firstName": "Max", "lastName": "Muster", "email": "max@example.org"},
			},
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := churchtools.NewClient(server.URL, "token", "", "")
	if err := client.Login(); err != nil {
		t.Fatal(err)
	}

	persons, err := client.ListPersons(churchtools.PersonListOptions{CampusID: 3})
	if err != nil {
		t.Fatal(err)
	}
	if len(persons) != 1 || persons[0].ID != 10 {
		t.Fatalf("unexpected persons: %+v", persons)
	}
}

func TestListGroupsByCampus(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/whoami", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 1, "firstName": "Admin", "lastName": "User", "email": "admin@example.org"},
		})
	})
	mux.HandleFunc("/api/csrftoken", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": "csrf"})
	})
	mux.HandleFunc("/api/groups", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("campus_ids[]"); got != "2" {
			t.Fatalf("campus_ids[] = %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": 7, "name": "Chor"},
			},
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := churchtools.NewClient(server.URL, "token", "", "")
	if err := client.Login(); err != nil {
		t.Fatal(err)
	}

	groups, err := client.ListGroups(churchtools.GroupListOptions{CampusID: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 || groups[0].Name != "Chor" {
		t.Fatalf("unexpected groups: %+v", groups)
	}
}
