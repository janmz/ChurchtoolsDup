package duplicates

import (
	"sort"
	"strconv"
	"strings"
	"unicode"

	churchtools "github.com/janmz/churchtools-dup/internal/churchtools"
	csvfile "github.com/janmz/churchtools-dup/internal/csvfile"
)

// Group is a set of persons considered duplicates.
type Group struct {
	DupID   int
	Persons []churchtools.Person
}

// FindGroups detects duplicate clusters for a campus within the full person list.
// Only groups with at least one person at campusID are returned.
func FindGroups(campusID int, allPersons []churchtools.Person) []Group {
	if campusID <= 0 || len(allPersons) < 2 {
		return nil
	}
	return findGroups(allPersons, campusID, true)
}

// FindAllGroups detects duplicate clusters across the full person list without
// filtering by campus.
func FindAllGroups(allPersons []churchtools.Person) []Group {
	if len(allPersons) < 2 {
		return nil
	}
	return findGroups(allPersons, 0, false)
}

func findGroups(allPersons []churchtools.Person, campusID int, requireCampus bool) []Group {
	byID := make(map[int]churchtools.Person, len(allPersons))
	var campusIDs map[int]struct{}
	if requireCampus {
		campusIDs = make(map[int]struct{})
		for _, person := range allPersons {
			byID[person.ID] = person
			if person.CampusID == campusID {
				campusIDs[person.ID] = struct{}{}
			}
		}
	} else {
		for _, person := range allPersons {
			byID[person.ID] = person
		}
	}

	uf := newUnionFind()
	linkByEmail(allPersons, uf)
	linkByNameBlocks(allPersons, uf)

	components := uf.components()
	groups := make([]Group, 0)
	dupID := 1

	for _, ids := range components {
		if len(ids) < 2 {
			continue
		}
		if requireCampus && !containsCampusPerson(ids, campusIDs) {
			continue
		}

		persons := make([]churchtools.Person, 0, len(ids))
		for _, id := range ids {
			if person, ok := byID[id]; ok {
				persons = append(persons, person)
			}
		}
		if len(persons) < 2 {
			continue
		}

		sort.Slice(persons, func(i, j int) bool {
			if persons[i].ID == persons[j].ID {
				return false
			}
			return persons[i].ID < persons[j].ID
		})

		groups = append(groups, Group{DupID: dupID, Persons: persons})
		dupID++
	}

	sort.Slice(groups, func(i, j int) bool {
		return groups[i].DupID < groups[j].DupID
	})
	return groups
}

// GroupsToEntries flattens duplicate groups into CSV rows.
func GroupsToEntries(groups []Group, campusNames map[int]string) []csvfile.DupEntry {
	entries := make([]csvfile.DupEntry, 0)
	for _, group := range groups {
		for _, person := range group.Persons {
			entries = append(entries, csvfile.DupEntryFromPerson(
				group.DupID,
				person,
				churchtools.CampusLabel(person, campusNames),
			))
		}
	}
	return entries
}

func containsCampusPerson(ids []int, campusIDs map[int]struct{}) bool {
	for _, id := range ids {
		if _, ok := campusIDs[id]; ok {
			return true
		}
	}
	return false
}

func linkByEmail(persons []churchtools.Person, uf *unionFind) {
	byEmail := make(map[string][]churchtools.Person)
	for _, person := range persons {
		email := normalizeEmail(person.PrimaryEmail())
		if email == "" {
			continue
		}
		byEmail[email] = append(byEmail[email], person)
	}

	for _, group := range byEmail {
		for i := 0; i < len(group); i++ {
			for j := i + 1; j < len(group); j++ {
				if isSharedEmailCouple(group[i], group[j]) {
					continue
				}
				uf.union(group[i].ID, group[j].ID)
			}
		}
	}
}

func linkByNameBlocks(persons []churchtools.Person, uf *unionFind) {
	blocks := make(map[string][]churchtools.Person)
	for _, person := range persons {
		for _, key := range blockingKeys(person) {
			blocks[key] = append(blocks[key], person)
		}
	}

	seen := make(map[string]struct{})
	for _, block := range blocks {
		if len(block) < 2 {
			continue
		}
		for i := 0; i < len(block); i++ {
			for j := i + 1; j < len(block); j++ {
				a := block[i]
				b := block[j]
				if a.ID == b.ID {
					continue
				}
				pairKey := pairKey(a.ID, b.ID)
				if _, ok := seen[pairKey]; ok {
					continue
				}
				if personsMatch(a, b) {
					seen[pairKey] = struct{}{}
					uf.union(a.ID, b.ID)
				}
			}
		}
	}
}

func pairKey(a, b int) string {
	if a > b {
		a, b = b, a
	}
	return strconv.Itoa(a) + ":" + strconv.Itoa(b)
}

func blockingKeys(person churchtools.Person) []string {
	cityMain, _ := normalizeCity(person.City)
	street := normalizeStreet(person.Street)
	first := nameTokens(person.FirstName)
	last := nameTokens(person.LastName)

	keys := make([]string, 0, 8)
	if len(first) > 0 && cityMain != "" && street != "" {
		keys = append(keys, "fcs:"+first[0]+"|"+cityMain+"|"+street)
	}
	if len(first) > 0 && len(last) > 0 {
		keys = append(keys, "fl:"+first[0]+"|"+last[0])
		keys = append(keys, "lf:"+last[0]+"|"+first[0])
	}
	return keys
}

func personsMatch(a, b churchtools.Person) bool {
	emailA := normalizeEmail(a.PrimaryEmail())
	emailB := normalizeEmail(b.PrimaryEmail())
	if emailA != "" && emailA == emailB {
		if isSharedEmailCouple(a, b) {
			return false
		}
		return true
	}

	streetA := normalizeStreet(a.Street)
	streetB := normalizeStreet(b.Street)
	sameCity := compareCity(a.City, b.City)
	sameStreet := streetA != "" && streetA == streetB

	if sameCity && sameStreet && firstNamesMatch(a.FirstName, b.FirstName) {
		return true
	}

	if firstNamesMatch(a.FirstName, b.FirstName) && lastNamesMatch(a.LastName, b.LastName) {
		return true
	}

	if firstNamesMatch(a.FirstName, b.LastName) && lastNamesMatch(a.LastName, b.FirstName) {
		return true
	}

	return false
}

// isSharedEmailCouple reports spouses/partners sharing one mailbox at the same address.
func isSharedEmailCouple(a, b churchtools.Person) bool {
	emailA := normalizeEmail(a.PrimaryEmail())
	emailB := normalizeEmail(b.PrimaryEmail())
	if emailA == "" || emailA != emailB {
		return false
	}
	if a.CampusID != b.CampusID {
		return false
	}
	if !compareCity(a.City, b.City) {
		return false
	}
	streetA := normalizeStreet(a.Street)
	streetB := normalizeStreet(b.Street)
	if streetA == "" || streetA != streetB {
		return false
	}
	return !firstNamesMatch(a.FirstName, b.FirstName)
}

func firstNamesMatch(a, b string) bool {
	tokensA := nameTokens(a)
	tokensB := nameTokens(b)
	if len(tokensA) == 0 || len(tokensB) == 0 {
		return false
	}

	if joinTokens(tokensA) == joinTokens(tokensB) {
		return true
	}

	if tokensSubset(tokensA, tokensB) || tokensSubset(tokensB, tokensA) {
		return true
	}

	return initialsMatch(tokensA, tokensB)
}

func lastNamesMatch(a, b string) bool {
	tokensA := nameTokens(a)
	tokensB := nameTokens(b)
	if len(tokensA) == 0 || len(tokensB) == 0 {
		return false
	}

	if joinTokens(tokensA) == joinTokens(tokensB) {
		return true
	}

	return tokensSubset(tokensA, tokensB) || tokensSubset(tokensB, tokensA)
}

func initialsMatch(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !tokenMatchesInitial(a[i], b[i]) {
			return false
		}
	}
	return true
}

func tokenMatchesInitial(a, b string) bool {
	if a == b {
		return true
	}
	if len(a) == 1 && strings.HasPrefix(b, a) {
		return true
	}
	if len(b) == 1 && strings.HasPrefix(a, b) {
		return true
	}
	if len(a) > 1 && len(b) == 1 && strings.HasPrefix(a, b) {
		return true
	}
	if len(b) > 1 && len(a) == 1 && strings.HasPrefix(b, a) {
		return true
	}
	return false
}

func tokensSubset(smaller, larger []string) bool {
	if len(smaller) == 0 || len(smaller) > len(larger) {
		return false
	}
	for _, token := range smaller {
		if !containsToken(larger, token) {
			return false
		}
	}
	return true
}

func containsToken(tokens []string, token string) bool {
	for _, candidate := range tokens {
		if candidate == token {
			return true
		}
	}
	return false
}

func joinTokens(tokens []string) string {
	return strings.Join(tokens, " ")
}

func nameTokens(value string) []string {
	value = normalizeName(value)
	if value == "" {
		return nil
	}
	parts := strings.Fields(value)
	tokens := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.Trim(part, ".")
		if part == "" {
			continue
		}
		tokens = append(tokens, part)
	}
	return tokens
}

func normalizeName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", " ")
	value = strings.ReplaceAll(value, ".", " ")
	value = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			return r
		}
		return ' '
	}, value)
	return strings.Join(strings.Fields(value), " ")
}

/*
 * compareCity compares two city names and returns true if they are the same.
 * The names are normalized and compared case-insensitively.
 * The names are split into a main name and a secondary name if present.
 * The main name is compared case-insensitively.
 * The secondary name is compared case-insensitively or could be omitted or abbreviated.
 */
func compareCity(a, b string) bool {
	aName, aByName := normalizeCity(a)
	bName, bByName := normalizeCity(b)
	if aName == "" || bName == "" {
		return false
	}
	if aByName == "" || bByName == "" {
		// If one byname is not present, just compare the main name.
		return aName == bName
	}
	if aName != bName {
		return false
	}
	if aByName == bByName {
		return true
	}
	if (len(aByName) == 1 && len(bByName) > 1 && aByName[0] == bByName[0]) ||
		(len(aByName) > 1 && len(bByName) == 1 && aByName[0] == bByName[0]) {
		return true
	}
	return false
}

// normalizeCity splits a city into main name and optional locality suffix.
func normalizeCity(value string) (string, string) {
	byNamePrefixes := []string{"an der", "a.d.", "a. d.", "in der", "i.d.", "i. d.", "am", "a.", "im", "i.", "in", "bei", "b.", "ob der", "o.d.", "o. d.", "vor der", "v.d.", "v. d.", "unter", "u."}
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "", ""
	}
	if index := strings.Index(value, "/"); index > 0 {
		main := strings.TrimSpace(value[:index])
		byName := strings.TrimRight(strings.TrimSpace(value[index+1:]), ".")
		return main, byName
	}
	for _, name := range byNamePrefixes {
		namePad := " " + name + " "
		if index := strings.Index(value, namePad); index > 0 {
			main := strings.TrimSpace(value[:index])
			byName := strings.TrimRight(strings.TrimSpace(value[index+len(namePad):]), ".")
			return main, byName
		}
	}
	return value, ""
}

func normalizeStreet(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}

	value = strings.ReplaceAll(value, "ß", "ss")
	value = insertSpaceAfterDots(value)
	value = strings.ReplaceAll(value, "-", " ")
	value = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			return r
		}
		return ' '
	}, value)

	tokens := strings.Fields(value)
	for i, token := range tokens {
		tokens[i] = canonicalizeStreetToken(token)
	}
	return strings.Join(tokens, " ")
}

func insertSpaceAfterDots(value string) string {
	runes := []rune(value)
	if len(runes) == 0 {
		return value
	}

	var b strings.Builder
	b.Grow(len(value) + 4)
	for i, r := range runes {
		b.WriteRune(r)
		if r != '.' || i+1 >= len(runes) {
			continue
		}
		next := runes[i+1]
		if next != ' ' && (unicode.IsLetter(next) || unicode.IsDigit(next)) {
			b.WriteRune(' ')
		}
	}
	return b.String()
}

func canonicalizeStreetToken(token string) string {
	token = strings.Trim(token, ".")
	if token == "" {
		return ""
	}

	if token == "str" || token == "strasse" {
		return "strasse"
	}

	if strings.HasSuffix(token, "straße") {
		return strings.TrimSuffix(token, "straße") + "strasse"
	}
	if strings.HasSuffix(token, "strasse") {
		return token
	}
	if strings.HasSuffix(token, "str") {
		return strings.TrimSuffix(token, "str") + "strasse"
	}
	return token
}

func normalizeEmail(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

type unionFind struct {
	parent map[int]int
}

func newUnionFind() *unionFind {
	return &unionFind{parent: make(map[int]int)}
}

func (uf *unionFind) find(id int) int {
	parent, ok := uf.parent[id]
	if !ok {
		uf.parent[id] = id
		return id
	}
	if parent != id {
		uf.parent[id] = uf.find(parent)
	}
	return uf.parent[id]
}

func (uf *unionFind) union(a, b int) {
	rootA := uf.find(a)
	rootB := uf.find(b)
	if rootA == rootB {
		return
	}
	uf.parent[rootB] = rootA
}

func (uf *unionFind) components() [][]int {
	buckets := make(map[int][]int)
	for id := range uf.parent {
		root := uf.find(id)
		buckets[root] = append(buckets[root], id)
	}

	roots := make([]int, 0, len(buckets))
	for root := range buckets {
		roots = append(roots, root)
	}
	sort.Ints(roots)

	result := make([][]int, 0, len(roots))
	for _, root := range roots {
		ids := append([]int(nil), buckets[root]...)
		sort.Ints(ids)
		result = append(result, ids)
	}
	return result
}
