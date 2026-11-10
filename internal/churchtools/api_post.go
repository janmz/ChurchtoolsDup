package churchtools

import "net/http"

// postAPI sends a JSON POST request and returns status code and body.
func (c *Client) postAPI(path string, payload any) (int, []byte, error) {
	return c.callAPI(http.MethodPost, path, nil, payload)
}

// putAPI sends a JSON PUT request and returns status code and body.
func (c *Client) putAPI(path string, payload any) (int, []byte, error) {
	return c.callAPI(http.MethodPut, path, nil, payload)
}
