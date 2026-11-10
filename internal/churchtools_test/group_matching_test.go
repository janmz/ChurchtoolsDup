package churchtools_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"

	churchtools "github.com/janmz/churchtools-dup/internal/churchtools"
)

func TestFindGroupByNamePrefersExactMatch(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/groups", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": 1, "name": "ChurchTools Admin (alt)"},
				{"id": 2, "name": "churchtools admin"},
			},
		})
	})
	client := newLoggedInClient(t, mux)

	group, err := client.FindGroupByName("ChurchTools Admin")
	if err != nil {
		t.Fatalf("FindGroupByName: %v", err)
	}
	if group.ID != 2 {
		t.Fatalf("group = %+v, want exact (case-insensitive) match ID 2", group)
	}
}

func TestFindGroupByNameIgnoresSubstrings(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/groups", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": 1, "name": "Admin Nord"},
				{"id": 2, "name": "Admin Süd"},
			},
		})
	})
	client := newLoggedInClient(t, mux)

	// "Kurs ChurchTools Admin"-like names must never be joined for "Admin".
	if _, err := client.FindGroupByName("Admin"); !errors.Is(err, churchtools.ErrGroupNotFound) {
		t.Fatalf("err = %v, want ErrGroupNotFound (no substring matches)", err)
	}
}

func TestPersonIsInGroupRequiresExactName(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/persons/5/groups", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{"id": 9, "name": "Duplikate (alt)"}},
		})
	})
	client := newLoggedInClient(t, mux)

	in, err := client.PersonIsInGroup(5, "duplikate")
	if err != nil {
		t.Fatal(err)
	}
	if in {
		t.Fatal(`"Duplikate (alt)" must not count as membership in "Duplikate"`)
	}
}

func TestRequestGroupMembershipKeepsExistingMembership(t *testing.T) {
	puts := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/api/groups/9/members/5", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			puts++
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"groupMemberStatus": "active", "comment": "wichtig"},
		})
	})
	client := newLoggedInClient(t, mux)

	result, err := client.RequestGroupMembership(9, 5)
	if err != nil {
		t.Fatal(err)
	}
	if puts != 0 {
		t.Fatalf("PUT on existing membership would clear comment/end date; puts = %d", puts)
	}
	if result.Status != churchtools.MembershipActive || !result.AlreadyMember {
		t.Fatalf("result = %+v", result)
	}
}

func TestRequestGroupMembershipUsesOnlyAdd(t *testing.T) {
	var body map[string]any
	mux := http.NewServeMux()
	mux.HandleFunc("/api/groups/9/members/5", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			http.NotFound(w, r)
			return
		}
		data, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(data, &body)
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"groupMemberStatus": "active"}})
	})
	client := newLoggedInClient(t, mux)

	result, err := client.RequestGroupMembership(9, 5)
	if err != nil {
		t.Fatal(err)
	}
	if body["only_add"] != true {
		t.Fatalf("PUT body = %v, want only_add=true", body)
	}
	if result.Status != churchtools.MembershipActive || result.AlreadyMember {
		t.Fatalf("result = %+v", result)
	}
}

// A permanently denied pre-join group must be requested only once per run.
func TestEnsurePreJoinGroupsDoesNotRetryDeniedGroup(t *testing.T) {
	puts := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/api/persons/1/groups", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	})
	mux.HandleFunc("/api/groups", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{"id": 3, "name": "Geschlossen"}},
		})
	})
	mux.HandleFunc("/api/groups/3/members/1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			http.NotFound(w, r)
			return
		}
		puts++
		http.Error(w, `{"message":"nicht erlaubt"}`, http.StatusConflict)
	})
	client := newLoggedInClient(t, mux)

	results, err := client.EnsurePreJoinGroups([]string{"Geschlossen", "Fehlt"})
	if err != nil {
		t.Fatal(err)
	}
	if puts != 1 {
		t.Fatalf("PUT requests = %d, want 1", puts)
	}
	if len(results) != 2 || results[0].Status != churchtools.MembershipDenied {
		t.Fatalf("results = %+v", results)
	}
}

// Groups joined during the run are left again; existing memberships stay.
func TestLeaveJoinedGroupsOnlyRemovesNewMemberships(t *testing.T) {
	deleted := map[string]int{}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/groups/3/members/1", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			http.NotFound(w, r) // not a member yet
		case http.MethodDelete:
			deleted["3"]++
			w.WriteHeader(http.StatusNoContent)
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"groupMemberStatus": "active"}})
		}
	})
	mux.HandleFunc("/api/groups/4/members/1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deleted["4"]++
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"groupMemberStatus": "active"}})
	})
	mux.HandleFunc("/api/groups/9/members/77", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			http.NotFound(w, r)
			return
		}
		if r.Method == http.MethodDelete {
			deleted["9"]++
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"groupMemberStatus": "active"}})
	})
	client := newLoggedInClient(t, mux)

	for _, req := range [][2]int{{3, 1}, {4, 1}, {9, 77}} {
		if _, err := client.RequestGroupMembership(req[0], req[1]); err != nil {
			t.Fatal(err)
		}
	}

	results := client.LeaveJoinedGroups()
	if len(results) != 1 || results[0].GroupID != 3 || results[0].Err != nil {
		t.Fatalf("results = %+v, want only group 3", results)
	}
	if deleted["3"] != 1 || deleted["4"] != 0 || deleted["9"] != 0 {
		t.Fatalf("deleted = %v: only the user's new membership in group 3 may be removed", deleted)
	}
	if again := client.LeaveJoinedGroups(); len(again) != 0 {
		t.Fatalf("second call must be a no-op, got %+v", again)
	}
}
