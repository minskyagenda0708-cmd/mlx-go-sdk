package mlx

import (
	"errors"
	"fmt"
	"net/http"
)

var (
	ErrMissingToken       = errors.New("mlx token is required")
	ErrNilContext         = errors.New("context must not be nil")
	ErrInvalidBaseURL     = errors.New("invalid base url")
	ErrInvalidLauncherURL = errors.New("invalid launcher url")
	ErrProfileNotFound    = errors.New("profile not found")
	ErrProfileAmbiguous   = errors.New("profile lookup matched multiple profiles")
)

// ArgError describes an invalid input argument.
type ArgError struct {
	arg    string
	reason string
}

func NewArgError(arg, reason string) *ArgError {
	return &ArgError{arg: arg, reason: reason}
}

func (e *ArgError) Error() string {
	return fmt.Sprintf("%s is invalid because %s", e.arg, e.reason)
}

// ErrorResponse represents a MultiloginX API error.
type ErrorResponse struct {
	Response *http.Response
	Status   Status `json:"status"`
}

func (e *ErrorResponse) Error() string {
	if e == nil {
		return "mlx api error"
	}
	if e.Response != nil {
		return fmt.Sprintf("%s %s: %d %s", e.Response.Request.Method, e.Response.Request.URL, e.Response.StatusCode, e.Status.Message)
	}
	return fmt.Sprintf("mlx api error: %s", e.Status.Message)
}
