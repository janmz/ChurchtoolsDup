package churchtools

import (
	"encoding/json"
	"strings"
)

// PrivacyPolicyAgreement holds consent metadata for a person record.
type PrivacyPolicyAgreement struct {
	Date *string `json:"date"`
}

// UnmarshalJSON accepts object or array payloads from ChurchTools.
func (p *PrivacyPolicyAgreement) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}

	type plain PrivacyPolicyAgreement

	if data[0] == '[' {
		var items []plain
		if err := json.Unmarshal(data, &items); err != nil {
			return err
		}
		for _, item := range items {
			agreement := PrivacyPolicyAgreement(item)
			if hasStringValue(agreement.Date) {
				*p = agreement
				return nil
			}
		}
		return nil
	}

	var item plain
	if err := json.Unmarshal(data, &item); err != nil {
		return err
	}
	*p = PrivacyPolicyAgreement(item)
	return nil
}

func hasStringValue(value *string) bool {
	return value != nil && strings.TrimSpace(*value) != ""
}
