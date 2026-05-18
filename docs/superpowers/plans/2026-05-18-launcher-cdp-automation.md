# Launcher CDP Automation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> superpowers:subagent-driven-development (recommended) or
> superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `AutomationRod` reliably use Playwright/CDP under the hood, add
CDP/Rod endpoint helpers, add a higher-level automation workflow, and verify
the change with TDD plus a focused quality pass.

**Architecture:** Keep `AutomationRod` as a public semantic alias, normalize it
to launcher `playwright` in one place, and centralize CDP endpoint resolution
in one helper file. Surface the ready-to-use endpoint data through both the
launcher response model and a workflow-level result so consumers stop building
URLs by hand.

**Tech Stack:** Go 1.26, `httptest`, existing SDK polling/workflow helpers,
local launcher `/json/version` CDP endpoint resolution, `gofmt`, `go test`.

---

### Task 1: Lock the bug and helper contract with failing tests

**Files:**
- Create: `automation_endpoints_test.go`
- Modify: `launcher_test.go`
- Test: `automation_endpoints_test.go`

- [ ] **Step 1: Write the failing launcher alias test**

```go
func TestLauncherStartUsesPlaywrightForRod(t *testing.T) {
	server, httpClient := testutil.NewServer(t, func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		if got := r.URL.Query().Get("automation_type"); got != "playwright" {
			t.Fatalf("expected automation_type=playwright, got %q", got)
		}
		fmt.Fprint(
			w,
			`{"status":{"http_code":200,"message":"ok"},"data":{"id":"profile-1","port":"55513"}}`,
		)
	})

	client, err := New(
		WithToken("test-token"),
		WithHTTPClient(httpClient),
		WithLauncherURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, _, err := client.Launcher.Start(
		context.Background(),
		"folder-1",
		"profile-1",
		StartProfileOptions{AutomationType: AutomationRod},
	)
	if err != nil {
		t.Fatalf("Launcher.Start returned error: %v", err)
	}
	if resp.Data.RequestedAutomation != AutomationRod {
		t.Fatalf(
			"expected requested automation %q, got %q",
			AutomationRod,
			resp.Data.RequestedAutomation,
		)
	}
	if resp.Data.LauncherAutomation != AutomationPlaywright {
		t.Fatalf(
			"expected launcher automation %q, got %q",
			AutomationPlaywright,
			resp.Data.LauncherAutomation,
		)
	}
}
```

- [ ] **Step 2: Write the failing endpoint helper tests**

```go
func TestStartedProfileDataResolveCDPWebSocketURL(t *testing.T) {
	cdpServer := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		if r.URL.Path != "/json/version" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		fmt.Fprint(
			w,
			`{"webSocketDebuggerUrl":"ws://127.0.0.1:45471/devtools/browser/demo"}`,
		)
	}))
	t.Cleanup(cdpServer.Close)

	cdpURL, err := url.Parse(cdpServer.URL)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	data := StartedProfileData{
		Port:               cdpURL.Port(),
		RequestedAutomation: AutomationRod,
		LauncherAutomation:  AutomationPlaywright,
	}

	controlURL, err := data.ResolveRodControlURL(context.Background())
	if err != nil {
		t.Fatalf("ResolveRodControlURL returned error: %v", err)
	}
	if controlURL != "ws://127.0.0.1:45471/devtools/browser/demo" {
		t.Fatalf("unexpected control url: %s", controlURL)
	}
}

func TestStartedProfileDataResolveCDPWebSocketURLEmptyPort(t *testing.T) {
	data := StartedProfileData{
		RequestedAutomation: AutomationRod,
		LauncherAutomation:  AutomationPlaywright,
	}

	_, err := data.ResolveCDPWebSocketURL(context.Background())
	if err == nil {
		t.Fatal("expected error for empty port")
	}

	var endpointErr *AutomationEndpointError
	if !errors.As(err, &endpointErr) {
		t.Fatalf("expected AutomationEndpointError, got %T", err)
	}
}
```

- [ ] **Step 3: Run the focused tests to verify they fail**

Run:

```powershell
go test . -run "TestLauncherStartUsesPlaywrightForRod|TestStartedProfileDataResolveCDPWebSocketURL|TestStartedProfileDataResolveCDPWebSocketURLEmptyPort"
```

Expected: FAIL because `RequestedAutomation`, `LauncherAutomation`,
`ResolveCDPWebSocketURL`, `ResolveRodControlURL`, and
`AutomationEndpointError` do not exist yet.

- [ ] **Step 4: Commit the red test state if using explicit red/green commits**

```powershell
git add launcher_test.go automation_endpoints_test.go
git commit -m "test: lock launcher rod alias behavior"
```

If you prefer to keep the failing-test phase uncommitted locally, skip the
commit and move directly to Task 2.

### Task 2: Implement launcher normalization and endpoint helpers

**Files:**
- Create: `automation_endpoints.go`
- Modify: `launcher.go`
- Test: `launcher_test.go`, `automation_endpoints_test.go`

- [ ] **Step 1: Add a focused helper file for automation normalization and CDP resolution**

```go
package mlx

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type AutomationEndpointError struct {
	RequestedAutomation AutomationType
	LauncherAutomation  AutomationType
	Port                string
	Message             string
}

func (e *AutomationEndpointError) Error() string {
	return fmt.Sprintf(
		"launcher did not return a usable cdp endpoint: %s (requested=%s launcher=%s port=%q)",
		e.Message,
		e.RequestedAutomation,
		e.LauncherAutomation,
		e.Port,
	)
}

type cdpVersionResponse struct {
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

func normalizeLauncherAutomation(
	requested AutomationType,
) AutomationType {
	if requested == AutomationRod {
		return AutomationPlaywright
	}
	return requested
}

func enrichStartedProfileData(
	data *StartedProfileData,
	requested AutomationType,
	launcher AutomationType,
) {
	if data == nil {
		return
	}
	data.RequestedAutomation = requested
	data.LauncherAutomation = launcher
	data.CDPPort = strings.TrimSpace(data.Port)
}
```

- [ ] **Step 2: Add endpoint resolution methods and error handling**

```go
func (d StartedProfileData) ResolveCDPWebSocketURL(
	ctx context.Context,
) (string, error) {
	port := strings.TrimSpace(d.CDPPort)
	if port == "" {
		port = strings.TrimSpace(d.Port)
	}
	if port == "" {
		return "", &AutomationEndpointError{
			RequestedAutomation: d.RequestedAutomation,
			LauncherAutomation:  d.LauncherAutomation,
			Port:                d.Port,
			Message:             "empty cdp port",
		}
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("http://127.0.0.1:%s/json/version", port),
		nil,
	)
	if err != nil {
		return "", err
	}

	httpClient := &http.Client{Timeout: 3 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var payload cdpVersionResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	if strings.TrimSpace(payload.WebSocketDebuggerURL) == "" {
		return "", &AutomationEndpointError{
			RequestedAutomation: d.RequestedAutomation,
			LauncherAutomation:  d.LauncherAutomation,
			Port:                port,
			Message:             "webSocketDebuggerUrl is empty",
		}
	}
	return payload.WebSocketDebuggerURL, nil
}

func (d StartedProfileData) ResolveRodControlURL(
	ctx context.Context,
) (string, error) {
	return d.ResolveCDPWebSocketURL(ctx)
}
```

- [ ] **Step 3: Enrich launcher start responses with requested and normalized automation**

```go
type StartedProfileData struct {
	BrowserType         string         `json:"browser_type"`
	CoreVersion         int            `json:"core_version"`
	ID                  string         `json:"id"`
	IsQuick             bool           `json:"is_quick"`
	Port                string         `json:"port"`
	RequestedAutomation AutomationType `json:"requested_automation,omitempty"`
	LauncherAutomation  AutomationType `json:"launcher_automation,omitempty"`
	CDPPort             string         `json:"cdp_port,omitempty"`
}

func (s *LauncherServiceOp) Start(
	ctx context.Context,
	folderID string,
	profileID string,
	opts StartProfileOptions,
) (*StartProfileResponse, *Response, error) {
	launcherAutomation := normalizeLauncherAutomation(opts.AutomationType)
	values := url.Values{}
	if launcherAutomation != "" {
		values.Set("automation_type", string(launcherAutomation))
	}
	values.Set("headless_mode", fmt.Sprintf("%t", opts.Headless))

	// existing request build and request execution

	if err == nil && out != nil {
		enrichStartedProfileData(
			&out.Data,
			opts.AutomationType,
			launcherAutomation,
		)
	}
	return out, resp, err
}
```

- [ ] **Step 4: Run the focused tests to verify they pass**

Run:

```powershell
go test . -run "TestLauncherStartUsesPlaywrightForRod|TestStartedProfileDataResolveCDPWebSocketURL|TestStartedProfileDataResolveCDPWebSocketURLEmptyPort"
```

Expected: PASS.

- [ ] **Step 5: Format and commit the launcher/helper implementation**

Run:

```powershell
gofmt -w launcher.go automation_endpoints.go launcher_test.go automation_endpoints_test.go
git add launcher.go automation_endpoints.go launcher_test.go automation_endpoints_test.go
git commit -m "feat: add launcher cdp endpoint helpers"
```

### Task 3: Add and implement the high-level automation workflow

**Files:**
- Modify: `workflows.go`
- Modify: `integration/workflows_test.go`
- Test: `integration/workflows_test.go`

- [ ] **Step 1: Write the failing integration test for automation-by-name workflow**

```go
func TestWorkflowStartProfileAutomationByName(t *testing.T) {
	cdpServer := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		if r.URL.Path != "/json/version" {
			t.Fatalf("unexpected cdp path: %s", r.URL.Path)
		}
		fmt.Fprint(
			w,
			`{"webSocketDebuggerUrl":"ws://127.0.0.1:45551/devtools/browser/demo"}`,
		)
	}))
	t.Cleanup(cdpServer.Close)

	cdpURL, err := url.Parse(cdpServer.URL)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	server, httpClient := testutil.NewServer(t, func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/profile/search":
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[{"id":"profile-1","name":"Demo","folder_id":"folder-1","browser_type":"mimic","os_type":"windows","core_version":137}],"total_count":1}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/profile/metas":
			fmt.Fprintf(w, `{"status":{"http_code":200,"message":""},"data":{"profiles":[%s]}}`, verifiedProfileMetaJSON("profile-1", "Demo", "folder-1"))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/profile/f/folder-1/p/profile-1/start":
			if got := r.URL.Query().Get("automation_type"); got != "playwright" {
				t.Fatalf("expected normalized automation_type=playwright, got %q", got)
			}
			fmt.Fprintf(
				w,
				`{"status":{"http_code":200,"message":"ok"},"data":{"id":"profile-1","browser_type":"mimic","core_version":137,"port":"%s"}}`,
				cdpURL.Port(),
			)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/profile/status/p/profile-1":
			fmt.Fprint(w, `{"status":{"http_code":200,"message":""},"data":{"profile_id":"profile-1","name":"Demo","status":"browser_running","browser_type":"mimic","core_version":137,"folder_id":"folder-1"}}`)
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

	result, err := client.Workflows.StartProfileAutomationByName(
		context.Background(),
		"Demo",
		mlx.StartProfileAutomationByNameOptions{
			StartOptions: mlx.StartProfileOptions{
				AutomationType: mlx.AutomationRod,
			},
			WaitForRunning: true,
			PollOptions: mlx.PollOptions{
				InitialInterval: time.Millisecond,
				MaxInterval:     time.Millisecond,
				Timeout:         time.Second,
			},
		},
	)
	if err != nil {
		t.Fatalf("StartProfileAutomationByName returned error: %v", err)
	}
	if result.RequestedAutomation != mlx.AutomationRod {
		t.Fatalf("unexpected requested automation: %q", result.RequestedAutomation)
	}
	if result.LauncherAutomation != mlx.AutomationPlaywright {
		t.Fatalf("unexpected launcher automation: %q", result.LauncherAutomation)
	}
	if result.CDPWebSocketURL == "" || result.RodControlURL == "" {
		t.Fatalf("expected resolved automation endpoints, got %#v", result)
	}
}
```

- [ ] **Step 2: Run the targeted integration test to verify it fails**

Run:

```powershell
go test ./integration -run TestWorkflowStartProfileAutomationByName -v
```

Expected: FAIL because `StartProfileAutomationByName`,
`StartProfileAutomationByNameOptions`, and the workflow result type do not
exist yet.

- [ ] **Step 3: Implement the workflow interface, options, result, and orchestration**

```go
type WorkflowService interface {
	// existing methods
	StartProfileAutomationByName(
		context.Context,
		string,
		StartProfileAutomationByNameOptions,
	) (*StartedProfileAutomationWorkflowResult, error)
}

type StartProfileAutomationByNameOptions struct {
	FindOptions    *FindProfileOptions
	StartOptions   StartProfileOptions
	WaitForRunning bool
	PollOptions    PollOptions
}

type StartedProfileAutomationWorkflowResult struct {
	Profile             *Profile
	StartResponse       *StartProfileResponse
	RuntimeStatus       *ProfileRuntimeStatusResponse
	RequestedAutomation AutomationType
	LauncherAutomation  AutomationType
	CDPPort             string
	CDPWebSocketURL     string
	RodControlURL       string
}

func (s *WorkflowServiceOp) StartProfileAutomationByName(
	ctx context.Context,
	profileName string,
	opts StartProfileAutomationByNameOptions,
) (*StartedProfileAutomationWorkflowResult, error) {
	verified, err := s.FindProfileByNameVerified(
		ctx,
		profileName,
		FindProfileByNameVerifiedOptions{FindOptions: opts.FindOptions},
	)
	if err != nil {
		return nil, err
	}

	startResp, _, err := s.client.Launcher.Start(
		ctx,
		verified.Profile.FolderID,
		verified.Profile.ID,
		opts.StartOptions,
	)
	if err != nil {
		return nil, err
	}

	cdpWSURL, err := startResp.Data.ResolveCDPWebSocketURL(ctx)
	if err != nil {
		return nil, err
	}

	result := &StartedProfileAutomationWorkflowResult{
		Profile:             verified.Profile,
		StartResponse:       startResp,
		RequestedAutomation: startResp.Data.RequestedAutomation,
		LauncherAutomation:  startResp.Data.LauncherAutomation,
		CDPPort:             startResp.Data.CDPPort,
		CDPWebSocketURL:     cdpWSURL,
		RodControlURL:       cdpWSURL,
	}

	if opts.WaitForRunning {
		statusResp, _, err := s.client.Launcher.WaitForRunning(
			ctx,
			verified.Profile.ID,
			opts.PollOptions,
		)
		if err != nil {
			return nil, err
		}
		result.RuntimeStatus = statusResp
	}

	return result, nil
}
```

- [ ] **Step 4: Run the integration tests to verify the workflow passes**

Run:

```powershell
go test ./integration -run TestWorkflowStartProfileAutomationByName -v
```

Expected: PASS.

- [ ] **Step 5: Commit the workflow slice**

Run:

```powershell
gofmt -w workflows.go integration/workflows_test.go
git add workflows.go integration/workflows_test.go
git commit -m "feat: add automation workflow by profile name"
```

### Task 4: Update docs, live coverage, and run the quality pass

**Files:**
- Modify: `docs/rod-example.md`
- Modify: `docs/consumer-guide.md`
- Modify: `README.md`
- Modify: `e2e/e2e_test.go`
- Modify: `launcher.go`, `workflows.go`, `automation_endpoints.go`
- Test: `go test ./...`

- [ ] **Step 1: Rewrite the Rod example to use the SDK helper instead of manual fallback**

```go
started, _, err := client.Launcher.Start(ctx, folderID, profileID, mlx.StartProfileOptions{
	AutomationType: mlx.AutomationRod,
})
if err != nil {
	log.Fatalf("start profile: %v", err)
}

controlURL, err := started.Data.ResolveRodControlURL(ctx)
if err != nil {
	log.Fatalf("resolve rod control url: %v", err)
}

browser := rod.New().
	ControlURL(controlURL).
	NoDefaultDevice().
	MustConnect()
```

Docs text to enforce:

- `AutomationRod` is a semantic alias backed by launcher `playwright`
- callers no longer need to manually retry with `playwright`
- the SDK helper resolves the DevTools WebSocket endpoint

- [ ] **Step 2: Update the live E2E test to verify the SDK contract**

```go
started, _, err := client.Launcher.Start(
	ctx,
	folderID,
	profileID,
	StartProfileOptions{AutomationType: AutomationRod},
)
if err != nil {
	t.Fatalf("Launcher.Start returned error: %v", err)
}

	if started.Data.LauncherAutomation != AutomationPlaywright {
		t.Fatalf(
			"expected launcher automation %q, got %q",
			AutomationPlaywright,
			started.Data.LauncherAutomation,
		)
	}

controlURL, err := started.Data.ResolveRodControlURL(ctx)
if err != nil {
	t.Fatalf("ResolveRodControlURL returned error: %v", err)
}

browser := rod.New().ControlURL(controlURL).NoDefaultDevice()
```

- [ ] **Step 3: Run a focused cleanup search for duplicates and obsolete fallback logic**

Run:

```powershell
rg -n "ResolveURL\\(|playwright fallback|automation_type=rod|empty port" .
```

Expected:

- no remaining primary-path docs or tests that tell callers to manually retry
- no duplicate endpoint normalization logic outside the new helper
- no stale comments claiming Rod is a distinct reliable launcher mode

- [ ] **Step 4: Format, run the full test suite, and optionally run the live E2E check**

Run:

```powershell
gofmt -w launcher.go automation_endpoints.go workflows.go launcher_test.go automation_endpoints_test.go integration/workflows_test.go e2e/e2e_test.go
go test ./...
```

Expected: PASS.

Optional live verification if environment is available:

```powershell
$env:MLX_RUN_E2E="1"; go test -tags=e2e ./e2e -run TestE2ERodConnection -count=1 -v
```

Expected: PASS with `AutomationRod` requested and `AutomationPlaywright`
normalized launcher mode.

- [ ] **Step 5: Commit docs, cleanup, and verification**

```powershell
git add README.md docs/rod-example.md docs/consumer-guide.md e2e/e2e_test.go launcher.go workflows.go automation_endpoints.go launcher_test.go automation_endpoints_test.go integration/workflows_test.go
git commit -m "docs: align rod automation with cdp helpers"
```
