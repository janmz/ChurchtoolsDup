package churchtools

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const maxOAuthRedirects = 15

// MeAPIToken returns the API login token for the current session.
func (c *Client) MeAPIToken() (string, error) {
	return c.meAPIToken(true)
}

func (c *Client) meAPIToken(allowRelogin bool) (string, error) {
	req, err := http.NewRequest(http.MethodGet, c.apiURL("/person/me/apitoken"), nil)
	if err != nil {
		return "", err
	}
	c.applyAuth(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("api-token abrufen: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusUnauthorized {
		if err := c.reloginOnce(allowRelogin); err != nil {
			return "", err
		}
		return c.meAPIToken(false)
	}
	if resp.StatusCode != http.StatusOK {
		return "", &APIError{
			StatusCode: resp.StatusCode,
			Message:    "API-Token konnte nicht gelesen werden",
			Body:       string(body),
		}
	}

	var envelope struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return "", fmt.Errorf("api-token parsen: %w", err)
	}
	return strings.TrimSpace(envelope.Data), nil
}

func (c *Client) authenticateSubInstanceViaOAuth(centralURL, subURL string) error {
	if err := c.postPasswordLogin(centralURL); err != nil {
		return fmt.Errorf("login zentralinstanz: %w", err)
	}

	startURL, err := discoverOAuthStartLoginURL(c.http, subURL)
	if err != nil {
		return err
	}

	if err := c.followOAuthRedirects(startURL); err != nil {
		return err
	}

	c.baseURL = strings.TrimSuffix(subURL, "/")
	return c.finishPasswordLogin()
}

func discoverOAuthStartLoginURL(httpClient *http.Client, subURL string) (string, error) {
	noRedirect := &http.Client{
		Timeout: httpClient.Timeout,
		Jar:     httpClient.Jar,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	subURL = strings.TrimSuffix(subURL, "/")
	for id := 1; id <= 10; id++ {
		startURL := fmt.Sprintf("%s/oauthclients/%d/startlogin", subURL, id)
		req, err := http.NewRequest(http.MethodGet, startURL, nil)
		if err != nil {
			continue
		}

		resp, err := noRedirect.Do(req)
		if err != nil {
			continue
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		switch resp.StatusCode {
		case http.StatusFound, http.StatusSeeOther, http.StatusTemporaryRedirect:
			if strings.Contains(resp.Header.Get("Location"), "/oauth/authorize") {
				return startURL, nil
			}
		}
	}

	return "", errors.New("oauth-startlogin der nebeninstanz nicht gefunden")
}

func (c *Client) followOAuthRedirects(startURL string) error {
	client := &http.Client{
		Timeout: c.http.Timeout,
		Jar:     c.http.Jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxOAuthRedirects {
				return errors.New("zu viele oauth-redirects")
			}
			return nil
		},
	}

	req, err := http.NewRequest(http.MethodGet, startURL, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("oauth-flow: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}
