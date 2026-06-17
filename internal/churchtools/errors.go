package churchtools

import "errors"

// IsForbidden reports whether err is a ChurchTools 403 response.
func IsForbidden(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == 403
	}
	return false
}
