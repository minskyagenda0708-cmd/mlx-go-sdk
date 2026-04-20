package mlx

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mlx-go-sdk/internal/testutil"
)

func TestArchiveManagerOrganizePreservesZipName(t *testing.T) {
	rootDir := t.TempDir()
	sourceDir := t.TempDir()
	sourcePath := filepath.Join(sourceDir, "export-1.zip")
	if err := os.WriteFile(sourcePath, []byte("zip-bytes"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	manager := &ArchiveManagerOp{}
	result, err := manager.OrganizeExport(sourcePath, rootDir, "John Doe")
	if err != nil {
		t.Fatalf("OrganizeExport returned error: %v", err)
	}

	if filepath.Base(result.ArchivePath) != "export-1.zip" {
		t.Fatalf("zip file name changed: %s", result.ArchivePath)
	}
	if _, err := os.Stat(sourcePath); !os.IsNotExist(err) {
		t.Fatalf("expected source zip to be moved away, stat err=%v", err)
	}
	if _, err := os.Stat(result.ArchivePath); err != nil {
		t.Fatalf("expected archive zip to exist, got %v", err)
	}
}

func TestArchiveManagerOrganizeUsesSeparateFolders(t *testing.T) {
	rootDir := t.TempDir()
	sourceDir := t.TempDir()

	sourceOne := filepath.Join(sourceDir, "first.zip")
	sourceTwo := filepath.Join(sourceDir, "second.zip")
	if err := os.WriteFile(sourceOne, []byte("one"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(sourceTwo, []byte("two"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	manager := &ArchiveManagerOp{}
	first, err := manager.OrganizeExport(sourceOne, rootDir, "persona-one")
	if err != nil {
		t.Fatalf("OrganizeExport first returned error: %v", err)
	}
	second, err := manager.OrganizeExport(sourceTwo, rootDir, "persona-two")
	if err != nil {
		t.Fatalf("OrganizeExport second returned error: %v", err)
	}

	if first.ArchiveDir == second.ArchiveDir {
		t.Fatalf("expected separate archive directories, got %s", first.ArchiveDir)
	}
	if filepath.Base(first.ArchivePath) != "first.zip" || filepath.Base(second.ArchivePath) != "second.zip" {
		t.Fatalf("zip names must remain unchanged")
	}
}

func TestArchiveManagerOrganizeRejectsMissingZip(t *testing.T) {
	manager := &ArchiveManagerOp{}
	_, err := manager.OrganizeExport(filepath.Join(t.TempDir(), "missing.zip"), t.TempDir(), "demo")
	if err == nil {
		t.Fatal("expected missing zip error")
	}
}

func TestArchiveManagerOrganizeRejectsDirectory(t *testing.T) {
	rootDir := t.TempDir()
	sourceDir := t.TempDir()

	manager := &ArchiveManagerOp{}
	_, err := manager.OrganizeExport(sourceDir, rootDir, "demo")
	if err == nil {
		t.Fatal("expected directory input error")
	}
}

func TestArchiveManagerOrganizeRejectsNonZip(t *testing.T) {
	rootDir := t.TempDir()
	sourceDir := t.TempDir()
	sourcePath := filepath.Join(sourceDir, "export-1.txt")
	if err := os.WriteFile(sourcePath, []byte("not-zip"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	manager := &ArchiveManagerOp{}
	_, err := manager.OrganizeExport(sourcePath, rootDir, "demo")
	if err == nil {
		t.Fatal("expected non-zip input error")
	}
}

func TestArchiveManagerOrganizeRejectsExistingDestination(t *testing.T) {
	rootDir := t.TempDir()
	sourceDir := t.TempDir()
	sourcePath := filepath.Join(sourceDir, "export-1.zip")
	if err := os.WriteFile(sourcePath, []byte("zip-bytes"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	existingDir := filepath.Join(rootDir, "John Doe")
	if err := os.MkdirAll(existingDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(existingDir, "export-1.zip"), []byte("existing"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	manager := &ArchiveManagerOp{}
	_, err := manager.OrganizeExport(sourcePath, rootDir, "John Doe")
	if err == nil {
		t.Fatal("expected archive exists error")
	}
}

func TestArchiveManagerExportProfileToFolder(t *testing.T) {
	workspace := t.TempDir()
	exportSource := filepath.Join(workspace, "export-1.zip")
	if err := os.WriteFile(exportSource, []byte("zip"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && r.URL.Path == "/api/v1/profile/profile-1/export":
			fmt.Fprintf(w, `{"status":{"http_code":200,"message":"Export in progress"},"data":{"export_id":"export-1","export_path":"%s","profile_id":"profile-1","status":"running","message":"","timestamp":1713552000000}}`, filepath.ToSlash(exportSource))
		case r.Method == "GET" && r.URL.Path == "/api/v1/profile/exports/export-1/status":
			fmt.Fprintf(w, `{"status":{"http_code":200,"message":""},"data":{"export_id":"export-1","export_path":"%s","profile_id":"profile-1","status":"done","message":"","timestamp":1713552000000}}`, filepath.ToSlash(exportSource))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})

	client, err := New(
		WithToken("test-token"),
		WithHTTPClient(httpClient),
		WithLauncherURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	result, err := client.Archives.ExportProfileToFolder(context.Background(), "profile-1", ExportProfileToFolderOptions{
		RootDir:      filepath.Join(workspace, "archives"),
		ProfileName:  "Jane / Doe",
		WaitTimeout:  5 * time.Second,
		PollInterval: 10 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("ExportProfileToFolder returned error: %v", err)
	}

	if result.ExportJob.Data.ExportID != "export-1" {
		t.Fatalf("unexpected export id: %s", result.ExportJob.Data.ExportID)
	}
	if filepath.Base(result.Archive.ArchivePath) != "export-1.zip" {
		t.Fatalf("zip file name changed: %s", result.Archive.ArchivePath)
	}
	if filepath.Base(result.Archive.ArchiveDir) == "export-1.zip" {
		t.Fatalf("archive directory must not be the zip file name")
	}
	if _, err := os.Stat(result.Archive.ArchivePath); err != nil {
		t.Fatalf("expected archive zip to exist, got %v", err)
	}
}

func TestArchiveManagerExportProfileToFolderNormalizesExtensionlessStatusPath(t *testing.T) {
	workspace := t.TempDir()
	exportSource := filepath.Join(workspace, "export-2.zip")
	if err := os.WriteFile(exportSource, []byte("zip"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && r.URL.Path == "/api/v1/profile/profile-2/export":
			fmt.Fprintf(w, `{"status":{"http_code":200,"message":"Export in progress"},"data":{"export_id":"export-2","export_path":"%s","profile_id":"profile-2","status":"running","message":"","timestamp":1713552000000}}`, filepath.ToSlash(exportSource))
		case r.Method == "GET" && r.URL.Path == "/api/v1/profile/exports/export-2/status":
			fmt.Fprintf(w, `{"status":{"http_code":200,"message":""},"data":{"export_id":"export-2","export_path":"%s","profile_id":"profile-2","status":"done","message":"","timestamp":1713552000000}}`, strings.TrimSuffix(filepath.ToSlash(exportSource), ".zip"))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})

	client, err := New(
		WithToken("test-token"),
		WithHTTPClient(httpClient),
		WithLauncherURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	result, err := client.Archives.ExportProfileToFolder(context.Background(), "profile-2", ExportProfileToFolderOptions{
		RootDir:      filepath.Join(workspace, "archives"),
		ProfileName:  "Jane Doe",
		WaitTimeout:  5 * time.Second,
		PollInterval: 10 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("ExportProfileToFolder returned error: %v", err)
	}
	if filepath.Ext(result.Archive.ArchivePath) != ".zip" {
		t.Fatalf("expected normalized archive path to keep .zip, got %s", result.Archive.ArchivePath)
	}
}

func TestArchiveManagerExportProfileToFolderTimesOut(t *testing.T) {
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && r.URL.Path == "/api/v1/profile/profile-timeout/export":
			fmt.Fprint(w, `{"status":{"http_code":200,"message":"Export in progress"},"data":{"export_id":"export-timeout","export_path":"C:/exports/export-timeout.zip","profile_id":"profile-timeout","status":"running","message":"","timestamp":1713552000000}}`)
		case r.Method == "GET" && r.URL.Path == "/api/v1/profile/exports/export-timeout/status":
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"export_id":"export-timeout","export_path":"C:/exports/export-timeout","profile_id":"profile-timeout","status":"running","message":"still-running","timestamp":1713552000000}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})

	client, err := New(
		WithToken("test-token"),
		WithHTTPClient(httpClient),
		WithLauncherURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	_, err = client.Archives.ExportProfileToFolder(context.Background(), "profile-timeout", ExportProfileToFolderOptions{
		RootDir:      t.TempDir(),
		ProfileName:  "Timeout Demo",
		WaitTimeout:  25 * time.Millisecond,
		PollInterval: 5 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestExportJobStateArchivePathNormalizesZip(t *testing.T) {
	tests := []struct {
		name string
		job  ExportJobState
		want string
	}{
		{
			name: "already zip",
			job:  ExportJobState{ExportPath: `C:\exports\job-1.zip`},
			want: `C:\exports\job-1.zip`,
		},
		{
			name: "missing extension",
			job:  ExportJobState{ExportPath: `C:\exports\job-2`},
			want: `C:\exports\job-2.zip`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.job.ArchivePath(); got != tt.want {
				t.Fatalf("ArchivePath() = %q, want %q", got, tt.want)
			}
		})
	}
}
