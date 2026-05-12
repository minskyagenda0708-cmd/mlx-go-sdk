package mlx

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type stubTimeoutError struct{}

func (stubTimeoutError) Error() string   { return "timeout" }
func (stubTimeoutError) Timeout() bool   { return true }
func (stubTimeoutError) Temporary() bool { return true }

func TestClassifyTransportError(t *testing.T) {
	err := &TransportError{Err: context.DeadlineExceeded}
	if got := ClassifyError(err); got != ErrorClassTimeout {
		t.Fatalf("unexpected class: %s", got)
	}
	if !IsRetryableError(err) {
		t.Fatal("expected timeout transport error to be retryable")
	}
}

func TestClassifyAPIError(t *testing.T) {
	err := &ErrorResponse{Response: &http.Response{StatusCode: http.StatusTooManyRequests, Header: http.Header{"Retry-After": []string{"3"}}}, Status: Status{HTTPCode: http.StatusTooManyRequests, Message: "rate limit"}}
	if got := ClassifyError(err); got != ErrorClassRateLimited {
		t.Fatalf("unexpected class: %s", got)
	}
	if !IsRetryableError(err) || !IsRateLimitedError(err) {
		t.Fatal("expected 429 api error to be retryable and rate-limited")
	}
	if ra := RetryAfter(err); ra != 3*time.Second {
		t.Fatalf("unexpected retry-after: %v", ra)
	}
}

func TestDoRetriesTransportErrors(t *testing.T) {
	attempts := 0
	client, err := New(
		WithToken("test-token"),
		WithHTTPClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			attempts++
			if attempts < 3 {
				return nil, stubTimeoutError{}
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       http.NoBody,
				Request:    req,
			}, nil
		})}),
		WithRetry(RetryOptions{MaxAttempts: 3, InitialInterval: time.Millisecond, MaxInterval: time.Millisecond, Jitter: 0}),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	req, err := client.newAPIRequest(context.Background(), http.MethodGet, "/workspace/folders", nil)
	if err != nil {
		t.Fatalf("newAPIRequest returned error: %v", err)
	}
	if _, err := client.do(req, nil); err != nil {
		t.Fatalf("do returned error: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestDoDoesNotRetryUnlessConfigured(t *testing.T) {
	attempts := 0
	client, err := New(
		WithToken("test-token"),
		WithHTTPClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			attempts++
			return nil, stubTimeoutError{}
		})}),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	req, err := client.newAPIRequest(context.Background(), http.MethodGet, "/workspace/folders", nil)
	if err != nil {
		t.Fatalf("newAPIRequest returned error: %v", err)
	}
	if _, err := client.do(req, nil); err == nil {
		t.Fatal("expected error")
	}
	if attempts != 1 {
		t.Fatalf("expected 1 attempt without retry config, got %d", attempts)
	}
}

func TestDoRetriesRateLimitResponse(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprint(w, `{"status":{"http_code":429,"message":"rate limit"}}`)
			return
		}
		fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"folders":[]}}`)
	}))
	defer server.Close()

	client, err := New(
		WithToken("test-token"),
		WithHTTPClient(server.Client()),
		WithBaseURL(server.URL),
		WithRetry(RetryOptions{MaxAttempts: 2, InitialInterval: time.Millisecond, MaxInterval: time.Millisecond, Jitter: 0}),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, _, err := client.Folders.List(context.Background())
	if err != nil {
		t.Fatalf("Folders.List returned error: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
	if len(resp.Data.Folders) != 0 {
		t.Fatalf("unexpected folders response: %#v", resp.Data.Folders)
	}
}

func TestDoDoesNotRetryNonIdempotentRequests(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, `{"status":{"http_code":429,"message":"rate limit"}}`)
	}))
	defer server.Close()

	client, err := New(
		WithToken("test-token"),
		WithHTTPClient(server.Client()),
		WithBaseURL(server.URL),
		WithRetry(RetryOptions{MaxAttempts: 3, InitialInterval: time.Millisecond, MaxInterval: time.Millisecond, Jitter: 0}),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	_, _, err = client.Profiles.Create(context.Background(), &CreateProfileRequest{
		Name:     "Demo",
		FolderID: "folder-1",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if attempts != 1 {
		t.Fatalf("expected non-idempotent POST to avoid retries, got %d attempts", attempts)
	}
}

func TestDoDoesNotRetryInvalidRequest(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"status":{"http_code":400,"message":"bad request"}}`)
	}))
	defer server.Close()

	client, err := New(
		WithToken("test-token"),
		WithHTTPClient(server.Client()),
		WithBaseURL(server.URL),
		WithRetry(RetryOptions{MaxAttempts: 3, InitialInterval: time.Millisecond, MaxInterval: time.Millisecond, Jitter: 0}),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	_, _, err = client.Folders.List(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if attempts != 1 {
		t.Fatalf("expected no retry for 400, got %d attempts", attempts)
	}
}

func TestRetryAfterParsesHTTPDate(t *testing.T) {
	when := time.Now().Add(2 * time.Second).UTC().Format(http.TimeFormat)
	err := &ErrorResponse{Response: &http.Response{Header: http.Header{"Retry-After": []string{when}}}}
	if got := RetryAfter(err); got <= 0 {
		t.Fatalf("expected positive retry-after, got %v", got)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestTransportErrorNetworkClass(t *testing.T) {
	err := &TransportError{Err: &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("refused")}}
	if got := ClassifyError(err); got != ErrorClassNetwork {
		t.Fatalf("unexpected class: %s", got)
	}
	if !IsRetryableError(err) {
		t.Fatal("expected network error to be retryable")
	}
}

func TestIsTemporaryError(t *testing.T) {
	if !IsTemporaryError(&TransportError{Err: context.DeadlineExceeded}) {
		t.Fatal("expected temporary transport error")
	}
	if !IsTemporaryError(&ErrorResponse{Response: &http.Response{StatusCode: http.StatusServiceUnavailable}, Status: Status{HTTPCode: http.StatusServiceUnavailable}}) {
		t.Fatal("expected temporary api error")
	}
	if IsTemporaryError(&ErrorResponse{Response: &http.Response{StatusCode: http.StatusBadRequest}, Status: Status{HTTPCode: http.StatusBadRequest}}) {
		t.Fatal("did not expect 400 api error to be temporary")
	}
}

func TestErrorResponseErrorWithoutRequest(t *testing.T) {
	err := &ErrorResponse{Response: &http.Response{StatusCode: http.StatusInternalServerError}, Status: Status{Message: "boom"}}
	if got := err.Error(); got != "mlx api error: 500 boom" {
		t.Fatalf("unexpected error string: %s", got)
	}
}
