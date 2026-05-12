package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	mlx "mlx-go-sdk"
)

const testFoldersListResponse = `{"status":{"http_code":200,"message":""},"data":{"folders":[{"folder_id":"folder-1","name":"Default folder","comment":"","profiles_count":1,"created_at":"2026-04-20T00:00:00Z"}]}}`

func TestNewClientFromConfigUsesTokenAndEndpointOverrides(t *testing.T) {
	t.Setenv(mlx.EnvToken, "test-token")

	var gotPath string
	var gotAuth string
	var gotUserAgent string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotUserAgent = r.Header.Get("User-Agent")
		fmt.Fprint(w, testFoldersListResponse)
	}))
	defer server.Close()

	cfg := DefaultConfig()
	cfg.Endpoints.BaseURL = server.URL
	cfg.Transport.UserAgent = "mlx-go-sdk-cli/test"
	cfg.Retry.Enabled = false

	client, token, err := NewClientFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewClientFromConfig returned error: %v", err)
	}
	if token != "test-token" {
		t.Fatalf("unexpected token: %q", token)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}

	resp, _, err := client.Folders.List(context.Background())
	if err != nil {
		t.Fatalf("Folders.List returned error: %v", err)
	}
	if gotPath != "/workspace/folders" {
		t.Fatalf("unexpected request path: %q", gotPath)
	}
	if gotAuth != "Bearer test-token" {
		t.Fatalf("unexpected authorization header: %q", gotAuth)
	}
	if gotUserAgent != "mlx-go-sdk-cli/test" {
		t.Fatalf("unexpected user agent: %q", gotUserAgent)
	}
	if len(resp.Data.Folders) != 1 {
		t.Fatalf("expected one folder, got %d", len(resp.Data.Folders))
	}
	if resp.Data.Folders[0].FolderID != "folder-1" {
		t.Fatalf("unexpected folder id: %q", resp.Data.Folders[0].FolderID)
	}
}

func TestNewClientFromConfigAppliesTimeout(t *testing.T) {
	t.Setenv(mlx.EnvToken, "test-token")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(60 * time.Millisecond)
		fmt.Fprint(w, testFoldersListResponse)
	}))
	defer server.Close()

	cfg := DefaultConfig()
	cfg.Endpoints.BaseURL = server.URL
	cfg.Transport.Timeout = Duration(15 * time.Millisecond)
	cfg.Retry.Enabled = false

	client, _, err := NewClientFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewClientFromConfig returned error: %v", err)
	}

	_, _, err = client.Folders.List(context.Background())
	if err == nil {
		t.Fatal("expected timeout error")
	}

	var transportErr *mlx.TransportError
	if !errors.As(err, &transportErr) {
		t.Fatalf("expected TransportError, got %T: %v", err, err)
	}
	if !transportErr.Timeout() && mlx.ClassifyError(err) != mlx.ErrorClassTimeout {
		t.Fatalf("expected timeout classification, got %q (%v)", mlx.ClassifyError(err), err)
	}
}

func TestLoadRuntimeLoadsConfigAndEnvOverrides(t *testing.T) {
	t.Setenv(mlx.EnvToken, "test-token")
	t.Setenv(mlx.EnvBaseURL, "")
	t.Setenv(EnvOutputFormat, "")
	t.Setenv(EnvTimeout, "")

	var gotAuth string
	var gotUserAgent string
	var gotPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotUserAgent = r.Header.Get("User-Agent")
		fmt.Fprint(w, testFoldersListResponse)
	}))
	defer server.Close()

	configPath := writeRuntimeConfigFile(t, `{
  "version": "1",
  "endpoints": {
    "base_url": "https://config.example.test"
  },
  "transport": {
    "timeout": "45s",
    "user_agent": "config-agent/1.0"
  },
  "output": {
    "format": "table"
  }
}`)

	t.Setenv(mlx.EnvBaseURL, server.URL)
	t.Setenv(EnvOutputFormat, "json")
	t.Setenv(EnvTimeout, "25ms")

	rt, err := LoadRuntime(configPath)
	if err != nil {
		t.Fatalf("LoadRuntime returned error: %v", err)
	}
	if rt == nil {
		t.Fatal("expected non-nil runtime")
	}
	if rt.Client == nil {
		t.Fatal("expected non-nil runtime client")
	}
	if rt.Token != "test-token" {
		t.Fatalf("unexpected token: %q", rt.Token)
	}
	if rt.ConfigPath != filepath.Clean(configPath) {
		t.Fatalf("unexpected config path: %q", rt.ConfigPath)
	}
	if rt.Config.Output.Format != "json" {
		t.Fatalf("expected env output override, got %q", rt.Config.Output.Format)
	}
	if rt.Config.Transport.Timeout.Duration() != 25*time.Millisecond {
		t.Fatalf("expected env timeout override, got %s", rt.Config.Transport.Timeout.Duration())
	}
	if rt.Config.Transport.UserAgent != "config-agent/1.0" {
		t.Fatalf("unexpected user agent in config: %q", rt.Config.Transport.UserAgent)
	}

	resp, _, err := rt.Client.Folders.List(context.Background())
	if err != nil {
		t.Fatalf("Folders.List returned error: %v", err)
	}
	if gotPath != "/workspace/folders" {
		t.Fatalf("unexpected request path: %q", gotPath)
	}
	if gotAuth != "Bearer test-token" {
		t.Fatalf("unexpected authorization header: %q", gotAuth)
	}
	if gotUserAgent != "config-agent/1.0" {
		t.Fatalf("unexpected user agent header: %q", gotUserAgent)
	}
	if len(resp.Data.Folders) != 1 {
		t.Fatalf("expected one folder, got %d", len(resp.Data.Folders))
	}
}

func TestLoadRuntimeRequiresToken(t *testing.T) {
	t.Setenv(mlx.EnvToken, "")

	configPath := writeRuntimeConfigFile(t, `{
  "version": "1",
  "transport": {
    "user_agent": "config-agent/1.0"
  }
}`)

	_, err := LoadRuntime(configPath)
	if err == nil {
		t.Fatal("expected missing token error")
	}
	if !strings.Contains(err.Error(), mlx.EnvToken) {
		t.Fatalf("expected error to mention %s, got %v", mlx.EnvToken, err)
	}
}

func writeRuntimeConfigFile(t *testing.T, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	return path
}
