package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	mlx "mlx-go-sdk"
)

func TestExecuteTemplateListJSONOutput(t *testing.T) {
	t.Setenv(mlx.EnvToken, "test-token")

	var gotQuery url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/resources/metas" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		gotQuery = r.URL.Query()
		fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"objects":[{"id":"tpl-1","object_type_id":"7e46e7f9-15d4-41b6-83b9-a652336793ec","object_name":"Template A","object_size":123,"current_version":"1","created_at":"2026-04-20T00:00:00Z","created_by":"me@example.com","update_at":"2026-04-20T00:00:00Z","update_by":"me@example.com","storage_type":"cloud","meta_info":"{}","is_default":false,"is_in_trashbin":false}]}}`)
	}))
	defer server.Close()

	configPath := writeRuntimeConfigFile(t, fmt.Sprintf(`{
  "version": "1",
  "endpoints": {
    "base_url": %q,
    "launcher_url": %q
  },
  "output": {
    "format": "json"
  },
  "retry": {
    "enabled": false
  }
}`, server.URL, server.URL))

	output, err := captureCLIStdout(func() error {
		return Execute([]string{"--config", configPath, "template", "list", "--name", "Template", "--limit", "20", "--offset", "5"})
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if got := gotQuery.Get("object_type_id"); got != mlx.ResourceTypeProfileTemplates {
		t.Fatalf("expected template object_type_id %q, got %q", mlx.ResourceTypeProfileTemplates, got)
	}
	if got := gotQuery.Get("object_name"); got != "Template" {
		t.Fatalf("expected object_name filter, got %q", got)
	}
	if got := gotQuery.Get("limit"); got != "20" {
		t.Fatalf("expected limit=20, got %q", got)
	}
	if got := gotQuery.Get("offset"); got != "5" {
		t.Fatalf("expected offset=5, got %q", got)
	}
	if !strings.Contains(output, `"object_name": "Template A"`) {
		t.Fatalf("expected JSON output to contain template name, got %s", output)
	}
}

func TestExecuteTemplateGetJSONOutputIncludesParsedTemplate(t *testing.T) {
	t.Setenv(mlx.EnvToken, "test-token")

	templatePath := filepath.Join(t.TempDir(), "template.json")
	templateBody := `{
  "name": "Template Body",
  "mainParams": {
    "name": "Template Profile",
    "browser_type": "mimic",
    "folder_id": "folder-1",
    "os_type": "windows",
    "parameters": {
      "storage": {
        "is_local": false
      }
    }
  }
}`
	if err := os.WriteFile(templatePath, []byte(templateBody), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/resources/tpl-1/meta":
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"id":"tpl-1","object_type_id":"7e46e7f9-15d4-41b6-83b9-a652336793ec","object_name":"Template A","object_size":123,"current_version":"1","created_at":"2026-04-20T00:00:00Z","created_by":"me@example.com","update_at":"2026-04-20T00:00:00Z","update_by":"me@example.com","storage_type":"cloud","meta_info":"{\"name\":\"Meta Template\"}","is_default":false,"is_in_trashbin":false}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/object_storage/tpl-1/download":
			fmt.Fprintf(w, `{"status":{"http_code":200,"message":%s}}`, jsonQuotedString("Object downloaded to the disk at "+templatePath))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	configPath := writeRuntimeConfigFile(t, fmt.Sprintf(`{
  "version": "1",
  "endpoints": {
    "base_url": %q,
    "launcher_url": %q
  },
  "output": {
    "format": "json"
  },
  "retry": {
    "enabled": false
  }
}`, server.URL, server.URL))

	output, err := captureCLIStdout(func() error {
		return Execute([]string{"--config", configPath, "template", "get", "--id", "tpl-1"})
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if !strings.Contains(output, `"object_name": "Template A"`) {
		t.Fatalf("expected JSON output to contain meta object_name, got %s", output)
	}
	if !strings.Contains(output, `"path": `) || !strings.Contains(output, filepath.Base(templatePath)) {
		t.Fatalf("expected JSON output to contain downloaded path, got %s", output)
	}
	if !strings.Contains(output, `"name": "Template Body"`) {
		t.Fatalf("expected JSON output to contain parsed template body, got %s", output)
	}
}

func TestExecuteProfileCreateFromTemplateWithWaitAndLocalOverride(t *testing.T) {
	t.Setenv(mlx.EnvToken, "test-token")

	templatePath := filepath.Join(t.TempDir(), "template.json")
	templateBody := `{
  "name": "Template Body",
  "mainParams": {
    "name": "Template Profile",
    "browser_type": "mimic",
    "folder_id": "folder-template",
    "os_type": "windows",
    "parameters": {
      "storage": {
        "is_local": false
      }
    }
  }
}`
	if err := os.WriteFile(templatePath, []byte(templateBody), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	var createBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/resources/tpl-1/meta":
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"id":"tpl-1","object_type_id":"7e46e7f9-15d4-41b6-83b9-a652336793ec","object_name":"Template A","object_size":123,"current_version":"1","created_at":"2026-04-20T00:00:00Z","created_by":"me@example.com","update_at":"2026-04-20T00:00:00Z","update_by":"me@example.com","storage_type":"cloud","meta_info":"{\"name\":\"Meta Template\"}","is_default":false,"is_in_trashbin":false}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/object_storage/tpl-1/download":
			fmt.Fprintf(w, `{"status":{"http_code":200,"message":%s}}`, jsonQuotedString("Object downloaded to the disk at "+templatePath))
		case r.Method == http.MethodPost && r.URL.Path == "/profile/create":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("ReadAll returned error: %v", err)
			}
			createBody = string(body)
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"ids":["profile-1"]}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/profile/metas":
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[{"id":"profile-1","name":"Demo Local","browser_type":"mimic","core_version":137,"is_auto_update":true,"is_local":false,"os_type":"windows","folder_id":"folder-override","workspace_id":"ws-1","created_at":"2026-04-20T00:00:00Z","created_by":"me@example.com","in_use_by":"","last_launched_at":"","last_launched_by":"","last_launched_on":"","last_update_at":"2026-04-20T00:00:00Z","last_updated_by":"me@example.com","removed_at":"","removed_by":"","status":"ready","parameters":{"storage":{"is_local":true}}}]}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	configPath := writeRuntimeConfigFile(t, fmt.Sprintf(`{
  "version": "1",
  "endpoints": {
    "base_url": %q,
    "launcher_url": %q
  },
  "output": {
    "format": "json"
  },
  "retry": {
    "enabled": false
  }
}`, server.URL, server.URL))

	output, err := captureCLIStdout(func() error {
		return Execute([]string{"--config", configPath, "profile", "create", "--template-id", "tpl-1", "--name", "Demo Local", "--folder-id", "folder-override", "--local", "--wait"})
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	checks := []string{
		`"name":"Demo Local"`,
		`"folder_id":"folder-override"`,
		`"browser_type":"mimic"`,
		`"is_local":true`,
	}
	for _, check := range checks {
		if !strings.Contains(createBody, check) {
			t.Fatalf("expected create request body to contain %s, got %s", check, createBody)
		}
	}
	if !strings.Contains(output, `"id": "profile-1"`) {
		t.Fatalf("expected verified workflow output to contain created profile id, got %s", output)
	}
	if !strings.Contains(output, `"is_local": true`) {
		t.Fatalf("expected verified workflow output to contain local storage confirmation, got %s", output)
	}
}

func captureCLIStdout(fn func() error) (string, error) {
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}
	os.Stdout = w

	runErr := fn()
	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		_ = r.Close()
		return "", err
	}
	_ = r.Close()
	return buf.String(), runErr
}

func jsonQuotedString(value string) string {
	body, _ := json.Marshal(value)
	return string(body)
}
