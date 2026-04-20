package mlx

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

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

func TestLauncherStatuses(t *testing.T) {
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/profile/statuses" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"active_counter":{"cloud":2,"local":0,"quick":1},"states":{"profile-1":{"browser_type":"mimic","core_version":137,"folder_id":"folder-1","in_use_by":"","is_quick":false,"last_launched_at":"2026-04-20T00:00:00Z","last_launched_by":"marvin@example.com","last_launched_on":"localhost","message":"","name":"Demo","profile_id":"profile-1","status":"browser_running","timestamp":1745100000000,"workspace_id":"workspace-1"}}}}`)
	})

	client, err := New(
		WithToken("test-token"),
		WithHTTPClient(httpClient),
		WithLauncherURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, _, err := client.Launcher.Statuses(context.Background())
	if err != nil {
		t.Fatalf("Launcher.Statuses returned error: %v", err)
	}
	if resp.Data.ActiveCounter.Cloud != 2 || resp.Data.ActiveCounter.Quick != 1 {
		t.Fatalf("unexpected active counter: %#v", resp.Data.ActiveCounter)
	}
	state := resp.Data.States["profile-1"]
	if state.LastLaunchedOn != "localhost" {
		t.Fatalf("unexpected last launched on: %s", state.LastLaunchedOn)
	}
	if state.Timestamp != 1745100000000 {
		t.Fatalf("unexpected timestamp: %d", state.Timestamp)
	}
}

func TestLauncherQuickStatuses(t *testing.T) {
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/profile/quick/statuses" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"active_counter":1,"states":{"quick-1":{"browser_type":"mimic","is_quick":true,"message":"57165","name":"test","status":"browser_running","timestamp":1744706373229}}}}`)
	})

	client, err := New(
		WithToken("test-token"),
		WithHTTPClient(httpClient),
		WithLauncherURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, _, err := client.Launcher.QuickStatuses(context.Background())
	if err != nil {
		t.Fatalf("Launcher.QuickStatuses returned error: %v", err)
	}
	if resp.Data.ActiveCounter != 1 {
		t.Fatalf("unexpected active counter: %d", resp.Data.ActiveCounter)
	}
	if resp.Data.States["quick-1"].Timestamp != 1744706373229 {
		t.Fatalf("unexpected timestamp: %d", resp.Data.States["quick-1"].Timestamp)
	}
}

func TestLauncherWaitForRunningRetries(t *testing.T) {
	var calls atomic.Int32
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/profile/status/p/profile-1" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if calls.Add(1) < 3 {
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"profile_id":"profile-1","name":"Demo","status":"start_browser","browser_type":"mimic","core_version":137,"folder_id":"folder-1","workspace_id":"workspace-1","message":""}}`)
			return
		}
		fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"profile_id":"profile-1","name":"Demo","status":"browser_running","browser_type":"mimic","core_version":137,"folder_id":"folder-1","workspace_id":"workspace-1","message":"","timestamp":1745100000000}}`)
	})

	client, err := New(
		WithToken("test-token"),
		WithHTTPClient(httpClient),
		WithLauncherURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, _, err := client.Launcher.WaitForRunning(context.Background(), "profile-1", PollOptions{InitialInterval: time.Millisecond, MaxInterval: 2 * time.Millisecond, Timeout: time.Second, Multiplier: 2})
	if err != nil {
		t.Fatalf("Launcher.WaitForRunning returned error: %v", err)
	}
	if resp.Data.Status != "browser_running" {
		t.Fatalf("unexpected status: %s", resp.Data.Status)
	}
}
