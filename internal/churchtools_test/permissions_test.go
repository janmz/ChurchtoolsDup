package churchtools_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	churchtools "github.com/janmz/churchtools-dup/internal/churchtools"
)

func TestHasModulePermission(t *testing.T) {
	perms := map[string]any{
		"churchdb": map[string]any{
			"export data": true,
			"write access": false,
			"view alldata": []any{1.0, 2.0},
		},
	}

	if !churchtools.HasModulePermission(perms, "churchdb", "export data") {
		t.Fatal("expected export data")
	}
	if churchtools.HasModulePermission(perms, "churchdb", "write access") {
		t.Fatal("expected no write access")
	}
	if !churchtools.HasModulePermission(perms, "churchdb", "view alldata") {
		t.Fatal("expected view alldata")
	}
}

func TestEnsurePermissionsRequestsGroupMembership(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/whoami", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 7, "firstName": "Admin", "lastName": "User", "email": "admin@example.org"},
		})
	})
	mux.HandleFunc("/api/csrftoken", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": "csrf"})
	})
	mux.HandleFunc("/api/permissions/global", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"churchdb": map[string]any{
					"export data": false,
				},
			},
		})
	})
	mux.HandleFunc("/api/groups", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": 12, "name": "Personen exportieren"},
			},
		})
	})
	mux.HandleFunc("/api/groups/12/members/7", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := churchtools.NewClient(server.URL, "token", "", "")
	if err := client.Login(); err != nil {
		t.Fatal(err)
	}

	notes, err := client.EnsurePermissions([]churchtools.PermissionRequirement{
		{
			Module:      churchtools.ModuleChurchDB,
			Permission:  churchtools.PermExportData,
			GroupNames:  []string{"Personen exportieren"},
			Description: "Personen exportieren",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) == 0 {
		t.Fatal("expected notes")
	}
}

func TestFindGroupByNameStripsHTML(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/whoami", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 1, "firstName": "A", "lastName": "B", "email": "a@example.org"},
		})
	})
	mux.HandleFunc("/api/csrftoken", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": "csrf"})
	})
	mux.HandleFunc("/api/groups", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": 3, "name": "<span>1</span> Personen bearbeiten"},
			},
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := churchtools.NewClient(server.URL, "token", "", "")
	if err := client.Login(); err != nil {
		t.Fatal(err)
	}

	group, err := client.FindGroupByName("Personen bearbeiten")
	if err != nil {
		t.Fatal(err)
	}
	if group.ID != 3 {
		t.Fatalf("group id = %d", group.ID)
	}
}

func TestFindGroupByNamesUsesFallback(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/whoami", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 1, "firstName": "A", "lastName": "B", "email": "a@example.org"},
		})
	})
	mux.HandleFunc("/api/csrftoken", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": "csrf"})
	})
	mux.HandleFunc("/api/groups", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		switch query {
		case "Personen bearbeiten":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{}})
		case "Personen Administration":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": 8, "name": "Personen Administration"},
				},
			})
		default:
			http.Error(w, "unexpected query", http.StatusBadRequest)
		}
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := churchtools.NewClient(server.URL, "token", "", "")
	if err := client.Login(); err != nil {
		t.Fatal(err)
	}

	group, name, err := client.FindGroupByNames([]string{
		"Personen bearbeiten",
		"Personen Administration",
	})
	if err != nil {
		t.Fatal(err)
	}
	if group.ID != 8 || name != "Personen Administration" {
		t.Fatalf("unexpected group: id=%d name=%q", group.ID, name)
	}
}
