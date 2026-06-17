package churchtools_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	churchtools "github.com/janmz/churchtools-dup/internal/churchtools"
)

func TestLogin(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/whoami", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Login test-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":        1,
				"campusId":  3,
				"firstName": "Admin",
				"lastName":  "User",
				"email":     "admin@example.org",
			},
		})
	})

	mux.HandleFunc("/api/csrftoken", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": "csrf-test"})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := churchtools.NewClient(server.URL, "test-token", "", "")
	if err := client.Login(); err != nil {
		t.Fatalf("Login: %v", err)
	}

	campusID, err := client.CurrentUserCampusID()
	if err != nil {
		t.Fatalf("CurrentUserCampusID: %v", err)
	}
	if campusID != 3 {
		t.Fatalf("campusID = %d", campusID)
	}
}

func TestIsForbidden(t *testing.T) {
	if !churchtools.IsForbidden(&churchtools.APIError{StatusCode: 403}) {
		t.Fatal("expected forbidden")
	}
	if churchtools.IsForbidden(&churchtools.APIError{StatusCode: 404}) {
		t.Fatal("expected not forbidden")
	}
}
