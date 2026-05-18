package mlx

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/minskyagenda0708-cmd/mlx-go-sdk/internal/testutil"
)

func TestLauncherStartQuick(t *testing.T) {
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v3/profile/quick" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if !contains(string(body), "\"browser_type\":\"mimic\"") {
			t.Fatalf("expected body to contain browser_type=mimic, got: %s", string(body))
		}
		if !contains(string(body), "\"is_headless\":true") {
			t.Fatalf("expected body to contain is_headless=true, got: %s", string(body))
		}
		fmt.Fprint(w, `{"status":{"http_code":200,"message":"Quick profile started successfully"},"data":{"browser_type":"mimic","core_version":132,"id":"quick-1","is_quick":true,"port":"55579"}}`)
	})

	client, err := New(
		WithToken("test-token"),
		WithHTTPClient(httpClient),
		WithLauncherURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, _, err := client.Launcher.StartQuick(context.Background(), &StartQuickProfileRequest{
		BrowserType: "mimic",
		OSType:      "windows",
		Headless:    true,
	})
	if err != nil {
		t.Fatalf("Launcher.StartQuick returned error: %v", err)
	}
	if resp.Data.ID != "quick-1" {
		t.Fatalf("unexpected id: %s", resp.Data.ID)
	}
	if !resp.Data.IsQuick {
		t.Fatalf("expected is_quick=true")
	}
	if resp.Data.Port != "55579" {
		t.Fatalf("unexpected port: %s", resp.Data.Port)
	}
}

func TestLauncherSaveQuick(t *testing.T) {
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/profile/quick/save" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if !contains(string(body), "\"profile_id\":\"quick-1\"") {
			t.Fatalf("expected body to contain quick-1, got: %s", string(body))
		}
		fmt.Fprint(w, `{"status":{"http_code":200,"message":"Quick profile saved"},"data":{}}`)
	})

	client, err := New(
		WithToken("test-token"),
		WithHTTPClient(httpClient),
		WithLauncherURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	_, _, err = client.Launcher.SaveQuick(context.Background(), &SaveQuickProfileRequest{
		Data: []SaveQuickProfileItem{{ProfileID: "quick-1"}},
	})
	if err != nil {
		t.Fatalf("Launcher.SaveQuick returned error: %v", err)
	}
}

func TestLauncherValidateProxy(t *testing.T) {
	var capturedBody string
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/proxy/validate" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		capturedBody = string(body)
		fmt.Fprint(w, `{"status":{"http_code":200,"message":"Proxy validated"},"data":{"accuracy":200,"altitude":100,"country_code":"US","ip":"194.71.130.189","latitude":37.7749,"longitude":-122.4194,"timezone":"America/Los_Angeles"}}`)
	})

	client, err := New(
		WithToken("test-token"),
		WithHTTPClient(httpClient),
		WithLauncherURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, _, err := client.Launcher.ValidateProxy(context.Background(), &ValidateProxyRequest{
		Type:     "socks5",
		Host:     "194.71.130.189",
		Port:     24745,
		Username: "modeler_aKZa4q",
		Password: "eS4prGYBpHMA",
	})
	if err != nil {
		t.Fatalf("Launcher.ValidateProxy returned error: %v", err)
	}
	if resp.Data.IP != "194.71.130.189" {
		t.Fatalf("unexpected ip: %s", resp.Data.IP)
	}
	if resp.Data.CountryCode != "US" {
		t.Fatalf("unexpected country code: %s", resp.Data.CountryCode)
	}
	if !contains(capturedBody, "\"type\":\"socks5\"") {
		t.Fatalf("expected body to contain socks5, got: %s", capturedBody)
	}
	if !contains(capturedBody, "\"host\":\"194.71.130.189\"") {
		t.Fatalf("expected body to contain host, got: %s", capturedBody)
	}
}
