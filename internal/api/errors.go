package api

import (
	"fmt"
	"net/http"
)

var (
	// ErrNotFound indicates the requested resource does not exist (HTTP 404).
	ErrNotFound = &APIError{StatusCode: http.StatusNotFound}
	// ErrPaymentRequired indicates the account lacks a required ability (HTTP 402).
	ErrPaymentRequired = &APIError{StatusCode: http.StatusPaymentRequired}
	// ErrRateLimited indicates the API rate limit was exceeded (HTTP 429).
	ErrRateLimited = &APIError{StatusCode: http.StatusTooManyRequests}
)

// APIError represents an error returned by the PagerDuty API.
type APIError struct {
	StatusCode int
	Code       int    `json:"code"`
	Message    string `json:"message"`
}

// Error returns the status code and message as a formatted string.
func (e *APIError) Error() string {
	return fmt.Sprintf("pagerduty API error (status %d, code %d): %s", e.StatusCode, e.Code, e.Message)
}

// Is matches by status code. errors.Is handles chain-walking on the
// receiver; this method only inspects the target directly.
func (e *APIError) Is(target error) bool {
	t, ok := target.(*APIError)
	if !ok {
		return false
	}
	return e.StatusCode == t.StatusCode
}
