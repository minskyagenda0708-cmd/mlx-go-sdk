package mlx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	ErrMissingToken       = errors.New("mlx token is required")
	ErrNilContext         = errors.New("context must not be nil")
	ErrInvalidBaseURL     = errors.New("invalid base url")
	ErrInvalidLauncherURL = errors.New("invalid launcher url")
	ErrProfileNotFound    = errors.New("profile not found")
	ErrProfileAmbiguous   = errors.New("profile lookup matched multiple profiles")
)

// ErrorClass describes a typed SDK error category.
type ErrorClass string

const (
	ErrorClassUnknown        ErrorClass = "unknown"
	ErrorClassCanceled       ErrorClass = "canceled"
	ErrorClassTimeout        ErrorClass = "timeout"
	ErrorClassNetwork        ErrorClass = "network"
	ErrorClassRateLimited    ErrorClass = "rate_limited"
	ErrorClassUnauthorized   ErrorClass = "unauthorized"
	ErrorClassForbidden      ErrorClass = "forbidden"
	ErrorClassNotFound       ErrorClass = "not_found"
	ErrorClassConflict       ErrorClass = "conflict"
	ErrorClassInvalidRequest ErrorClass = "invalid_request"
	ErrorClassServer         ErrorClass = "server"
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

// TransportError wraps network/transport failures from the underlying HTTP client.
type TransportError struct {
	Request *http.Request
	Err     error
}

func (e *TransportError) Error() string {
	if e == nil {
		return "mlx transport error"
	}
	if e.Request != nil && e.Request.URL != nil {
		return fmt.Sprintf("%s %s: %v", e.Request.Method, e.Request.URL.String(), e.Err)
	}
	return fmt.Sprintf("mlx transport error: %v", e.Err)
}

func (e *TransportError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// Class returns the typed transport error category.
func (e *TransportError) Class() ErrorClass {
	if e == nil {
		return ErrorClassUnknown
	}
	if errors.Is(e.Err, context.Canceled) {
		return ErrorClassCanceled
	}
	if errors.Is(e.Err, context.DeadlineExceeded) || e.Timeout() {
		return ErrorClassTimeout
	}
	return ErrorClassNetwork
}

// Timeout reports whether the wrapped transport failure is a timeout.
func (e *TransportError) Timeout() bool {
	if e == nil {
		return false
	}
	type timeoutError interface{ Timeout() bool }
	var target timeoutError
	return errors.As(e.Err, &target) && target.Timeout()
}

// Temporary reports whether the transport failure is likely transient.
func (e *TransportError) Temporary() bool {
	if e == nil {
		return false
	}
	if errors.Is(e.Err, context.Canceled) {
		return false
	}
	if errors.Is(e.Err, context.DeadlineExceeded) || e.Timeout() {
		return true
	}
	type temporaryError interface{ Temporary() bool }
	var temp temporaryError
	if errors.As(e.Err, &temp) && temp.Temporary() {
		return true
	}
	var urlErr *url.Error
	if errors.As(e.Err, &urlErr) {
		return urlErr.Timeout()
	}
	return false
}

// Retryable reports whether retry/backoff helpers should retry the transport error.
func (e *TransportError) Retryable() bool {
	if e == nil {
		return false
	}
	if errors.Is(e.Err, context.Canceled) {
		return false
	}
	return e.Temporary() || e.Class() == ErrorClassNetwork
}

// ErrorResponse represents a MultiloginX API error.
type ErrorResponse struct {
	Response *http.Response
	Status   Status `json:"status"`
	Body     []byte `json:"-"`
}

func (e *ErrorResponse) Error() string {
	if e == nil {
		return "mlx api error"
	}
	if e.Response != nil && e.Response.Request != nil && e.Response.Request.URL != nil {
		return fmt.Sprintf("%s %s: %d %s", e.Response.Request.Method, e.Response.Request.URL, e.Response.StatusCode, e.Status.Message)
	}
	if e.Response != nil {
		return fmt.Sprintf("mlx api error: %d %s", e.Response.StatusCode, e.Status.Message)
	}
	return fmt.Sprintf("mlx api error: %s", e.Status.Message)
}

// StatusCode returns the best available HTTP/status code for the response.
func (e *ErrorResponse) StatusCode() int {
	if e == nil {
		return 0
	}
	if e.Response != nil && e.Response.StatusCode != 0 {
		return e.Response.StatusCode
	}
	return e.Status.HTTPCode
}

// Class returns the typed error category for the API response.
func (e *ErrorResponse) Class() ErrorClass {
	switch code := e.StatusCode(); {
	case code == http.StatusTooManyRequests || strings.Contains(strings.ToLower(strings.TrimSpace(e.Status.Message)), "rate limit"):
		return ErrorClassRateLimited
	case code == http.StatusUnauthorized:
		return ErrorClassUnauthorized
	case code == http.StatusForbidden:
		return ErrorClassForbidden
	case code == http.StatusNotFound:
		return ErrorClassNotFound
	case code == http.StatusConflict:
		return ErrorClassConflict
	case code == http.StatusRequestTimeout || code == http.StatusGatewayTimeout:
		return ErrorClassTimeout
	case code >= 500:
		return ErrorClassServer
	case code >= 400:
		return ErrorClassInvalidRequest
	default:
		return ErrorClassUnknown
	}
}

// RetryAfter returns the parsed Retry-After header when present.
func (e *ErrorResponse) RetryAfter() time.Duration {
	if e == nil || e.Response == nil {
		return 0
	}
	raw := strings.TrimSpace(e.Response.Header.Get("Retry-After"))
	if raw == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(raw); err == nil {
		if seconds <= 0 {
			return 0
		}
		return time.Duration(seconds) * time.Second
	}
	if at, err := http.ParseTime(raw); err == nil {
		if d := time.Until(at); d > 0 {
			return d
		}
	}
	return 0
}

// Temporary reports whether the API error is likely transient.
func (e *ErrorResponse) Temporary() bool {
	if e == nil {
		return false
	}
	switch e.Class() {
	case ErrorClassRateLimited, ErrorClassServer, ErrorClassTimeout:
		return true
	default:
		return false
	}
}

// Retryable reports whether retry/backoff helpers should retry the API error.
func (e *ErrorResponse) Retryable() bool {
	if e == nil {
		return false
	}
	code := e.StatusCode()
	if e.Temporary() {
		return true
	}
	return code == http.StatusBadGateway || code == http.StatusServiceUnavailable || code == http.StatusTooEarly
}

// IsRateLimited reports whether the API error is a rate limit response.
func (e *ErrorResponse) IsRateLimited() bool {
	return e != nil && e.Class() == ErrorClassRateLimited
}

// ClassifyError returns a typed classification for SDK errors.
func ClassifyError(err error) ErrorClass {
	if err == nil {
		return ErrorClassUnknown
	}
	var transportErr *TransportError
	if errors.As(err, &transportErr) {
		return transportErr.Class()
	}
	var apiErr *ErrorResponse
	if errors.As(err, &apiErr) {
		return apiErr.Class()
	}
	if errors.Is(err, context.Canceled) {
		return ErrorClassCanceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrorClassTimeout
	}
	return ErrorClassUnknown
}

// IsRetryableError reports whether the given error is safe to retry automatically.
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}
	var transportErr *TransportError
	if errors.As(err, &transportErr) {
		return transportErr.Retryable()
	}
	var apiErr *ErrorResponse
	if errors.As(err, &apiErr) {
		return apiErr.Retryable()
	}
	return false
}

// IsTemporaryError reports whether the given error is likely transient.
func IsTemporaryError(err error) bool {
	if err == nil {
		return false
	}
	var transportErr *TransportError
	if errors.As(err, &transportErr) {
		return transportErr.Temporary()
	}
	var apiErr *ErrorResponse
	if errors.As(err, &apiErr) {
		return apiErr.Temporary()
	}
	return false
}

// IsRateLimitedError reports whether the error represents an MLX/API rate limit condition.
func IsRateLimitedError(err error) bool {
	var apiErr *ErrorResponse
	return errors.As(err, &apiErr) && apiErr.IsRateLimited()
}

// RetryAfter returns the recommended delay extracted from a typed SDK error when present.
func RetryAfter(err error) time.Duration {
	var apiErr *ErrorResponse
	if errors.As(err, &apiErr) {
		return apiErr.RetryAfter()
	}
	return 0
}
