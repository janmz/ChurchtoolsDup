package csvfile

import "strings"

var idColumns = []string{"id", "person_id", "personid", "ct_id", "churchtools_id"}
var firstNameColumns = []string{"firstname", "first_name", "vorname"}
var lastNameColumns = []string{"lastname", "last_name", "nachname"}
var emailColumns = []string{"email", "e-mail", "mail"}

func normalizeHeader(header []string) []string {
	normalized := make([]string, len(header))
	for i, name := range header {
		normalized[i] = normalizeColumnName(name)
	}
	return normalized
}

func normalizeColumnName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.TrimPrefix(name, "\ufeff")
	return strings.ToLower(name)
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

// formulaPrefix is prepended to cells that spreadsheet programs would
// interpret as a formula (CSV/formula injection, OWASP).
const formulaPrefix = "'"

// isFormulaStart reports whether a cell would start a formula in Excel or
// LibreOffice Calc.
func isFormulaStart(value string) bool {
	if value == "" {
		return false
	}
	switch value[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return true
	}
	return false
}

// escapeFormula neutralises cells that would be evaluated as a formula.
func escapeFormula(value string) string {
	if isFormulaStart(value) {
		return formulaPrefix + value
	}
	return value
}

// unescapeFormula removes the prefix added by escapeFormula on import.
func unescapeFormula(value string) string {
	if strings.HasPrefix(value, formulaPrefix) && isFormulaStart(value[len(formulaPrefix):]) {
		return value[len(formulaPrefix):]
	}
	return value
}
