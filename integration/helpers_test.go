// Package integration_test contains HTTP-backed integration helpers for the
// exported mlx-go-sdk surface.
//
// Integration tests in this package should exercise SDK behavior through
// exported APIs and controlled httptest-backed fixtures while remaining safe for
// routine `go test ./...` execution.
package integration_test

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func verifiedProfileMetaJSON(id, name, folderID string) string {
	return fmt.Sprintf(`{"id":%q,"name":%q,"folder_id":%q,"browser_type":"mimic","core_version":137,"os_type":"windows","workspace_id":"ws-1","created_at":"2026-04-20T00:00:00Z","created_by":"me@example.com","last_update_at":"2026-04-20T00:00:00Z","last_updated_by":"me@example.com","status":"ready","parameters":{"storage":{"is_local":true}}}`, id, name, folderID)
}

func extractBodyField(t *testing.T, r *http.Request, prefix string) string {
	t.Helper()
	body := mustReadBody(t, r)
	text := string(body)

	start := strings.Index(text, prefix)
	if start == -1 {
		return ""
	}
	start += len(prefix)

	end := strings.Index(text[start:], `"`)
	if end == -1 {
		return ""
	}
	return text[start : start+end]
}

func readRequestBody(t *testing.T, r *http.Request) string {
	t.Helper()
	return string(mustReadBody(t, r))
}

func mustReadBody(t *testing.T, r *http.Request) []byte {
	t.Helper()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("failed to read request body: %v", err)
	}
	return body
}

func contains(text, part string) bool {
	return strings.Contains(text, part)
}
