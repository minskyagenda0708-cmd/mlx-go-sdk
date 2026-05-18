package integration_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	mlx "github.com/minskyagenda0708-cmd/mlx-go-sdk"
	"github.com/minskyagenda0708-cmd/mlx-go-sdk/internal/testutil"
)

func TestWorkflowBatchStartProfilesByName(t *testing.T) {
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/profile/search":
			if extractBodyField(t, r, `"search_text":"`) == "Alpha" {
				fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[{"id":"profile-1","name":"Alpha","folder_id":"folder-1","browser_type":"mimic","os_type":"windows","core_version":137}],"total_count":1}}`)
				return
			}
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[],"total_count":0}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/profile/metas":
			fmt.Fprintf(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[%s]}}`, verifiedProfileMetaJSON("profile-1", "Alpha", "folder-1"))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/profile/f/folder-1/p/profile-1/start":
			fmt.Fprint(w, `{"status":{"http_code":200,"message":"started"},"data":{"browser_type":"mimic","core_version":137,"id":"profile-1","is_quick":false,"port":"55513"}}`)
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

	result, err := client.Workflows.StartProfilesByName(context.Background(), []string{"Alpha", "Missing"}, mlx.StartProfileByNameOptions{})
	if err == nil {
		t.Fatal("expected aggregated batch error")
	}
	var batchErr *mlx.BatchProfileOperationError
	if !errors.As(err, &batchErr) {
		t.Fatalf("expected BatchProfileOperationError, got %v", err)
	}
	if result.Summary.Total != 2 || result.Summary.Succeeded != 1 || result.Summary.Failed != 1 {
		t.Fatalf("unexpected summary: %#v", result.Summary)
	}
	if result.Items[0].Result == nil || result.Items[0].Result.Profile.ID != "profile-1" {
		t.Fatalf("unexpected success item: %#v", result.Items[0])
	}
	if result.Items[1].Err == nil || !errors.Is(result.Items[1].Err, mlx.ErrProfileNotFound) {
		t.Fatalf("unexpected failure item: %#v", result.Items[1])
	}
}

func TestWorkflowBatchStopProfilesByName(t *testing.T) {
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/profile/search":
			search := extractBodyField(t, r, `"search_text":"`)
			fmt.Fprintf(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[{"id":%q,"name":%q,"folder_id":"folder-1","browser_type":"mimic","os_type":"windows","core_version":137}],"total_count":1}}`, "profile-"+search, search)
		case r.Method == http.MethodPost && r.URL.Path == "/profile/metas":
			body := readRequestBody(t, r)
			if contains(body, "profile-Alpha") {
				fmt.Fprintf(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[%s]}}`, verifiedProfileMetaJSON("profile-Alpha", "Alpha", "folder-1"))
				return
			}
			fmt.Fprintf(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[%s]}}`, verifiedProfileMetaJSON("profile-Beta", "Beta", "folder-1"))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/profile/stop/p/profile-Alpha":
			fmt.Fprint(w, `{"status":{"http_code":200,"message":"stopped"},"data":null}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/profile/stop/p/profile-Beta":
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

	result, err := client.Workflows.StopProfilesByName(context.Background(), []string{"Alpha", "Beta"}, mlx.StopProfileByNameOptions{IgnoreAlreadyStopped: true})
	if err != nil {
		t.Fatalf("expected stop batch to succeed, got %v", err)
	}
	if result.Summary.Succeeded != 2 || result.Summary.Failed != 0 {
		t.Fatalf("unexpected summary: %#v", result.Summary)
	}
}

func TestWorkflowBatchExportProfilesByNameToFolder(t *testing.T) {
	workspace := t.TempDir()
	exportOne := filepath.Join(workspace, "alpha.zip")
	exportTwo := filepath.Join(workspace, "beta.zip")
	if err := os.WriteFile(exportOne, []byte("zip"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(exportTwo, []byte("zip"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/profile/search":
			search := extractBodyField(t, r, `"search_text":"`)
			fmt.Fprintf(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[{"id":%q,"name":%q,"folder_id":"folder-1","browser_type":"mimic","os_type":"windows","core_version":137}],"total_count":1}}`, "profile-"+search, search)
		case r.Method == http.MethodPost && r.URL.Path == "/profile/metas":
			body := readRequestBody(t, r)
			if contains(body, "profile-Alpha") {
				fmt.Fprintf(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[%s]}}`, verifiedProfileMetaJSON("profile-Alpha", "Alpha", "folder-1"))
				return
			}
			fmt.Fprintf(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[%s]}}`, verifiedProfileMetaJSON("profile-Beta", "Beta", "folder-1"))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/profile/profile-Alpha/export":
			fmt.Fprintf(w, `{"status":{"http_code":200,"message":"Export in progress"},"data":{"export_id":"export-alpha","export_path":"%s","profile_id":"profile-Alpha","status":"running","message":"","timestamp":1713552000000}}`, filepath.ToSlash(exportOne))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/profile/profile-Beta/export":
			fmt.Fprintf(w, `{"status":{"http_code":200,"message":"Export in progress"},"data":{"export_id":"export-beta","export_path":"%s","profile_id":"profile-Beta","status":"running","message":"","timestamp":1713552000000}}`, filepath.ToSlash(exportTwo))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/profile/exports/export-alpha/status":
			fmt.Fprintf(w, `{"status":{"http_code":200,"message":""},"data":{"export_id":"export-alpha","export_path":"%s","profile_id":"profile-Alpha","status":"done","message":"","timestamp":1713552000000}}`, filepath.ToSlash(exportOne))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/profile/exports/export-beta/status":
			fmt.Fprintf(w, `{"status":{"http_code":200,"message":""},"data":{"export_id":"export-beta","export_path":"%s","profile_id":"profile-Beta","status":"done","message":"","timestamp":1713552000000}}`, filepath.ToSlash(exportTwo))
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

	result, err := client.Workflows.ExportProfilesByNameToFolder(context.Background(), []string{"Alpha", "Beta"}, mlx.ExportProfileByNameToFolderOptions{
		ExportOptions: mlx.ExportProfileToFolderOptions{
			RootDir:      filepath.Join(workspace, "archives"),
			PollInterval: time.Millisecond,
			WaitTimeout:  time.Second,
		},
	})
	if err != nil {
		t.Fatalf("expected export batch to succeed, got %v", err)
	}
	if result.Summary.Succeeded != 2 {
		t.Fatalf("unexpected summary: %#v", result.Summary)
	}
	for _, item := range result.Items {
		if item.Result == nil || item.Result.Export == nil || item.Result.Export.Archive == nil {
			t.Fatalf("unexpected export item: %#v", item)
		}
	}
}

func TestWorkflowBatchEnableExtensionForProfilesByName(t *testing.T) {
	usageCalls := map[string]int{}
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/profile/search":
			search := extractBodyField(t, r, `"search_text":"`)
			fmt.Fprintf(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[{"id":%q,"name":%q,"folder_id":"folder-1","browser_type":"mimic","os_type":"windows","core_version":137}],"total_count":1}}`, "profile-"+search, search)
		case r.Method == http.MethodPost && r.URL.Path == "/profile/metas":
			body := readRequestBody(t, r)
			if contains(body, "profile-Alpha") {
				fmt.Fprintf(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[%s]}}`, verifiedProfileMetaJSON("profile-Alpha", "Alpha", "folder-1"))
				return
			}
			fmt.Fprintf(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[%s]}}`, verifiedProfileMetaJSON("profile-Beta", "Beta", "folder-1"))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/resources/ext-1/enable_for_profiles":
			fmt.Fprint(w, `{"status":{"http_code":200,"message":"enabled"},"data":"ok"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/resources/object_profile_usages":
			usageCalls[r.URL.RawQuery]++
			if usageCalls[r.URL.RawQuery] == 1 {
				fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":[]}`)
				return
			}
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":[{"id":"profile-Alpha","object_id":"ext-1"},{"id":"profile-Beta","object_id":"ext-1"}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/resources/profile_object_usages":
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":[]}`)
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

	result, err := client.Workflows.EnableExtensionForProfilesByName(
		context.Background(),
		[]string{"Alpha", "Beta"},
		"ext-1",
		mlx.EnableExtensionForProfileByNameOptions{
			PollOptions: mlx.PollOptions{InitialInterval: time.Millisecond, MaxInterval: time.Millisecond, Timeout: time.Second},
		},
	)
	if err != nil {
		t.Fatalf("expected extension batch to succeed, got %v", err)
	}
	if result.Summary.Succeeded != 2 {
		t.Fatalf("unexpected summary: %#v", result.Summary)
	}
}
