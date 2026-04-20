package mlx

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"mlx-go-sdk/internal/testutil"
)

func TestCookiesListWebsites(t *testing.T) {
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/cookies/metadata/websites" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":[{"key":"google","value":"google.com"}]}`)
	})

	client, err := New(
		WithToken("test-token"),
		WithHTTPClient(httpClient),
		WithCookiesURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, _, err := client.Cookies.ListWebsites(context.Background())
	if err != nil {
		t.Fatalf("Cookies.ListWebsites returned error: %v", err)
	}
	if len(resp.Data) != 1 || resp.Data[0].Key != "google" {
		t.Fatalf("unexpected websites response: %#v", resp.Data)
	}
}

func TestCookiesCreateMetadata(t *testing.T) {
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/cookies/metadata" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("X-Strict-Mode"); got != "true" {
			t.Fatalf("expected X-Strict-Mode header, got %q", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		text := string(body)
		if !strings.Contains(text, `"profile_id":"profile-1"`) || !strings.Contains(text, `"target_website":"google"`) {
			t.Fatalf("unexpected request body: %s", text)
		}
		fmt.Fprint(w, `{"status":{"http_code":201,"message":"cookies metadata successfully created"},"data":{"profile_id":"profile-1"}}`)
	})

	client, err := New(
		WithToken("test-token"),
		WithHTTPClient(httpClient),
		WithCookiesURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, _, err := client.Cookies.CreateMetadata(context.Background(), &CreateCookiesMetadataRequest{
		ProfileID:     "profile-1",
		TargetWebsite: "google",
		StrictMode:    true,
	})
	if err != nil {
		t.Fatalf("Cookies.CreateMetadata returned error: %v", err)
	}
	if resp.Data.ProfileID != "profile-1" {
		t.Fatalf("unexpected profile id: %s", resp.Data.ProfileID)
	}
}

func TestCookiesList(t *testing.T) {
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/cookies/profile-1" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"cookies":[{"id":24521,"created_at":"2023-07-11T04:37:45.917Z","data":[{"name":"session","value":"abc","domain":"google.com","path":"/","secure":true,"httpOnly":false,"session":false,"expirationDate":1740851445102}]}]}}`)
	})

	client, err := New(
		WithToken("test-token"),
		WithHTTPClient(httpClient),
		WithCookiesURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, _, err := client.Cookies.List(context.Background(), "profile-1")
	if err != nil {
		t.Fatalf("Cookies.List returned error: %v", err)
	}
	if len(resp.Data.Cookies) != 1 || len(resp.Data.Cookies[0].Data) != 1 {
		t.Fatalf("unexpected cookies response: %#v", resp.Data)
	}
	if resp.Data.Cookies[0].Data[0].Domain != "google.com" {
		t.Fatalf("unexpected cookie domain: %s", resp.Data.Cookies[0].Data[0].Domain)
	}
}

func TestCookiesImportMarshalsCookieString(t *testing.T) {
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/cookie_import" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		text := string(body)
		if !strings.Contains(text, `"import_advanced_cookies":false`) {
			t.Fatalf("expected import_advanced_cookies=false, got %s", text)
		}
		if !strings.Contains(text, `"cookies":"[{\"name\":\"session\"`) {
			t.Fatalf("expected cookies string payload, got %s", text)
		}
		fmt.Fprint(w, `{"status":{"http_code":200,"message":"Cookies successfully imported"},"data":null}`)
	})

	client, err := New(
		WithToken("test-token"),
		WithHTTPClient(httpClient),
		WithLauncherURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	_, _, err = client.Cookies.Import(context.Background(), &CookieImportRequest{
		ProfileID:             "profile-1",
		FolderID:              "folder-1",
		ImportAdvancedCookies: false,
		Cookies: []BrowserCookie{{
			Name:   "session",
			Value:  "abc",
			Domain: "google.com",
			Path:   "/",
		}},
	})
	if err != nil {
		t.Fatalf("Cookies.Import returned error: %v", err)
	}
}

func TestCookiesExport(t *testing.T) {
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/cookie_export" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"status":{"http_code":200,"message":"Cookies downloaded successfully."},"data":{"cookies":"[cookies]","profile_id":"profile-1","timestamp":1738595753833}}`)
	})

	client, err := New(
		WithToken("test-token"),
		WithHTTPClient(httpClient),
		WithLauncherURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, _, err := client.Cookies.Export(context.Background(), &CookieExportRequest{ProfileID: "profile-1", FolderID: "folder-1"})
	if err != nil {
		t.Fatalf("Cookies.Export returned error: %v", err)
	}
	if resp.Data.ProfileID != "profile-1" {
		t.Fatalf("unexpected profile id: %s", resp.Data.ProfileID)
	}
}

func TestCookiesSeedProfileCookies(t *testing.T) {
	var metadataCalls atomic.Int32
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/cookies/metadata":
			metadataCalls.Add(1)
			fmt.Fprint(w, `{"status":{"http_code":201,"message":"cookies metadata successfully created"},"data":{"profile_id":"profile-1"}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/cookies/profile-1":
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"cookies":[{"id":24521,"created_at":"2023-07-11T04:37:45.917Z","data":[{"name":"session","value":"abc","domain":"google.com","path":"/","secure":true,"httpOnly":false,"session":false,"expirationDate":1740851445102}]}]}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/cookie_import":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("failed to read import request body: %v", err)
			}
			text := string(body)
			if !strings.Contains(text, `"folder_id":"folder-1"`) {
				t.Fatalf("expected resolved folder id in import body, got %s", text)
			}
			fmt.Fprint(w, `{"status":{"http_code":200,"message":"Cookies successfully imported"},"data":null}`)
		case r.Method == http.MethodPost && r.URL.Path == "/profile/metas":
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[{"id":"profile-1","name":"Demo","folder_id":"folder-1","browser_type":"mimic","core_version":137,"os_type":"windows","workspace_id":"ws-1","created_at":"2026-04-20T00:00:00Z","created_by":"me@example.com","last_update_at":"2026-04-20T00:00:00Z","last_updated_by":"me@example.com","status":"ready"}]}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})

	client, err := New(
		WithToken("test-token"),
		WithHTTPClient(httpClient),
		WithBaseURL(server.URL),
		WithLauncherURL(server.URL),
		WithCookiesURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	result, err := client.Cookies.SeedProfileCookies(context.Background(), SeedProfileCookiesOptions{
		ProfileID:               "profile-1",
		TargetWebsite:           "google",
		CreateMetadataIfMissing: true,
	})
	if err != nil {
		t.Fatalf("Cookies.SeedProfileCookies returned error: %v", err)
	}
	if !result.MetadataCreated {
		t.Fatalf("expected metadata to be created")
	}
	if result.CookieCount != 1 {
		t.Fatalf("expected one imported cookie, got %d", result.CookieCount)
	}
	if result.FolderID != "folder-1" {
		t.Fatalf("unexpected folder id: %s", result.FolderID)
	}
	if metadataCalls.Load() != 1 {
		t.Fatalf("expected one metadata create call, got %d", metadataCalls.Load())
	}
	if result.SelectedBundle == nil || result.SelectedBundle.ID != 24521 {
		t.Fatalf("unexpected selected bundle: %#v", result.SelectedBundle)
	}
}

func TestCookiesSeedProfileCookiesFallsBackToUpdate(t *testing.T) {
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/cookies/metadata":
			w.WriteHeader(http.StatusConflict)
			fmt.Fprint(w, `{"status":{"http_code":409,"message":"metadata already exists"}}`)
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/cookies/metadata":
			fmt.Fprint(w, `{"status":{"http_code":200,"message":"cookies metadata successfully updated"},"data":null}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/cookies/profile-1":
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"cookies":[{"id":24521,"created_at":"2023-07-11T04:37:45.917Z","data":[{"name":"session","value":"abc","domain":"google.com","path":"/"}]}]}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/cookie_import":
			fmt.Fprint(w, `{"status":{"http_code":200,"message":"Cookies successfully imported"},"data":null}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})

	client, err := New(
		WithToken("test-token"),
		WithHTTPClient(httpClient),
		WithLauncherURL(server.URL),
		WithCookiesURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	result, err := client.Cookies.SeedProfileCookies(context.Background(), SeedProfileCookiesOptions{
		ProfileID:               "profile-1",
		FolderID:                "folder-1",
		TargetWebsite:           "google",
		AdditionalWebsite:       "bing",
		CreateMetadataIfMissing: true,
	})
	if err != nil {
		t.Fatalf("Cookies.SeedProfileCookies returned error: %v", err)
	}
	if result.MetadataCreated {
		t.Fatalf("did not expect metadata create success on conflict fallback")
	}
	if !result.MetadataUpdated {
		t.Fatalf("expected metadata update fallback")
	}
}
