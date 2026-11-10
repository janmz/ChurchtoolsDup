package churchtools

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

var flexibleAddressFieldKeys = []string{
	"street",
	"strasse",
	"straße",
	"address",
	"addresses",
	"homeAddress",
	"home_address",
	"city",
	"stadt",
	"createdAt",
	"created_at",
	"creationDate",
	"creation_date",
	"cdate",
	"meta",
	"campus",
	"campusId",
	"campus_id",
}

func enrichPersonFields(person *Person, data json.RawMessage) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return
	}

	enrichAddressFields(person, fields)
	enrichCreatedAt(person, fields)
	enrichCampusFields(person, fields)
}

func enrichAddressFields(person *Person, fields map[string]json.RawMessage) {
	if strings.TrimSpace(person.Street) == "" {
		for _, key := range []string{"street", "strasse", "straße", "address"} {
			if value := parseFlexibleString(fields[key]); value != nil {
				person.Street = *value
				break
			}
		}
	}

	if strings.TrimSpace(person.City) == "" {
		for _, key := range []string{"city", "stadt"} {
			if value := parseFlexibleString(fields[key]); value != nil {
				person.City = *value
				break
			}
		}
	}

	if strings.TrimSpace(person.Street) != "" && strings.TrimSpace(person.City) != "" {
		return
	}

	for _, key := range []string{"homeAddress", "home_address", "addresses", "address"} {
		raw, ok := fields[key]
		if !ok {
			continue
		}
		street, city := parseAddressValue(raw)
		if strings.TrimSpace(person.Street) == "" && street != "" {
			person.Street = street
		}
		if strings.TrimSpace(person.City) == "" && city != "" {
			person.City = city
		}
	}
}

func parseAddressValue(raw json.RawMessage) (street, city string) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", ""
	}

	if raw[0] == '[' {
		var items []map[string]any
		if err := json.Unmarshal(raw, &items); err != nil {
			return "", ""
		}
		for _, item := range items {
			street, city = addressFromMap(item)
			if street != "" || city != "" {
				return street, city
			}
		}
		return "", ""
	}

	var item map[string]any
	if err := json.Unmarshal(raw, &item); err != nil {
		return "", ""
	}
	return addressFromMap(item)
}

func addressFromMap(item map[string]any) (street, city string) {
	for _, key := range []string{"street", "strasse", "straße", "address"} {
		if value, ok := item[key].(string); ok && strings.TrimSpace(value) != "" {
			street = strings.TrimSpace(value)
			break
		}
	}
	for _, key := range []string{"city", "stadt", "place"} {
		if value, ok := item[key].(string); ok && strings.TrimSpace(value) != "" {
			city = strings.TrimSpace(value)
			break
		}
	}
	return street, city
}

func enrichCreatedAt(person *Person, fields map[string]json.RawMessage) {
	if strings.TrimSpace(person.CreatedAt) != "" {
		person.CreatedAt = FormatExportDate(person.CreatedAt)
		return
	}

	for _, key := range []string{"createdAt", "created_at", "createdDate", "creationDate", "creation_date", "cdate"} {
		if value := parseFlexibleString(fields[key]); value != nil {
			person.CreatedAt = FormatExportDate(*value)
			return
		}
	}

	if raw, ok := fields["meta"]; ok {
		if value := parseMetaDateString(raw); value != "" {
			person.CreatedAt = FormatExportDate(value)
		}
	}
}

func parseMetaDateString(raw json.RawMessage) string {
	var meta map[string]any
	if err := json.Unmarshal(raw, &meta); err != nil {
		return ""
	}
	for _, key := range []string{"createdDate", "createdAt", "created_at", "dateCreated", "creationDate"} {
		if value, ok := meta[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// FormatExportDate formats API timestamps for CSV export (DD.MM.YYYY).
// Timestamps with a time zone (ChurchTools sends UTC, e.g. "…T23:30:00Z")
// are converted to local time first, so a person created shortly after
// midnight local time gets the correct day. Values without zone are taken as
// local time; pure dates stay unchanged.
func FormatExportDate(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	const exportLayout = "02.01.2006"
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t.In(time.Local).Format(exportLayout)
	}
	for _, layout := range []string{"2006-01-02T15:04:05", "2006-01-02 15:04:05", "2006-01-02"} {
		if t, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return t.Format(exportLayout)
		}
	}

	if len(value) >= 10 && value[4] == '-' && value[7] == '-' {
		if t, err := time.Parse("2006-01-02", value[:10]); err == nil {
			return t.Format(exportLayout)
		}
	}

	return value
}

func enrichCampusFields(person *Person, fields map[string]json.RawMessage) {
	if person.CampusID <= 0 {
		for _, key := range []string{"campusId", "campus_id"} {
			if id, ok := intFromFlexibleField(fields[key]); ok {
				person.CampusID = id
				break
			}
		}
	}

	if strings.TrimSpace(person.CampusName) != "" {
		return
	}

	if raw, ok := fields["campus"]; ok {
		var campus map[string]any
		if err := json.Unmarshal(raw, &campus); err == nil {
			if person.CampusID <= 0 {
				if id, ok := intFromAny(campus["id"]); ok {
					person.CampusID = id
				}
			}
			if name, ok := campus["name"].(string); ok {
				person.CampusName = strings.TrimSpace(name)
			}
		}
	}
}

func intFromFlexibleField(raw json.RawMessage) (int, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return 0, false
	}
	var number int
	if err := json.Unmarshal(raw, &number); err == nil {
		return number, number > 0
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		number, err := strconv.Atoi(strings.TrimSpace(text))
		return number, err == nil && number > 0
	}
	return 0, false
}

// PrimaryEmail returns the default e-mail address for duplicate matching.
func (p Person) PrimaryEmail() string {
	email := strings.TrimSpace(p.Email)
	if email != "" {
		return email
	}
	for _, item := range p.Emails {
		if strings.TrimSpace(item.Email) != "" {
			return strings.TrimSpace(item.Email)
		}
	}
	return ""
}
