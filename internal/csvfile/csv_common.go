package csvfile

import "strings"

var idColumns = []string{"id", "person_id", "personid", "ct_id", "churchtools_id"}
var firstNameColumns = []string{"firstname", "first_name", "vorname"}
var lastNameColumns = []string{"lastname", "last_name", "nachname"}
var emailColumns = []string{"email", "e-mail", "mail"}

func normalizeHeader(header []string) []string {
	normalized := make([]string, len(header))
	for i, name := range header {
		normalized[i] = strings.ToLower(strings.TrimSpace(name))
	}
	return normalized
}

func isEmptyRecord(record []string) bool {
	for _, field := range record {
		if strings.TrimSpace(field) != "" {
			return false
		}
	}
	return true
}

func findColumn(index map[string]int, names []string) (int, bool) {
	for _, name := range names {
		if col, ok := index[name]; ok {
			return col, true
		}
	}
	return 0, false
}

func fieldAt(record []string, col int) string {
	if col < 0 || col >= len(record) {
		return ""
	}
	return record[col]
}
