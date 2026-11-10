package churchtools

import (
	"encoding/json"
	"errors"
	"fmt"
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
	// mainURL is the explicitly configured main (central) instance for
	// sub-instance logins. Credentials are never sent to a derived host.
	mainURL string
	// confirmPublicSignup must approve every public group signup.
	confirmPublicSignup PublicSignupConfirm
	// joinedGroups lists groups the logged-in user joined in this run.
	joinedGroups []int
}

// NewClient creates a client without authenticating yet. The password is used
// verbatim; leading or trailing spaces are part of it.
func NewClient(baseURL, loginToken, username, password string) *Client {
	normalized := normalizeInstanceURL(baseURL)
	c := &Client{
		baseURL:           normalized,
		configuredBaseURL: normalized,
		loginToken:        strings.TrimSpace(loginToken),
		username:          strings.TrimSpace(username),
		password:          password,
	}
	c.resetHTTPClient()
	return c
}

// SetMainInstanceURL configures the main (central) instance used when login
// on a sub-instance fails. Without it, no fallback host is ever contacted.
func (c *Client) SetMainInstanceURL(mainURL string) {
	c.mainURL = normalizeInstanceURL(mainURL)
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

func (c *Client) hasPassword() bool {
	return c.username != "" && c.password != ""
}

// Login authenticates and loads a CSRF token for legacy calls.
func (c *Client) Login() error {
	c.baseURL = c.configuredBaseURL
	c.loginRedirectNote = ""

	lastErr := c.loginAttempt()
	if lastErr == nil {
		return nil
	}

	// Fallback only to an explicitly configured main instance: a host
	// derived from the instance name could belong to another church.
	mainURL := c.mainURL
	if mainURL == "" || mainURL == c.configuredBaseURL {
		return lastErr
	}

	if c.loginToken == "" && c.hasPassword() {
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
	status, body, err := c.doRequest(http.MethodGet, c.apiURL("/whoami"), nil, true)
	if err != nil {
		return fmt.Errorf("login mit token: %w", err)
	}
	if status != http.StatusOK {
		return newAPIError(status, "Login-Token ungültig oder abgelaufen", body)
	}

	var envelope apiEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("whoami parsen: %w", err)
	}
	if envelope.Data.ID <= 0 {
		return newAPIError(status, "Login-Token ungültig oder abgelaufen", body)
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

	status, body, err := c.doRequest(http.MethodPost, passwordLoginURL(baseURL), payload, false)
	if err != nil {
		return fmt.Errorf("login mit passwort: %w", err)
	}
	if status != http.StatusOK {
		return newAPIError(status, "Benutzername oder Passwort ungültig", body)
	}
	return nil
}

// finishPasswordLogin loads the user without re-login, so a failing session
// cannot recurse into another login.
func (c *Client) finishPasswordLogin() error {
	user, err := c.whoAmI(false)
	if err != nil {
		return err
	}
	c.personID = user.ID
	return nil
}

func (c *Client) resetHTTPClient() {
	jar, _ := cookiejar.New(nil)
	c.http = &http.Client{
		Timeout:       60 * time.Second,
		Jar:           jar,
		CheckRedirect: sameOriginRedirect,
	}
}

// sameOriginRedirect refuses redirects to another host or scheme, so request
// bodies (e.g. the password login) and auth headers never leave the instance.
func sameOriginRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return errors.New("zu viele weiterleitungen")
	}
	origin := via[0].URL
	if req.URL.Scheme != origin.Scheme || !strings.EqualFold(req.URL.Host, origin.Host) {
		return fmt.Errorf("weiterleitung zu %s://%s abgelehnt (nur gleicher Host erlaubt)", req.URL.Scheme, req.URL.Host)
	}
	return nil
}

func passwordLoginURL(baseURL string) string {
	return strings.TrimSuffix(baseURL, "/") + "/api/login"
}

func (c *Client) fetchCSRFToken() (string, error) {
	status, body, err := c.doRequest(http.MethodGet, c.apiURL("/csrftoken"), nil, true)
	if err != nil {
		return "", fmt.Errorf("csrf-token abrufen: %w", err)
	}
	if status != http.StatusOK {
		return "", newAPIError(status, "CSRF-Token konnte nicht geladen werden", body)
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
	status, body, err := c.callAPIRelogin(http.MethodGet, "/whoami", nil, nil, allowRelogin)
	if err != nil {
		return Person{}, fmt.Errorf("whoami: %w", err)
	}
	if status != http.StatusOK {
		return Person{}, newAPIError(status, "", body)
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
	data, err := c.getJSONData(
		fmt.Sprintf("/persons/%d/logintoken", personID),
		"Login-Token konnte nicht gelesen werden (Berechtigung?)",
	)
	if err != nil {
		return "", err
	}
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return "", fmt.Errorf("login-token parsen: %w", err)
	}
	return token, nil
}

// PersonByID loads a single person record.
func (c *Client) PersonByID(id int) (Person, error) {
	data, err := c.getJSONData("/persons/"+strconv.Itoa(id), "")
	if err != nil {
		return Person{}, err
	}
	return decodePerson(data)
}

// GlobalPermissions returns permission metadata for the current user.
func (c *Client) GlobalPermissions() (map[string]any, error) {
	data, err := c.getJSONData("/permissions/global", "")
	if err != nil {
		return nil, fmt.Errorf("berechtigungen laden: %w", err)
	}
	var perms map[string]any
	if err := json.Unmarshal(data, &perms); err != nil {
		return nil, fmt.Errorf("berechtigungen parsen: %w", err)
	}
	return perms, nil
}

// Ping checks whether the base URL points to a ChurchTools instance.
func (c *Client) Ping() error {
	status, body, err := c.doRequest(http.MethodGet, c.apiURL("/whoami"), nil, true)
	if err != nil {
		return fmt.Errorf("verbindung testen: %w", err)
	}
	if status == http.StatusUnauthorized {
		return errors.New("nicht authentifiziert – Login-Token oder Passwort prüfen")
	}
	if status != http.StatusOK {
		return newAPIError(status, "", body)
	}
	return nil
}

// PersonID returns the authenticated user's person ID after login.
func (c *Client) PersonID() int {
	return c.personID
}

// relogin restores an expired session with the configured credentials:
// the login token when present, otherwise username and password (including
// the sub-instance OAuth flow handled by Login).
func (c *Client) relogin() error {
	if c.loginToken != "" {
		return c.authenticateWithToken()
	}
	if c.hasPassword() {
		c.resetHTTPClient()
		if err := c.Login(); err != nil {
			return fmt.Errorf("erneute anmeldung: %w", err)
		}
		return nil
	}
	return errors.New("session abgelaufen – keine Zugangsdaten für erneute Anmeldung")
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

func isRelationPermissionText(text string) bool {
	lower := strings.ToLower(text)
	return strings.Contains(lower, "relation") ||
		strings.Contains(lower, "bezieh") ||
		strings.Contains(lower, "administer") ||
		strings.Contains(lower, "export")
}

func collectRelationStrings(value any, found *[]string) {
	switch v := value.(type) {
	case map[string]any:
		for key, nested := range v {
			if isRelationPermissionText(key) {
				*found = append(*found, key)
			}
			collectRelationStrings(nested, found)
		}
	case []any:
		for _, item := range v {
			collectRelationStrings(item, found)
		}
	case string:
		if isRelationPermissionText(v) {
			*found = append(*found, v)
		}
	}
}
