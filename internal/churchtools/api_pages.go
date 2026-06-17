package churchtools

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

const maxAPIPages = 10_000

// fetchAPIPages loads all pages from a paginated ChurchTools list endpoint.
func (c *Client) fetchAPIPages(path string, query url.Values) ([]json.RawMessage, error) {
	if query == nil {
		query = url.Values{}
	}
	if query.Get("limit") == "" {
		query.Set("limit", strconv.Itoa(defaultPersonPageSize))
	}

	page := 1
	var all []json.RawMessage

	for {
		if page > maxAPIPages {
			return nil, fmt.Errorf("paginierung abgebrochen: mehr als %d seiten", maxAPIPages)
		}
		query.Set("page", strconv.Itoa(page))
		body, err := c.getAPI(path, query)
		if err != nil {
			return nil, err
		}

		var envelope struct {
			Data json.RawMessage `json:"data"`
			Meta struct {
				Pagination *struct {
					Current  int `json:"current"`
					LastPage int `json:"lastPage"`
				} `json:"pagination"`
			} `json:"meta"`
		}
		if err := json.Unmarshal(body, &envelope); err != nil {
			return nil, fmt.Errorf("antwort parsen: %w", err)
		}

		items, err := rawItems(envelope.Data)
		if err != nil {
			return nil, err
		}
		all = append(all, items...)

		if envelope.Meta.Pagination == nil || page >= envelope.Meta.Pagination.LastPage {
			break
		}
		page++
	}

	return all, nil
}

func (c *Client) fetchAPIList(path string, query url.Values) ([]json.RawMessage, error) {
	body, err := c.getAPI(path, query)
	if err != nil {
		return nil, err
	}

	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("antwort parsen: %w", err)
	}

	return rawItems(envelope.Data)
}

func (c *Client) getAPI(path string, query url.Values) ([]byte, error) {
	return c.getAPIRetry(path, query, true)
}

func (c *Client) getAPIRetry(path string, query url.Values, allowRelogin bool) ([]byte, error) {
	reqURL := c.apiURL(path)
	if len(query) > 0 {
		reqURL += "?" + query.Encode()
	}

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	c.applyAuth(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("api GET %s: %w", path, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusUnauthorized {
		if err := c.reloginOnce(allowRelogin); err != nil {
			return nil, err
		}
		return c.getAPIRetry(path, query, false)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    "daten konnten nicht geladen werden",
			Body:       string(body),
		}
	}
	return body, nil
}

func rawItems(data json.RawMessage) ([]json.RawMessage, error) {
	if len(data) == 0 || string(data) == "null" {
		return nil, nil
	}

	var items []json.RawMessage
	if err := json.Unmarshal(data, &items); err == nil {
		return items, nil
	}

	return []json.RawMessage{data}, nil
}
