package mlx

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// AutomationType describes the launcher automation mode.
type AutomationType string

const (
	AutomationSelenium   AutomationType = "selenium"
	AutomationPlaywright AutomationType = "playwright"
	AutomationPuppeteer  AutomationType = "puppeteer"
	AutomationRod        AutomationType = "rod"
)

// LauncherService manages running profiles via the local launcher.
type LauncherService interface {
	Start(context.Context, string, string, StartProfileOptions) (*StartProfileResponse, *Response, error)
	StartQuick(context.Context, *StartQuickProfileRequest) (*StartQuickProfileResponse, *Response, error)
	SaveQuick(context.Context, *SaveQuickProfileRequest) (*EmptyDataResponse, *Response, error)
	Stop(context.Context, string) (*EmptyDataResponse, *Response, error)
	StopAll(context.Context, StopAllProfilesOptions) (*EmptyDataResponse, *Response, error)
	Health(context.Context) (*LauncherHealthResponse, *Response, error)
	Status(context.Context, string) (*ProfileRuntimeStatusResponse, *Response, error)
	WaitForRunning(context.Context, string, PollOptions) (*ProfileRuntimeStatusResponse, *Response, error)
	Statuses(context.Context) (*AllProfileStatusesResponse, *Response, error)
	QuickStatuses(context.Context) (*QuickProfileStatusesResponse, *Response, error)
	Version(context.Context) (*LauncherVersionResponse, *Response, error)
	ValidateProxy(context.Context, *ValidateProxyRequest) (*ValidateProxyResponse, *Response, error)
}

// LauncherServiceOp is the concrete launcher service.
type LauncherServiceOp struct {
	client *Client
}

// StartProfileOptions configures profile start requests.
type StartProfileOptions struct {
	AutomationType AutomationType
	Headless       bool
	StrictMode     bool
}

// StopAllProfilesOptions controls stop-all behavior.
type StopAllProfilesOptions struct {
	Type string
}

// StartProfileResponse contains the launcher port and runtime profile info.
type StartProfileResponse struct {
	Status Status             `json:"status"`
	Data   StartedProfileData `json:"data"`
}

func (r *StartProfileResponse) GetStatus() Status { return r.Status }

// StartedProfileData contains launcher startup output.
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

// StartQuickProfileRequest configures a quick profile start.
type StartQuickProfileRequest struct {
	BrowserType      string             `json:"browser_type,omitempty"`
	OSType           string             `json:"os_type,omitempty"`
	ScriptFile       string             `json:"script_file,omitempty"`
	AutomationType   AutomationType     `json:"automation,omitempty"`
	CoreVersion      int                `json:"core_version,omitempty"`
	CoreMinorVersion int                `json:"core_minor_version,omitempty"`
	Headless         bool               `json:"is_headless,omitempty"`
	Parameters       *ProfileParameters `json:"parameters,omitempty"`
	CustomStartURLs  []string           `json:"custom_start_urls,omitempty"`
}

// StartQuickProfileResponse contains the launcher port and runtime info for a quick profile.
type StartQuickProfileResponse struct {
	Status Status             `json:"status"`
	Data   StartedProfileData `json:"data"`
}

func (r *StartQuickProfileResponse) GetStatus() Status { return r.Status }

// SaveQuickProfileRequest configures saving quick profiles.
type SaveQuickProfileRequest struct {
	Data []SaveQuickProfileItem `json:"data"`
}

// SaveQuickProfileItem identifies a quick profile to save.
type SaveQuickProfileItem struct {
	ProfileID string `json:"profile_id"`
}

// ValidateProxyRequest configures proxy validation.
type ValidateProxyRequest struct {
	Type     string `json:"type"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// ValidateProxyResponse contains proxy validation results.
type ValidateProxyResponse struct {
	Status Status              `json:"status"`
	Data   ProxyValidationData `json:"data"`
}

func (r *ValidateProxyResponse) GetStatus() Status { return r.Status }

// ProxyValidationData contains geolocation and accuracy data for a proxy.
type ProxyValidationData struct {
	Accuracy    float64 `json:"accuracy"`
	Altitude    float64 `json:"altitude"`
	CountryCode string  `json:"country_code"`
	IP          string  `json:"ip"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	Timezone    string  `json:"timezone"`
}

// ProfileRuntimeStatusResponse contains a single profile status.
type ProfileRuntimeStatusResponse struct {
	Status Status               `json:"status"`
	Data   ProfileRuntimeStatus `json:"data"`
}

func (r *ProfileRuntimeStatusResponse) GetStatus() Status { return r.Status }

// ProfileRuntimeStatus describes the running state of one profile.
type ProfileRuntimeStatus struct {
	ProfileID      string `json:"profile_id"`
	Name           string `json:"name"`
	Status         string `json:"status"`
	BrowserType    string `json:"browser_type"`
	CoreVersion    int    `json:"core_version"`
	FolderID       string `json:"folder_id"`
	WorkspaceID    string `json:"workspace_id"`
	InUseBy        string `json:"in_use_by"`
	LastLaunchedAt string `json:"last_launched_at"`
	LastLaunchedBy string `json:"last_launched_by"`
	LastLaunchedOn string `json:"last_launched_on"`
	Message        string `json:"message"`
	IsQuick        bool   `json:"is_quick"`
	Timestamp      int64  `json:"timestamp"`
}

// AllProfileStatusesResponse contains all profile runtime states.
type AllProfileStatusesResponse struct {
	Status Status                 `json:"status"`
	Data   AllProfileStatusesData `json:"data"`
}

func (r *AllProfileStatusesResponse) GetStatus() Status { return r.Status }

// AllProfileStatusesData wraps all launcher states.
type AllProfileStatusesData struct {
	ActiveCounter LauncherActiveCounter           `json:"active_counter"`
	States        map[string]ProfileRuntimeStatus `json:"states"`
}

// LauncherActiveCounter reports running profile counts by storage type.
type LauncherActiveCounter struct {
	Cloud int `json:"cloud"`
	Local int `json:"local"`
	Quick int `json:"quick"`
}

// QuickProfileStatusesResponse contains quick profile states.
type QuickProfileStatusesResponse struct {
	Status Status                   `json:"status"`
	Data   QuickProfileStatusesData `json:"data"`
}

func (r *QuickProfileStatusesResponse) GetStatus() Status { return r.Status }

// QuickProfileStatusesData wraps quick profile states.
type QuickProfileStatusesData struct {
	ActiveCounter int                                  `json:"active_counter"`
	States        map[string]QuickProfileRuntimeStatus `json:"states"`
}

// QuickProfileRuntimeStatus describes quick profile state.
type QuickProfileRuntimeStatus struct {
	Name        string `json:"name"`
	Status      string `json:"status"`
	Message     string `json:"message"`
	BrowserType string `json:"browser_type"`
	IsQuick     bool   `json:"is_quick"`
	Timestamp   int64  `json:"timestamp"`
}

// LauncherVersionResponse returns launcher version info.
type LauncherVersionResponse struct {
	Status Status              `json:"status"`
	Data   LauncherVersionData `json:"data"`
}

func (r *LauncherVersionResponse) GetStatus() Status { return r.Status }

// LauncherVersionData contains version info.
type LauncherVersionData struct {
	Env     string `json:"env"`
	Version string `json:"version"`
}

// LauncherHealthResponse reports whether the local launcher is reachable.
//
// Multilogin X does not currently expose a dedicated health endpoint in the
// checked-in Postman collection, so this helper probes `/api/v1/version` as the
// launcher liveness/readiness check.
type LauncherHealthResponse struct {
	Status Status             `json:"status"`
	Data   LauncherHealthData `json:"data"`
}

func (r *LauncherHealthResponse) GetStatus() Status { return r.Status }

// LauncherHealthData contains the launcher readiness state.
type LauncherHealthData struct {
	Alive   bool   `json:"alive"`
	Env     string `json:"env,omitempty"`
	Version string `json:"version,omitempty"`
}

func (s *LauncherServiceOp) Start(ctx context.Context, folderID, profileID string, opts StartProfileOptions) (*StartProfileResponse, *Response, error) {
	if folderID == "" {
		return nil, nil, NewArgError("folderID", "it must not be empty")
	}
	if profileID == "" {
		return nil, nil, NewArgError("profileID", "it must not be empty")
	}
	launcherAutomation := normalizeLauncherAutomation(opts.AutomationType)
	values := url.Values{}
	if launcherAutomation != "" {
		values.Set("automation_type", string(launcherAutomation))
	}
	values.Set("headless_mode", fmt.Sprintf("%t", opts.Headless))
	path := fmt.Sprintf("/api/v2/profile/f/%s/p/%s/start?%s", url.PathEscape(folderID), url.PathEscape(profileID), values.Encode())
	req, err := s.client.newLauncherRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	if opts.StrictMode {
		req.Header.Set("X-Strict-Mode", "true")
	}
	out := new(StartProfileResponse)
	resp, err := s.client.do(req, out)
	if err == nil {
		enrichStartedProfileData(&out.Data, opts.AutomationType, launcherAutomation)
	}
	return out, resp, err
}

func (s *LauncherServiceOp) Stop(ctx context.Context, profileID string) (*EmptyDataResponse, *Response, error) {
	if profileID == "" {
		return nil, nil, NewArgError("profileID", "it must not be empty")
	}
	path := fmt.Sprintf("/api/v1/profile/stop/p/%s", url.PathEscape(profileID))
	req, err := s.client.newLauncherRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	out := new(EmptyDataResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *LauncherServiceOp) StopAll(ctx context.Context, opts StopAllProfilesOptions) (*EmptyDataResponse, *Response, error) {
	path := "/api/v1/profile/stop_all"
	if opts.Type != "" {
		path = fmt.Sprintf("%s?type=%s", path, url.QueryEscape(opts.Type))
	}
	req, err := s.client.newLauncherRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	out := new(EmptyDataResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *LauncherServiceOp) Health(ctx context.Context) (*LauncherHealthResponse, *Response, error) {
	version, resp, err := s.Version(ctx)
	if err != nil {
		return nil, resp, err
	}
	out := &LauncherHealthResponse{
		Status: version.Status,
		Data: LauncherHealthData{
			Alive:   true,
			Env:     version.Data.Env,
			Version: version.Data.Version,
		},
	}
	if resp != nil {
		resp.Status = out.Status
		resp.Raw = out.Data
	}
	return out, resp, nil
}

func (s *LauncherServiceOp) Status(ctx context.Context, profileID string) (*ProfileRuntimeStatusResponse, *Response, error) {
	if profileID == "" {
		return nil, nil, NewArgError("profileID", "it must not be empty")
	}
	path := fmt.Sprintf("/api/v1/profile/status/p/%s", url.PathEscape(profileID))
	req, err := s.client.newLauncherRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	out := new(ProfileRuntimeStatusResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *LauncherServiceOp) WaitForRunning(ctx context.Context, profileID string, opts PollOptions) (*ProfileRuntimeStatusResponse, *Response, error) {
	if profileID == "" {
		return nil, nil, NewArgError("profileID", "it must not be empty")
	}
	return pollUntil(ctx, opts, fmt.Sprintf("profile %s did not reach running status", profileID), func(ctx context.Context) (*ProfileRuntimeStatusResponse, *Response, error) {
		return s.Status(ctx, profileID)
	}, func(resp *ProfileRuntimeStatusResponse) bool {
		if resp == nil {
			return false
		}
		status := resp.Data.Status
		return status == "browser_running" || strings.Contains(strings.ToLower(status), "running")
	}, func(resp *ProfileRuntimeStatusResponse) string {
		if resp == nil {
			return ""
		}
		return resp.Data.Status
	})
}

func (s *LauncherServiceOp) Statuses(ctx context.Context) (*AllProfileStatusesResponse, *Response, error) {
	req, err := s.client.newLauncherRequest(ctx, http.MethodGet, "/api/v1/profile/statuses", nil)
	if err != nil {
		return nil, nil, err
	}
	out := new(AllProfileStatusesResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *LauncherServiceOp) QuickStatuses(ctx context.Context) (*QuickProfileStatusesResponse, *Response, error) {
	req, err := s.client.newLauncherRequest(ctx, http.MethodGet, "/api/v1/profile/quick/statuses", nil)
	if err != nil {
		return nil, nil, err
	}
	out := new(QuickProfileStatusesResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *LauncherServiceOp) Version(ctx context.Context) (*LauncherVersionResponse, *Response, error) {
	req, err := s.client.newLauncherRequest(ctx, http.MethodGet, "/api/v1/version", nil)
	if err != nil {
		return nil, nil, err
	}
	out := new(LauncherVersionResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *LauncherServiceOp) StartQuick(ctx context.Context, reqBody *StartQuickProfileRequest) (*StartQuickProfileResponse, *Response, error) {
	if reqBody == nil {
		return nil, nil, NewArgError("reqBody", "it must not be nil")
	}
	reqBodyCopy := *reqBody
	reqBodyCopy.AutomationType = normalizeLauncherAutomation(reqBodyCopy.AutomationType)
	req, err := s.client.newLauncherRequest(ctx, http.MethodPost, "/api/v3/profile/quick", &reqBodyCopy)
	if err != nil {
		return nil, nil, err
	}
	out := new(StartQuickProfileResponse)
	resp, err := s.client.do(req, out)
	if err == nil {
		enrichStartedProfileData(&out.Data, reqBody.AutomationType, reqBodyCopy.AutomationType)
	}
	return out, resp, err
}

func (s *LauncherServiceOp) SaveQuick(ctx context.Context, reqBody *SaveQuickProfileRequest) (*EmptyDataResponse, *Response, error) {
	if reqBody == nil {
		return nil, nil, NewArgError("reqBody", "it must not be nil")
	}
	req, err := s.client.newLauncherRequest(ctx, http.MethodPost, "/api/v1/profile/quick/save", reqBody)
	if err != nil {
		return nil, nil, err
	}
	out := new(EmptyDataResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *LauncherServiceOp) ValidateProxy(ctx context.Context, reqBody *ValidateProxyRequest) (*ValidateProxyResponse, *Response, error) {
	if reqBody == nil {
		return nil, nil, NewArgError("reqBody", "it must not be nil")
	}
	req, err := s.client.newLauncherRequest(ctx, http.MethodPost, "/api/v1/proxy/validate", reqBody)
	if err != nil {
		return nil, nil, err
	}
	out := new(ValidateProxyResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}
