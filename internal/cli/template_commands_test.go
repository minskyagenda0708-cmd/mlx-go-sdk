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

	mlx "github.com/minskyagenda0708-cmd/mlx-go-sdk"
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

func TestExecuteLauncherStartUsesConfigBoolDefaults(t *testing.T) {
	t.Setenv(mlx.EnvToken, "test-token")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/profile/metas":
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[{"id":"profile-1","name":"Demo","folder_id":"folder-1","browser_type":"mimic","core_version":137,"os_type":"windows","workspace_id":"ws-1","created_at":"2026-04-20T00:00:00Z","created_by":"me@example.com","last_update_at":"2026-04-20T00:00:00Z","last_updated_by":"me@example.com","status":"ready","parameters":{"storage":{"is_local":false}}}]}}`)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v2/profile/f/folder-1/p/profile-1/start"):
			if got := r.URL.Query().Get("headless_mode"); got != "true" {
				t.Fatalf("expected headless_mode=true from config default, got %q", got)
			}
			if got := r.URL.Query().Get("automation_type"); got != "playwright" {
				t.Fatalf("expected automation_type=playwright from config default, got %q", got)
			}
			if got := r.Header.Get("X-Strict-Mode"); got != "true" {
				t.Fatalf("expected strict mode header from config default, got %q", got)
			}
			fmt.Fprint(w, `{"status":{"http_code":200,"message":"Profile started successfully"},"data":{"browser_type":"mimic","core_version":137,"id":"profile-1","is_quick":false,"port":"55513"}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/profile/status/p/profile-1":
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"profile_id":"profile-1","name":"Demo","status":"browser_running","browser_type":"mimic","core_version":137,"folder_id":"folder-1","workspace_id":"workspace-1","message":"","timestamp":1745100000000}}`)
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
  },
  "defaults": {
    "launcher": {
      "automation_type": "playwright",
      "headless": true,
      "strict_mode": true,
      "wait_for_running": true
    },
    "proxy": {
      "proxy_continuity": {
        "enabled": false
      }
    }
  }
}`, server.URL, server.URL))

	output, err := captureCLIStdout(func() error {
		return Execute([]string{"--config", configPath, "launcher", "start", "--profile-id", "profile-1"})
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !strings.Contains(output, `"runtime_status"`) {
		t.Fatalf("expected launcher start output to include runtime status when wait default is enabled, got %s", output)
	}
}

func TestExecuteLauncherStartSkipProxyCheckSucceeds(t *testing.T) {
	t.Setenv(mlx.EnvToken, "test-token")

	var proxyHit bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/profile/metas":
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[{"id":"profile-1","name":"Demo","folder_id":"folder-1","browser_type":"mimic","core_version":137,"os_type":"windows","workspace_id":"ws-1","created_at":"2026-04-20T00:00:00Z","created_by":"me@example.com","last_update_at":"2026-04-20T00:00:00Z","last_updated_by":"me@example.com","status":"ready","parameters":{"storage":{"is_local":false}}}]}}`)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v2/profile/f/folder-1/p/profile-1/start"):
			fmt.Fprint(w, `{"status":{"http_code":200,"message":"Profile started successfully"},"data":{"browser_type":"mimic","core_version":137,"id":"profile-1","is_quick":false,"port":"55513"}}`)
		case strings.Contains(r.URL.Path, "/proxy/"):
			proxyHit = true
			t.Fatalf("proxy endpoint must not be hit when --skip-proxy-check is set: %s %s", r.Method, r.URL.Path)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	// Continuity is ENABLED in config, proving --skip-proxy-check bypasses it.
	configPath := writeRuntimeConfigFile(t, fmt.Sprintf(`{
  "version": "1",
  "endpoints": {
    "base_url": %q,
    "launcher_url": %q,
    "proxy_url": %q
  },
  "output": {
    "format": "json"
  },
  "retry": {
    "enabled": false
  },
  "defaults": {
    "launcher": {
      "automation_type": "playwright",
      "wait_for_running": false
    },
    "proxy": {
      "proxy_continuity": {
        "enabled": true,
        "latency_threshold_ms": 2000,
        "latency_hard_cap_ms": 3000,
        "candidates_per_round": 3,
        "check_targets": ["http://127.0.0.1:1"],
        "check_timeout": "1s"
      }
    }
  }
}`, server.URL, server.URL, server.URL))

	output, err := captureCLIStdout(func() error {
		return Execute([]string{"--config", configPath, "launcher", "start", "--profile-id", "profile-1", "--skip-proxy-check"})
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if proxyHit {
		t.Fatalf("proxy endpoint was hit despite --skip-proxy-check")
	}
	if !strings.Contains(output, `"id": "profile-1"`) {
		t.Fatalf("expected launcher start output to contain started profile id, got %s", output)
	}
}

func TestExecuteLauncherStartFailsClosedWhenProxyCheckFails(t *testing.T) {
	t.Setenv(mlx.EnvToken, "test-token")

	var startHit bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/profile/metas":
			// No proxy on the profile (parameters.proxy is null) -> current is nil,
			// exercising the nil-guard and forcing a generation round.
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[{"id":"profile-1","name":"Demo","folder_id":"folder-1","browser_type":"mimic","core_version":137,"os_type":"windows","workspace_id":"ws-1","created_at":"2026-04-20T00:00:00Z","created_by":"me@example.com","last_update_at":"2026-04-20T00:00:00Z","last_updated_by":"me@example.com","status":"ready","parameters":{"storage":{"is_local":false}}}]}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/v1/user":
			// Proxy backend (GetUsage, first call in generation) unreachable/failing -> fail-closed.
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"status":{"http_code":500,"message":"proxy backend unavailable"}}`)
		case strings.HasPrefix(r.URL.Path, "/api/v2/profile/f/") && strings.HasSuffix(r.URL.Path, "/start"):
			startHit = true
			t.Fatalf("start endpoint must NEVER be hit when the proxy check fails (fail-closed): %s %s", r.Method, r.URL.Path)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	configPath := writeRuntimeConfigFile(t, fmt.Sprintf(`{
  "version": "1",
  "endpoints": {
    "base_url": %q,
    "launcher_url": %q,
    "proxy_url": %q
  },
  "output": {
    "format": "json"
  },
  "retry": {
    "enabled": false
  },
  "defaults": {
    "launcher": {
      "automation_type": "playwright",
      "wait_for_running": false
    },
    "proxy": {
      "proxy_continuity": {
        "enabled": true,
        "latency_threshold_ms": 2000,
        "latency_hard_cap_ms": 3000,
        "candidates_per_round": 3,
        "check_targets": ["http://127.0.0.1:1"],
        "check_timeout": "1s"
      }
    }
  }
}`, server.URL, server.URL, server.URL))

	_, err := captureCLIStdout(func() error {
		return Execute([]string{"--config", configPath, "launcher", "start", "--profile-id", "profile-1"})
	})
	if err == nil {
		t.Fatalf("expected fail-closed error when proxy check fails, got nil")
	}
	if !strings.Contains(err.Error(), "proxy check failed") {
		t.Fatalf("expected pre-launch proxy check error, got %v", err)
	}
	if startHit {
		t.Fatalf("start endpoint was hit despite failed proxy check (NOT fail-closed)")
	}
}

func TestExecuteLauncherStartByNameFailsClosedWhenProxyCheckFails(t *testing.T) {
	t.Setenv(mlx.EnvToken, "test-token")

	var startHit bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/profile/search":
			// FindByName (name resolution) -> single exact match.
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[{"id":"profile-1","name":"Demo","folder_id":"folder-1","browser_type":"mimic","os_type":"windows","core_version":137,"created_by":"me@example.com","is_local":false}],"total_count":1}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/profile/metas":
			// GetMeta: name-verify + ensureProxyBeforeStart both hit this.
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[{"id":"profile-1","name":"Demo","folder_id":"folder-1","browser_type":"mimic","core_version":137,"os_type":"windows","workspace_id":"ws-1","created_at":"2026-04-20T00:00:00Z","created_by":"me@example.com","last_update_at":"2026-04-20T00:00:00Z","last_updated_by":"me@example.com","status":"ready","parameters":{"storage":{"is_local":false}}}]}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/v1/user":
			// Proxy backend unreachable/failing -> fail-closed on the name path too.
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"status":{"http_code":500,"message":"proxy backend unavailable"}}`)
		case strings.HasPrefix(r.URL.Path, "/api/v2/profile/f/") && strings.HasSuffix(r.URL.Path, "/start"):
			startHit = true
			t.Fatalf("start endpoint must NEVER be hit on the --profile-name path when the proxy check fails: %s %s", r.Method, r.URL.Path)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	configPath := writeRuntimeConfigFile(t, fmt.Sprintf(`{
  "version": "1",
  "endpoints": {
    "base_url": %q,
    "launcher_url": %q,
    "proxy_url": %q
  },
  "output": {
    "format": "json"
  },
  "retry": {
    "enabled": false
  },
  "defaults": {
    "launcher": {
      "automation_type": "playwright",
      "wait_for_running": false
    },
    "proxy": {
      "proxy_continuity": {
        "enabled": true,
        "latency_threshold_ms": 2000,
        "latency_hard_cap_ms": 3000,
        "candidates_per_round": 3,
        "check_targets": ["http://127.0.0.1:1"],
        "check_timeout": "1s"
      }
    }
  }
}`, server.URL, server.URL, server.URL))

	_, err := captureCLIStdout(func() error {
		return Execute([]string{"--config", configPath, "launcher", "start", "--profile-name", "Demo"})
	})
	if err == nil {
		t.Fatalf("expected fail-closed error on --profile-name path when proxy check fails, got nil")
	}
	if !strings.Contains(err.Error(), "proxy check failed") {
		t.Fatalf("expected pre-launch proxy check error, got %v", err)
	}
	if startHit {
		t.Fatalf("start endpoint was hit despite failed proxy check on name path (NOT fail-closed)")
	}
}

func TestExecuteImportRunUsesConfigBoolDefaults(t *testing.T) {
	t.Setenv(mlx.EnvToken, "test-token")

	var importBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/profile/import":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("ReadAll returned error: %v", err)
			}
			importBody = string(body)
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"import_id":"import-1","import_path":"C:/exports/demo.zip","status":"running","message":"","timestamp":1745100000000}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/profile/imports/import-1/status":
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"import_id":"import-1","import_path":"C:/exports/demo.zip","new_profile_id":"profile-1","status":"done","message":"","timestamp":1745100000000}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/profile/metas":
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[{"id":"profile-1","name":"Imported Demo","folder_id":"folder-1","browser_type":"mimic","core_version":137,"os_type":"windows","workspace_id":"ws-1","created_at":"2026-04-20T00:00:00Z","created_by":"me@example.com","last_update_at":"2026-04-20T00:00:00Z","last_updated_by":"me@example.com","status":"ready","parameters":{"storage":{"is_local":true}}}]}}`)
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
  },
  "defaults": {
    "import": {
      "is_local": true,
      "wait": true
    }
  }
}`, server.URL, server.URL))

	output, err := captureCLIStdout(func() error {
		return Execute([]string{"--config", configPath, "import", "run", "--import-path", "C:\\exports\\demo.zip"})
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !strings.Contains(importBody, `"is_local":true`) {
		t.Fatalf("expected import request body to inherit is_local=true from config, got %s", importBody)
	}
	if !strings.Contains(output, `"profile_meta"`) && !strings.Contains(output, `"ProfileMeta"`) && !strings.Contains(output, `"Imported Demo"`) {
		t.Fatalf("expected verified import output when wait default is enabled, got %s", output)
	}
}

func TestExecuteLauncherStopWaitRequiresExplicitStoppedStatus(t *testing.T) {
	t.Setenv(mlx.EnvToken, "test-token")

	statusCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/profile/metas":
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[{"id":"profile-1","name":"Demo","folder_id":"folder-1","browser_type":"mimic","core_version":137,"os_type":"windows","workspace_id":"ws-1","created_at":"2026-04-20T00:00:00Z","created_by":"me@example.com","last_update_at":"2026-04-20T00:00:00Z","last_updated_by":"me@example.com","status":"ready","parameters":{"storage":{"is_local":false}}}]}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/profile/stop/p/profile-1":
			fmt.Fprint(w, `{"status":{"http_code":200,"message":"Profile stopped successfully"},"data":null}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/profile/status/p/profile-1":
			statusCalls++
			if statusCalls == 1 {
				fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"profile_id":"profile-1","name":"Demo","status":"stopping","browser_type":"mimic","core_version":137,"folder_id":"folder-1","workspace_id":"workspace-1","message":"","timestamp":1745100000000}}`)
				return
			}
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"profile_id":"profile-1","name":"Demo","status":"stopped","browser_type":"mimic","core_version":137,"folder_id":"folder-1","workspace_id":"workspace-1","message":"","timestamp":1745100001000}}`)
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
  },
  "poll": {
    "initial_interval": "1ms",
    "max_interval": "1ms",
    "timeout": "200ms",
    "multiplier": 1.5
  }
}`, server.URL, server.URL))

	output, err := captureCLIStdout(func() error {
		return Execute([]string{"--config", configPath, "launcher", "stop", "--profile-id", "profile-1", "--wait"})
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if statusCalls < 2 {
		t.Fatalf("expected stop wait to poll until explicit stopped status, got %d status calls", statusCalls)
	}
	if !strings.Contains(output, `"stopped"`) {
		t.Fatalf("expected stopped runtime status in output, got %s", output)
	}
}

func TestExecuteExtensionEnableByIDWaitsForObjectUsageBinding(t *testing.T) {
	t.Setenv(mlx.EnvToken, "test-token")

	objectUsageCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/profile/metas":
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[{"id":"profile-1","name":"Demo","folder_id":"folder-1","browser_type":"mimic","core_version":137,"os_type":"windows","workspace_id":"ws-1","created_at":"2026-04-20T00:00:00Z","created_by":"me@example.com","last_update_at":"2026-04-20T00:00:00Z","last_updated_by":"me@example.com","status":"ready","parameters":{"storage":{"is_local":false}}}]}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/resources/ext-1/enable_for_profiles":
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":"enabled"}`)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/resources/object_profile_usages"):
			objectUsageCalls++
			if objectUsageCalls == 1 {
				fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":[]}`)
				return
			}
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":[{"id":"profile-1","object_id":"ext-1"}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/resources/profile_object_usages":
			w.WriteHeader(http.StatusNotImplemented)
			fmt.Fprint(w, `{"status":{"http_code":501,"message":"not implemented"}}`)
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
  },
  "poll": {
    "initial_interval": "1ms",
    "max_interval": "1ms",
    "timeout": "200ms",
    "multiplier": 1.5
  },
  "defaults": {
    "extension": {
      "require_profile_usage_read": false
    }
  }
}`, server.URL, server.URL))

	output, err := captureCLIStdout(func() error {
		return Execute([]string{"--config", configPath, "extension", "enable", "--id", "ext-1", "--profile-id", "profile-1"})
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if objectUsageCalls < 2 {
		t.Fatalf("expected extension enable to wait for attached object usage, got %d usage calls", objectUsageCalls)
	}
	if !strings.Contains(output, `"object_id": "ext-1"`) && !strings.Contains(output, `"object_id":"ext-1"`) {
		t.Fatalf("expected verified object usage binding in output, got %s", output)
	}
}

func TestExecuteProfileCreateFromFlagsLocalizesForCountry(t *testing.T) {
	t.Setenv(mlx.EnvToken, "test-token")

	var createBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/profile/create":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("ReadAll returned error: %v", err)
			}
			createBody = string(body)
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"ids":["profile-1"]}}`)
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
		return Execute([]string{"--config", configPath, "profile", "create", "--name", "x", "--country", "de", "--browser", "mimic", "--os", "windows", "--folder-id", "f"})
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	checks := []string{
		`"name":"x"`,
		`"browser_type":"mimic"`,
		`"os_type":"windows"`,
		`"folder_id":"f"`,
		`"locale":"de-DE"`,
		`"zone":"Europe/Berlin"`,
		`"localization_masking":"custom"`,
	}
	for _, check := range checks {
		if !strings.Contains(createBody, check) {
			t.Fatalf("expected create request body to contain %s, got %s", check, createBody)
		}
	}
	if !strings.Contains(output, `"ids"`) {
		t.Fatalf("expected create output to contain ids, got %s", output)
	}
}

func TestExecuteProfileCreateStartLaunchesCreatedProfile(t *testing.T) {
	t.Setenv(mlx.EnvToken, "test-token")

	var startHit bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/profile/create":
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"ids":["profile-1"]}}`)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v2/profile/f/f/p/profile-1/start"):
			startHit = true
			fmt.Fprint(w, `{"status":{"http_code":200,"message":"Profile started successfully"},"data":{"browser_type":"mimic","core_version":137,"id":"profile-1","is_quick":false,"port":"55513"}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	// Continuity disabled keeps the test network-free (no proxy backend hit).
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
  },
  "defaults": {
    "launcher": {
      "automation_type": "playwright"
    },
    "proxy": {
      "proxy_continuity": {
        "enabled": false
      }
    }
  }
}`, server.URL, server.URL))

	output, err := captureCLIStdout(func() error {
		return Execute([]string{"--config", configPath, "profile", "create", "--name", "x", "--browser", "mimic", "--os", "windows", "--folder-id", "f", "--start"})
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !startHit {
		t.Fatalf("expected --start to launch the created profile, but the start endpoint was not hit")
	}
	if !strings.Contains(output, `"create"`) || !strings.Contains(output, `"start"`) {
		t.Fatalf("expected combined create+start output, got %s", output)
	}
	if !strings.Contains(output, `"port": "55513"`) {
		t.Fatalf("expected start response in output, got %s", output)
	}
}

func TestExecuteProfileCreateStartFailsClosedWhenProxyCheckFails(t *testing.T) {
	t.Setenv(mlx.EnvToken, "test-token")

	var startHit bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/profile/create":
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"ids":["profile-1"]}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/profile/metas":
			// ensureProxyBeforeStart fetches the created profile's meta.
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[{"id":"profile-1","name":"x","folder_id":"f","browser_type":"mimic","core_version":137,"os_type":"windows","workspace_id":"ws-1","created_at":"2026-04-20T00:00:00Z","created_by":"me@example.com","last_update_at":"2026-04-20T00:00:00Z","last_updated_by":"me@example.com","status":"ready","parameters":{"storage":{"is_local":false}}}]}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/v1/user":
			// Proxy backend unreachable -> fail-closed before launch.
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"status":{"http_code":500,"message":"proxy backend unavailable"}}`)
		case strings.HasPrefix(r.URL.Path, "/api/v2/profile/f/") && strings.HasSuffix(r.URL.Path, "/start"):
			startHit = true
			t.Fatalf("start endpoint must NEVER be hit when the proxy check fails (fail-closed): %s %s", r.Method, r.URL.Path)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	configPath := writeRuntimeConfigFile(t, fmt.Sprintf(`{
  "version": "1",
  "endpoints": {
    "base_url": %q,
    "launcher_url": %q,
    "proxy_url": %q
  },
  "output": {
    "format": "json"
  },
  "retry": {
    "enabled": false
  },
  "defaults": {
    "launcher": {
      "automation_type": "playwright"
    },
    "proxy": {
      "proxy_continuity": {
        "enabled": true,
        "latency_threshold_ms": 2000,
        "latency_hard_cap_ms": 3000,
        "candidates_per_round": 3,
        "check_targets": ["http://127.0.0.1:1"],
        "check_timeout": "1s"
      }
    }
  }
}`, server.URL, server.URL, server.URL))

	_, err := captureCLIStdout(func() error {
		return Execute([]string{"--config", configPath, "profile", "create", "--name", "x", "--browser", "mimic", "--os", "windows", "--folder-id", "f", "--start"})
	})
	if err == nil {
		t.Fatalf("expected fail-closed error when proxy check fails on create --start, got nil")
	}
	if !strings.Contains(err.Error(), "proxy check failed") {
		t.Fatalf("expected pre-launch proxy check error, got %v", err)
	}
	if startHit {
		t.Fatalf("start endpoint was hit despite failed proxy check on create --start (NOT fail-closed)")
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
