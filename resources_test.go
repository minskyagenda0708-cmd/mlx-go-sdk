package mlx

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"mlx-go-sdk/internal/testutil"
)

func TestResourcesListTypes(t *testing.T) {
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/resources/types" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"types":[{"id":"7e46e7f9-15d4-41b6-83b9-a652336793ec","name":"Profile templates"}]}}`)
	})

	client, err := New(WithToken("test-token"), WithHTTPClient(httpClient), WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, _, err := client.Resources.ListTypes(context.Background())
	if err != nil {
		t.Fatalf("Resources.ListTypes returned error: %v", err)
	}
	if len(resp.Data.Types) != 1 || resp.Data.Types[0].ID != ResourceTypeProfileTemplates {
		t.Fatalf("unexpected types response: %#v", resp.Data.Types)
	}
}

func TestResourcesListProfileTemplates(t *testing.T) {
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/resources/metas" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("object_type_id"); got != ResourceTypeProfileTemplates {
			t.Fatalf("unexpected object_type_id: %s", got)
		}
		fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"objects":[{"id":"tpl-1","object_type_id":"7e46e7f9-15d4-41b6-83b9-a652336793ec","object_name":"Template.txt","object_size":2,"current_version":"1","created_at":"2026-04-20","created_by":"user","update_at":"2026-04-20","update_by":"user","storage_type":"cloud","meta_info":"{}","is_default":false,"is_in_trashbin":false}]}}`)
	})

	client, err := New(WithToken("test-token"), WithHTTPClient(httpClient), WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, _, err := client.Resources.ListProfileTemplates(context.Background(), &ListResourceMetasOptions{ObjectName: "Template"})
	if err != nil {
		t.Fatalf("Resources.ListProfileTemplates returned error: %v", err)
	}
	if len(resp.Data.Objects) != 1 || resp.Data.Objects[0].ID != "tpl-1" {
		t.Fatalf("unexpected objects response: %#v", resp.Data.Objects)
	}
}

func TestResourcesCreateProfileTemplate(t *testing.T) {
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/object_storage/create_and_upload" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		text := string(body)
		checks := []string{
			`"object_type_id":"7e46e7f9-15d4-41b6-83b9-a652336793ec"`,
			`"object_name":"Template"`,
			`"object_body":"{}`,
		}
		for _, check := range checks {
			if !strings.Contains(text, check) {
				t.Fatalf("expected request body to contain %s, got %s", check, text)
			}
		}
		fmt.Fprint(w, `{"status":{"http_code":200,"message":"saved"},"data":{"meta_id":"tpl-1"}}`)
	})

	client, err := New(WithToken("test-token"), WithHTTPClient(httpClient), WithLauncherURL(server.URL))
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, _, err := client.Resources.CreateProfileTemplate(context.Background(), &CreateProfileTemplateRequest{Name: "Template", Body: `{}`, Meta: `{}`})
	if err != nil {
		t.Fatalf("Resources.CreateProfileTemplate returned error: %v", err)
	}
	if resp.Data.MetaID != "tpl-1" {
		t.Fatalf("unexpected meta id: %s", resp.Data.MetaID)
	}
}

func TestResourcesDownloadParsesPath(t *testing.T) {
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/object_storage/tpl-1/download" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		fmt.Fprint(w, `{"status":{"http_code":200,"message":"Object downloaded to the disk at C:\\Users\\bath0ry\\mlx\\ObjectStorage\\Profile templates\\tpl-1\\Template.txt"}}`)
	})

	client, err := New(WithToken("test-token"), WithHTTPClient(httpClient), WithLauncherURL(server.URL))
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, _, err := client.Resources.Download(context.Background(), "tpl-1")
	if err != nil {
		t.Fatalf("Resources.Download returned error: %v", err)
	}
	if !strings.Contains(resp.Path, `Template.txt`) {
		t.Fatalf("unexpected download path: %s", resp.Path)
	}
}

func TestResourcesWorkflowDefaultsToLocalStorage(t *testing.T) {
	gotStorageType := ""
	server, httpClient := testutil.NewServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/profile/search":
			gotStorageType = extractBodyField(t, r, `"storage_type":"`)
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[{"id":"profile-1","name":"Demo","folder_id":"folder-1","browser_type":"mimic","os_type":"windows","core_version":137,"notes":"","created_by":"me@example.com","in_use_by":"","last_launched_by":"","is_local":true}],"total_count":1}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/profile/stop/p/profile-1":
			fmt.Fprint(w, `{"status":{"http_code":200,"message":"Profile stopped successfully"},"data":null}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})

	client, err := New(WithToken("test-token"), WithHTTPClient(httpClient), WithBaseURL(server.URL), WithLauncherURL(server.URL))
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	_, err = client.Workflows.StopProfileByName(context.Background(), "Demo", StopProfileByNameOptions{IgnoreAlreadyStopped: true})
	if err != nil {
		t.Fatalf("Workflows.StopProfileByName returned error: %v", err)
	}
	if gotStorageType != "local" {
		t.Fatalf("expected workflow search to default to local storage, got %q", gotStorageType)
	}
}

func extractBodyField(t *testing.T, r *http.Request, prefix string) string {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("failed to read request body: %v", err)
	}
	text := string(body)
	start := strings.Index(text, prefix)
	if start == -1 {
		return ""
	}
	start += len(prefix)
	end := strings.Index(text[start:], `"`)
	if end == -1 {
		return ""
	}
	return text[start : start+end]
}
