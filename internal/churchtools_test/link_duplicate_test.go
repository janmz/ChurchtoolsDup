package churchtools_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	churchtools "github.com/janmz/churchtools-dup/internal/churchtools"
)

// newLoggedInClient starts a mock ChurchTools server with whoami/csrftoken
// handlers added to mux and returns a logged-in token client.
func newLoggedInClient(t *testing.T, mux *http.ServeMux) *churchtools.Client {
	t.Helper()
	mux.HandleFunc("/api/whoami", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 1, "firstName": "A", "lastName": "B", "email": "a@example.org"},
		})
	})
	mux.HandleFunc("/api/csrftoken", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": "csrf"})
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := churchtools.NewClient(server.URL, "token", "", "")
	if err := client.Login(); err != nil {
		t.Fatal(err)
	}
	return client
}

var dupType = churchtools.RelationshipType{ID: 5, Name: "Duplikat"}

func TestLinkAsDuplicateSendsOtherPersonID(t *testing.T) {
	var got map[string]any
	mux := http.NewServeMux()
	mux.HandleFunc("/api/persons/10005/relationships", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatal(err)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"data":{"id":77,"relationshipTypeId":5}}`))
	})
	client := newLoggedInClient(t, mux)

	if err := client.LinkAsDuplicate(10005, 10006, dupType); err != nil {
		t.Fatalf("LinkAsDuplicate: %v", err)
	}
	if got["otherPersonId"] != float64(10006) || got["relationshipTypeId"] != float64(5) {
		t.Fatalf("POST body = %v, want otherPersonId 10006 and relationshipTypeId 5", got)
	}
}

func TestLinkAsDuplicateReportsRESTError(t *testing.T) {
	legacyCalled := false
	mux := http.NewServeMux()
	mux.HandleFunc("/api/persons/10005/relationships", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"kein Recht"}`, http.StatusForbidden)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		legacyCalled = true
	})
	client := newLoggedInClient(t, mux)

	if err := client.LinkAsDuplicate(10005, 10006, dupType); err == nil {
		t.Fatal("expected error for HTTP 403")
	}
	if legacyCalled {
		t.Fatal("legacy fallback must only be used for HTTP 404")
	}
}

func TestLinkAsDuplicateLegacyNeedsExplicitSuccess(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{name: "html login page", body: "<html>Login</html>", wantErr: true},
		{name: "json without status", body: `{"data":null}`, wantErr: true},
		{name: "explicit error", body: `{"status":"error"}`, wantErr: true},
		{name: "explicit success", body: `{"status":"success"}`, wantErr: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/api/persons/10005/relationships", func(w http.ResponseWriter, r *http.Request) {
				http.NotFound(w, r)
			})
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("q") != "churchdb/ajax" {
					http.NotFound(w, r)
					return
				}
				_, _ = w.Write([]byte(tc.body))
			})
			client := newLoggedInClient(t, mux)

			err := client.LinkAsDuplicate(10005, 10006, dupType)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
