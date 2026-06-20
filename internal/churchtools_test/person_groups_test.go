package churchtools_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	churchtools "github.com/janmz/churchtools-dup/internal/churchtools"
)

func TestListPersonGroups(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/whoami", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 7, "firstName": "Max", "lastName": "Muster", "email": "max@example.org"},
		})
	})
	mux.HandleFunc("/api/csrftoken", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": "csrf"})
	})
	mux.HandleFunc("/api/persons/7/groups", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{
					"personId": 7,
					"group": map[string]any{
						"title":  "Chor",
						"apiUrl": "/api/groups/12",
					},
				},
				{
					"personId": 7,
					"group": map[string]any{
						"title": "<span>1</span> Personen bearbeiten",
						"apiUrl": "/api/groups/3",
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

	groups, err := client.ListPersonGroups(7)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	byID := make(map[int]string, len(groups))
	for _, group := range groups {
		byID[group.ID] = churchtools.PlainGroupName(group.Name)
	}
	if byID[3] != "1 Personen bearbeiten" {
		t.Fatalf("group 3 = %q", byID[3])
	}
	if byID[12] != "Chor" {
		t.Fatalf("group 12 = %q", byID[12])
	}
}

func TestListPersonGroupsUsesGroupIdNotMembershipId(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/whoami", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 1, "firstName": "A", "lastName": "B", "email": "a@example.org"},
		})
	})
	mux.HandleFunc("/api/csrftoken", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": "csrf"})
	})
	mux.HandleFunc("/api/persons/9/groups", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": 501, "groupId": 5, "name": "Duplikate"},
			},
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := churchtools.NewClient(server.URL, "token", "", "")
	if err := client.Login(); err != nil {
		t.Fatal(err)
	}

	groups, err := client.ListPersonGroups(9)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 || groups[0].ID != 5 || groups[0].Name != "Duplikate" {
		t.Fatalf("unexpected groups: %+v", groups)
	}
}

func TestListPersonGroupsFlatItems(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/whoami", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 1, "firstName": "A", "lastName": "B", "email": "a@example.org"},
		})
	})
	mux.HandleFunc("/api/csrftoken", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": "csrf"})
	})
	mux.HandleFunc("/api/persons/9/groups", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": 5, "name": "Duplikate"},
			},
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := churchtools.NewClient(server.URL, "token", "", "")
	if err := client.Login(); err != nil {
		t.Fatal(err)
	}

	groups, err := client.ListPersonGroups(9)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 || groups[0].ID != 5 || groups[0].Name != "Duplikate" {
		t.Fatalf("unexpected groups: %+v", groups)
	}
}
