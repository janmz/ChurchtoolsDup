package duplicates_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	churchtools "github.com/janmz/churchtools-dup/internal/churchtools"
	csvfile "github.com/janmz/churchtools-dup/internal/csvfile"
	"github.com/janmz/churchtools-dup/internal/duplicates"
)

func TestImportRunnerDryRunDetectsExistingRelationship(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/whoami", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 1, "firstName": "A", "lastName": "B", "email": "a@example.org"},
		})
	})
	mux.HandleFunc("/api/csrftoken", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": "csrf"})
	})
	mux.HandleFunc("/api/persons/10005/relationships", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{
					"relationshipTypeId": 5,
					"relative": map[string]any{
						"apiUrl": "/api/persons/10006",
					},
				},
			},
		})
	})
	mux.HandleFunc("/api/persons/10005/groups", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": 9, "name": "Duplikate"},
			},
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := churchtools.NewClient(server.URL, "token", "", "")
	if err := client.Login(); err != nil {
		t.Fatal(err)
	}

	runner := duplicates.ImportRunner{
		Client:  client,
		RelType: churchtools.RelationshipType{ID: 5, Name: "Duplikat"},
	}
	groups := [][]csvfile.DupEntry{{
		{DupID: 3, PersonID: 10005},
		{DupID: 3, PersonID: 10006},
	}}

	results, err := runner.Run(groups, duplicates.ImportOptions{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || !results[0].Success {
		t.Fatalf("unexpected results: %+v", results)
	}
	if results[0].Linked != 0 || results[0].AlreadyLinked != 1 {
		t.Fatalf("counts = linked %d already %d", results[0].Linked, results[0].AlreadyLinked)
	}
	if !strings.Contains(results[0].Message, "bereits mit 10006 verknüpft") {
		t.Fatalf("message = %q", results[0].Message)
	}
	if !strings.Contains(results[0].Message, "bereits in Gruppe \"Duplikate\"") {
		t.Fatalf("message = %q", results[0].Message)
	}
	if duplicates.ImportResultStatus(results[0]) != "skipped" {
		t.Fatalf("expected skipped status, got %q", duplicates.ImportResultStatus(results[0]))
	}
}

func TestImportRunnerSkipsExistingRelationship(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/whoami", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 1, "firstName": "A", "lastName": "B", "email": "a@example.org"},
		})
	})
	mux.HandleFunc("/api/csrftoken", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": "csrf"})
	})
	mux.HandleFunc("/api/persons/10005/relationships", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			t.Fatal("unexpected POST for existing relationship")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{
					"relationshipTypeId": 5,
					"relative": map[string]any{
						"apiUrl": "/api/persons/10006",
					},
				},
			},
		})
	})
	mux.HandleFunc("/api/groups", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{"id": 9, "name": "Duplikate"}},
		})
	})
	mux.HandleFunc("/api/groups/9/members/10005", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/api/persons/10005/groups", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{"id": 9, "name": "Duplikate"}},
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := churchtools.NewClient(server.URL, "token", "", "")
	if err := client.Login(); err != nil {
		t.Fatal(err)
	}

	runner := duplicates.ImportRunner{
		Client:  client,
		RelType: churchtools.RelationshipType{ID: 5, Name: "Duplikat"},
	}
	groups := [][]csvfile.DupEntry{{
		{DupID: 3, PersonID: 10005},
		{DupID: 3, PersonID: 10006},
	}}

	results, err := runner.Run(groups, duplicates.ImportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || !results[0].Success {
		t.Fatalf("unexpected results: %+v", results)
	}
	if results[0].Message != "Person 10005: alle 1 Duplikat-Beziehung(en) bereits vorhanden; Person 10005 in Gruppe \"Duplikate\"" {
		t.Fatalf("message = %q", results[0].Message)
	}
	if results[0].Linked != 0 || results[0].AlreadyLinked != 1 {
		t.Fatalf("counts = linked %d already %d", results[0].Linked, results[0].AlreadyLinked)
	}
	if duplicates.ImportResultStatus(results[0]) != "skipped" {
		t.Fatalf("expected skipped status, got %q", duplicates.ImportResultStatus(results[0]))
	}
}

func TestImportRunnerCountsAlreadyLinkedWhenLinkIsNoOp(t *testing.T) {
	posts := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/api/whoami", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 1, "firstName": "A", "lastName": "B", "email": "a@example.org"},
		})
	})
	mux.HandleFunc("/api/csrftoken", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": "csrf"})
	})
	mux.HandleFunc("/api/persons/10005/relationships", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			posts++
			w.WriteHeader(http.StatusCreated)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{}})
	})
	mux.HandleFunc("/api/persons/10006/relationships", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{
					"relationshipTypeId": 5,
					"relative": map[string]any{
						"apiUrl": "/api/persons/10005",
					},
				},
			},
		})
	})
	mux.HandleFunc("/api/persons/10005/groups", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{"id": 9, "name": "Duplikate"}},
		})
	})
	mux.HandleFunc("/api/groups", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{"id": 9, "name": "Duplikate"}},
		})
	})
	mux.HandleFunc("/api/groups/9/members/10005", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := churchtools.NewClient(server.URL, "token", "", "")
	if err := client.Login(); err != nil {
		t.Fatal(err)
	}

	runner := duplicates.ImportRunner{
		Client:  client,
		RelType: churchtools.RelationshipType{ID: 5, Name: "Duplikat"},
	}
	groups := [][]csvfile.DupEntry{{
		{DupID: 3, PersonID: 10005},
		{DupID: 3, PersonID: 10006},
	}}

	results, err := runner.Run(groups, duplicates.ImportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if posts != 0 {
		t.Fatalf("expected no POST, got %d", posts)
	}
	if results[0].Linked != 0 || results[0].AlreadyLinked != 1 {
		t.Fatalf("counts = linked %d already %d", results[0].Linked, results[0].AlreadyLinked)
	}
	if duplicates.ImportResultStatus(results[0]) != "skipped" {
		t.Fatalf("expected skipped status, got %q", duplicates.ImportResultStatus(results[0]))
	}
}

func TestImportResultStatusDryRunVorgemerkt(t *testing.T) {
	cases := []struct {
		name   string
		result duplicates.ImportResult
		want   string
	}{
		{
			name:   "new duplicate",
			result: duplicates.ImportResult{Success: true, Linked: 1, AlreadyLinked: 0},
			want:   "ok",
		},
		{
			name:   "already linked",
			result: duplicates.ImportResult{Success: true, Linked: 0, AlreadyLinked: 1},
			want:   "skipped",
		},
		{
			name:   "already in duplicate group",
			result: duplicates.ImportResult{Success: true, Linked: 1, Vorgemerkt: true},
			want:   "skipped",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := duplicates.ImportResultStatus(tc.result); got != tc.want {
				t.Fatalf("status = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestImportRunnerDryRunSummaryVorgemerkt(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/whoami", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 1, "firstName": "A", "lastName": "B", "email": "a@example.org"},
		})
	})
	mux.HandleFunc("/api/csrftoken", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": "csrf"})
	})
	registerEmptyRelationships := func(personID int) {
		path := "/api/persons/" + strconv.Itoa(personID) + "/relationships"
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{}})
		})
	}
	registerDuplicateGroups := func(personID int) {
		path := "/api/persons/" + strconv.Itoa(personID) + "/groups"
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{{"id": 9, "name": "Duplikate"}},
			})
		})
	}
	for _, personID := range []int{1537, 12318, 7696, 12310, 7699, 12213} {
		registerEmptyRelationships(personID)
	}
	mux.HandleFunc("/api/persons/1537/groups", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{}})
	})
	registerDuplicateGroups(7696)
	registerDuplicateGroups(7699)

	server := httptest.NewServer(mux)
	defer server.Close()

	client := churchtools.NewClient(server.URL, "token", "", "")
	if err := client.Login(); err != nil {
		t.Fatal(err)
	}

	runner := duplicates.ImportRunner{
		Client:  client,
		RelType: churchtools.RelationshipType{ID: 5, Name: "Duplikat"},
	}
	groups := [][]csvfile.DupEntry{
		{{DupID: 1, PersonID: 1537}, {DupID: 1, PersonID: 12318}},
		{{DupID: 3, PersonID: 7696}, {DupID: 3, PersonID: 12310}},
		{{DupID: 4, PersonID: 7699}, {DupID: 4, PersonID: 12213}},
	}

	results, err := runner.Run(groups, duplicates.ImportOptions{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}

	ok, skipped, linkedTotal := 0, 0, 0
	for _, result := range results {
		switch duplicates.ImportResultStatus(result) {
		case "ok":
			ok++
			linkedTotal += result.Linked
		case "skipped":
			skipped++
		}
	}
	if ok != 1 || skipped != 2 || linkedTotal != 1 {
		t.Fatalf("summary = ok %d skipped %d linked %d, want 1/2/1", ok, skipped, linkedTotal)
	}
}

func TestPrintImportSummaryCountsExistingAsSkipped(t *testing.T) {
	results := []duplicates.ImportResult{
		{DupID: 2, Success: true, Linked: 1, AlreadyLinked: 0, Message: "Person 10003 mit 1 Dublette(n) verknüpft"},
		{DupID: 3, Success: true, Linked: 0, AlreadyLinked: 1, Message: "Person 10005: alle 1 Duplikat-Beziehung(en) bereits vorhanden"},
	}

	if duplicates.ImportResultStatus(results[0]) != "ok" {
		t.Fatalf("dup 2 status = %q", duplicates.ImportResultStatus(results[0]))
	}
	if duplicates.ImportResultStatus(results[1]) != "skipped" {
		t.Fatalf("dup 3 status = %q", duplicates.ImportResultStatus(results[1]))
	}
}
