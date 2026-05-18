package mlx

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/bath0ry/mlx-go-sdk/internal/testutil"
)

func TestFoldersList(t *testing.T) {
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/workspace/folders" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"folders":[{"folder_id":"folder-1","name":"Default folder","comment":"","profiles_count":2,"created_at":"2026-04-19T00:00:00Z"}]}}`)
	})

	client, err := New(
		WithToken("test-token"),
		WithHTTPClient(httpClient),
		WithBaseURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, _, err := client.Folders.List(context.Background())
	if err != nil {
		t.Fatalf("Folders.List returned error: %v", err)
	}

	if len(resp.Data.Folders) != 1 {
		t.Fatalf("expected 1 folder, got %d", len(resp.Data.Folders))
	}
	if resp.Data.Folders[0].FolderID != "folder-1" {
		t.Fatalf("unexpected folder id: %s", resp.Data.Folders[0].FolderID)
	}
}
