package mlx

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
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
	Stop(context.Context, string) (*EmptyDataResponse, *Response, error)
	StopAll(context.Context, StopAllProfilesOptions) (*EmptyDataResponse, *Response, error)
	Health(context.Context) (*LauncherHealthResponse, *Response, error)
	Status(context.Context, string) (*ProfileRuntimeStatusResponse, *Response, error)
	Statuses(context.Context) (*AllProfileStatusesResponse, *Response, error)
	QuickStatuses(context.Context) (*QuickProfileStatusesResponse, *Response, error)
	Version(context.Context) (*LauncherVersionResponse, *Response, error)
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
	BrowserType string `json:"browser_type"`
	CoreVersion int    `json:"core_version"`
	ID          string `json:"id"`
	IsQuick     bool   `json:"is_quick"`
	Port        string `json:"port"`
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
	LastLaunchedBy string `json:"last_launched_by"`
	Message        string `json:"message"`
	IsQuick        bool   `json:"is_quick"`
}

// AllProfileStatusesResponse contains all profile runtime states.
type AllProfileStatusesResponse struct {
	Status Status                 `json:"status"`
	Data   AllProfileStatusesData `json:"data"`
}

func (r *AllProfileStatusesResponse) GetStatus() Status { return r.Status }

// AllProfileStatusesData wraps all launcher states.
type AllProfileStatusesData struct {
	States map[string]ProfileRuntimeStatus `json:"states"`
}

// QuickProfileStatusesResponse contains quick profile states.
type QuickProfileStatusesResponse struct {
	Status Status                   `json:"status"`
	Data   QuickProfileStatusesData `json:"data"`
}

func (r *QuickProfileStatusesResponse) GetStatus() Status { return r.Status }

// QuickProfileStatusesData wraps quick profile states.
type QuickProfileStatusesData struct {
	ActiveCounter any                                  `json:"active_counter"`
	States        map[string]QuickProfileRuntimeStatus `json:"states"`
}

// QuickProfileRuntimeStatus describes quick profile state.
type QuickProfileRuntimeStatus struct {
	Name        string `json:"name"`
	Status      string `json:"status"`
	Message     string `json:"message"`
	BrowserType string `json:"browser_type"`
	IsQuick     bool   `json:"is_quick"`
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
	values := url.Values{}
	if opts.AutomationType != "" {
		values.Set("automation_type", string(opts.AutomationType))
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
