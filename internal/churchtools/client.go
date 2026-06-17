package churchtools

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strconv"
	"strings"
	"time"
)

// APIError describes a failed ChurchTools response.
type APIError struct {
	StatusCode int
	Message    string
	Body       string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("churchtools API (%d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("churchtools API (%d): %s", e.StatusCode, e.Body)
}

// Client talks to ChurchTools REST and legacy AJAX APIs.
type Client struct {
	baseURL           string
	configuredBaseURL string
	loginToken        string
	username          string
	password          string
	http              *http.Client
	csrfToken         string
	personID          int
	loginRedirectNote string
}

// NewClient creates a client without authenticating yet.
func NewClient(baseURL, loginToken, username, password string) *Client {
	jar, _ := cookiejar.New(nil)
	normalized := normalizeInstanceURL(baseURL)
	return &Client{
		baseURL:           normalized,
		configuredBaseURL: normalized,
		loginToken:        strings.TrimSpace(loginToken),
		username:          strings.TrimSpace(username),
		password:          strings.TrimSpace(password),
		http: &http.Client{
			Timeout: 60 * time.Second,
			Jar:     jar,
		},
	}
}

// BaseURL returns the ChurchTools instance URL used for API calls.
func (c *Client) BaseURL() string {
	return c.baseURL
}

// LoginRedirectNote is non-empty when password login succeeded on the main
// instance instead of the configured sub-instance URL.
func (c *Client) LoginRedirectNote() string {
	return c.loginRedirectNote
}

// Login authenticates and loads a CSRF token for legacy calls.
func (c *Client) Login() error {
	c.baseURL = c.configuredBaseURL
	c.loginRedirectNote = ""

	if err := c.loginAttempt(); err == nil {
		return nil
	} else {
		lastErr := err
		mainURL, ok := mainInstanceURLForLogin(c.configuredBaseURL)
		if !ok || mainURL == c.configuredBaseURL {
			return lastErr
		}

		if c.loginToken == "" && strings.TrimSpace(c.password) != "" {
			c.resetHTTPClient()
			c.loginRedirectNote = SubInstanceOAuthLoginNote(c.configuredBaseURL, mainURL)
			if err := c.authenticateSubInstanceViaOAuth(mainURL, c.configuredBaseURL); err != nil {
				return fmt.Errorf("anmeldung nebeninstanz fehlgeschlagen: %w", err)
			}
			token, err := c.fetchCSRFToken()
			if err != nil {
				return err
			}
			c.csrfToken = token
			return nil
		}

		if c.loginToken != "" {
			c.resetHTTPClient()
			c.baseURL = mainURL
			c.loginRedirectNote = MainInstanceLoginNote(c.configuredBaseURL, mainURL)
			if err := c.loginAttempt(); err != nil {
				return lastErr
			}
			return nil
		}

		return lastErr
	}
}

func (c *Client) loginAttempt() error {
	if c.loginToken != "" {
		if err := c.authenticateWithToken(); err != nil {
			return err
		}
	} else if err := c.authenticateWithPassword(); err != nil {
		return err
	}

	token, err := c.fetchCSRFToken()
	if err != nil {
		return err
	}
	c.csrfToken = token
	return nil
}

func (c *Client) authenticateWithToken() error {
	req, err := http.NewRequest(http.MethodGet, c.apiURL("/whoami"), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Login "+c.loginToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("login mit token: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    "Login-Token ungültig oder abgelaufen",
			Body:       string(body),
		}
	}

	var envelope apiEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("whoami parsen: %w", err)
	}
	if envelope.Data.ID <= 0 {
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    "Login-Token ungültig oder abgelaufen",
			Body:       string(body),
		}
	}
	c.personID = envelope.Data.ID
	return nil
}

func (c *Client) authenticateWithPassword() error {
	if err := c.postPasswordLogin(c.baseURL); err != nil {
		return err
	}
	return c.finishPasswordLogin()
}

func (c *Client) postPasswordLogin(baseURL string) error {
	payload, err := json.Marshal(map[string]string{
		"username": c.username,
		"password": c.password,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, passwordLoginURL(baseURL), bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("login mit passwort: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    "Benutzername oder Passwort ungültig",
			Body:       string(body),
		}
	}
	return nil
}

func (c *Client) finishPasswordLogin() error {
	user, err := c.WhoAmI()
	if err != nil {
		return err
	}
	c.personID = user.ID
	return nil
}

func (c *Client) resetHTTPClient() {
	jar, _ := cookiejar.New(nil)
	c.http = &http.Client{
		Timeout: 60 * time.Second,
		Jar:     jar,
	}
}

func passwordLoginURL(baseURL string) string {
	return strings.TrimSuffix(baseURL, "/") + "/api/login"
}

func (c *Client) fetchCSRFToken() (string, error) {
	req, err := http.NewRequest(http.MethodGet, c.apiURL("/csrftoken"), nil)
	if err != nil {
		return "", err
	}
	c.applyAuth(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("csrf-token abrufen: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", &APIError{
			StatusCode: resp.StatusCode,
			Message:    "CSRF-Token konnte nicht geladen werden",
			Body:       string(body),
		}
	}

	var envelope struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return "", fmt.Errorf("csrf-token parsen: %w", err)
	}
	if envelope.Data == "" {
		return "", errors.New("leerer CSRF-Token")
	}
	return envelope.Data, nil
}

// WhoAmI returns the authenticated user.
func (c *Client) WhoAmI() (Person, error) {
	return c.whoAmI(true)
}

func (c *Client) whoAmI(allowRelogin bool) (Person, error) {
	req, err := http.NewRequest(http.MethodGet, c.apiURL("/whoami"), nil)
	if err != nil {
		return Person{}, err
	}
	c.applyAuth(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return Person{}, fmt.Errorf("whoami: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusUnauthorized {
		if err := c.reloginOnce(allowRelogin); err != nil {
			return Person{}, err
		}
		return c.whoAmI(false)
	}
	if resp.StatusCode != http.StatusOK {
		return Person{}, &APIError{StatusCode: resp.StatusCode, Body: string(body)}
	}

	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return Person{}, fmt.Errorf("whoami parsen: %w", err)
	}
	person, err := decodePerson(envelope.Data)
	if err != nil {
		return Person{}, fmt.Errorf("whoami parsen: %w", err)
	}
	c.personID = person.ID
	return person, nil
}

// LoginToken returns the API login token for a person (requires permission).
func (c *Client) LoginToken(personID int) (string, error) {
	return c.fetchLoginToken(personID, true)
}

func (c *Client) fetchLoginToken(personID int, allowRelogin bool) (string, error) {
	path := fmt.Sprintf("/persons/%d/logintoken", personID)
	req, err := http.NewRequest(http.MethodGet, c.apiURL(path), nil)
	if err != nil {
		return "", err
	}
	c.applyAuth(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("login-token abrufen: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusUnauthorized {
		if err := c.reloginOnce(allowRelogin); err != nil {
			return "", err
		}
		return c.fetchLoginToken(personID, false)
	}
	if resp.StatusCode != http.StatusOK {
		return "", &APIError{
			StatusCode: resp.StatusCode,
			Message:    "Login-Token konnte nicht gelesen werden (Berechtigung?)",
			Body:       string(body),
		}
	}

	var envelope struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return "", fmt.Errorf("login-token parsen: %w", err)
	}
	return envelope.Data, nil
}

// PersonByID loads a single person record.
func (c *Client) PersonByID(id int) (Person, error) {
	return c.personByID(id, true)
}

func (c *Client) personByID(id int, allowRelogin bool) (Person, error) {
	req, err := http.NewRequest(http.MethodGet, c.apiURL("/persons/"+strconv.Itoa(id)), nil)
	if err != nil {
		return Person{}, err
	}
	c.applyAuth(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return Person{}, fmt.Errorf("person laden: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusUnauthorized {
		if err := c.reloginOnce(allowRelogin); err != nil {
			return Person{}, err
		}
		return c.personByID(id, false)
	}
	if resp.StatusCode != http.StatusOK {
		return Person{}, &APIError{StatusCode: resp.StatusCode, Body: string(body)}
	}

	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return Person{}, fmt.Errorf("person parsen: %w", err)
	}
	return decodePerson(envelope.Data)
}

// GlobalPermissions returns permission metadata for the current user.
func (c *Client) GlobalPermissions() (map[string]any, error) {
	return c.globalPermissions(true)
}

func (c *Client) globalPermissions(allowRelogin bool) (map[string]any, error) {
	req, err := http.NewRequest(http.MethodGet, c.apiURL("/permissions/global"), nil)
	if err != nil {
		return nil, err
	}
	c.applyAuth(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("berechtigungen laden: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusUnauthorized {
		if err := c.reloginOnce(allowRelogin); err != nil {
			return nil, err
		}
		return c.globalPermissions(false)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &APIError{StatusCode: resp.StatusCode, Body: string(body)}
	}

	var envelope struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("berechtigungen parsen: %w", err)
	}
	return envelope.Data, nil
}

// Ping checks whether the base URL points to a ChurchTools instance.
func (c *Client) Ping() error {
	req, err := http.NewRequest(http.MethodGet, c.apiURL("/whoami"), nil)
	if err != nil {
		return err
	}
	c.applyAuth(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("verbindung testen: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return errors.New("nicht authentifiziert – Login-Token oder Passwort prüfen")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return &APIError{StatusCode: resp.StatusCode, Body: string(body)}
	}
	return nil
}

// PersonID returns the authenticated user's person ID after login.
func (c *Client) PersonID() int {
	return c.personID
}

func (c *Client) relogin() error {
	if c.loginToken == "" {
		return errors.New("session abgelaufen – bitte erneut mit login_token einloggen")
	}
	return c.authenticateWithToken()
}

func (c *Client) applyAuth(req *http.Request) {
	if c.loginToken != "" {
		req.Header.Set("Authorization", "Login "+c.loginToken)
	}
}

func (c *Client) apiURL(path string) string {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return c.baseURL + "/api" + path
}

type apiEnvelope struct {
	Data Person `json:"data"`
}

// Person is a ChurchTools person record used for duplicate export and import.
type Person struct {
	ID                     int                     `json:"id"`
	CampusID               int                     `json:"campusId"`
	CampusName             string                  `json:"-"`
	FirstName              string                  `json:"firstName"`
	LastName               string                  `json:"lastName"`
	Email                  string                  `json:"email"`
	Emails                 []PersonEmail           `json:"emails"`
	CMSUserID              string                  `json:"cmsUserId"`
	InvitationStatus       string                  `json:"invitationStatus"`
	IsSystemUser           *bool                   `json:"isSystemUser"`
	IsAllowedToLogin       *bool                   `json:"isAllowedToLogin,omitempty"`
	AcceptedSecurity       *string                 `json:"acceptedsecurity"`
	LastLogin              *string                 `json:"lastLogin,omitempty"`
	PrivacyPolicyAgreement *PrivacyPolicyAgreement `json:"privacyPolicyAgreement"`
	Street                 string                  `json:"street"`
	City                   string                  `json:"city"`
	CreatedAt              string                  `json:"createdAt"`
}

// PersonEmail is an additional e-mail address on a person record.
type PersonEmail struct {
	Email          string `json:"email"`
	IsDefault      bool   `json:"isDefault"`
	ContactLabelID int    `json:"contactLabelId"`
}

// CurrentUserCampusID returns the campus of the authenticated user.
func (c *Client) CurrentUserCampusID() (int, error) {
	user, err := c.WhoAmI()
	if err != nil {
		return 0, err
	}
	return user.CampusID, nil
}

// PermissionHints documents required rights for duplicate export/import.
var PermissionHints = []string{
	"Personen lesen/exportieren: GET /persons (export data)",
	"Beziehungen bearbeiten: POST /persons/{id}/relationships (edit relations)",
	"Personen verwalten: Dubletten verknüpfen und zusammenführen (administer persons)",
	"Gruppe Duplikate: Mitgliedschaft für markierte Originale",
	"Login-Token lesen: GET /persons/{id}/logintoken (für Setup token)",
}

// FindRelationPermissions searches permission payloads for relation/person entries.
func FindRelationPermissions(perms map[string]any) []string {
	var found []string
	collectRelationStrings(perms, &found)
	return found
}

func collectRelationStrings(value any, found *[]string) {
	switch v := value.(type) {
	case map[string]any:
		for key, nested := range v {
			lower := strings.ToLower(key)
			if strings.Contains(lower, "relation") ||
				strings.Contains(lower, "bezieh") ||
				strings.Contains(lower, "administer") ||
				strings.Contains(lower, "export") {
				*found = append(*found, key)
			}
			collectRelationStrings(nested, found)
		}
	case []any:
		for _, item := range v {
			collectRelationStrings(item, found)
		}
	case string:
		lower := strings.ToLower(v)
		if strings.Contains(lower, "relation") ||
			strings.Contains(lower, "bezieh") ||
			strings.Contains(lower, "administer") ||
			strings.Contains(lower, "export") {
			*found = append(*found, v)
		}
	}
}
