package churchtools_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	churchtools "github.com/janmz/churchtools-dup/internal/churchtools"
)

func TestEnsurePreJoinGroupsSkipsExistingMembership(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/whoami", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 7, "firstName": "Admin", "lastName": "User", "email": "admin@example.org"},
		})
	})
	mux.HandleFunc("/api/csrftoken", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": "csrf"})
	})
	mux.HandleFunc("/api/persons/7/groups", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": 1, "name": "ChurchTools Admin"},
			},
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := churchtools.NewClient(server.URL, "token", "", "")
	if err := client.Login(); err != nil {
		t.Fatal(err)
	}

	results, err := client.EnsurePreJoinGroups([]string{"ChurchTools Admin"})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("results = %d, want 1 skipped group", len(results))
	}
	if !results[0].Skipped || results[0].GroupName != "ChurchTools Admin" {
		t.Fatalf("unexpected first result: %+v", results[0])
	}
}

func TestEnsurePreJoinGroupsRetriesHiddenGroupAfterEarlierJoin(t *testing.T) {
	adminVisible := false
	mux := http.NewServeMux()
	mux.HandleFunc("/api/whoami", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 7, "firstName": "Admin", "lastName": "User", "email": "admin@example.org"},
		})
	})
	mux.HandleFunc("/api/csrftoken", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": "csrf"})
	})
	mux.HandleFunc("/api/persons/7/groups", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	})
	mux.HandleFunc("/api/groups", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		var data []map[string]any
		switch query {
		case "ChurchTools Admin":
			if adminVisible {
				data = []map[string]any{{"id": 1, "name": "ChurchTools Admin"}}
			}
		case "ChurchTools Verwaltung":
			data = []map[string]any{{"id": 2, "name": "ChurchTools Verwaltung"}}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
	})
	mux.HandleFunc("/api/groups/2/members/7", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		adminVisible = true
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/api/groups/1/members/7", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := churchtools.NewClient(server.URL, "token", "", "")
	if err := client.Login(); err != nil {
		t.Fatal(err)
	}

	results, err := client.EnsurePreJoinGroups([]string{"ChurchTools Admin", "ChurchTools Verwaltung"})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("results = %+v", results)
	}
	if results[0].GroupName != "ChurchTools Admin" || results[0].Status != churchtools.MembershipActive {
		t.Fatalf("admin result = %+v", results[0])
	}
	if results[1].GroupName != "ChurchTools Verwaltung" || results[1].Status != churchtools.MembershipActive {
		t.Fatalf("verwaltung result = %+v", results[1])
	}
}

func TestEnsurePreJoinGroupsJoinsInOrder(t *testing.T) {
	var joinOrder []string

	mux := http.NewServeMux()
	mux.HandleFunc("/api/whoami", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 7, "firstName": "Admin", "lastName": "User", "email": "admin@example.org"},
		})
	})
	mux.HandleFunc("/api/csrftoken", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": "csrf"})
	})
	mux.HandleFunc("/api/persons/7/groups", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{}})
	})
	mux.HandleFunc("/api/groups", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		switch query {
		case "ChurchTools Admin":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{{"id": 10, "name": "ChurchTools Admin"}},
			})
		case "Personen Administration":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{{"id": 11, "name": "Personen Administration"}},
			})
		default:
			http.Error(w, "unexpected query", http.StatusBadRequest)
		}
	})
	mux.HandleFunc("/api/groups/10/members/7", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		joinOrder = append(joinOrder, "ChurchTools Admin")
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/api/groups/11/members/7", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		joinOrder = append(joinOrder, "Personen Administration")
		w.WriteHeader(http.StatusOK)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := churchtools.NewClient(server.URL, "token", "", "")
	if err := client.Login(); err != nil {
		t.Fatal(err)
	}

	results, err := client.EnsurePreJoinGroups([]string{
		"ChurchTools Admin",
		"Personen Administration",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("results = %d, want 2", len(results))
	}
	if results[0].Status != churchtools.MembershipActive || results[0].Skipped {
		t.Fatalf("first join: %+v", results[0])
	}
	if results[1].Status != churchtools.MembershipActive || results[1].Skipped {
		t.Fatalf("second join: %+v", results[1])
	}
	if len(joinOrder) != 2 || joinOrder[0] != "ChurchTools Admin" || joinOrder[1] != "Personen Administration" {
		t.Fatalf("join order = %v", joinOrder)
	}
}
