package churchtools

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoginMainInstanceFallback(t *testing.T) {
	const token = "shared-token"

	mainMux := http.NewServeMux()
	mainMux.HandleFunc("/api/whoami", authWhoAmI(token))
	mainMux.HandleFunc("/api/csrftoken", authCSRF(token))
	mainSrv := httptest.NewServer(mainMux)
	defer mainSrv.Close()

	subMux := http.NewServeMux()
	subMux.HandleFunc("/api/whoami", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
	subMux.HandleFunc("/api/csrftoken", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
	subSrv := httptest.NewServer(subMux)
	defer subSrv.Close()

	orig := mainInstanceURLForLogin
	mainInstanceURLForLogin = func(instanceURL string) (string, bool) {
		if instanceURL == subSrv.URL {
			return mainSrv.URL, true
		}
		return "", false
	}
	t.Cleanup(func() { mainInstanceURLForLogin = orig })

	client := NewClient(subSrv.URL, token, "", "")
	if err := client.Login(); err != nil {
		t.Fatalf("Login: %v", err)
	}
	if client.BaseURL() != mainSrv.URL {
		t.Fatalf("baseURL = %q, want %q", client.BaseURL(), mainSrv.URL)
	}
	if client.LoginRedirectNote() == "" {
		t.Fatal("expected login redirect note")
	}
}

func authWhoAmI(token string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Login "+token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 1, "firstName": "Max", "lastName": "Mustermann"},
		})
	}
}

func authCSRF(token string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Login "+token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": "csrf-test"})
	}
}
