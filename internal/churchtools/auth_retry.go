package churchtools

import "errors"

// errAuthRetryExhausted is returned when a 401 persists after one re-login attempt.
var errAuthRetryExhausted = errors.New("session abgelaufen – bitte erneut einloggen")
