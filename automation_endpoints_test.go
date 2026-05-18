package mlx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestStartedProfileDataResolveRodControlURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/json/version" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		port := mustPortFromURL(t, "http://"+r.Host)
		fmt.Fprintf(w, `{"webSocketDebuggerUrl":"ws://127.0.0.1:%s/devtools/browser/abc123"}`, port)
	}))
	t.Cleanup(server.Close)

	data := &StartedProfileData{Port: mustPortFromURL(t, server.URL)}
	got, err := data.ResolveRodControlURL(context.Background())
	if err != nil {
		t.Fatalf("ResolveRodControlURL returned error: %v", err)
	}

	want := fmt.Sprintf("ws://127.0.0.1:%s/devtools/browser/abc123", data.Port)
	if got != want {
		t.Fatalf("unexpected rod control url: got %q want %q", got, want)
	}
}

func TestStartedProfileDataResolveCDPWebSocketURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/json/version" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		port := mustPortFromURL(t, "http://"+r.Host)
		fmt.Fprintf(w, `{"webSocketDebuggerUrl":"ws://127.0.0.1:%s/devtools/browser/def456"}`, port)
	}))
	t.Cleanup(server.Close)

	data := &StartedProfileData{Port: mustPortFromURL(t, server.URL)}
	got, err := data.ResolveCDPWebSocketURL(context.Background())
	if err != nil {
		t.Fatalf("ResolveCDPWebSocketURL returned error: %v", err)
	}

	want := fmt.Sprintf("ws://127.0.0.1:%s/devtools/browser/def456", data.Port)
	if got != want {
		t.Fatalf("unexpected cdp websocket url: got %q want %q", got, want)
	}
}

func TestStartedProfileDataResolveCDPWebSocketURLNormalizesHost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/json/version" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		port := mustPortFromURL(t, "http://"+r.Host)
		fmt.Fprintf(w, `{"webSocketDebuggerUrl":"ws://example.com:%s/devtools/browser/ghi789"}`, port)
	}))
	t.Cleanup(server.Close)

	data := &StartedProfileData{Port: mustPortFromURL(t, server.URL)}
	got, err := data.ResolveCDPWebSocketURL(context.Background())
	if err != nil {
		t.Fatalf("ResolveCDPWebSocketURL returned error: %v", err)
	}

	want := fmt.Sprintf("ws://127.0.0.1:%s/devtools/browser/ghi789", data.Port)
	if got != want {
		t.Fatalf("unexpected normalized cdp websocket url: got %q want %q", got, want)
	}
}

func TestStartedProfileDataResolveCDPWebSocketURLEmptyPort(t *testing.T) {
	data := &StartedProfileData{
		RequestedAutomation: AutomationRod,
		LauncherAutomation:  AutomationPlaywright,
	}
	_, err := data.ResolveCDPWebSocketURL(context.Background())
	if err == nil {
		t.Fatal("expected error for empty port")
	}

	var endpointErr *AutomationEndpointError
	if !errors.As(err, &endpointErr) {
		t.Fatalf("expected *AutomationEndpointError, got %T: %v", err, err)
	}
	if endpointErr.RequestedAutomation != AutomationRod {
		t.Fatalf("unexpected RequestedAutomation: got %q want %q", endpointErr.RequestedAutomation, AutomationRod)
	}
	if endpointErr.LauncherAutomation != AutomationPlaywright {
		t.Fatalf("unexpected LauncherAutomation: got %q want %q", endpointErr.LauncherAutomation, AutomationPlaywright)
	}
}

func mustPortFromURL(t *testing.T, rawURL string) string {
	t.Helper()

	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("failed to parse url %q: %v", rawURL, err)
	}
	if parsed.Port() == "" {
		t.Fatalf("expected url to include a port: %s", rawURL)
	}
	return parsed.Port()
}
