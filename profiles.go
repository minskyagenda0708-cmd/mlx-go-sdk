package mlx

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// ProfilesService describes profile lifecycle operations.
type ProfilesService interface {
	Create(context.Context, *CreateProfileRequest) (*CreateProfileResponse, *Response, error)
	Search(context.Context, *SearchProfilesRequest) (*SearchProfilesResponse, *Response, error)
	FindByName(context.Context, string, *FindProfileOptions) (*Profile, *Response, error)
	Update(context.Context, *UpdateProfileRequest) (*EmptyDataResponse, *Response, error)
	Patch(context.Context, *PatchProfileRequest) (*EmptyDataResponse, *Response, error)
	Delete(context.Context, *DeleteProfilesRequest) (*EmptyDataResponse, *Response, error)
	Restore(context.Context, *RestoreProfilesRequest) (*EmptyDataResponse, *Response, error)
	Clone(context.Context, *CloneProfileRequest) (*CreateProfileResponse, *Response, error)
	Move(context.Context, *MoveProfilesRequest) (*EmptyDataResponse, *Response, error)
	GetMeta(context.Context, string) (*ProfileMeta, *Response, error)
	GetMetas(context.Context, *ProfileMetasRequest) (*ProfileMetasResponse, *Response, error)
	GetSummary(context.Context, string) (*ProfileSummaryResponse, *Response, error)
}

// ProfilesServiceOp is the concrete ProfilesService implementation.
type ProfilesServiceOp struct {
	client *Client
}

// CreateProfileRequest creates one or more profiles.
type CreateProfileRequest struct {
	Name             string             `json:"name"`
	BrowserType      string             `json:"browser_type"`
	FolderID         string             `json:"folder_id"`
	OSType           string             `json:"os_type"`
	CoreVersion      int                `json:"core_version,omitempty"`
	CoreMinorVersion int                `json:"core_minor_version,omitempty"`
	AutoUpdateCore   *bool              `json:"auto_update_core,omitempty"`
	Times            int                `json:"times,omitempty"`
	Notes            string             `json:"notes,omitempty"`
	Parameters       *ProfileParameters `json:"parameters,omitempty"`
	Tags             []string           `json:"tags,omitempty"`
}

// SearchProfilesRequest searches profiles.
type SearchProfilesRequest struct {
	IsRemoved   bool     `json:"is_removed"`
	Limit       int      `json:"limit"`
	Offset      int      `json:"offset"`
	SearchText  string   `json:"search_text"`
	StorageType string   `json:"storage_type"`
	FolderID    string   `json:"folder_id,omitempty"`
	BrowserType string   `json:"browser_type,omitempty"`
	OSType      string   `json:"os_type,omitempty"`
	OrderBy     string   `json:"order_by,omitempty"`
	Sort        string   `json:"sort,omitempty"`
	CoreVersion int      `json:"core_version,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// FindProfileOptions narrows convenience profile lookups.
type FindProfileOptions struct {
	IsRemoved   bool
	StorageType string
	FolderID    string
	BrowserType string
	OSType      string
	Limit       int
	Tags        []string
}

// UpdateProfileRequest fully updates a profile.
type UpdateProfileRequest struct {
	ProfileID        string             `json:"profile_id"`
	Name             string             `json:"name"`
	AutoUpdateCore   *bool              `json:"auto_update_core,omitempty"`
	CoreVersion      int                `json:"core_version,omitempty"`
	CoreMinorVersion int                `json:"core_minor_version,omitempty"`
	Parameters       *ProfileParameters `json:"parameters,omitempty"`
	Notes            string             `json:"notes,omitempty"`
	Tags             []string           `json:"tags,omitempty"`
}

// PatchProfileRequest partially updates a profile.
type PatchProfileRequest struct {
	ProfileID        string             `json:"profile_id"`
	Name             string             `json:"name,omitempty"`
	AutoUpdateCore   *bool              `json:"auto_update_core,omitempty"`
	CoreVersion      int                `json:"core_version,omitempty"`
	CoreMinorVersion int                `json:"core_minor_version,omitempty"`
	Proxy            *Proxy             `json:"proxy,omitempty"`
	CustomStartURLs  []string           `json:"custom_start_urls,omitempty"`
	Notes            string             `json:"notes,omitempty"`
	Parameters       *ProfileParameters `json:"parameters,omitempty"`
	Tags             []string           `json:"tags,omitempty"`
}

// DeleteProfilesRequest removes profiles.
type DeleteProfilesRequest struct {
	IDs         []string `json:"ids"`
	Permanently bool     `json:"permanently"`
}

// RestoreProfilesRequest restores soft-deleted profiles.
type RestoreProfilesRequest struct {
	IDs []string `json:"ids"`
}

// CloneProfileRequest clones a profile.
type CloneProfileRequest struct {
	ProfileID string `json:"profile_id"`
	Times     int    `json:"times"`
}

// MoveProfilesRequest moves profiles into another folder.
type MoveProfilesRequest struct {
	DestinationFolderID string   `json:"dest_folder_id"`
	IDs                 []string `json:"ids,omitempty"`
}

// ProfileMetasRequest requests metadata for profiles.
type ProfileMetasRequest struct {
	IDs []string `json:"ids"`
}

// ProfileParameters contains flags, storage, proxy, and fingerprint settings.
type ProfileParameters struct {
	Flags           *ProfileFlags `json:"flags,omitempty"`
	Storage         *Storage      `json:"storage,omitempty"`
	Fingerprint     *Fingerprint  `json:"fingerprint,omitempty"`
	Proxy           *Proxy        `json:"proxy,omitempty"`
	CustomStartURLs []string      `json:"custom_start_urls,omitempty"`
}

// Storage contains storage-related profile settings.
type Storage struct {
	IsLocal           bool `json:"is_local"`
	SaveServiceWorker bool `json:"save_service_worker,omitempty"`
}

// ProfileFlags contains the fingerprint masking flags used by profile create/update APIs.
type ProfileFlags struct {
	AudioMasking        string `json:"audio_masking,omitempty"`
	FontsMasking        string `json:"fonts_masking,omitempty"`
	GeolocationMasking  string `json:"geolocation_masking,omitempty"`
	GeolocationPopup    string `json:"geolocation_popup,omitempty"`
	GraphicsMasking     string `json:"graphics_masking,omitempty"`
	GraphicsNoise       string `json:"graphics_noise,omitempty"`
	LocalizationMasking string `json:"localization_masking,omitempty"`
	MediaDevicesMasking string `json:"media_devices_masking,omitempty"`
	NavigatorMasking    string `json:"navigator_masking,omitempty"`
	PortsMasking        string `json:"ports_masking,omitempty"`
	ProxyMasking        string `json:"proxy_masking,omitempty"`
	QuicMode            string `json:"quic_mode,omitempty"`
	ScreenMasking       string `json:"screen_masking,omitempty"`
	TimezoneMasking     string `json:"timezone_masking,omitempty"`
	WebRTCMasking       string `json:"webrtc_masking,omitempty"`
	CanvasNoise         string `json:"canvas_noise,omitempty"`
	StartupBehavior     string `json:"startup_behavior,omitempty"`
}

// Fingerprint contains typed browser fingerprint settings.
type Fingerprint struct {
	Navigator    *NavigatorFingerprint    `json:"navigator,omitempty"`
	Localization *LocalizationFingerprint `json:"localization,omitempty"`
	Timezone     *TimezoneFingerprint     `json:"timezone,omitempty"`
	Graphic      *GraphicFingerprint      `json:"graphic,omitempty"`
	WebRTC       *WebRTCFingerprint       `json:"webrtc,omitempty"`
	MediaDevices *MediaDevicesFingerprint `json:"media_devices,omitempty"`
	Screen       *ScreenFingerprint       `json:"screen,omitempty"`
	Geolocation  *GeolocationFingerprint  `json:"geolocation,omitempty"`
	Ports        []int                    `json:"ports,omitempty"`
	Fonts        []string                 `json:"fonts,omitempty"`
	CMDParams    *CommandParams           `json:"cmd_params,omitempty"`
}

// NavigatorFingerprint contains navigator-related values.
type NavigatorFingerprint struct {
	HardwareConcurrency int    `json:"hardware_concurrency,omitempty"`
	Platform            string `json:"platform,omitempty"`
	UserAgent           string `json:"user_agent,omitempty"`
	OSCPU               string `json:"os_cpu,omitempty"`
	MaxTouchPoints      int    `json:"max_touch_points,omitempty"`
}

// LocalizationFingerprint contains language and locale settings.
type LocalizationFingerprint struct {
	Languages       string `json:"languages,omitempty"`
	Locale          string `json:"locale,omitempty"`
	AcceptLanguages string `json:"accept_languages,omitempty"`
}

// TimezoneFingerprint contains timezone settings.
type TimezoneFingerprint struct {
	Zone string `json:"zone,omitempty"`
}

// GraphicFingerprint contains GPU information.
type GraphicFingerprint struct {
	Renderer string `json:"renderer,omitempty"`
	Vendor   string `json:"vendor,omitempty"`
	VendorID string `json:"vendor_id,omitempty"`
	DeviceID string `json:"device_id,omitempty"`
}

// WebRTCFingerprint contains WebRTC information.
type WebRTCFingerprint struct {
	PublicIP string `json:"public_ip,omitempty"`
}

// MediaDevicesFingerprint contains media device counts.
type MediaDevicesFingerprint struct {
	AudioInputs  int `json:"audio_inputs,omitempty"`
	AudioOutputs int `json:"audio_outputs,omitempty"`
	VideoInputs  int `json:"video_inputs,omitempty"`
}

// ScreenFingerprint contains screen-related values.
type ScreenFingerprint struct {
	Height     int     `json:"height,omitempty"`
	PixelRatio float64 `json:"pixel_ratio,omitempty"`
	Width      int     `json:"width,omitempty"`
}

// GeolocationFingerprint contains geolocation values.
type GeolocationFingerprint struct {
	Accuracy  float64 `json:"accuracy,omitempty"`
	Altitude  float64 `json:"altitude,omitempty"`
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
}

// CommandParams wraps browser command-line parameters.
type CommandParams struct {
	Params []CommandParam `json:"params,omitempty"`
}

// CommandParam describes a single command-line parameter.
type CommandParam struct {
	Flag  string `json:"flag,omitempty"`
	Value string `json:"value,omitempty"`
}

// Proxy contains proxy settings.
type Proxy struct {
	Host             string `json:"host,omitempty"`
	Type             string `json:"type,omitempty"`
	Port             int    `json:"port,omitempty"`
	Username         string `json:"username,omitempty"`
	Password         string `json:"password,omitempty"`
	SaveTraffic      bool   `json:"save_traffic,omitempty"`
	Country          string `json:"country,omitempty"`
	Region           string `json:"region,omitempty"`
	City             string `json:"city,omitempty"`
	SessionID        string `json:"session_id,omitempty"`
	Provider         string `json:"provider,omitempty"`
	ConnectionString string `json:"connection_string,omitempty"`
	RetentionKey     string `json:"retention_key,omitempty"`
	RetentionSecret  string `json:"retention_secret,omitempty"`
}

// CreateProfileResponse contains created profile IDs.
type CreateProfileResponse struct {
	Status Status            `json:"status"`
	Data   CreatedProfileIDs `json:"data"`
}

func (r *CreateProfileResponse) GetStatus() Status { return r.Status }

// CreatedProfileIDs wraps returned profile IDs.
type CreatedProfileIDs struct {
	IDs []string `json:"ids"`
}

// EmptyDataResponse is used by endpoints returning null data.
type EmptyDataResponse struct {
	Status Status `json:"status"`
	Data   any    `json:"data"`
}

func (r *EmptyDataResponse) GetStatus() Status { return r.Status }

// SearchProfilesResponse contains search results.
type SearchProfilesResponse struct {
	Status Status             `json:"status"`
	Data   SearchProfilesData `json:"data"`
}

func (r *SearchProfilesResponse) GetStatus() Status { return r.Status }

// SearchProfilesData contains a page of profiles.
type SearchProfilesData struct {
	Profiles   []Profile `json:"profiles"`
	TotalCount int       `json:"total_count"`
}

// Profile is the lightweight profile view returned by search.
type Profile struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	FolderID          string `json:"folder_id"`
	ABPStatus         bool   `json:"abp_status"`
	BrowserType       string `json:"browser_type"`
	OSType            string `json:"os_type"`
	CoreVersion       int    `json:"core_version"`
	Notes             string `json:"notes"`
	CreatedBy         string `json:"created_by"`
	CreatedAt         string `json:"created_at"`
	InUseBy           string `json:"in_use_by"`
	LockedBy          string `json:"locked_by"`
	LastLaunchedAt    string `json:"last_launched_at"`
	LastLaunchedBy    string `json:"last_launched_by"`
	LastLaunchedOn    string `json:"last_launched_on"`
	UpdatedAt         string `json:"updated_at"`
	PasswordProtected bool   `json:"password_protected"`
	IsLocal           bool   `json:"is_local"`
}

// ProfileMetasResponse returns detailed profile metadata.
type ProfileMetasResponse struct {
	Status Status           `json:"status"`
	Data   ProfileMetasData `json:"data"`
}

func (r *ProfileMetasResponse) GetStatus() Status { return r.Status }

// ProfileMetasData contains detailed profile metadata records.
type ProfileMetasData struct {
	Profiles []ProfileMeta `json:"profiles"`
}

// ProfileMeta is a detailed profile metadata record.
type ProfileMeta struct {
	ID             string             `json:"id"`
	Name           string             `json:"name"`
	Notes          string             `json:"notes"`
	BrowserType    string             `json:"browser_type"`
	CoreVersion    int                `json:"core_version"`
	IsAutoUpdate   bool               `json:"is_auto_update"`
	IsLocal        bool               `json:"is_local"`
	OSType         string             `json:"os_type"`
	FolderID       string             `json:"folder_id"`
	WorkspaceID    string             `json:"workspace_id"`
	CreatedAt      string             `json:"created_at"`
	CreatedBy      string             `json:"created_by"`
	InUseBy        string             `json:"in_use_by"`
	LastLaunchedAt string             `json:"last_launched_at"`
	LastLaunchedBy string             `json:"last_launched_by"`
	LastLaunchedOn string             `json:"last_launched_on"`
	LastUpdatedAt  string             `json:"last_update_at"`
	LastUpdatedBy  string             `json:"last_updated_by"`
	RemovedAt      string             `json:"removed_at"`
	RemovedBy      string             `json:"removed_by"`
	Status         string             `json:"status"`
	Parameters     *ProfileParameters `json:"parameters,omitempty"`
}

/*
Multilogin X live validation showed that the top-level `is_local` flag returned by
`/profile/metas` can be incorrect for real local profiles created or imported with
local storage semantics. In contrast, these signals were confirmed to match actual
behavior in live tests and should be treated as more reliable:

  - `profile/search` filtered with `storage_type=local|cloud|all`
  - `parameters.storage.is_local` inside profile metas
  - launcher active counters (`/api/v1/profile/statuses`)

Because of that MLX API bug/quirk, SDK code must treat `ProfileMeta.IsLocal` as a
diagnostic/raw field only and avoid using it for local/cloud decisions.
*/

// CheckLocal returns the confirmed local/cloud storage signal for a profile meta.
//
// It intentionally ignores the top-level `ProfileMeta.IsLocal` field because live
// MLX responses from `/profile/metas` returned incorrect values for real local
// profiles. When storage parameters are absent, this helper returns false instead of
// falling back to the buggy top-level flag.
func (m *ProfileMeta) CheckLocal() bool {
	if m == nil || m.Parameters == nil || m.Parameters.Storage == nil {
		return false
	}
	return m.Parameters.Storage.IsLocal
}

// ProfileSummaryResponse contains fingerprint summary details.
type ProfileSummaryResponse struct {
	Status Status         `json:"status"`
	Data   ProfileSummary `json:"data"`
}

func (r *ProfileSummaryResponse) GetStatus() Status { return r.Status }

// ProfileSummary contains ready-to-start fingerprint summary information.
type ProfileSummary struct {
	Fonts          []string                 `json:"fonts,omitempty"`
	Geolocation    *GeolocationFingerprint  `json:"geolocation,omitempty"`
	Graphic        *GraphicFingerprint      `json:"graphic,omitempty"`
	Localization   *LocalizationFingerprint `json:"localization,omitempty"`
	MaskingOptions map[string]any           `json:"masking_options,omitempty"`
	MediaDevices   *MediaDevicesFingerprint `json:"media_devices,omitempty"`
	Navigator      *NavigatorFingerprint    `json:"navigator,omitempty"`
	Ports          []int                    `json:"ports,omitempty"`
	Screen         *ScreenFingerprint       `json:"screen,omitempty"`
	Timezone       *TimezoneFingerprint     `json:"timezone,omitempty"`
	WebRTC         *WebRTCFingerprint       `json:"webrtc,omitempty"`
}

func (s *ProfilesServiceOp) Create(ctx context.Context, reqBody *CreateProfileRequest) (*CreateProfileResponse, *Response, error) {
	if reqBody == nil {
		return nil, nil, NewArgError("reqBody", "it must not be nil")
	}
	req, err := s.client.newAPIRequest(ctx, http.MethodPost, "/profile/create", reqBody)
	if err != nil {
		return nil, nil, err
	}
	out := new(CreateProfileResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *ProfilesServiceOp) Search(ctx context.Context, reqBody *SearchProfilesRequest) (*SearchProfilesResponse, *Response, error) {
	if reqBody == nil {
		return nil, nil, NewArgError("reqBody", "it must not be nil")
	}
	req, err := s.client.newAPIRequest(ctx, http.MethodPost, "/profile/search", reqBody)
	if err != nil {
		return nil, nil, err
	}
	out := new(SearchProfilesResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *ProfilesServiceOp) FindByName(ctx context.Context, profileName string, opts *FindProfileOptions) (*Profile, *Response, error) {
	if strings.TrimSpace(profileName) == "" {
		return nil, nil, NewArgError("profileName", "it must not be empty")
	}
	searchReq := &SearchProfilesRequest{
		IsRemoved:   false,
		Limit:       100,
		Offset:      0,
		SearchText:  profileName,
		StorageType: "all",
	}
	if opts != nil {
		searchReq.IsRemoved = opts.IsRemoved
		if opts.StorageType != "" {
			searchReq.StorageType = opts.StorageType
		}
		searchReq.FolderID = opts.FolderID
		searchReq.BrowserType = opts.BrowserType
		searchReq.OSType = opts.OSType
		searchReq.Tags = opts.Tags
		if opts.Limit > 0 {
			searchReq.Limit = opts.Limit
		}
	}

	resp, httpResp, err := s.Search(ctx, searchReq)
	if err != nil {
		return nil, httpResp, err
	}

	trimmedName := strings.TrimSpace(profileName)
	matches := make([]Profile, 0, len(resp.Data.Profiles))
	for _, profile := range resp.Data.Profiles {
		if strings.EqualFold(strings.TrimSpace(profile.Name), trimmedName) {
			matches = append(matches, profile)
		}
	}

	if len(matches) == 0 {
		return nil, httpResp, fmt.Errorf("%w: %s", ErrProfileNotFound, profileName)
	}
	if len(matches) > 1 {
		return nil, httpResp, fmt.Errorf("%w: %q matched %d profiles", ErrProfileAmbiguous, profileName, len(matches))
	}

	match := matches[0]
	return &match, httpResp, nil
}

func (s *ProfilesServiceOp) Update(ctx context.Context, reqBody *UpdateProfileRequest) (*EmptyDataResponse, *Response, error) {
	if reqBody == nil {
		return nil, nil, NewArgError("reqBody", "it must not be nil")
	}
	req, err := s.client.newAPIRequest(ctx, http.MethodPost, "/profile/update", reqBody)
	if err != nil {
		return nil, nil, err
	}
	out := new(EmptyDataResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *ProfilesServiceOp) Patch(ctx context.Context, reqBody *PatchProfileRequest) (*EmptyDataResponse, *Response, error) {
	if reqBody == nil {
		return nil, nil, NewArgError("reqBody", "it must not be nil")
	}
	req, err := s.client.newAPIRequest(ctx, http.MethodPost, "/profile/partial_update", reqBody)
	if err != nil {
		return nil, nil, err
	}
	out := new(EmptyDataResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *ProfilesServiceOp) Delete(ctx context.Context, reqBody *DeleteProfilesRequest) (*EmptyDataResponse, *Response, error) {
	if reqBody == nil {
		return nil, nil, NewArgError("reqBody", "it must not be nil")
	}
	req, err := s.client.newAPIRequest(ctx, http.MethodPost, "/profile/remove", reqBody)
	if err != nil {
		return nil, nil, err
	}
	out := new(EmptyDataResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *ProfilesServiceOp) Restore(ctx context.Context, reqBody *RestoreProfilesRequest) (*EmptyDataResponse, *Response, error) {
	if reqBody == nil {
		return nil, nil, NewArgError("reqBody", "it must not be nil")
	}
	req, err := s.client.newAPIRequest(ctx, http.MethodPost, "/profile/restore", reqBody)
	if err != nil {
		return nil, nil, err
	}
	out := new(EmptyDataResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *ProfilesServiceOp) Clone(ctx context.Context, reqBody *CloneProfileRequest) (*CreateProfileResponse, *Response, error) {
	if reqBody == nil {
		return nil, nil, NewArgError("reqBody", "it must not be nil")
	}
	req, err := s.client.newAPIRequest(ctx, http.MethodPost, "/profile/clone", reqBody)
	if err != nil {
		return nil, nil, err
	}
	out := new(CreateProfileResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *ProfilesServiceOp) Move(ctx context.Context, reqBody *MoveProfilesRequest) (*EmptyDataResponse, *Response, error) {
	if reqBody == nil {
		return nil, nil, NewArgError("reqBody", "it must not be nil")
	}
	req, err := s.client.newAPIRequest(ctx, http.MethodPost, "/profile/move", reqBody)
	if err != nil {
		return nil, nil, err
	}
	out := new(EmptyDataResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *ProfilesServiceOp) GetMeta(ctx context.Context, profileID string) (*ProfileMeta, *Response, error) {
	if strings.TrimSpace(profileID) == "" {
		return nil, nil, NewArgError("profileID", "it must not be empty")
	}
	resp, httpResp, err := s.GetMetas(ctx, &ProfileMetasRequest{IDs: []string{profileID}})
	if err != nil {
		return nil, httpResp, err
	}
	if len(resp.Data.Profiles) == 0 {
		return nil, httpResp, fmt.Errorf("%w: %s", ErrProfileNotFound, profileID)
	}
	meta := resp.Data.Profiles[0]
	return &meta, httpResp, nil
}

func (s *ProfilesServiceOp) GetMetas(ctx context.Context, reqBody *ProfileMetasRequest) (*ProfileMetasResponse, *Response, error) {
	if reqBody == nil {
		return nil, nil, NewArgError("reqBody", "it must not be nil")
	}
	req, err := s.client.newAPIRequest(ctx, http.MethodPost, "/profile/metas", reqBody)
	if err != nil {
		return nil, nil, err
	}
	out := new(ProfileMetasResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *ProfilesServiceOp) GetSummary(ctx context.Context, metaID string) (*ProfileSummaryResponse, *Response, error) {
	if metaID == "" {
		return nil, nil, NewArgError("metaID", "it must not be empty")
	}
	path := fmt.Sprintf("/profile/summary?meta_id=%s", url.QueryEscape(metaID))
	req, err := s.client.newAPIRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	out := new(ProfileSummaryResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}
