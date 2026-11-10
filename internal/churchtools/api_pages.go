package churchtools

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/janmz/churchtools-dup/internal/iolimit"
)

// maxAPIPages guards against endless pagination; a variable so tests can lower it.
var maxAPIPages = 10_000

// maxAPITotalBytes limits the sum of all pages of one list request, so a
// misbehaving server cannot fill memory page by page (variable for tests).
var maxAPITotalBytes = iolimit.MaxAPITotalBytes

// defaultPageSize is the page size for paginated list endpoints.
const defaultPageSize = 100

// fetchAPIPages loads all pages from a paginated ChurchTools list endpoint.
func (c *Client) fetchAPIPages(path string, query url.Values) ([]json.RawMessage, error) {
	if query == nil {
		query = url.Values{}
	}
	if query.Get("limit") == "" {
		query.Set("limit", strconv.Itoa(defaultPageSize))
	}

	page := 1
	var all []json.RawMessage
	var total int64

	for {
		if page > maxAPIPages {
			return nil, fmt.Errorf("paginierung abgebrochen: mehr als %d seiten", maxAPIPages)
		}
		query.Set("page", strconv.Itoa(page))
		body, err := c.getAPI(path, query)
		if err != nil {
			return nil, err
		}
		total += int64(len(body))
		if total > maxAPITotalBytes {
			return nil, fmt.Errorf("paginierung abgebrochen: %w (mehr als %d Bytes für %s)", iolimit.ErrTooLarge, maxAPITotalBytes, path)
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

// getAPI performs a GET request and returns the body of a 200 response.
func (c *Client) getAPI(path string, query url.Values) ([]byte, error) {
	status, body, err := c.callAPI(http.MethodGet, path, query, nil)
	if err != nil {
		return nil, fmt.Errorf("api GET %s: %w", path, err)
	}
	if status != http.StatusOK {
		return nil, newAPIError(status, "daten konnten nicht geladen werden", body)
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
