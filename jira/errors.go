package jira

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	devhttp "github.com/randalmurphal/devflow/http"
)

// Configuration errors.
var (
	ErrConfigURLRequired       = errors.New("jira url is required")
	ErrConfigAuthTypeRequired  = errors.New("jira auth type is required")
	ErrConfigAuthTypeInvalid   = errors.New("jira auth type must be api_token, oauth2, basic, or pat")
	ErrConfigAPITokenAuth      = errors.New("api_token auth requires email and token")
	ErrConfigBasicAuth         = errors.New("basic auth requires username and password")
	ErrConfigPATAuth           = errors.New("pat auth requires token")
	ErrConfigOAuth2Auth        = errors.New("oauth2 auth requires client_id and client_secret")
	ErrConfigAPIVersionInvalid = errors.New("api_version must be auto, v2, or v3")
)

// Issue errors.
var (
	ErrIssueNotFound    = errors.New("jira issue not found")
	ErrProjectNotFound  = errors.New("jira project not found")
	ErrIssueKeyRequired = errors.New("issue key is required")
	ErrIssueKeyInvalid  = errors.New("invalid issue key format")
)

// Transition errors.
var (
	ErrTransitionNotFound   = errors.New("transition not found for issue")
	ErrTransitionNotAllowed = errors.New("transition not allowed from current status")
	ErrTransitionIDRequired = errors.New("transition id is required")
)

// Comment errors.
var (
	ErrCommentNotFound   = errors.New("comment not found")
	ErrCommentIDRequired = errors.New("comment id is required")
)

// Webhook errors.
var (
	ErrWebhookInvalidSignature = errors.New("invalid webhook signature")
	ErrWebhookInvalidPayload   = errors.New("invalid webhook payload")
	ErrWebhookEventUnknown     = errors.New("unknown webhook event type")
)

// ADF errors.
var (
	ErrADFInvalid     = errors.New("invalid ADF document")
	ErrADFVersionOnly = errors.New("ADF version must be 1")
	ErrADFTypeInvalid = errors.New("ADF root type must be 'doc'")
)

// APIError represents an error response from the Jira API.
// It wraps devhttp.APIError for consistent error handling.
type APIError struct {
	StatusCode    int               `json:"-"`
	ErrorMessages []string          `json:"errorMessages,omitempty"`
	Errors        map[string]string `json:"errors,omitempty"`
	Endpoint      string            `json:"-"`
	RequestID     string            `json:"-"`
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if len(e.ErrorMessages) > 0 {
		return fmt.Sprintf("jira api error (%d): %s", e.StatusCode, e.ErrorMessages[0])
	}
	if len(e.Errors) > 0 {
		for field, msg := range e.Errors {
			return fmt.Sprintf("jira api error (%d): %s: %s", e.StatusCode, field, msg)
		}
	}
	if e.RequestID != "" {
		return fmt.Sprintf("jira api error (%d) at %s [%s]", e.StatusCode, e.Endpoint, e.RequestID)
	}
	return fmt.Sprintf("jira api error (%d)", e.StatusCode)
}

// Unwrap returns the underlying sentinel error based on status code.
func (e *APIError) Unwrap() error {
	switch e.StatusCode {
	case http.StatusBadRequest:
		return devhttp.ErrBadRequest
	case http.StatusUnauthorized:
		return devhttp.ErrUnauthorized
	case http.StatusForbidden:
		return devhttp.ErrForbidden
	case http.StatusNotFound:
		return devhttp.ErrNotFound
	case http.StatusTooManyRequests:
		return devhttp.ErrRateLimited
	default:
		if e.StatusCode >= 500 {
			return devhttp.ErrServerError
		}
		return nil
	}
}

// IsNotFound returns true if this is a 404 error.
func (e *APIError) IsNotFound() bool {
	return e.StatusCode == http.StatusNotFound
}

// IsUnauthorized returns true if this is a 401 error.
func (e *APIError) IsUnauthorized() bool {
	return e.StatusCode == http.StatusUnauthorized
}

// IsForbidden returns true if this is a 403 error.
func (e *APIError) IsForbidden() bool {
	return e.StatusCode == http.StatusForbidden
}

// IsRateLimited returns true if this is a 429 error.
func (e *APIError) IsRateLimited() bool {
	return e.StatusCode == http.StatusTooManyRequests
}

// NewAPIError creates a new APIError from status code and error messages.
func NewAPIError(statusCode int, messages []string, fieldErrors map[string]string) *APIError {
	return &APIError{
		StatusCode:    statusCode,
		ErrorMessages: messages,
		Errors:        fieldErrors,
	}
}

// parseAPIError parses an error response from the Jira API.
func parseAPIError(resp *http.Response, endpoint string) error {
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	apiErr := &APIError{
		StatusCode: resp.StatusCode,
		Endpoint:   endpoint,
		RequestID:  resp.Header.Get("X-Request-Id"),
	}

	// Try to parse Jira error response format
	if json.Unmarshal(body, apiErr) != nil {
		// Fall back to generic message
		apiErr.ErrorMessages = []string{http.StatusText(resp.StatusCode)}
	}

	return apiErr
}

// IsNotFound reports whether the error indicates a resource was not found.
func IsNotFound(err error) bool {
	return errors.Is(err, devhttp.ErrNotFound) || errors.Is(err, ErrIssueNotFound) ||
		errors.Is(err, ErrProjectNotFound) || errors.Is(err, ErrCommentNotFound)
}

// IsUnauthorized reports whether the error indicates authentication failed.
func IsUnauthorized(err error) bool {
	return errors.Is(err, devhttp.ErrUnauthorized)
}

// IsForbidden reports whether the error indicates permission was denied.
func IsForbidden(err error) bool {
	return errors.Is(err, devhttp.ErrForbidden)
}

// IsRateLimited reports whether the error indicates rate limiting.
func IsRateLimited(err error) bool {
	return errors.Is(err, devhttp.ErrRateLimited)
}

// IsRetryable reports whether the error is transient and should be retried.
func IsRetryable(err error) bool {
	return devhttp.IsRetryable(err)
}
