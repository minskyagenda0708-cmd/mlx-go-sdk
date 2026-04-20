package mlx

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mlx-go-sdk/internal/testutil"
)

func TestWorkflowStartProfileByName(t *testing.T) {
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/profile/search":
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[{"id":"profile-1","name":"Demo","folder_id":"folder-1","browser_type":"mimic","os_type":"windows","core_version":137,"notes":"","created_by":"me@example.com","in_use_by":"","last_launched_by":"","is_local":false}],"total_count":1}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/profile/f/folder-1/p/profile-1/start":
			fmt.Fprint(w, `{"status":{"http_code":200,"message":"Profile started successfully"},"data":{"browser_type":"mimic","core_version":137,"id":"profile-1","is_quick":false,"port":"55513"}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/profile/status/p/profile-1":
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"profile_id":"profile-1","name":"Demo","status":"browser_running","browser_type":"mimic","core_version":137,"folder_id":"folder-1","workspace_id":"workspace-1","message":"","timestamp":1745100000000}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})

	client, err := New(
		WithToken("test-token"),
		WithHTTPClient(httpClient),
		WithBaseURL(server.URL),
		WithLauncherURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	result, err := client.Workflows.StartProfileByName(context.Background(), "Demo", StartProfileByNameOptions{
		WaitForRunning: true,
		PollOptions:    PollOptions{InitialInterval: time.Millisecond, MaxInterval: time.Millisecond, Timeout: time.Second},
	})
	if err != nil {
		t.Fatalf("Workflows.StartProfileByName returned error: %v", err)
	}
	if result.Profile.ID != "profile-1" {
		t.Fatalf("unexpected profile id: %s", result.Profile.ID)
	}
	if result.StartResponse.Data.Port != "55513" {
		t.Fatalf("unexpected start port: %s", result.StartResponse.Data.Port)
	}
	if result.RuntimeStatus == nil || result.RuntimeStatus.Data.Status != "browser_running" {
		t.Fatalf("unexpected runtime status: %#v", result.RuntimeStatus)
	}
}

func TestWorkflowExportProfileByNameToFolder(t *testing.T) {
	workspace := t.TempDir()
	exportSource := filepath.Join(workspace, "export-1.zip")
	if err := os.WriteFile(exportSource, []byte("zip"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/profile/search":
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[{"id":"profile-1","name":"Demo","folder_id":"folder-1","browser_type":"mimic","os_type":"windows","core_version":137,"notes":"","created_by":"me@example.com","in_use_by":"","last_launched_by":"","is_local":false}],"total_count":1}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/profile/stop/p/profile-1":
			fmt.Fprint(w, `{"status":{"http_code":200,"message":"Profile stopped successfully"},"data":null}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/profile/profile-1/export":
			fmt.Fprintf(w, `{"status":{"http_code":200,"message":"Export in progress"},"data":{"export_id":"export-1","export_path":"%s","profile_id":"profile-1","status":"running","message":"","timestamp":1713552000000}}`, filepath.ToSlash(exportSource))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/profile/exports/export-1/status":
			fmt.Fprintf(w, `{"status":{"http_code":200,"message":""},"data":{"export_id":"export-1","export_path":"%s","profile_id":"profile-1","status":"done","message":"","timestamp":1713552000000}}`, filepath.ToSlash(exportSource))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})

	client, err := New(
		WithToken("test-token"),
		WithHTTPClient(httpClient),
		WithBaseURL(server.URL),
		WithLauncherURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	result, err := client.Workflows.ExportProfileByNameToFolder(context.Background(), "Demo", ExportProfileByNameToFolderOptions{
		ExportOptions: ExportProfileToFolderOptions{
			RootDir:      filepath.Join(workspace, "archives"),
			PollInterval: 10 * time.Millisecond,
			WaitTimeout:  time.Second,
		},
		StopBeforeExport: true,
	})
	if err != nil {
		t.Fatalf("Workflows.ExportProfileByNameToFolder returned error: %v", err)
	}
	if result.Profile.ID != "profile-1" {
		t.Fatalf("unexpected profile id: %s", result.Profile.ID)
	}
	if result.Export == nil || result.Export.Archive == nil {
		t.Fatalf("unexpected export result: %#v", result.Export)
	}
	if filepath.Ext(result.Export.Archive.ArchivePath) != ".zip" {
		t.Fatalf("unexpected archive path: %s", result.Export.Archive.ArchivePath)
	}
}

func TestWorkflowStopProfileByNameIgnoresAlreadyStopped(t *testing.T) {
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/profile/search":
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[{"id":"profile-1","name":"Demo","folder_id":"folder-1","browser_type":"mimic","os_type":"windows","core_version":137,"notes":"","created_by":"me@example.com","in_use_by":"","last_launched_by":"","is_local":false}],"total_count":1}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/profile/stop/p/profile-1":
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"status":{"http_code":500,"message":"profile already stopped"}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})

	client, err := New(
		WithToken("test-token"),
		WithHTTPClient(httpClient),
		WithBaseURL(server.URL),
		WithLauncherURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	result, err := client.Workflows.StopProfileByName(context.Background(), "Demo", StopProfileByNameOptions{IgnoreAlreadyStopped: true})
	if err != nil {
		t.Fatalf("Workflows.StopProfileByName returned error: %v", err)
	}
	if result.Profile.ID != "profile-1" {
		t.Fatalf("unexpected profile id: %s", result.Profile.ID)
	}
	if result.StopResponse != nil {
		t.Fatalf("expected nil stop response when already stopped, got %#v", result.StopResponse)
	}
}

func TestWorkflowGenerateProfileProxyByNameAndPatch(t *testing.T) {
	step := 0
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch step {
		case 0:
			if r.Method != http.MethodPost || r.URL.Path != "/profile/search" {
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[{"id":"profile-1","name":"Demo","folder_id":"folder-1","browser_type":"mimic","os_type":"windows","core_version":137,"notes":"","created_by":"me@example.com","in_use_by":"","last_launched_by":"","is_local":false}],"total_count":1}}`)
		case 1:
			if r.Method != http.MethodGet || r.URL.Path != "/v1/user" {
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			fmt.Fprint(w, `{"traffic":1501700871,"billingId":"2235470499"}`)
		case 2:
			if r.Method != http.MethodPost || r.URL.Path != "/v1/proxy/connection_url" {
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("failed to read body: %v", err)
			}
			text := string(body)
			if !strings.Contains(text, `"protocol":"socks5"`) || !strings.Contains(text, `"region":"new_jersey"`) || !strings.Contains(text, `"city":"east_brunswick"`) {
				t.Fatalf("unexpected generate body: %s", text)
			}
			fmt.Fprint(w, `{"status":200,"data":["gate.multilogin.com:1080:2235470499_bc98e4f8_multilogin_com-country-us-region-new_jersey-city-east_brunswick-sid-demo:secret"]}`)
		case 3:
			if r.Method != http.MethodPost || r.URL.Path != "/profile/partial_update" {
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("failed to read body: %v", err)
			}
			text := string(body)
			checks := []string{
				`"profile_id":"profile-1"`,
				`"type":"socks5"`,
				`"country":"us"`,
				`"region":"new_jersey"`,
				`"city":"east_brunswick"`,
			}
			for _, check := range checks {
				if !strings.Contains(text, check) {
					t.Fatalf("expected patch body to contain %s, got %s", check, text)
				}
			}
			fmt.Fprint(w, `{"status":{"http_code":200,"message":"updated"},"data":null}`)
		default:
			t.Fatalf("unexpected request step %d", step)
		}
		step++
	})

	client, err := New(
		WithToken("test-token"),
		WithHTTPClient(httpClient),
		WithBaseURL(server.URL),
		WithProxyURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	result, err := client.Workflows.GenerateProfileProxyByName(context.Background(), "Demo", GenerateProfileProxyByNameOptions{
		PatchProfile: true,
		GenerateOptions: GenerateProfileProxyRequest{
			GenerateProxyRequest: GenerateProxyRequest{
				Country:     "us",
				Region:      "new_jersey",
				City:        "east_brunswick",
				SessionType: ProxySessionSticky,
			},
			PreferSOCKS5: true,
		},
	})
	if err != nil {
		t.Fatalf("Workflows.GenerateProfileProxyByName returned error: %v", err)
	}
	if result.Profile.ID != "profile-1" {
		t.Fatalf("unexpected profile id: %s", result.Profile.ID)
	}
	if result.ProfileProxy == nil || result.ProfileProxy.Type != "socks5" {
		t.Fatalf("unexpected profile proxy: %#v", result.ProfileProxy)
	}
	if result.Connection == nil || result.Connection.Region != "new_jersey" {
		t.Fatalf("unexpected connection: %#v", result.Connection)
	}
	if result.Usage == nil || result.Usage.BillingID != "2235470499" {
		t.Fatalf("unexpected usage: %#v", result.Usage)
	}
	if result.PatchResponse == nil || result.PatchResponse.Status.HTTPCode != 200 {
		t.Fatalf("unexpected patch response: %#v", result.PatchResponse)
	}
}
