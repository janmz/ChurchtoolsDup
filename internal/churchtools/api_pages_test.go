package churchtools

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetAPIRetryStopsAfterSecondUnauthorized(t *testing.T) {
	attempts := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/api/whoami", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 1, "firstName": "A", "lastName": "B", "email": "a@b.de"},
		})
	})
	mux.HandleFunc("/api/csrftoken", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": "csrf-test"})
	})
	mux.HandleFunc("/api/test", func(w http.ResponseWriter, r *http.Request) {
		attempts++
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(server.URL, "token", "", "")
	if err := client.Login(); err != nil {
		t.Fatalf("Login: %v", err)
	}

	_, err := client.getAPI("/test", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}

func TestFetchAPIPagesAbortsAtLimit(t *testing.T) {
	origMax := maxAPIPages
	maxAPIPages = 5 // keep the test fast: 6 instead of 10 001 requests
	t.Cleanup(func() { maxAPIPages = origMax })

	mux := http.NewServeMux()
	mux.HandleFunc("/api/whoami", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 1, "firstName": "A", "lastName": "B", "email": "a@b.de"},
		})
	})
	mux.HandleFunc("/api/csrftoken", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": "csrf-test"})
	})
	mux.HandleFunc("/api/items", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{"id": 1}},
			"meta": map[string]any{
				"pagination": map[string]any{"current": 1, "lastPage": maxAPIPages + 1},
			},
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(server.URL, "token", "", "")
	if err := client.Login(); err != nil {
		t.Fatalf("Login: %v", err)
	}

	_, err := client.fetchAPIPages("/items", nil)
	if err == nil {
		t.Fatal("expected pagination limit error")
	}
}

func TestFetchAPIPagesAbortsAtTotalSize(t *testing.T) {
	origMax := maxAPITotalBytes
	maxAPITotalBytes = 300 // a few small pages
	t.Cleanup(func() { maxAPITotalBytes = origMax })

	pages := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/api/whoami", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 1, "firstName": "A", "lastName": "B", "email": "a@b.de"},
		})
	})
	mux.HandleFunc("/api/csrftoken", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": "csrf-test"})
	})
	mux.HandleFunc("/api/items", func(w http.ResponseWriter, r *http.Request) {
		pages++
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{"id": 1, "name": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"}},
			"meta": map[string]any{"pagination": map[string]any{"current": 1, "lastPage": 100}},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(server.URL, "token", "", "")
	if err := client.Login(); err != nil {
		t.Fatal(err)
	}
	if _, err := client.fetchAPIPages("/items", nil); err == nil {
		t.Fatal("expected total size error")
	}
	if pages >= 100 {
		t.Fatalf("pages = %d, must stop early", pages)
	}
}
