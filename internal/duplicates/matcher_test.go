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

	if groups := FindGroups(30, all); len(groups) != 0 {
		t.Fatalf("FindGroups(30) = %d groups, want 0", len(groups))
	}

	groups := FindAllGroups(all)
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

	groups := FindGroups(10, all)
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

	groups := FindGroups(10, all)
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
	if !personsMatch(a, b) {
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

	groups := FindGroups(10, all)
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
	if !personsMatch(a, b) {
		t.Fatal("expected swapped first/last name to match")
	}
}

func TestPersonsMatchCityStreetAndFirstName(t *testing.T) {
	a := person(1, 10, "Peter", "Lang", "", "Bahnhofstr. 7", "Darmstadt")
	b := person(2, 0, "Peter", "Kurz", "", "Bahnhofstr. 7", "Darmstadt")
	if !personsMatch(a, b) {
		t.Fatal("expected same first name, city and street to match")
	}
}

func TestPersonsMatchCityAndFirstNameRequiresStreet(t *testing.T) {
	a := person(1, 10, "Peter", "Lang", "", "Bahnhofstr. 7", "Darmstadt")
	b := person(2, 0, "Peter", "Kurz", "", "Marktplatz 1", "Darmstadt")
	if personsMatch(a, b) {
		t.Fatal("expected different streets not to match via city rule")
	}
}
