package churchtools

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestOAuthSubInstanceLogin(t *testing.T) {
	const password = "secret"

	var centralSrv *httptest.Server
	var subSrv *httptest.Server

	centralMux := http.NewServeMux()
	centralMux.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "central_session", Value: "ok", Path: "/"})
		w.WriteHeader(http.StatusOK)
	})
	centralMux.HandleFunc("/oauth/authorize", func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie("central_session"); err != nil || cookie.Value != "ok" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		redirectURI := r.URL.Query().Get("redirect_uri")
		u, err := url.Parse(redirectURI)
		if err != nil {
			http.Error(w, "bad redirect", http.StatusBadRequest)
			return
		}
		q := u.Query()
		q.Set("code", "test-code")
		u.RawQuery = q.Encode()
		http.Redirect(w, r, u.String(), http.StatusFound)
	})

	subMux := http.NewServeMux()
	subMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	subMux.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "use oauth", http.StatusUnauthorized)
	})
	subMux.HandleFunc("/oauthclients/2/startlogin", func(w http.ResponseWriter, r *http.Request) {
		callback := subSrv.URL + "/oauthclients/2/login/callback"
		authorize := centralSrv.URL + "/oauth/authorize?redirect_uri=" +
			url.QueryEscape(callback) + "&response_type=code&client_id=test"
		http.Redirect(w, r, authorize, http.StatusFound)
	})
	subMux.HandleFunc("/oauthclients/2/login/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("code") == "" {
			http.Error(w, "missing code", http.StatusBadRequest)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "sub_session", Value: "ok", Path: "/"})
		http.Redirect(w, r, "/", http.StatusFound)
	})
	subMux.HandleFunc("/api/whoami", func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie("sub_session"); err != nil || cookie.Value != "ok" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 42, "firstName": "Max", "lastName": "Mustermann"},
		})
	})
	subMux.HandleFunc("/api/csrftoken", func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie("sub_session"); err != nil || cookie.Value != "ok" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": "csrf-sub"})
	})
	subMux.HandleFunc("/api/person/me/apitoken", func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie("sub_session"); err != nil || cookie.Value != "ok" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": "sub-api-token"})
	})

	centralSrv = httptest.NewServer(centralMux)
	defer centralSrv.Close()
	subSrv = httptest.NewServer(subMux)
	defer subSrv.Close()

	orig := mainInstanceURLForLogin
	mainInstanceURLForLogin = func(instanceURL string) (string, bool) {
		if instanceURL == subSrv.URL {
			return centralSrv.URL, true
		}
		return "", false
	}
	t.Cleanup(func() { mainInstanceURLForLogin = orig })

	client := NewClient(subSrv.URL, "", "jan@example.org", password)
	if err := client.Login(); err != nil {
		t.Fatalf("Login: %v", err)
	}
	if client.BaseURL() != subSrv.URL {
		t.Fatalf("baseURL = %q, want sub %q", client.BaseURL(), subSrv.URL)
	}
	if client.LoginRedirectNote() == "" {
		t.Fatal("expected oauth login note")
	}

	token, err := client.MeAPIToken()
	if err != nil {
		t.Fatalf("MeAPIToken: %v", err)
	}
	if token != "sub-api-token" {
		t.Fatalf("token = %q", token)
	}
}

func TestDiscoverOAuthStartLoginURL(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/oauthclients/1/startlogin", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "missing", http.StatusInternalServerError)
	})
	mux.HandleFunc("/oauthclients/2/startlogin", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://central.example/oauth/authorize", http.StatusFound)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	jarClient := NewClient(srv.URL, "", "", "").http
	got, err := discoverOAuthStartLoginURL(jarClient, srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	want := srv.URL + "/oauthclients/2/startlogin"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
