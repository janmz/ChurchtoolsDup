package churchtools

import (
	"encoding/json"
	"strings"
)

var flexiblePersonFieldKeys = []string{
	"isSystemUser",
	"is_system_user",
	"isAllowedToLogin",
	"is_allowed_to_login",
	"cmsUserId",
	"cms_user_id",
	"acceptedsecurity",
	"acceptedSecurity",
	"accepted_security",
	"lastLogin",
	"last_login",
	"privacyPolicyAgreement",
	"privacy_policy_agreement",
}

func decodePerson(data json.RawMessage) (Person, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return Person{}, err
	}

	clean := make(map[string]json.RawMessage, len(fields))
	for key, value := range fields {
		clean[key] = value
	}
	for _, key := range flexiblePersonFieldKeys {
		delete(clean, key)
	}
	for _, key := range flexibleAddressFieldKeys {
		delete(clean, key)
	}

	cleanData, err := json.Marshal(clean)
	if err != nil {
		return Person{}, err
	}

	var person Person
	if err := json.Unmarshal(cleanData, &person); err != nil {
		return Person{}, err
	}
	enrichPersonAccount(&person, data)
	enrichPersonFields(&person, data)
	return person, nil
}

func enrichPersonAccount(person *Person, data json.RawMessage) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return
	}

	if person.IsSystemUser == nil {
		for _, key := range []string{"isSystemUser", "is_system_user"} {
			if value := parseFlexibleBool(fields[key]); value != nil {
				person.IsSystemUser = value
				break
			}
		}
	}

	if strings.TrimSpace(person.CMSUserID) == "" {
		for _, key := range []string{"cmsUserId", "cms_user_id"} {
			if value := parseFlexibleString(fields[key]); value != nil {
				person.CMSUserID = *value
				break
			}
		}
	}

	if !hasStringValue(person.AcceptedSecurity) {
		for _, key := range []string{"acceptedsecurity", "acceptedSecurity", "accepted_security"} {
			if value := parseAcceptedSecurity(fields[key]); value != nil {
				person.AcceptedSecurity = value
				break
			}
		}
	}

	if person.PrivacyPolicyAgreement == nil || !hasStringValue(person.PrivacyPolicyAgreement.Date) {
		for _, key := range []string{"privacyPolicyAgreement", "privacy_policy_agreement"} {
			if raw, ok := fields[key]; ok {
				var agreement PrivacyPolicyAgreement
				if err := json.Unmarshal(raw, &agreement); err == nil && hasStringValue(agreement.Date) {
					person.PrivacyPolicyAgreement = &agreement
					break
				}
			}
		}
	}

	if !hasStringValue(person.LastLogin) {
		for _, key := range []string{"lastLogin", "last_login"} {
			if value := parseFlexibleString(fields[key]); value != nil {
				person.LastLogin = value
				break
			}
		}
	}

	if person.IsAllowedToLogin == nil {
		for _, key := range []string{"isAllowedToLogin", "is_allowed_to_login"} {
			if value := parseFlexibleBool(fields[key]); value != nil {
				person.IsAllowedToLogin = value
				break
			}
		}
	}

	if strings.TrimSpace(person.InvitationStatus) == "" {
		for _, key := range []string{"invitationStatus", "invitation_status"} {
			if value := parseFlexibleString(fields[key]); value != nil {
				person.InvitationStatus = *value
				break
			}
		}
	}
}

func parseFlexibleBool(raw json.RawMessage) *bool {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}

	var value bool
	if err := json.Unmarshal(raw, &value); err == nil {
		return &value
	}

	var number int
	if err := json.Unmarshal(raw, &number); err == nil {
		result := number != 0
		return &result
	}

	text := strings.Trim(strings.TrimSpace(string(raw)), `"`)
	if text == "" {
		return nil
	}
	if text == "1" || strings.EqualFold(text, "true") {
		result := true
		return &result
	}
	if text == "0" || strings.EqualFold(text, "false") {
		result := false
		return &result
	}
	return nil
}

func parseFlexibleString(raw json.RawMessage) *string {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}

	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil
		}
		return &value
	}
	return nil
}

func parseAcceptedSecurity(raw json.RawMessage) *string {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}

	if value := parseFlexibleString(raw); value != nil {
		return value
	}

	var object struct {
		Date *string `json:"date"`
	}
	if err := json.Unmarshal(raw, &object); err == nil && hasStringValue(object.Date) {
		return object.Date
	}
	return nil
}
