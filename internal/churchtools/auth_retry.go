package churchtools

import "errors"

// errAuthRetryExhausted is returned when a 401 persists after one re-login attempt.
var errAuthRetryExhausted = errors.New("session abgelaufen – bitte erneut einloggen")

// reloginOnce retries authentication at most once after HTTP 401.
// allowRelogin is false on the second attempt to prevent unbounded recursion.
func (c *Client) reloginOnce(allowRelogin bool) error {
	if !allowRelogin {
		return errAuthRetryExhausted
	}
	return c.relogin()
}
