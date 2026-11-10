package churchtools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/janmz/churchtools-dup/internal/iolimit"
)

// Retry policy for HTTP 429 (Too Many Requests).
const (
	defaultRetryAfter = 2 * time.Second
	maxRetryAfter     = 30 * time.Second
)

// sleep is replaceable in tests to avoid real waiting.
var sleep = time.Sleep

// newAPIError builds an APIError with a shortened, printable response body.
func newAPIError(status int, message string, body []byte) *APIError {
	return &APIError{
		StatusCode: status,
		Message:    message,
		Body:       cleanMessage(string(body)),
	}
}

// cleanMessage makes server text safe for terminal output: control
// characters (incl. ANSI escape sequences) become spaces, whitespace is
// collapsed and the text is cut to iolimit.MaxErrorBodyBytes.
func cleanMessage(text string) string {
	text = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, text)
	text = strings.Join(strings.Fields(text), " ")
	return iolimit.Truncate(text, iolimit.MaxErrorBodyBytes)
}

// doRequest sends one HTTP request with authentication, retries once after
// HTTP 429 and returns status code and size-limited body. It does not handle
// HTTP 401; see callAPI.
func (c *Client) doRequest(method, rawURL string, payload []byte, auth bool) (int, []byte, error) {
	for attempt := 0; ; attempt++ {
		var body *bytes.Reader
		if payload != nil {
			body = bytes.NewReader(payload)
		}
		var req *http.Request
		var err error
		if body != nil {
			req, err = http.NewRequest(method, rawURL, body)
		} else {
			req, err = http.NewRequest(method, rawURL, nil)
		}
		if err != nil {
			return 0, nil, err
		}
		if payload != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if auth {
			c.applyAuth(req)
			c.applyCSRF(req)
		}

		resp, err := c.http.Do(req)
		if err != nil {
			return 0, nil, fmt.Errorf("%s %s: %w", method, rawURL, err)
		}
		respBody, readErr := readResponseBody(resp)
		resp.Body.Close()
		if readErr != nil {
			return resp.StatusCode, nil, fmt.Errorf("%s %s: antwort lesen: %w", method, rawURL, readErr)
		}

		if resp.StatusCode == http.StatusTooManyRequests && attempt == 0 {
			sleep(retryAfter(resp.Header.Get("Retry-After")))
			continue
		}
		return resp.StatusCode, respBody, nil
	}
}

// csrfHeader is the header ChurchTools expects for the CSRF token (REST write
// requests and the legacy AJAX API), as sent by the official JS client.
const csrfHeader = "CSRF-Token"

// applyCSRF adds the session's CSRF token to write requests. GET/HEAD need
// none, and the login request is sent before a token exists.
func (c *Client) applyCSRF(req *http.Request) {
	if c.csrfToken == "" || req.Method == http.MethodGet || req.Method == http.MethodHead {
		return
	}
	if strings.HasSuffix(req.URL.Path, "/api/login") {
		return
	}
	req.Header.Set(csrfHeader, c.csrfToken)
}

// readResponseBody reads an HTTP response body with the API size limit.
func readResponseBody(resp *http.Response) ([]byte, error) {
	return iolimit.ReadAll(resp.Body, iolimit.MaxAPIResponseBytes)
}

// retryAfter interprets a Retry-After header given in seconds.
func retryAfter(value string) time.Duration {
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return defaultRetryAfter
	}
	wait := time.Duration(seconds) * time.Second
	if wait > maxRetryAfter {
		return maxRetryAfter
	}
	return wait
}

// callAPI performs a REST call below /api. On HTTP 401 it logs in again once
// and repeats the request; a second 401 returns errAuthRetryExhausted.
func (c *Client) callAPI(method, path string, query url.Values, payload any) (int, []byte, error) {
	return c.callAPIRelogin(method, path, query, payload, true)
}

func (c *Client) callAPIRelogin(method, path string, query url.Values, payload any, allowRelogin bool) (int, []byte, error) {
	var data []byte
	if payload != nil {
		var err error
		data, err = json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
	}

	reqURL := c.apiURL(path)
	if len(query) > 0 {
		reqURL += "?" + query.Encode()
	}

	status, body, err := c.doRequest(method, reqURL, data, true)
	if err != nil {
		return status, body, err
	}
	if status == http.StatusUnauthorized {
		if !allowRelogin {
			return status, body, errAuthRetryExhausted
		}
		if err := c.relogin(); err != nil {
			return status, body, err
		}
		return c.callAPIRelogin(method, path, query, payload, false)
	}
	return status, body, nil
}

// getJSONData performs a GET request and returns the raw "data" member of the
// ChurchTools response envelope. Non-200 responses become an APIError.
func (c *Client) getJSONData(path, errMessage string) (json.RawMessage, error) {
	status, body, err := c.callAPI(http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, newAPIError(status, errMessage, body)
	}
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("antwort von %s parsen: %w", path, err)
	}
	return envelope.Data, nil
}
