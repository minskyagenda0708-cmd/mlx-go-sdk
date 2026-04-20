package mlx

import (
	"context"
	"fmt"
	"io"
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

func TestLauncherHealth(t *testing.T) {
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/version" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		if len(body) != 0 {
			t.Fatalf("expected empty body, got %q", string(body))
		}
		fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"env":"desktop","version":"1.11.1"}}`)
	})

	client, err := New(
		WithToken("test-token"),
		WithHTTPClient(httpClient),
		WithLauncherURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, _, err := client.Launcher.Health(context.Background())
	if err != nil {
		t.Fatalf("Launcher.Health returned error: %v", err)
	}

	if !resp.Data.Alive {
		t.Fatal("expected launcher to be reported alive")
	}
	if resp.Data.Version != "1.11.1" {
		t.Fatalf("unexpected version: %s", resp.Data.Version)
	}
	if resp.Data.Env != "desktop" {
		t.Fatalf("unexpected env: %s", resp.Data.Env)
	}
}
