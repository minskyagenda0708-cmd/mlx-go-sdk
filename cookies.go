package mlx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

// CookiesService manages pre-made cookies metadata and launcher cookie import/export flows.
type CookiesService interface {
	ListWebsites(context.Context) (*CookieWebsitesResponse, *Response, error)
	CreateMetadata(context.Context, *CreateCookiesMetadataRequest) (*CookiesMetadataResponse, *Response, error)
	UpdateMetadata(context.Context, *UpdateCookiesMetadataRequest) (*EmptyDataResponse, *Response, error)
	List(context.Context, string) (*CookieListResponse, *Response, error)
	Import(context.Context, *CookieImportRequest) (*EmptyDataResponse, *Response, error)
	Export(context.Context, *CookieExportRequest) (*CookieExportResponse, *Response, error)
	SeedProfileCookies(context.Context, SeedProfileCookiesOptions) (*SeedProfileCookiesResult, error)
}

// CookiesServiceOp is the concrete cookie service.
type CookiesServiceOp struct {
	client *Client
}

// CookieWebsite describes an available pre-made cookie target.
type CookieWebsite struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// CookieWebsitesResponse contains target website options.
type CookieWebsitesResponse struct {
	Status Status          `json:"status"`
	Data   []CookieWebsite `json:"data"`
}

func (r *CookieWebsitesResponse) GetStatus() Status { return r.Status }

// CreateCookiesMetadataRequest binds a profile to a pre-made cookie target website.
type CreateCookiesMetadataRequest struct {
	ProfileID     string `json:"profile_id"`
	TargetWebsite string `json:"target_website"`
	StrictMode    bool   `json:"-"`
}

// UpdateCookiesMetadataRequest changes cookie metadata for a profile.
type UpdateCookiesMetadataRequest struct {
	ProfileID         string `json:"profile_id"`
	TargetWebsite     string `json:"target_website"`
	AdditionalWebsite string `json:"additional_website,omitempty"`
	StrictMode        bool   `json:"-"`
}

// CookiesMetadataResponse returns the profile id affected by metadata creation.
type CookiesMetadataResponse struct {
	Status Status              `json:"status"`
	Data   CookiesMetadataData `json:"data"`
}

func (r *CookiesMetadataResponse) GetStatus() Status { return r.Status }

// CookiesMetadataData contains metadata creation output.
type CookiesMetadataData struct {
	ProfileID string `json:"profile_id"`
}

// CookieListResponse contains pre-made cookie bundles for a profile.
type CookieListResponse struct {
	Status Status         `json:"status"`
	Data   CookieListData `json:"data"`
}

func (r *CookieListResponse) GetStatus() Status { return r.Status }

// CookieListData wraps cookie bundles.
type CookieListData struct {
	Cookies []CookieBundle `json:"cookies"`
}

// CookieBundle groups a generated cookie set.
type CookieBundle struct {
	ID        int             `json:"id"`
	CreatedAt string          `json:"created_at"`
	Data      []BrowserCookie `json:"data"`
}

// BrowserCookie describes an individual browser cookie.
type BrowserCookie struct {
	Name           string `json:"name,omitempty"`
	Value          string `json:"value,omitempty"`
	Domain         string `json:"domain,omitempty"`
	Path           string `json:"path,omitempty"`
	Secure         bool   `json:"secure,omitempty"`
	HTTPOnly       bool   `json:"httpOnly,omitempty"`
	Session        bool   `json:"session,omitempty"`
	HostOnly       bool   `json:"hostOnly,omitempty"`
	StoreID        string `json:"storeId,omitempty"`
	SameSite       string `json:"sameSite,omitempty"`
	SameParty      bool   `json:"sameParty,omitempty"`
	SourcePort     int    `json:"sourcePort,omitempty"`
	SourceScheme   string `json:"sourceScheme,omitempty"`
	ExpirationDate int64  `json:"expirationDate,omitempty"`
	Size           int    `json:"size,omitempty"`
}

// CookieImportRequest imports either advanced pre-made cookies or explicit cookie JSON into a profile.
type CookieImportRequest struct {
	ProfileID             string          `json:"profile_id"`
	FolderID              string          `json:"folder_id,omitempty"`
	ImportAdvancedCookies bool            `json:"import_advanced_cookies"`
	Cookies               []BrowserCookie `json:"-"`
	StrictMode            bool            `json:"-"`
}

// MarshalJSON converts cookie arrays into the quoted JSON string format expected by the launcher endpoint.
func (r CookieImportRequest) MarshalJSON() ([]byte, error) {
	type alias struct {
		ProfileID             string `json:"profile_id"`
		FolderID              string `json:"folder_id,omitempty"`
		ImportAdvancedCookies bool   `json:"import_advanced_cookies"`
		Cookies               string `json:"cookies,omitempty"`
	}
	var cookies string
	if len(r.Cookies) > 0 {
		payload, err := json.Marshal(r.Cookies)
		if err != nil {
			return nil, err
		}
		cookies = string(payload)
	}
	return json.Marshal(alias{
		ProfileID:             r.ProfileID,
		FolderID:              r.FolderID,
		ImportAdvancedCookies: r.ImportAdvancedCookies,
		Cookies:               cookies,
	})
}

// CookieExportRequest exports profile cookies from the launcher.
type CookieExportRequest struct {
	ProfileID string `json:"profile_id"`
	FolderID  string `json:"folder_id,omitempty"`
}

// CookieExportResponse contains exported cookie JSON.
type CookieExportResponse struct {
	Status Status           `json:"status"`
	Data   CookieExportData `json:"data"`
}

func (r *CookieExportResponse) GetStatus() Status { return r.Status }

// CookieExportData contains launcher cookie export output.
type CookieExportData struct {
	Cookies   string `json:"cookies"`
	ProfileID string `json:"profile_id"`
	Timestamp int64  `json:"timestamp"`
}

// SeedProfileCookiesOptions configures the high-level cookie seeding helper.
type SeedProfileCookiesOptions struct {
	ProfileID               string
	FolderID                string
	TargetWebsite           string
	AdditionalWebsite       string
	CreateMetadataIfMissing bool
	StrictMode              bool
	ImportAdvancedCookies   bool
	CookieBundleIndex       int
}

// SeedProfileCookiesResult contains the outcome of metadata creation/update, cookie selection, and import.
type SeedProfileCookiesResult struct {
	MetadataCreated bool
	MetadataUpdated bool
	ProfileID       string
	FolderID        string
	TargetWebsite   string
	CookieCount     int
	SelectedBundle  *CookieBundle
	ImportResponse  *EmptyDataResponse
}

func (s *CookiesServiceOp) ListWebsites(ctx context.Context) (*CookieWebsitesResponse, *Response, error) {
	req, err := s.client.newCookiesRequest(ctx, http.MethodGet, "/api/v1/cookies/metadata/websites", nil)
	if err != nil {
		return nil, nil, err
	}
	out := new(CookieWebsitesResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *CookiesServiceOp) CreateMetadata(ctx context.Context, reqBody *CreateCookiesMetadataRequest) (*CookiesMetadataResponse, *Response, error) {
	if reqBody == nil {
		return nil, nil, NewArgError("reqBody", "it must not be nil")
	}
	if reqBody.ProfileID == "" {
		return nil, nil, NewArgError("profileID", "it must not be empty")
	}
	if reqBody.TargetWebsite == "" {
		return nil, nil, NewArgError("targetWebsite", "it must not be empty")
	}
	req, err := s.client.newCookiesRequest(ctx, http.MethodPost, "/api/v1/cookies/metadata", reqBody)
	if err != nil {
		return nil, nil, err
	}
	if reqBody.StrictMode {
		req.Header.Set("X-Strict-Mode", "true")
	}
	out := new(CookiesMetadataResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *CookiesServiceOp) UpdateMetadata(ctx context.Context, reqBody *UpdateCookiesMetadataRequest) (*EmptyDataResponse, *Response, error) {
	if reqBody == nil {
		return nil, nil, NewArgError("reqBody", "it must not be nil")
	}
	if reqBody.ProfileID == "" {
		return nil, nil, NewArgError("profileID", "it must not be empty")
	}
	if reqBody.TargetWebsite == "" {
		return nil, nil, NewArgError("targetWebsite", "it must not be empty")
	}
	req, err := s.client.newCookiesRequest(ctx, http.MethodPut, "/api/v1/cookies/metadata", reqBody)
	if err != nil {
		return nil, nil, err
	}
	if reqBody.StrictMode {
		req.Header.Set("X-Strict-Mode", "true")
	}
	out := new(EmptyDataResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *CookiesServiceOp) List(ctx context.Context, profileID string) (*CookieListResponse, *Response, error) {
	if profileID == "" {
		return nil, nil, NewArgError("profileID", "it must not be empty")
	}
	path := fmt.Sprintf("/api/v1/cookies/%s", url.PathEscape(profileID))
	req, err := s.client.newCookiesRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	out := new(CookieListResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *CookiesServiceOp) Import(ctx context.Context, reqBody *CookieImportRequest) (*EmptyDataResponse, *Response, error) {
	if reqBody == nil {
		return nil, nil, NewArgError("reqBody", "it must not be nil")
	}
	if reqBody.ProfileID == "" {
		return nil, nil, NewArgError("profileID", "it must not be empty")
	}
	req, err := s.client.newLauncherRequest(ctx, http.MethodPost, "/api/v1/cookie_import", reqBody)
	if err != nil {
		return nil, nil, err
	}
	if reqBody.StrictMode {
		req.Header.Set("X-Strict-Mode", "true")
	}
	out := new(EmptyDataResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *CookiesServiceOp) Export(ctx context.Context, reqBody *CookieExportRequest) (*CookieExportResponse, *Response, error) {
	if reqBody == nil {
		return nil, nil, NewArgError("reqBody", "it must not be nil")
	}
	if reqBody.ProfileID == "" {
		return nil, nil, NewArgError("profileID", "it must not be empty")
	}
	req, err := s.client.newLauncherRequest(ctx, http.MethodPost, "/api/v1/cookie_export", reqBody)
	if err != nil {
		return nil, nil, err
	}
	out := new(CookieExportResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

// SeedProfileCookies creates or updates pre-made cookies metadata, fetches generated cookies, and imports them into the profile.
func (s *CookiesServiceOp) SeedProfileCookies(ctx context.Context, opts SeedProfileCookiesOptions) (*SeedProfileCookiesResult, error) {
	if opts.ProfileID == "" {
		return nil, NewArgError("profileID", "it must not be empty")
	}
	if opts.TargetWebsite == "" {
		return nil, NewArgError("targetWebsite", "it must not be empty")
	}

	folderID := opts.FolderID
	if folderID == "" {
		resolvedFolderID, err := s.resolveFolderID(ctx, opts.ProfileID)
		if err != nil {
			return nil, err
		}
		folderID = resolvedFolderID
	}

	result := &SeedProfileCookiesResult{
		ProfileID:     opts.ProfileID,
		FolderID:      folderID,
		TargetWebsite: opts.TargetWebsite,
	}

	if opts.CreateMetadataIfMissing {
		if _, _, err := s.CreateMetadata(ctx, &CreateCookiesMetadataRequest{
			ProfileID:     opts.ProfileID,
			TargetWebsite: opts.TargetWebsite,
			StrictMode:    opts.StrictMode,
		}); err != nil {
			var apiErr *ErrorResponse
			if !errors.As(err, &apiErr) || apiErr == nil || apiErr.Response == nil || apiErr.Response.StatusCode != http.StatusConflict {
				return nil, err
			}
		} else {
			result.MetadataCreated = true
		}
	}

	if !result.MetadataCreated {
		if _, _, err := s.UpdateMetadata(ctx, &UpdateCookiesMetadataRequest{
			ProfileID:         opts.ProfileID,
			TargetWebsite:     opts.TargetWebsite,
			AdditionalWebsite: opts.AdditionalWebsite,
			StrictMode:        opts.StrictMode,
		}); err != nil {
			return nil, err
		}
		result.MetadataUpdated = true
	}

	listResp, _, err := s.List(ctx, opts.ProfileID)
	if err != nil {
		return nil, err
	}
	if len(listResp.Data.Cookies) == 0 {
		return nil, NewArgError("cookies", "no pre-made cookie bundles were returned for the profile")
	}

	bundleIndex := opts.CookieBundleIndex
	if bundleIndex < 0 || bundleIndex >= len(listResp.Data.Cookies) {
		bundleIndex = 0
	}
	selectedBundle := listResp.Data.Cookies[bundleIndex]
	if len(selectedBundle.Data) == 0 {
		return nil, NewArgError("cookies", "selected pre-made cookie bundle does not contain cookies")
	}
	result.SelectedBundle = &selectedBundle
	result.CookieCount = len(selectedBundle.Data)

	importResp, _, err := s.Import(ctx, &CookieImportRequest{
		ProfileID:             opts.ProfileID,
		FolderID:              folderID,
		ImportAdvancedCookies: opts.ImportAdvancedCookies,
		Cookies:               selectedBundle.Data,
		StrictMode:            opts.StrictMode,
	})
	if err != nil {
		return nil, err
	}
	result.ImportResponse = importResp

	return result, nil
}

func (s *CookiesServiceOp) resolveFolderID(ctx context.Context, profileID string) (string, error) {
	resp, _, err := s.client.Profiles.Search(ctx, &SearchProfilesRequest{
		IsRemoved:   false,
		Limit:       100,
		Offset:      0,
		SearchText:  "",
		StorageType: "all",
	})
	if err != nil {
		return "", err
	}
	for _, profile := range resp.Data.Profiles {
		if profile.ID == profileID {
			if profile.FolderID == "" {
				return "", NewArgError("folderID", "resolved profile has empty folder id")
			}
			return profile.FolderID, nil
		}
	}
	return "", NewArgError("profileID", "profile was not found while resolving folder id")
}
