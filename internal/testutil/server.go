package testutil

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// NewServer creates a test HTTP server and returns its client.
func NewServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *http.Client) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server, server.Client()
}
