package mlx

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/minskyagenda0708-cmd/mlx-go-sdk/internal/testutil"
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

func TestTransfersExportStatuses(t *testing.T) {
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/profile/exports/statuses" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"statuses":[{"export_id":"export-1","export_path":"C:\\\\exports\\\\export-1","profile_id":"profile-1","status":"done","message":"","timestamp":123}]}}`)
	})

	client, err := New(
		WithToken("test-token"),
		WithHTTPClient(httpClient),
		WithLauncherURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, _, err := client.Transfers.ExportStatuses(context.Background())
	if err != nil {
		t.Fatalf("Transfers.ExportStatuses returned error: %v", err)
	}
	if len(resp.Data.Statuses) != 1 || resp.Data.Statuses[0].Timestamp != 123 {
		t.Fatalf("unexpected export statuses payload: %#v", resp.Data.Statuses)
	}
}

func TestTransfersImportStatuses(t *testing.T) {
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/profile/imports/statuses" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"statuses":[{"export_id":"export-1","import_id":"import-1","import_path":"C:\\\\exports\\\\export-1.zip","extracted_path":"","new_profile_id":"profile-2","status":"done","message":"","timestamp":456}]}}`)
	})

	client, err := New(
		WithToken("test-token"),
		WithHTTPClient(httpClient),
		WithLauncherURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, _, err := client.Transfers.ImportStatuses(context.Background())
	if err != nil {
		t.Fatalf("Transfers.ImportStatuses returned error: %v", err)
	}
	if len(resp.Data.Statuses) != 1 || resp.Data.Statuses[0].ImportID != "import-1" {
		t.Fatalf("unexpected import statuses payload: %#v", resp.Data.Statuses)
	}
}

func TestTransfersWaitForExportDoneRetries(t *testing.T) {
	var calls atomic.Int32
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/profile/exports/export-1/status" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if calls.Add(1) < 3 {
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"export_id":"export-1","export_path":"C:\\\\exports\\\\export-1","profile_id":"profile-1","status":"running","message":"","timestamp":123}}`)
			return
		}
		fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"export_id":"export-1","export_path":"C:\\\\exports\\\\export-1","profile_id":"profile-1","status":"done","message":"","timestamp":124}}`)
	})

	client, err := New(
		WithToken("test-token"),
		WithHTTPClient(httpClient),
		WithLauncherURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, _, err := client.Transfers.WaitForExportDone(context.Background(), "export-1", PollOptions{InitialInterval: time.Millisecond, MaxInterval: 2 * time.Millisecond, Timeout: time.Second, Multiplier: 2})
	if err != nil {
		t.Fatalf("Transfers.WaitForExportDone returned error: %v", err)
	}
	if resp.Data.Status != "done" {
		t.Fatalf("unexpected export status: %s", resp.Data.Status)
	}
}

func TestTransfersWaitForImportDoneRetries(t *testing.T) {
	var calls atomic.Int32
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/profile/imports/import-1/status" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if calls.Add(1) < 3 {
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"import_id":"import-1","import_path":"C:\\\\exports\\\\export-1.zip","status":"running","message":"","timestamp":456}}`)
			return
		}
		fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"import_id":"import-1","import_path":"C:\\\\exports\\\\export-1.zip","new_profile_id":"profile-2","status":"done","message":"","timestamp":457}}`)
	})

	client, err := New(
		WithToken("test-token"),
		WithHTTPClient(httpClient),
		WithLauncherURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, _, err := client.Transfers.WaitForImportDone(context.Background(), "import-1", PollOptions{InitialInterval: time.Millisecond, MaxInterval: 2 * time.Millisecond, Timeout: time.Second, Multiplier: 2})
	if err != nil {
		t.Fatalf("Transfers.WaitForImportDone returned error: %v", err)
	}
	if resp.Data.Status != "done" {
		t.Fatalf("unexpected import status: %s", resp.Data.Status)
	}
}
