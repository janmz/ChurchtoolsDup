package churchtools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func (c *Client) postAPI(path string, payload any, allowRelogin bool) (int, []byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, nil, err
	}

	req, err := http.NewRequest(http.MethodPost, c.apiURL(path), bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.applyAuth(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("api POST %s: %w", path, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusUnauthorized {
		if err := c.reloginOnce(allowRelogin); err != nil {
			return resp.StatusCode, respBody, err
		}
		return c.postAPI(path, payload, false)
	}
	return resp.StatusCode, respBody, nil
}
