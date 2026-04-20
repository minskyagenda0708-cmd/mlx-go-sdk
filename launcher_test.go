package mlx

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"mlx-go-sdk/internal/testutil"
)

func TestLauncherStart(t *testing.T) {
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v2/profile/f/folder-1/p/profile-1/start" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("automation_type"); got != string(AutomationPlaywright) {
			t.Fatalf("unexpected automation_type: %s", got)
		}
		if got := r.URL.Query().Get("headless_mode"); got != "true" {
			t.Fatalf("unexpected headless_mode: %s", got)
		}
		fmt.Fprint(w, `{"status":{"http_code":200,"message":"Profile started successfully"},"data":{"browser_type":"mimic","core_version":132,"id":"profile-1","is_quick":false,"port":"55513"}}`)
	})

	client, err := New(
		WithToken("test-token"),
		WithHTTPClient(httpClient),
		WithLauncherURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, _, err := client.Launcher.Start(context.Background(), "folder-1", "profile-1", StartProfileOptions{
		AutomationType: AutomationPlaywright,
		Headless:       true,
	})
	if err != nil {
		t.Fatalf("Launcher.Start returned error: %v", err)
	}

	if resp.Data.Port != "55513" {
		t.Fatalf("unexpected port: %s", resp.Data.Port)
	}
}
