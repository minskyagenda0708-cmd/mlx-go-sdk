package integration_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	mlx "github.com/bath0ry/mlx-go-sdk"
	"github.com/bath0ry/mlx-go-sdk/internal/testutil"
)

func TestWorkflowStartProfileByName(t *testing.T) {
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/profile/search":
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[{"id":"profile-1","name":"Demo","folder_id":"folder-1","browser_type":"mimic","os_type":"windows","core_version":137,"notes":"","created_by":"me@example.com","in_use_by":"","last_launched_by":"","is_local":false}],"total_count":1}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/profile/metas":
			fmt.Fprintf(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[%s]}}`, verifiedProfileMetaJSON("profile-1", "Demo", "folder-1"))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/profile/f/folder-1/p/profile-1/start":
			fmt.Fprint(w, `{"status":{"http_code":200,"message":"Profile started successfully"},"data":{"browser_type":"mimic","core_version":137,"id":"profile-1","is_quick":false,"port":"55513"}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/profile/status/p/profile-1":
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"profile_id":"profile-1","name":"Demo","status":"browser_running","browser_type":"mimic","core_version":137,"folder_id":"folder-1","workspace_id":"workspace-1","message":"","timestamp":1745100000000}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})

	client, err := mlx.New(
		mlx.WithToken("test-token"),
		mlx.WithHTTPClient(httpClient),
		mlx.WithBaseURL(server.URL),
		mlx.WithLauncherURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	result, err := client.Workflows.StartProfileByName(context.Background(), "Demo", mlx.StartProfileByNameOptions{
		WaitForRunning: true,
		PollOptions:    mlx.PollOptions{InitialInterval: time.Millisecond, MaxInterval: time.Millisecond, Timeout: time.Second},
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

func TestWorkflowStartProfileAutomationByName(t *testing.T) {
	cdpPort := ""
	cdpCalls := 0
	statusCalls := 0
	cdpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/json/version" {
			t.Fatalf("unexpected cdp path: %s", r.URL.Path)
		}
		cdpCalls++
		if cdpCalls > 1 {
			t.Fatalf("expected CDP endpoint to be resolved once, got %d calls", cdpCalls)
		}
		if statusCalls < 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprint(w, `{"message":"not ready"}`)
			return
		}
		fmt.Fprintf(w, `{"webSocketDebuggerUrl":"ws://127.0.0.1:%s/devtools/browser/demo"}`, cdpPort)
	}))
	t.Cleanup(cdpServer.Close)

	cdpURL, err := url.Parse(cdpServer.URL)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	cdpPort = cdpURL.Port()

	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/profile/search":
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[{"id":"profile-1","name":"Demo","folder_id":"folder-1","browser_type":"mimic","os_type":"windows","core_version":137,"notes":"","created_by":"me@example.com","in_use_by":"","last_launched_by":"","is_local":false}],"total_count":1}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/profile/metas":
			fmt.Fprintf(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[%s]}}`, verifiedProfileMetaJSON("profile-1", "Demo", "folder-1"))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/profile/f/folder-1/p/profile-1/start":
			if got := r.URL.Query().Get("automation_type"); got != "playwright" {
				t.Fatalf("expected normalized automation_type=playwright, got %q", got)
			}
			fmt.Fprintf(w, `{"status":{"http_code":200,"message":"ok"},"data":{"browser_type":"mimic","core_version":137,"id":"profile-1","is_quick":false,"port":"%s"}}`, cdpURL.Port())
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/profile/status/p/profile-1":
			statusCalls++
			status := "starting"
			if statusCalls > 1 {
				status = "browser_running"
			}
			fmt.Fprintf(w, `{"status":{"http_code":200,"message":""},"data":{"profile_id":"profile-1","name":"Demo","status":%q,"browser_type":"mimic","core_version":137,"folder_id":"folder-1","workspace_id":"workspace-1","message":"","timestamp":1745100000000}}`, status)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})

	client, err := mlx.New(
		mlx.WithToken("test-token"),
		mlx.WithHTTPClient(httpClient),
		mlx.WithBaseURL(server.URL),
		mlx.WithLauncherURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	result, err := client.Workflows.StartProfileAutomationByName(context.Background(), "Demo", mlx.StartProfileAutomationByNameOptions{
		StartOptions: mlx.StartProfileOptions{
			AutomationType: mlx.AutomationRod,
		},
		WaitForRunning: true,
		PollOptions:    mlx.PollOptions{InitialInterval: time.Millisecond, MaxInterval: time.Millisecond, Timeout: time.Second},
	})
	if err != nil {
		t.Fatalf("Workflows.StartProfileAutomationByName returned error: %v", err)
	}
	if result.Profile.ID != "profile-1" {
		t.Fatalf("unexpected profile id: %s", result.Profile.ID)
	}
	if result.StartResponse == nil || result.StartResponse.Data.Port != cdpURL.Port() {
		t.Fatalf("unexpected start response: %#v", result.StartResponse)
	}
	if result.StartResponse.Data.RequestedAutomation != mlx.AutomationRod {
		t.Fatalf("unexpected requested automation in start response: %q", result.StartResponse.Data.RequestedAutomation)
	}
	if result.StartResponse.Data.LauncherAutomation != mlx.AutomationPlaywright {
		t.Fatalf("unexpected launcher automation in start response: %q", result.StartResponse.Data.LauncherAutomation)
	}
	if result.RuntimeStatus == nil || result.RuntimeStatus.Data.Status != "browser_running" {
		t.Fatalf("unexpected runtime status: %#v", result.RuntimeStatus)
	}
	if result.RequestedAutomation != mlx.AutomationRod {
		t.Fatalf("unexpected requested automation: %q", result.RequestedAutomation)
	}
	if result.LauncherAutomation != mlx.AutomationPlaywright {
		t.Fatalf("unexpected launcher automation: %q", result.LauncherAutomation)
	}
	if result.CDPPort != cdpURL.Port() {
		t.Fatalf("unexpected cdp port: %q", result.CDPPort)
	}
	wantCDPURL := fmt.Sprintf("ws://127.0.0.1:%s/devtools/browser/demo", cdpURL.Port())
	if result.CDPWebSocketURL != wantCDPURL {
		t.Fatalf("unexpected cdp websocket url: got %q want %q", result.CDPWebSocketURL, wantCDPURL)
	}
	if result.RodControlURL != wantCDPURL {
		t.Fatalf("unexpected rod control url: got %q want %q", result.RodControlURL, wantCDPURL)
	}
	if cdpCalls != 1 {
		t.Fatalf("expected a single CDP probe, got %d", cdpCalls)
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
		case r.Method == http.MethodPost && r.URL.Path == "/profile/metas":
			fmt.Fprintf(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[%s]}}`, verifiedProfileMetaJSON("profile-1", "Demo", "folder-1"))
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

	client, err := mlx.New(
		mlx.WithToken("test-token"),
		mlx.WithHTTPClient(httpClient),
		mlx.WithBaseURL(server.URL),
		mlx.WithLauncherURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	result, err := client.Workflows.ExportProfileByNameToFolder(context.Background(), "Demo", mlx.ExportProfileByNameToFolderOptions{
		ExportOptions: mlx.ExportProfileToFolderOptions{
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
		case r.Method == http.MethodPost && r.URL.Path == "/profile/metas":
			fmt.Fprintf(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[%s]}}`, verifiedProfileMetaJSON("profile-1", "Demo", "folder-1"))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/profile/stop/p/profile-1":
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"status":{"http_code":500,"message":"profile already stopped"}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})

	client, err := mlx.New(
		mlx.WithToken("test-token"),
		mlx.WithHTTPClient(httpClient),
		mlx.WithBaseURL(server.URL),
		mlx.WithLauncherURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	result, err := client.Workflows.StopProfileByName(context.Background(), "Demo", mlx.StopProfileByNameOptions{IgnoreAlreadyStopped: true})
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
			if r.Method != http.MethodPost || r.URL.Path != "/profile/metas" {
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			fmt.Fprintf(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[%s]}}`, verifiedProfileMetaJSON("profile-1", "Demo", "folder-1"))
		case 2:
			if r.Method != http.MethodGet || r.URL.Path != "/v1/user" {
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			fmt.Fprint(w, `{"traffic":1501700871,"billingId":"2235470499"}`)
		case 3:
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
		case 4:
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

	client, err := mlx.New(
		mlx.WithToken("test-token"),
		mlx.WithHTTPClient(httpClient),
		mlx.WithBaseURL(server.URL),
		mlx.WithProxyURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	result, err := client.Workflows.GenerateProfileProxyByName(context.Background(), "Demo", mlx.GenerateProfileProxyByNameOptions{
		PatchProfile: true,
		GenerateOptions: mlx.GenerateProfileProxyRequest{
			GenerateProxyRequest: mlx.GenerateProxyRequest{
				Country:     "us",
				Region:      "new_jersey",
				City:        "east_brunswick",
				SessionType: mlx.ProxySessionSticky,
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

func TestWorkflowCreateProfilesAndVerify(t *testing.T) {
	metaCalls := 0
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/profile/create":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("failed to read create body: %v", err)
			}
			text := string(body)
			if !strings.Contains(text, `"name":"Demo"`) {
				t.Fatalf("unexpected create body: %s", text)
			}
			fmt.Fprint(w, `{"status":{"http_code":200,"message":"created"},"data":{"ids":["profile-1","profile-2"]}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/profile/metas":
			metaCalls++
			if metaCalls == 1 {
				fmt.Fprintf(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[%s]}}`, verifiedProfileMetaJSON("profile-1", "Demo", "folder-1"))
				return
			}
			fmt.Fprintf(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[%s,%s]}}`, verifiedProfileMetaJSON("profile-1", "Demo", "folder-1"), verifiedProfileMetaJSON("profile-2", "Demo", "folder-1"))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})

	client, err := mlx.New(
		mlx.WithToken("test-token"),
		mlx.WithHTTPClient(httpClient),
		mlx.WithBaseURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	result, err := client.Workflows.CreateProfilesAndVerify(context.Background(), &mlx.CreateProfileRequest{
		Name:        "Demo",
		BrowserType: "mimic",
		FolderID:    "folder-1",
		OSType:      "windows",
		Times:       2,
	}, mlx.CreateProfilesAndVerifyOptions{
		PollOptions: mlx.PollOptions{InitialInterval: time.Millisecond, MaxInterval: time.Millisecond, Timeout: time.Second},
	})
	if err != nil {
		t.Fatalf("Workflows.CreateProfilesAndVerify returned error: %v", err)
	}
	if metaCalls < 2 {
		t.Fatalf("expected verification polling, got %d meta calls", metaCalls)
	}
	if len(result.Profiles) != 2 {
		t.Fatalf("expected 2 verified profiles, got %d", len(result.Profiles))
	}
	ids := []string{result.Profiles[0].ID, result.Profiles[1].ID}
	if !slices.Equal(ids, []string{"profile-1", "profile-2"}) {
		t.Fatalf("unexpected verified ids: %#v", ids)
	}
}

func TestWorkflowFindProfileByNameVerified(t *testing.T) {
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/profile/search":
			storageType := extractBodyField(t, r, `"storage_type":"`)
			if storageType != "local" {
				t.Fatalf("expected local workflow default storage, got %q", storageType)
			}
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[{"id":"profile-1","name":"Demo","folder_id":"folder-1","browser_type":"mimic","os_type":"windows","core_version":137,"notes":"","created_by":"me@example.com","in_use_by":"","last_launched_by":"","is_local":true}],"total_count":1}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/profile/metas":
			fmt.Fprintf(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[%s]}}`, verifiedProfileMetaJSON("profile-1", "Demo", "folder-1"))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})

	client, err := mlx.New(
		mlx.WithToken("test-token"),
		mlx.WithHTTPClient(httpClient),
		mlx.WithBaseURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	result, err := client.Workflows.FindProfileByNameVerified(context.Background(), "Demo", mlx.FindProfileByNameVerifiedOptions{})
	if err != nil {
		t.Fatalf("Workflows.FindProfileByNameVerified returned error: %v", err)
	}
	if result.Profile.ID != "profile-1" || result.Meta.ID != "profile-1" {
		t.Fatalf("unexpected verified result: %#v", result)
	}
	if !result.Meta.CheckLocal() {
		t.Fatalf("expected verified meta to preserve local storage signal")
	}
}

func TestWorkflowStopProfileByNameWaitsForStopped(t *testing.T) {
	statusCalls := 0
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/profile/search":
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[{"id":"profile-1","name":"Demo","folder_id":"folder-1","browser_type":"mimic","os_type":"windows","core_version":137,"notes":"","created_by":"me@example.com","in_use_by":"","last_launched_by":"","is_local":false}],"total_count":1}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/profile/metas":
			fmt.Fprintf(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[%s]}}`, verifiedProfileMetaJSON("profile-1", "Demo", "folder-1"))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/profile/stop/p/profile-1":
			fmt.Fprint(w, `{"status":{"http_code":200,"message":"Profile stopped successfully"},"data":null}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/profile/status/p/profile-1":
			statusCalls++
			status := "stopping"
			if statusCalls > 1 {
				status = "stopped"
			}
			fmt.Fprintf(w, `{"status":{"http_code":200,"message":""},"data":{"profile_id":"profile-1","name":"Demo","status":%q,"browser_type":"mimic","core_version":137,"folder_id":"folder-1","workspace_id":"workspace-1","message":"","timestamp":1745100000000}}`, status)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})

	client, err := mlx.New(
		mlx.WithToken("test-token"),
		mlx.WithHTTPClient(httpClient),
		mlx.WithBaseURL(server.URL),
		mlx.WithLauncherURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	result, err := client.Workflows.StopProfileByName(context.Background(), "Demo", mlx.StopProfileByNameOptions{
		WaitForStopped: true,
		PollOptions:    mlx.PollOptions{InitialInterval: time.Millisecond, MaxInterval: time.Millisecond, Timeout: time.Second},
	})
	if err != nil {
		t.Fatalf("Workflows.StopProfileByName returned error: %v", err)
	}
	if statusCalls < 2 {
		t.Fatalf("expected status polling, got %d calls", statusCalls)
	}
	if result.RuntimeStatus == nil || result.RuntimeStatus.Data.Status != "stopped" {
		t.Fatalf("unexpected runtime status: %#v", result.RuntimeStatus)
	}
}

func TestWorkflowImportProfileAndVerify(t *testing.T) {
	statusCalls := 0
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/profile/import":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("failed to read import body: %v", err)
			}
			text := string(body)
			if !strings.Contains(text, `"import_path":"C:\\exports\\demo.zip"`) {
				t.Fatalf("unexpected import body: %s", text)
			}
			fmt.Fprint(w, `{"status":{"http_code":200,"message":"import started"},"data":{"import_id":"import-1","import_path":"C:\\exports\\demo.zip","status":"running","message":"","timestamp":456}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/profile/imports/import-1/status":
			statusCalls++
			if statusCalls == 1 {
				fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"import_id":"import-1","import_path":"C:\\exports\\demo.zip","status":"running","message":"","timestamp":456}}`)
				return
			}
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"import_id":"import-1","import_path":"C:\\exports\\demo.zip","new_profile_id":"profile-2","status":"done","message":"","timestamp":457}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/profile/metas":
			fmt.Fprintf(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[%s]}}`, verifiedProfileMetaJSON("profile-2", "Imported Demo", "folder-2"))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})

	client, err := mlx.New(
		mlx.WithToken("test-token"),
		mlx.WithHTTPClient(httpClient),
		mlx.WithLauncherURL(server.URL),
		mlx.WithBaseURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	result, err := client.Workflows.ImportProfileAndVerify(context.Background(), &mlx.ImportProfileRequest{
		ImportPath: `C:\exports\demo.zip`,
		IsLocal:    true,
	}, mlx.ImportProfileWorkflowOptions{
		PollOptions: mlx.PollOptions{InitialInterval: time.Millisecond, MaxInterval: time.Millisecond, Timeout: time.Second},
	})
	if err != nil {
		t.Fatalf("Workflows.ImportProfileAndVerify returned error: %v", err)
	}
	if statusCalls < 2 {
		t.Fatalf("expected import polling, got %d calls", statusCalls)
	}
	if result.ImportStatus.Data.NewProfileID != "profile-2" || result.ProfileMeta.ID != "profile-2" {
		t.Fatalf("unexpected import result: %#v", result)
	}
}

func TestWorkflowEnableExtensionForProfileByName(t *testing.T) {
	objectUsageCalls := 0
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/profile/search":
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[{"id":"profile-1","name":"Demo","folder_id":"folder-1","browser_type":"mimic","os_type":"windows","core_version":137,"notes":"","created_by":"me@example.com","in_use_by":"","last_launched_by":"","is_local":true}],"total_count":1}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/profile/metas":
			fmt.Fprintf(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[%s]}}`, verifiedProfileMetaJSON("profile-1", "Demo", "folder-1"))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/resources/ext-1/enable_for_profiles":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("failed to read enable body: %v", err)
			}
			if !strings.Contains(string(body), `"profile_ids":["profile-1"]`) {
				t.Fatalf("unexpected enable body: %s", string(body))
			}
			fmt.Fprint(w, `{"status":{"http_code":200,"message":"enabled"},"data":"ok"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/resources/object_profile_usages":
			if got := r.URL.Query().Get("object_id"); got != "ext-1" {
				t.Fatalf("unexpected object id: %s", got)
			}
			objectUsageCalls++
			if objectUsageCalls == 1 {
				fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":[]}`)
				return
			}
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":[{"id":"profile-1","object_id":"ext-1"}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/resources/profile_object_usages":
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":[{"id":"ext-1","name":"Demo","type":"extension","meta_info":{},"is_enabled":true}]}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})

	client, err := mlx.New(
		mlx.WithToken("test-token"),
		mlx.WithHTTPClient(httpClient),
		mlx.WithBaseURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	result, err := client.Workflows.EnableExtensionForProfileByName(context.Background(), "Demo", "ext-1", mlx.EnableExtensionForProfileByNameOptions{
		PollOptions:             mlx.PollOptions{InitialInterval: time.Millisecond, MaxInterval: time.Millisecond, Timeout: time.Second},
		RequireProfileUsageRead: true,
	})
	if err != nil {
		t.Fatalf("Workflows.EnableExtensionForProfileByName returned error: %v", err)
	}
	if objectUsageCalls < 2 {
		t.Fatalf("expected extension usage polling, got %d calls", objectUsageCalls)
	}
	if result.Profile.ID != "profile-1" || result.EnableResponse.Data != "ok" {
		t.Fatalf("unexpected enable result: %#v", result)
	}
	if result.ProfileUsages == nil || len(result.ProfileUsages.Data) != 1 {
		t.Fatalf("expected profile usage confirmation, got %#v", result.ProfileUsages)
	}
}

func TestWorkflowCreateLocalProfile(t *testing.T) {
	var capturedBody string
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/profile/create":
			capturedBody = readRequestBody(t, r)
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"ids":["profile-local-1"]}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/profile/metas":
			fmt.Fprintf(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[%s]}}`, verifiedProfileMetaJSON("profile-local-1", "LocalDemo", "folder-1"))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})

	client, err := mlx.New(
		mlx.WithToken("test-token"),
		mlx.WithHTTPClient(httpClient),
		mlx.WithBaseURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	result, err := client.Workflows.CreateLocalProfile(context.Background(), &mlx.CreateProfileRequest{
		Name:        "LocalDemo",
		BrowserType: "mimic",
		FolderID:    "folder-1",
		OSType:      "windows",
	}, mlx.CreateProfilesAndVerifyOptions{})
	if err != nil {
		t.Fatalf("Workflows.CreateLocalProfile returned error: %v", err)
	}
	if result.CreateResponse.Data.IDs[0] != "profile-local-1" {
		t.Fatalf("unexpected profile id: %s", result.CreateResponse.Data.IDs[0])
	}
	if !contains(capturedBody, `"is_local":true`) {
		t.Fatalf("expected local profile request to contain is_local=true, got: %s", capturedBody)
	}
}

func TestWorkflowCreateCloudProfile(t *testing.T) {
	var capturedBody string
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/profile/create":
			capturedBody = readRequestBody(t, r)
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"ids":["profile-cloud-1"]}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/profile/metas":
			fmt.Fprintf(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[%s]}}`, verifiedProfileMetaJSON("profile-cloud-1", "CloudDemo", "folder-1"))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})

	client, err := mlx.New(
		mlx.WithToken("test-token"),
		mlx.WithHTTPClient(httpClient),
		mlx.WithBaseURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	result, err := client.Workflows.CreateCloudProfile(context.Background(), &mlx.CreateProfileRequest{
		Name:        "CloudDemo",
		BrowserType: "mimic",
		FolderID:    "folder-1",
		OSType:      "windows",
	}, mlx.CreateProfilesAndVerifyOptions{})
	if err != nil {
		t.Fatalf("Workflows.CreateCloudProfile returned error: %v", err)
	}
	if result.CreateResponse.Data.IDs[0] != "profile-cloud-1" {
		t.Fatalf("unexpected profile id: %s", result.CreateResponse.Data.IDs[0])
	}
	if !contains(capturedBody, `"is_local":false`) {
		t.Fatalf("expected cloud profile request to contain is_local=false, got: %s", capturedBody)
	}
}
