package churchtools

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func noSleep(t *testing.T) {
	t.Helper()
	orig := sleep
	sleep = func(time.Duration) {}
	t.Cleanup(func() { sleep = orig })
}

func writeWhoAmI(w http.ResponseWriter) {
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": map[string]any{"id": 1, "firstName": "A", "lastName": "B", "email": "a@b.de"},
	})
}

func TestCallAPIRetriesOnceAfter429(t *testing.T) {
	noSleep(t)
	calls := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/api/whoami", func(w http.ResponseWriter, r *http.Request) { writeWhoAmI(w) })
	mux.HandleFunc("/api/csrftoken", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": "csrf"})
	})
	mux.HandleFunc("/api/persons/7", func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.Header().Set("Retry-After", "1")
			http.Error(w, "slow down", http.StatusTooManyRequests)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": 7, "firstName": "X"}})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(server.URL, "token", "", "")
	if err := client.Login(); err != nil {
		t.Fatal(err)
	}
	person, err := client.PersonByID(7)
	if err != nil || person.ID != 7 {
		t.Fatalf("PersonByID = %+v, %v", person, err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}

// A password session that expires must be restored with username/password.
func TestPasswordSessionReloginAfter401(t *testing.T) {
	logins := 0
	sessionValid := false
	mux := http.NewServeMux()
	mux.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
		logins++
		sessionValid = true
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"status": "success"}})
	})
	mux.HandleFunc("/api/whoami", func(w http.ResponseWriter, r *http.Request) {
		if !sessionValid {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		writeWhoAmI(w)
	})
	mux.HandleFunc("/api/csrftoken", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": "csrf"})
	})
	mux.HandleFunc("/api/test", func(w http.ResponseWriter, r *http.Request) {
		if !sessionValid {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(server.URL, "", "user", " pass word ")
	if err := client.Login(); err != nil {
		t.Fatal(err)
	}
	sessionValid = false // session expires

	if _, err := client.getAPI("/test", nil); err != nil {
		t.Fatalf("getAPI after expiry: %v", err)
	}
	if logins != 2 {
		t.Fatalf("logins = %d, want 2 (initial + relogin)", logins)
	}
}

func TestEnrichPersonsKeepsListDataOnFailure(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/whoami", func(w http.ResponseWriter, r *http.Request) { writeWhoAmI(w) })
	mux.HandleFunc("/api/csrftoken", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": "csrf"})
	})
	mux.HandleFunc("/api/persons/1", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": 1, "firstName": "Voll", "city": "Mainz"}})
	})
	mux.HandleFunc("/api/persons/2", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "gone", http.StatusNotFound)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(server.URL, "token", "", "")
	if err := client.Login(); err != nil {
		t.Fatal(err)
	}

	progress := 0
	persons, failures, err := client.EnrichPersons(
		[]Person{{ID: 1, FirstName: "Liste"}, {ID: 2, FirstName: "Nur Liste"}},
		EnrichOptions{Progress: func(done, total int) { progress = done }},
	)
	if err != nil {
		t.Fatalf("EnrichPersons: %v", err)
	}
	if len(failures) != 1 || failures[0].PersonID != 2 {
		t.Fatalf("failures = %+v", failures)
	}
	if persons[0].FirstName != "Voll" || persons[1].FirstName != "Nur Liste" {
		t.Fatalf("persons = %+v", persons)
	}
	if progress != 2 {
		t.Fatalf("progress = %d, want 2", progress)
	}
}

func TestFormatExportDateUsesLocalTime(t *testing.T) {
	orig := time.Local
	time.Local = time.FixedZone("CET", 3600)
	t.Cleanup(func() { time.Local = orig })

	cases := map[string]string{
		"2026-02-28T23:30:00Z":      "01.03.2026", // 00:30 local
		"2026-02-28T22:30:00Z":      "28.02.2026", // 23:30 local
		"2026-02-28":                "28.02.2026", // pure date stays
		"2026-02-28T23:30:00+01:00": "28.02.2026",
		"28.02.2026":                "28.02.2026", // already formatted
	}
	for in, want := range cases {
		if got := FormatExportDate(in); got != want {
			t.Errorf("FormatExportDate(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIntFromFlexibleFieldString(t *testing.T) {
	if id, ok := intFromFlexibleField([]byte(`"3"`)); !ok || id != 3 {
		t.Fatalf("intFromFlexibleField(\"3\") = %d, %v", id, ok)
	}
	if _, ok := intFromFlexibleField([]byte(`"abc"`)); ok {
		t.Fatal("non-numeric string must not parse")
	}
}

func TestNewAPIErrorTruncatesBody(t *testing.T) {
	err := newAPIError(500, "", []byte(strings.Repeat("x", 10_000)))
	if len(err.Body) > 2100 {
		t.Fatalf("body length = %d, want ≤ ~2 KB", len(err.Body))
	}
}

// Password sessions must send the CSRF token on write requests (REST and
// legacy AJAX) under the header name "CSRF-Token" – not on GET or /api/login.
func TestCSRFHeaderOnWriteRequests(t *testing.T) {
	seen := map[string]string{}
	record := func(w http.ResponseWriter, r *http.Request) {
		seen[r.Method+" "+r.URL.Path] = r.Header.Get("CSRF-Token")
		if r.Header.Get("X-CSRF-Token") != "" {
			t.Errorf("%s %s: X-CSRF-Token must not be used", r.Method, r.URL.Path)
		}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
		record(w, r)
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"status": "success"}})
	})
	mux.HandleFunc("/api/whoami", func(w http.ResponseWriter, r *http.Request) {
		record(w, r)
		writeWhoAmI(w)
	})
	mux.HandleFunc("/api/csrftoken", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": "csrf-secret"})
	})
	mux.HandleFunc("/api/persons/10/relationships", func(w http.ResponseWriter, r *http.Request) {
		record(w, r)
		http.NotFound(w, r) // forces the legacy fallback
	})
	mux.HandleFunc("/api/groups/3/members/10", func(w http.ResponseWriter, r *http.Request) {
		record(w, r)
		if r.Method == http.MethodGet {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		record(w, r)
		_, _ = w.Write([]byte(`{"status":"success"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(server.URL, "", "user", "pass")
	if err := client.Login(); err != nil {
		t.Fatal(err)
	}
	if err := client.LinkAsDuplicate(10, 11, RelationshipType{ID: 5}); err != nil {
		t.Fatalf("LinkAsDuplicate: %v", err)
	}
	if _, err := client.RequestGroupMembership(3, 10); err != nil {
		t.Fatalf("RequestGroupMembership: %v", err)
	}

	want := map[string]string{
		"POST /api/login":                    "",
		"GET /api/whoami":                    "",
		"GET /api/groups/3/members/10":       "",
		"POST /api/persons/10/relationships": "csrf-secret",
		"PUT /api/groups/3/members/10":       "csrf-secret",
		"POST /":                             "csrf-secret", // legacy AJAX API
	}
	for key, header := range want {
		got, ok := seen[key]
		if !ok {
			t.Errorf("%s: request not seen", key)
			continue
		}
		if got != header {
			t.Errorf("%s: CSRF-Token = %q, want %q", key, got, header)
		}
	}
}
