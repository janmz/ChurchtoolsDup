package churchtools_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	churchtools "github.com/janmz/churchtools-dup/internal/churchtools"
)

func TestRequestGroupMembershipUsesPublicSignupOnForbidden(t *testing.T) {
	var tokenCalls int
	mux := http.NewServeMux()
	mux.HandleFunc("/api/whoami", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 7696, "firstName": "Jan", "lastName": "Neuhaus", "email": "jan@example.org"},
		})
	})
	mux.HandleFunc("/api/csrftoken", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": "csrf"})
	})
	mux.HandleFunc("/api/groups/955/members/7696", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"error.forbidden.add.group.member"}`))
	})
	mux.HandleFunc("/api/publicgroups/955/form", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"token": "form-token",
				"group": map[string]any{
					"id":         955,
					"name":       "ChurchTools Admin",
					"autoAccept": true,
				},
			},
		})
	})
	mux.HandleFunc("/api/publicgroups/955/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		tokenCalls++
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"personId":7696`) {
			t.Fatalf("unexpected token body: %s", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"token": "signup-token"},
		})
	})
	mux.HandleFunc("/api/publicgroups/955/signup", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"token":"signup-token"`) {
			t.Fatalf("unexpected signup body: %s", body)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := churchtools.NewClient(server.URL, "token", "", "")
	if err := client.Login(); err != nil {
		t.Fatal(err)
	}

	result, err := client.RequestGroupMembership(955, 7696)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != churchtools.MembershipActive {
		t.Fatalf("status = %q message = %q", result.Status, result.Message)
	}
	if result.Message == "" {
		t.Fatalf("expected non-empty message for active signup, got %q", result.Message)
	}
	if tokenCalls != 1 {
		t.Fatalf("tokenCalls = %d", tokenCalls)
	}
}
