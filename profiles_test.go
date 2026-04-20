package mlx

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"mlx-go-sdk/internal/testutil"
)

func TestProfilesSearch(t *testing.T) {
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/profile/search" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("unexpected authorization header: %s", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		if !strings.Contains(string(body), `"search_text":"demo"`) {
			t.Fatalf("expected request body to contain search_text, got %s", string(body))
		}
		fmt.Fprint(w, `{"status":{"http_code":200,"message":"Search profile successfully result"},"data":{"profiles":[{"id":"profile-1","name":"Demo","folder_id":"folder-1","browser_type":"mimic","os_type":"windows","core_version":137,"notes":"","created_by":"me@example.com","in_use_by":"","last_launched_by":"","is_local":false}],"total_count":1}}`)
	})

	client, err := New(
		WithToken("test-token"),
		WithHTTPClient(httpClient),
		WithBaseURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, _, err := client.Profiles.Search(context.Background(), &SearchProfilesRequest{
		IsRemoved:   false,
		Limit:       10,
		Offset:      0,
		SearchText:  "demo",
		StorageType: "cloud",
	})
	if err != nil {
		t.Fatalf("Profiles.Search returned error: %v", err)
	}

	if resp.Data.TotalCount != 1 {
		t.Fatalf("expected total count 1, got %d", resp.Data.TotalCount)
	}
	if resp.Data.Profiles[0].ID != "profile-1" {
		t.Fatalf("unexpected profile id: %s", resp.Data.Profiles[0].ID)
	}
}
