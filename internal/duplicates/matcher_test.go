package duplicates

import (
	"testing"

	churchtools "github.com/janmz/churchtools-dup/internal/churchtools"
)

func person(id, campus int, first, last, email, street, city string) churchtools.Person {
	return churchtools.Person{
		ID:        id,
		CampusID:  campus,
		FirstName: first,
		LastName:  last,
		Email:     email,
		Street:    street,
		City:      city,
	}
}

func TestFindAllGroupsAcrossCampuses(t *testing.T) {
	all := []churchtools.Person{
		person(1, 10, "Max", "Muster", "max@example.org", "Hauptstr. 1", "Frankfurt"),
		person(2, 20, "Max", "Muster", "max@example.org", "Nebenweg 2", "Offenbach"),
		person(3, 30, "Erika", "Beispiel", "erika@example.org", "Gasse 3", "Mainz"),
	}

	if groups, _ := FindGroups(30, all); len(groups) != 0 {
		t.Fatalf("FindGroups(30) = %d groups, want 0", len(groups))
	}

	groups, _ := FindAllGroups(all)
	if len(groups) != 1 {
		t.Fatalf("FindAllGroups = %d groups, want 1", len(groups))
	}
	if len(groups[0].Persons) != 2 {
		t.Fatalf("expected 2 persons, got %d", len(groups[0].Persons))
	}
}

func TestFindGroupsIgnoresSharedEmailCouple(t *testing.T) {
	all := []churchtools.Person{
		person(1, 10, "Anna", "Beispiel", "paar@example.org", "Gartenweg 3", "Musterdorf"),
		person(2, 10, "Bernd", "Beispiel", "paar@example.org", "Gartenweg 3", "Musterdorf"),
	}

	groups, _ := FindGroups(10, all)
	if len(groups) != 0 {
		t.Fatalf("expected no groups for shared-email couple, got %d", len(groups))
	}
}

func TestFindGroupsByEmailDifferentAddressStillDuplicate(t *testing.T) {
	all := []churchtools.Person{
		person(1, 10, "Max", "Muster", "max@example.org", "Hauptstr. 1", "Frankfurt"),
		person(2, 0, "Max", "Muster", "max@example.org", "Nebenweg 2", "Offenbach"),
		person(3, 20, "Erika", "Beispiel", "erika@example.org", "Gasse 3", "Mainz"),
	}

	groups, _ := FindGroups(10, all)
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if len(groups[0].Persons) != 2 {
		t.Fatalf("expected 2 persons, got %d", len(groups[0].Persons))
	}
}

func TestCompareCityVariants(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"Frankfurt", "Frankfurt", true},
		{"Frankfurt", "Frankfurt am Main", true},
		{"Frankfurt/M.", "Frankfurt/Main", true},
		{"Frankfurt/M.", "Frankfurt/M", true},
		{"Frankfurt am Main", "Frankfurt a. Main", true},
		{"Trebur/Astheim", "Trebur/Astheim", true},
		{"Mainz", "Wiesbaden", false},
		{"Frankfurt a.d. Oder", "Frankfurt am Main", false},
	}
	for _, tc := range cases {
		if got := compareCity(tc.a, tc.b); got != tc.want {
			t.Errorf("compareCity(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestPersonsMatchCityWithLocalitySuffix(t *testing.T) {
	a := person(1, 10, "Peter", "Lang", "", "Bahnhofstr. 7", "Frankfurt")
	b := person(2, 0, "Peter", "Kurz", "", "Bahnhofstr. 7", "Frankfurt am Main")
	if !(matcher{}).personsMatch(a, b) {
		t.Fatal("expected Frankfurt and Frankfurt am Main to match with same street")
	}
}

func TestNormalizeStreetVariants(t *testing.T) {
	cases := []struct {
		a, b string
	}{
		{"Klarstr.", "Klarstraße"},
		{"Klarstr.", "Klarstrasse"},
		{"Friedrich-Ebert-Straße 16", "Friedrich Ebert Str.16"},
		{"Lindenweg 4", "Lindenweg  4"},
	}
	for _, tc := range cases {
		if normalizeStreet(tc.a) != normalizeStreet(tc.b) {
			t.Errorf("normalizeStreet(%q)=%q != normalizeStreet(%q)=%q",
				tc.a, normalizeStreet(tc.a), tc.b, normalizeStreet(tc.b))
		}
	}
}

func TestIsSharedEmailCouple(t *testing.T) {
	a := person(1, 10, "Anna", "Beispiel", "x@example.org", "Hauptstr. 1", "Mainz")
	b := person(2, 10, "Bernd", "Beispiel", "x@example.org", "Hauptstr. 1", "Mainz")
	if !isSharedEmailCouple(a, b) {
		t.Fatal("expected shared email couple")
	}
	c := person(3, 10, "Anna", "Beispiel", "x@example.org", "Hauptstr. 1", "Mainz")
	if isSharedEmailCouple(a, c) {
		t.Fatal("same first name should not be shared-email couple exclusion")
	}
}

func TestFindGroupsIgnoresUnrelatedCampus(t *testing.T) {
	all := []churchtools.Person{
		person(1, 10, "Max", "Muster", "a@example.org", "Hauptstr. 1", "Frankfurt"),
		person(2, 20, "Erika", "Beispiel", "b@example.org", "Gasse 2", "Mainz"),
	}

	groups, _ := FindGroups(10, all)
	if len(groups) != 0 {
		t.Fatalf("expected no groups, got %d", len(groups))
	}
}

func TestFirstNamesMatchVariants(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"Jan Oliver", "Jan", true},
		{"Maria-Luisa", "Maria Luisa", true},
		{"Jan O.", "Jan Oliver", true},
		{"Max", "Moritz", false},
	}
	for _, tc := range cases {
		if got := firstNamesMatch(tc.a, tc.b); got != tc.want {
			t.Errorf("firstNamesMatch(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestLastNamesMatchDoubleName(t *testing.T) {
	if !lastNamesMatch("Müller-Schmidt", "Müller") {
		t.Fatal("expected partial double last name to match")
	}
}

func TestPersonsMatchSwappedNames(t *testing.T) {
	a := person(1, 10, "Anna", "Berger", "", "Lindenweg 4", "Wiesbaden")
	b := person(2, 0, "Berger", "Anna", "", "Lindenweg 4", "Wiesbaden")
	if !(matcher{}).personsMatch(a, b) {
		t.Fatal("expected swapped first/last name to match")
	}
}

func TestPersonsMatchCityStreetAndFirstName(t *testing.T) {
	a := person(1, 10, "Peter", "Lang", "", "Bahnhofstr. 7", "Darmstadt")
	b := person(2, 0, "Peter", "Kurz", "", "Bahnhofstr. 7", "Darmstadt")
	if !(matcher{}).personsMatch(a, b) {
		t.Fatal("expected same first name, city and street to match")
	}
}

func TestPersonsMatchCityAndFirstNameRequiresStreet(t *testing.T) {
	a := person(1, 10, "Peter", "Lang", "", "Bahnhofstr. 7", "Darmstadt")
	b := person(2, 0, "Peter", "Kurz", "", "Marktplatz 1", "Darmstadt")
	if (matcher{}).personsMatch(a, b) {
		t.Fatal("expected different streets not to match via city rule")
	}
}

func TestFindAllGroupsSwappedNamesAndInitials(t *testing.T) {
	all := []churchtools.Person{
		person(1, 10, "Hans", "Meier", "", "", ""),
		person(2, 10, "Meier", "Hans", "", "", ""),
		person(3, 10, "A.", "Schulz", "", "", ""),
		person(4, 10, "Anna", "Schulz", "", "", ""),
		person(5, 10, "Clara", "Schmidt", "", "", ""),
		person(6, 10, "Clara", "Müller-Schmidt", "", "", ""),
	}

	groups, _ := FindAllGroups(all)
	if len(groups) != 3 {
		t.Fatalf("groups = %d, want 3 (swapped, initial, compound last name): %+v", len(groups), groups)
	}
	want := [][2]int{{1, 2}, {3, 4}, {5, 6}}
	for i, group := range groups {
		if len(group.Persons) != 2 || group.Persons[0].ID != want[i][0] || group.Persons[1].ID != want[i][1] {
			t.Fatalf("group %d = %+v, want IDs %v", i, group.Persons, want[i])
		}
	}
}

func TestFindAllGroupsStableDupIDs(t *testing.T) {
	all := []churchtools.Person{
		person(30, 10, "Max", "Muster", "max@example.org", "", ""),
		person(10, 10, "Max", "Muster", "", "Hauptstr. 1", "Mainz"),
		person(20, 10, "M.", "Muster", "", "Hauptstr. 1", "Mainz"),
		person(5, 10, "Erika", "Beispiel", "e@example.org", "", ""),
		person(40, 10, "Erika", "Beispiel", "e@example.org", "", ""),
	}

	first, _ := FindAllGroups(all)
	for i := 0; i < 50; i++ {
		groups, _ := FindAllGroups(all)
		if len(groups) != len(first) {
			t.Fatalf("run %d: %d groups, want %d", i, len(groups), len(first))
		}
		for g := range groups {
			if groups[g].DupID != first[g].DupID || groups[g].Persons[0].ID != first[g].Persons[0].ID {
				t.Fatalf("run %d: DupIDs differ between runs", i)
			}
		}
	}
	// Ordered by smallest person ID: Erika (5) gets DupID 1, Max (10) DupID 2.
	if first[0].Persons[0].ID != 5 || first[1].Persons[0].ID != 10 {
		t.Fatalf("unexpected order: %+v", first)
	}
}

func TestFindAllGroupsSkipsMassEmail(t *testing.T) {
	names := []string{"Anton", "Berta", "Cäsar", "Dora", "Emil", "Friedrich", "Gustav"}
	all := make([]churchtools.Person, 0, len(names))
	for i, name := range names {
		all = append(all, person(i+1, 10, name, "Nachname"+name, "info@gemeinde.de", "", ""))
	}

	groups, report := FindAllGroups(all)
	if len(groups) != 0 {
		t.Fatalf("groups = %d, want 0 for a mass e-mail address", len(groups))
	}
	if len(report.SharedEmails) != 1 || report.SharedEmails[0].Email != "info@gemeinde.de" || report.SharedEmails[0].Count != len(names) {
		t.Fatalf("report = %+v", report)
	}
}
