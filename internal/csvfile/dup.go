package csvfile

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	churchtools "github.com/janmz/churchtools-dup/internal/churchtools"
)

// DupHeader is the canonical CSV header for duplicate export/import files.
var DupHeader = []string{
	"DupID",
	"ID",
	"Vorname",
	"Nachname",
	"E-Mail",
	"Straße",
	"Stadt",
	"Standort",
	"Erstellungsdatum",
	"Einladungsstatus",
}

// DupEntry is one row in a duplicate CSV file.
type DupEntry struct {
	Line        int
	DupID       int
	PersonID    int
	FirstName   string
	LastName    string
	Email       string
	Street      string
	City        string
	CampusName       string
	CreatedAt        string
	InvitationStatus string
}

// DupEntryFromPerson maps a ChurchTools person to a duplicate export row.
func DupEntryFromPerson(dupID int, person churchtools.Person, campusName string) DupEntry {
	return DupEntry{
		DupID:       dupID,
		PersonID:    person.ID,
		FirstName:   person.FirstName,
		LastName:    person.LastName,
		Email:       person.PrimaryEmail(),
		Street:      person.Street,
		City:        person.City,
		CampusName:  campusName,
		CreatedAt:        churchtools.FormatExportDate(person.CreatedAt),
		InvitationStatus: person.ExportStatusLabel(),
	}
}

// WriteDup stores duplicate rows in the canonical CSV format.
func WriteDup(path string, entries []DupEntry) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("csv erstellen: %w", err)
	}
	defer file.Close()

	if err := WriteDupTo(file, entries); err != nil {
		return err
	}
	return file.Close()
}

// WriteDupTo writes duplicate rows to w.
func WriteDupTo(w io.Writer, entries []DupEntry) error {
	if _, err := w.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return fmt.Errorf("bom schreiben: %w", err)
	}

	writer := newCSVWriter(w)
	if err := writer.Write(DupHeader); err != nil {
		return fmt.Errorf("kopfzeile schreiben: %w", err)
	}

	for _, entry := range entries {
		if err := writer.Write([]string{
			strconv.Itoa(entry.DupID),
			strconv.Itoa(entry.PersonID),
			entry.FirstName,
			entry.LastName,
			entry.Email,
			entry.Street,
			entry.City,
			entry.CampusName,
			entry.CreatedAt,
			entry.InvitationStatus,
		}); err != nil {
			return fmt.Errorf("zeile schreiben: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("csv abschließen: %w", err)
	}
	return nil
}

// ReadDup parses a duplicate CSV file.
func ReadDup(path string) ([]DupEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("csv öffnen: %w", err)
	}

	reader, err := newCSVReader(data)
	if err != nil {
		return nil, err
	}

	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("csv-kopfzeile lesen: %w", err)
	}

	index := mapDupColumns(normalizeHeader(header))
	if _, ok := index["dupid"]; !ok {
		return nil, errors.New("csv benötigt eine DupID-spalte")
	}
	if _, ok := index["id"]; !ok {
		return nil, errors.New("csv benötigt eine ID-spalte")
	}

	var entries []DupEntry
	line := 1
	for {
		line++
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("zeile %d lesen: %w", line, err)
		}
		if isEmptyRecord(record) {
			continue
		}

		entry, err := parseDupRecord(line, record, index)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	if len(entries) == 0 {
		return nil, errors.New("csv enthält keine datensätze")
	}
	return entries, nil
}

// GroupDupEntries groups rows by DupID preserving file order within each group.
func GroupDupEntries(entries []DupEntry) [][]DupEntry {
	byDupID := make(map[int][]DupEntry)
	order := make([]int, 0)
	seen := make(map[int]struct{})

	for _, entry := range entries {
		if _, ok := seen[entry.DupID]; !ok {
			seen[entry.DupID] = struct{}{}
			order = append(order, entry.DupID)
		}
		byDupID[entry.DupID] = append(byDupID[entry.DupID], entry)
	}

	sort.Ints(order)
	groups := make([][]DupEntry, 0, len(order))
	for _, dupID := range order {
		groups = append(groups, byDupID[dupID])
	}
	return groups
}

func mapDupColumns(header []string) map[string]int {
	index := make(map[string]int, len(header))
	for i, name := range header {
		index[name] = i
	}

	result := make(map[string]int)
	if col, ok := findColumn(index, []string{"dupid", "dup_id", "gruppe"}); ok {
		result["dupid"] = col
	}
	if col, ok := findColumn(index, idColumns); ok {
		result["id"] = col
	}
	if col, ok := findColumn(index, firstNameColumns); ok {
		result["firstname"] = col
	}
	if col, ok := findColumn(index, lastNameColumns); ok {
		result["lastname"] = col
	}
	if col, ok := findColumn(index, emailColumns); ok {
		result["email"] = col
	}
	if col, ok := findColumn(index, []string{"straße", "strasse", "street", "address"}); ok {
		result["street"] = col
	}
	if col, ok := findColumn(index, []string{"stadt", "city", "ort"}); ok {
		result["city"] = col
	}
	if col, ok := findColumn(index, []string{"standort", "campus", "campus_name"}); ok {
		result["campus"] = col
	}
	if col, ok := findColumn(index, []string{"erstellungsdatum", "created_at", "createdat"}); ok {
		result["created_at"] = col
	}
	if col, ok := findColumn(index, []string{"einladungsstatus", "invitation_status", "status"}); ok {
		result["invitation_status"] = col
	}
	return result
}

func parseDupRecord(line int, record []string, index map[string]int) (DupEntry, error) {
	dupText := strings.TrimSpace(fieldAt(record, index["dupid"]))
	if dupText == "" {
		return DupEntry{}, fmt.Errorf("zeile %d: DupID fehlt", line)
	}
	dupID, err := strconv.Atoi(dupText)
	if err != nil || dupID <= 0 {
		return DupEntry{}, fmt.Errorf("zeile %d: ungültige DupID %q", line, dupText)
	}

	idText := strings.TrimSpace(fieldAt(record, index["id"]))
	if idText == "" {
		return DupEntry{}, fmt.Errorf("zeile %d: ID fehlt", line)
	}
	personID, err := strconv.Atoi(idText)
	if err != nil || personID <= 0 {
		return DupEntry{}, fmt.Errorf("zeile %d: ungültige ID %q", line, idText)
	}

	entry := DupEntry{
		Line:     line,
		DupID:    dupID,
		PersonID: personID,
	}
	if col, ok := index["firstname"]; ok {
		entry.FirstName = strings.TrimSpace(fieldAt(record, col))
	}
	if col, ok := index["lastname"]; ok {
		entry.LastName = strings.TrimSpace(fieldAt(record, col))
	}
	if col, ok := index["email"]; ok {
		entry.Email = strings.TrimSpace(fieldAt(record, col))
	}
	if col, ok := index["street"]; ok {
		entry.Street = strings.TrimSpace(fieldAt(record, col))
	}
	if col, ok := index["city"]; ok {
		entry.City = strings.TrimSpace(fieldAt(record, col))
	}
	if col, ok := index["campus"]; ok {
		entry.CampusName = strings.TrimSpace(fieldAt(record, col))
	}
	if col, ok := index["created_at"]; ok {
		entry.CreatedAt = strings.TrimSpace(fieldAt(record, col))
	}
	if col, ok := index["invitation_status"]; ok {
		entry.InvitationStatus = strings.TrimSpace(fieldAt(record, col))
	}
	return entry, nil
}
