package mlx

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"mlx-go-sdk/internal/testutil"
)

func TestTransfersExportStatus(t *testing.T) {
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/profile/exports/export-1/status" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"export_id":"export-1","export_path":"C:\\\\exports\\\\export-1.zip","profile_id":"profile-1","status":"done","message":"","timestamp":123}}`)
	})

	client, err := New(
		WithToken("test-token"),
		WithHTTPClient(httpClient),
		WithLauncherURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, _, err := client.Transfers.ExportStatus(context.Background(), "export-1")
	if err != nil {
		t.Fatalf("Transfers.ExportStatus returned error: %v", err)
	}

	if resp.Data.Status != "done" {
		t.Fatalf("unexpected export status: %s", resp.Data.Status)
	}
}
