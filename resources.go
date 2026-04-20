package mlx

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const (
	ResourceTypeProfileTemplates         = "7e46e7f9-15d4-41b6-83b9-a652336793ec"
	ResourceTypeProxyConfiguration       = "3c1a0080-5282-436b-885c-ab27d5004aa8"
	ResourceTypeExtensions               = "6811b909-2e4b-45db-ab62-f14f515523cf"
	ResourceTypeCookies                  = "58268a18-02b8-4d2d-ac59-9cc166ea4064"
	ResourceTypePasswords                = "bb80e9b9-b2bb-43b5-968b-c2ea9b509d7a"
	ResourceTypeAutomationScripts        = "8dfc6cec-4aad-41f0-ac87-ff44a4be0b3a"
	ResourceTypeLaunchParameterTemplates = "42d592bc-df3a-47b5-8d50-4b338df6ade2"
)

// ResourcesService manages template/resource objects backed by Multilogin resources and launcher object storage.
type ResourcesService interface {
	ListTypes(context.Context) (*ResourceTypesResponse, *Response, error)
	ListMetas(context.Context, *ListResourceMetasOptions) (*ResourceMetasResponse, *Response, error)
	ListProfileTemplates(context.Context, *ListResourceMetasOptions) (*ResourceMetasResponse, *Response, error)
	ListExtensions(context.Context, *ListResourceMetasOptions) (*ResourceMetasResponse, *Response, error)
	GetMeta(context.Context, string) (*ResourceMetaResponse, *Response, error)
	Delete(context.Context, string, bool) (*EmptyDataResponse, *Response, error)
	Restore(context.Context, string) (*EmptyDataResponse, *Response, error)
	ObjectProfileUsages(context.Context, string) (*ObjectProfileUsagesResponse, *Response, error)
	ProfileObjectUsages(context.Context, *ProfileObjectUsagesRequest) (*ProfileObjectUsagesResponse, *Response, error)
	ProfileExtensionUsages(context.Context, string) (*ProfileObjectUsagesResponse, *Response, error)
	Upload(context.Context, *UploadObjectRequest) (*CreateAndUploadObjectResponse, *Response, error)
	CreateAndUpload(context.Context, *CreateAndUploadObjectRequest) (*CreateAndUploadObjectResponse, *Response, error)
	CreateProfileTemplate(context.Context, *CreateProfileTemplateRequest) (*CreateAndUploadObjectResponse, *Response, error)
	UploadExtension(context.Context, *UploadExtensionRequest) (*CreateAndUploadObjectResponse, *Response, error)
	LocalToCloud(context.Context, *LocalToCloudObjectRequest) (*CreateAndUploadObjectResponse, *Response, error)
	CreateExtensionFromURL(context.Context, *CreateExtensionFromURLRequest) (*EmptyDataResponse, *Response, error)
	CreateExtensionFromChromeWebStore(context.Context, *CreateChromeWebStoreExtensionRequest) (*EmptyDataResponse, *Response, error)
	EnableExtensionForProfiles(context.Context, string, *SetResourceProfilesRequest) (*StringDataResponse, *Response, error)
	DisableExtensionForProfiles(context.Context, string, *SetResourceProfilesRequest) (*StringDataResponse, *Response, error)
	Download(context.Context, string) (*DownloadResourceResponse, *Response, error)
}

// ResourcesServiceOp is the concrete resources service implementation.
type ResourcesServiceOp struct {
	client *Client
}

// ResourceTypesResponse contains available resource object types.
type ResourceTypesResponse struct {
	Status Status            `json:"status"`
	Data   ResourceTypesData `json:"data"`
}

func (r *ResourceTypesResponse) GetStatus() Status { return r.Status }

// ResourceTypesData wraps available resource object types.
type ResourceTypesData struct {
	Types []ResourceType `json:"types"`
}

// ResourceType identifies one resource object type.
type ResourceType struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ListResourceMetasOptions controls resource meta listing.
type ListResourceMetasOptions struct {
	Limit           int
	Offset          int
	ObjectName      string
	ObjectTypeID    string
	StorageType     string
	Creator         string
	Trashbin        *bool
	CreateStartDate string
	CreateEndDate   string
	UpdateStartDate string
	UpdateEndDate   string
}

// ResourceMetasResponse contains listed resource objects.
type ResourceMetasResponse struct {
	Status Status            `json:"status"`
	Data   ResourceMetasData `json:"data"`
}

func (r *ResourceMetasResponse) GetStatus() Status { return r.Status }

// ResourceMetasData wraps listed resource objects.
type ResourceMetasData struct {
	Objects []ResourceMeta `json:"objects"`
}

// ResourceMetaResponse returns one resource object metadata record.
type ResourceMetaResponse struct {
	Status Status       `json:"status"`
	Data   ResourceMeta `json:"data"`
}

func (r *ResourceMetaResponse) GetStatus() Status { return r.Status }

// ResourceMeta describes one resource object metadata record.
type ResourceMeta struct {
	ID             string `json:"id"`
	ObjectTypeID   string `json:"object_type_id"`
	ObjectName     string `json:"object_name"`
	ObjectSize     int64  `json:"object_size"`
	CurrentVersion string `json:"current_version"`
	CreatedAt      string `json:"created_at"`
	CreatedBy      string `json:"created_by"`
	UpdateAt       string `json:"update_at"`
	UpdateBy       string `json:"update_by"`
	StorageType    string `json:"storage_type"`
	MetaInfo       string `json:"meta_info"`
	IsDefault      bool   `json:"is_default"`
	IsInTrashbin   bool   `json:"is_in_trashbin"`
}

// ObjectProfileUsagesResponse lists profile usages for one object.
type ObjectProfileUsagesResponse struct {
	Status Status               `json:"status"`
	Data   []ObjectProfileUsage `json:"data"`
}

func (r *ObjectProfileUsagesResponse) GetStatus() Status { return r.Status }

// ObjectProfileUsage identifies one object-to-profile usage record.
type ObjectProfileUsage struct {
	ID       string `json:"id"`
	ObjectID string `json:"object_id"`
}

// ProfileObjectUsagesRequest queries resource usages for a profile and object type.
type ProfileObjectUsagesRequest struct {
	ObjectType string `json:"object_type"`
	ProfileID  string `json:"profile_id"`
}

// ProfileObjectUsagesResponse lists objects associated with one profile.
type ProfileObjectUsagesResponse struct {
	Status Status               `json:"status"`
	Data   []ProfileObjectUsage `json:"data"`
}

func (r *ProfileObjectUsagesResponse) GetStatus() Status { return r.Status }

// ProfileObjectUsage describes one object associated with a profile.
type ProfileObjectUsage struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Type      string          `json:"type"`
	MetaInfo  json.RawMessage `json:"meta_info"`
	IsEnabled bool            `json:"is_enabled"`
}

// UploadObjectRequest uploads an existing local object into launcher-backed storage.
type UploadObjectRequest struct {
	ObjectTypeID string `json:"object_type_id"`
	ObjectPath   string `json:"object_path"`
	StorageType  string `json:"storage_type"`
	ObjectMeta   string `json:"object_meta,omitempty"`
	Encrypt      *bool  `json:"encrypt,omitempty"`
}

// CreateAndUploadObjectRequest creates a new launcher-backed object from body content.
type CreateAndUploadObjectRequest struct {
	ObjectName      string `json:"object_name"`
	ObjectExtension string `json:"object_extension"`
	ObjectTypeID    string `json:"object_type_id"`
	ObjectBody      string `json:"object_body"`
	ObjectMeta      string `json:"object_meta,omitempty"`
	Encrypt         *bool  `json:"encrypt,omitempty"`
}

// CreateAndUploadObjectResponse returns the created resource meta id.
type CreateAndUploadObjectResponse struct {
	Status Status                    `json:"status"`
	Data   CreateAndUploadObjectData `json:"data"`
}

func (r *CreateAndUploadObjectResponse) GetStatus() Status { return r.Status }

// CreateAndUploadObjectData contains the created resource id.
type CreateAndUploadObjectData struct {
	MetaID string `json:"meta_id"`
}

// CreateProfileTemplateRequest creates a profile template resource.
type CreateProfileTemplateRequest struct {
	Name      string
	Extension string
	Body      string
	Meta      string
	Encrypt   *bool
}

// UploadExtensionRequest uploads an extension archive as a resource object.
//
// Live validation showed that applying a local zip extension to a local profile
// works reliably when the extension object reference is cloud-backed, so this
// helper defaults StorageType to "cloud".
type UploadExtensionRequest struct {
	ObjectPath  string
	StorageType string
	ObjectMeta  string
	Encrypt     *bool
}

// LocalToCloudObjectRequest promotes a local object into cloud storage.
type LocalToCloudObjectRequest struct {
	ObjectPath string `json:"object_path"`
	ObjectID   string `json:"object_id,omitempty"`
}

// CreateExtensionFromURLRequest materializes an extension object from a downloadable URL.
//
// StorageType defaults to "cloud" because cloud-backed extension references are
// the safest choice for both cloud and local profiles.
type CreateExtensionFromURLRequest struct {
	URL         string `json:"url"`
	BrowserType string `json:"browser_type"`
	StorageType string `json:"storage_type"`
}

// CreateChromeWebStoreExtensionRequest creates an extension from a Chrome Web Store ID.
type CreateChromeWebStoreExtensionRequest struct {
	ExtensionID string
	BrowserType string
	StorageType string
}

// SetResourceProfilesRequest enables or disables a resource for a set of profiles.
type SetResourceProfilesRequest struct {
	ProfileIDs []string `json:"profile_ids"`
}

// StringDataResponse captures endpoints that return a simple string payload.
type StringDataResponse struct {
	Status Status `json:"status"`
	Data   string `json:"data"`
}

func (r *StringDataResponse) GetStatus() Status { return r.Status }

// DownloadResourceResponse contains the downloaded path materialized by the launcher.
type DownloadResourceResponse struct {
	Status Status `json:"status"`
	Path   string `json:"-"`
}

func (r *DownloadResourceResponse) GetStatus() Status { return r.Status }

func (s *ResourcesServiceOp) ListTypes(ctx context.Context) (*ResourceTypesResponse, *Response, error) {
	req, err := s.client.newAPIRequest(ctx, http.MethodGet, "/api/v1/resources/types", nil)
	if err != nil {
		return nil, nil, err
	}
	out := new(ResourceTypesResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *ResourcesServiceOp) ListMetas(ctx context.Context, opts *ListResourceMetasOptions) (*ResourceMetasResponse, *Response, error) {
	values := url.Values{}
	limit := 10
	offset := 0
	if opts != nil {
		if opts.Limit > 0 {
			limit = opts.Limit
		}
		if opts.Offset >= 0 {
			offset = opts.Offset
		}
		if opts.ObjectName != "" {
			values.Set("object_name", opts.ObjectName)
		}
		if opts.ObjectTypeID != "" {
			values.Set("object_type_id", opts.ObjectTypeID)
		}
		if opts.StorageType != "" {
			values.Set("storage_type", opts.StorageType)
		}
		if opts.Creator != "" {
			values.Set("creator", opts.Creator)
		}
		if opts.Trashbin != nil {
			values.Set("trashbin", fmt.Sprintf("%t", *opts.Trashbin))
		}
		if opts.CreateStartDate != "" {
			values.Set("create_start_date", opts.CreateStartDate)
		}
		if opts.CreateEndDate != "" {
			values.Set("create_end_date", opts.CreateEndDate)
		}
		if opts.UpdateStartDate != "" {
			values.Set("update_start_date", opts.UpdateStartDate)
		}
		if opts.UpdateEndDate != "" {
			values.Set("update_end_date", opts.UpdateEndDate)
		}
	}
	values.Set("limit", fmt.Sprintf("%d", limit))
	values.Set("offset", fmt.Sprintf("%d", offset))
	path := "/api/v1/resources/metas"
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	req, err := s.client.newAPIRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	out := new(ResourceMetasResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *ResourcesServiceOp) ListProfileTemplates(ctx context.Context, opts *ListResourceMetasOptions) (*ResourceMetasResponse, *Response, error) {
	cloned := &ListResourceMetasOptions{ObjectTypeID: ResourceTypeProfileTemplates}
	if opts != nil {
		*cloned = *opts
		cloned.ObjectTypeID = ResourceTypeProfileTemplates
	}
	return s.ListMetas(ctx, cloned)
}

func (s *ResourcesServiceOp) ListExtensions(ctx context.Context, opts *ListResourceMetasOptions) (*ResourceMetasResponse, *Response, error) {
	cloned := &ListResourceMetasOptions{ObjectTypeID: ResourceTypeExtensions}
	if opts != nil {
		*cloned = *opts
		cloned.ObjectTypeID = ResourceTypeExtensions
	}
	return s.ListMetas(ctx, cloned)
}

func (s *ResourcesServiceOp) GetMeta(ctx context.Context, resourceID string) (*ResourceMetaResponse, *Response, error) {
	if strings.TrimSpace(resourceID) == "" {
		return nil, nil, NewArgError("resourceID", "it must not be empty")
	}
	path := fmt.Sprintf("/api/v1/resources/%s/meta", url.PathEscape(resourceID))
	req, err := s.client.newAPIRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	out := new(ResourceMetaResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *ResourcesServiceOp) Delete(ctx context.Context, resourceID string, permanently bool) (*EmptyDataResponse, *Response, error) {
	if strings.TrimSpace(resourceID) == "" {
		return nil, nil, NewArgError("resourceID", "it must not be empty")
	}
	path := fmt.Sprintf("/api/v1/resources/%s/delete?permanently=%t", url.PathEscape(resourceID), permanently)
	req, err := s.client.newAPIRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	out := new(EmptyDataResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *ResourcesServiceOp) Restore(ctx context.Context, resourceID string) (*EmptyDataResponse, *Response, error) {
	if strings.TrimSpace(resourceID) == "" {
		return nil, nil, NewArgError("resourceID", "it must not be empty")
	}
	path := fmt.Sprintf("/api/v1/resources/%s/restore", url.PathEscape(resourceID))
	req, err := s.client.newAPIRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	out := new(EmptyDataResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *ResourcesServiceOp) ObjectProfileUsages(ctx context.Context, objectID string) (*ObjectProfileUsagesResponse, *Response, error) {
	if strings.TrimSpace(objectID) == "" {
		return nil, nil, NewArgError("objectID", "it must not be empty")
	}
	path := fmt.Sprintf("/api/v1/resources/object_profile_usages?object_id=%s", url.QueryEscape(objectID))
	req, err := s.client.newAPIRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	out := new(ObjectProfileUsagesResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *ResourcesServiceOp) ProfileObjectUsages(ctx context.Context, reqBody *ProfileObjectUsagesRequest) (*ProfileObjectUsagesResponse, *Response, error) {
	if reqBody == nil {
		return nil, nil, NewArgError("reqBody", "it must not be nil")
	}
	req, err := s.client.newAPIRequest(ctx, http.MethodPost, "/api/v1/resources/profile_object_usages", reqBody)
	if err != nil {
		return nil, nil, err
	}
	out := new(ProfileObjectUsagesResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *ResourcesServiceOp) ProfileExtensionUsages(ctx context.Context, profileID string) (*ProfileObjectUsagesResponse, *Response, error) {
	if strings.TrimSpace(profileID) == "" {
		return nil, nil, NewArgError("profileID", "it must not be empty")
	}
	/*
		Live MLX validation showed that `/api/v1/resources/profile_object_usages` can be
		unreliable for extension lookups even when the extension is actually attached to a
		local profile: the endpoint returned `501` while the inverse lookup via
		`ObjectProfileUsages` and launcher behavior still confirmed the binding.

		Use this helper when the endpoint is available, but prefer object-centric checks or
		other stronger signals in critical workflows.
	*/
	return s.ProfileObjectUsages(ctx, &ProfileObjectUsagesRequest{
		ObjectType: ResourceTypeExtensions,
		ProfileID:  profileID,
	})
}

func (s *ResourcesServiceOp) Upload(ctx context.Context, reqBody *UploadObjectRequest) (*CreateAndUploadObjectResponse, *Response, error) {
	if reqBody == nil {
		return nil, nil, NewArgError("reqBody", "it must not be nil")
	}
	if strings.TrimSpace(reqBody.ObjectTypeID) == "" {
		return nil, nil, NewArgError("objectTypeID", "it must not be empty")
	}
	if strings.TrimSpace(reqBody.ObjectPath) == "" {
		return nil, nil, NewArgError("objectPath", "it must not be empty")
	}
	if strings.TrimSpace(reqBody.StorageType) == "" {
		return nil, nil, NewArgError("storageType", "it must not be empty")
	}
	req, err := s.client.newLauncherRequest(ctx, http.MethodPost, "/api/v1/object_storage/upload", reqBody)
	if err != nil {
		return nil, nil, err
	}
	out := new(CreateAndUploadObjectResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *ResourcesServiceOp) CreateAndUpload(ctx context.Context, reqBody *CreateAndUploadObjectRequest) (*CreateAndUploadObjectResponse, *Response, error) {
	if reqBody == nil {
		return nil, nil, NewArgError("reqBody", "it must not be nil")
	}
	req, err := s.client.newLauncherRequest(ctx, http.MethodPost, "/api/v1/object_storage/create_and_upload", reqBody)
	if err != nil {
		return nil, nil, err
	}
	out := new(CreateAndUploadObjectResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *ResourcesServiceOp) CreateProfileTemplate(ctx context.Context, reqBody *CreateProfileTemplateRequest) (*CreateAndUploadObjectResponse, *Response, error) {
	if reqBody == nil {
		return nil, nil, NewArgError("reqBody", "it must not be nil")
	}
	ext := reqBody.Extension
	if ext == "" {
		ext = "txt"
	}
	return s.CreateAndUpload(ctx, &CreateAndUploadObjectRequest{
		ObjectName:      reqBody.Name,
		ObjectExtension: ext,
		ObjectTypeID:    ResourceTypeProfileTemplates,
		ObjectBody:      reqBody.Body,
		ObjectMeta:      reqBody.Meta,
		Encrypt:         reqBody.Encrypt,
	})
}

func (s *ResourcesServiceOp) UploadExtension(ctx context.Context, reqBody *UploadExtensionRequest) (*CreateAndUploadObjectResponse, *Response, error) {
	if reqBody == nil {
		return nil, nil, NewArgError("reqBody", "it must not be nil")
	}
	storageType := reqBody.StorageType
	if strings.TrimSpace(storageType) == "" {
		storageType = "cloud"
	}
	return s.Upload(ctx, &UploadObjectRequest{
		ObjectTypeID: ResourceTypeExtensions,
		ObjectPath:   reqBody.ObjectPath,
		StorageType:  storageType,
		ObjectMeta:   reqBody.ObjectMeta,
		Encrypt:      reqBody.Encrypt,
	})
}

func (s *ResourcesServiceOp) LocalToCloud(ctx context.Context, reqBody *LocalToCloudObjectRequest) (*CreateAndUploadObjectResponse, *Response, error) {
	if reqBody == nil {
		return nil, nil, NewArgError("reqBody", "it must not be nil")
	}
	if strings.TrimSpace(reqBody.ObjectPath) == "" {
		return nil, nil, NewArgError("objectPath", "it must not be empty")
	}
	req, err := s.client.newLauncherRequest(ctx, http.MethodPost, "/api/v1/object_storage/local_to_cloud", reqBody)
	if err != nil {
		return nil, nil, err
	}
	out := new(CreateAndUploadObjectResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *ResourcesServiceOp) CreateExtensionFromURL(ctx context.Context, reqBody *CreateExtensionFromURLRequest) (*EmptyDataResponse, *Response, error) {
	if reqBody == nil {
		return nil, nil, NewArgError("reqBody", "it must not be nil")
	}
	if strings.TrimSpace(reqBody.URL) == "" {
		return nil, nil, NewArgError("url", "it must not be empty")
	}
	if strings.TrimSpace(reqBody.BrowserType) == "" {
		return nil, nil, NewArgError("browserType", "it must not be empty")
	}
	storageType := reqBody.StorageType
	if strings.TrimSpace(storageType) == "" {
		storageType = "cloud"
	}
	req, err := s.client.newLauncherRequest(ctx, http.MethodPost, "/api/v1/create_extension_from_url", &CreateExtensionFromURLRequest{
		URL:         reqBody.URL,
		BrowserType: reqBody.BrowserType,
		StorageType: storageType,
	})
	if err != nil {
		return nil, nil, err
	}
	out := new(EmptyDataResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *ResourcesServiceOp) CreateExtensionFromChromeWebStore(ctx context.Context, reqBody *CreateChromeWebStoreExtensionRequest) (*EmptyDataResponse, *Response, error) {
	if reqBody == nil {
		return nil, nil, NewArgError("reqBody", "it must not be nil")
	}
	downloadURL, err := ChromeWebStoreExtensionDownloadURL(reqBody.ExtensionID)
	if err != nil {
		return nil, nil, err
	}
	return s.CreateExtensionFromURL(ctx, &CreateExtensionFromURLRequest{
		URL:         downloadURL,
		BrowserType: reqBody.BrowserType,
		StorageType: reqBody.StorageType,
	})
}

func (s *ResourcesServiceOp) EnableExtensionForProfiles(ctx context.Context, resourceID string, reqBody *SetResourceProfilesRequest) (*StringDataResponse, *Response, error) {
	if strings.TrimSpace(resourceID) == "" {
		return nil, nil, NewArgError("resourceID", "it must not be empty")
	}
	if reqBody == nil {
		return nil, nil, NewArgError("reqBody", "it must not be nil")
	}
	if len(reqBody.ProfileIDs) == 0 {
		return nil, nil, NewArgError("profileIDs", "it must not be empty")
	}
	path := fmt.Sprintf("/api/v1/resources/%s/enable_for_profiles", url.PathEscape(resourceID))
	req, err := s.client.newAPIRequest(ctx, http.MethodPost, path, reqBody)
	if err != nil {
		return nil, nil, err
	}
	out := new(StringDataResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *ResourcesServiceOp) DisableExtensionForProfiles(ctx context.Context, resourceID string, reqBody *SetResourceProfilesRequest) (*StringDataResponse, *Response, error) {
	if strings.TrimSpace(resourceID) == "" {
		return nil, nil, NewArgError("resourceID", "it must not be empty")
	}
	if reqBody == nil {
		return nil, nil, NewArgError("reqBody", "it must not be nil")
	}
	if len(reqBody.ProfileIDs) == 0 {
		return nil, nil, NewArgError("profileIDs", "it must not be empty")
	}
	path := fmt.Sprintf("/api/v1/resources/%s/disable_for_profiles", url.PathEscape(resourceID))
	req, err := s.client.newAPIRequest(ctx, http.MethodPost, path, reqBody)
	if err != nil {
		return nil, nil, err
	}
	out := new(StringDataResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *ResourcesServiceOp) Download(ctx context.Context, resourceID string) (*DownloadResourceResponse, *Response, error) {
	if strings.TrimSpace(resourceID) == "" {
		return nil, nil, NewArgError("resourceID", "it must not be empty")
	}
	path := fmt.Sprintf("/api/v1/object_storage/%s/download", url.PathEscape(resourceID))
	req, err := s.client.newLauncherRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	out := new(DownloadResourceResponse)
	resp, err := s.client.do(req, out)
	if err != nil {
		return nil, resp, err
	}
	out.Path = parseDownloadedObjectPath(out.Status.Message)
	if resp != nil {
		resp.Raw = out.Path
	}
	return out, resp, nil
}

func parseDownloadedObjectPath(message string) string {
	const prefix = "Object downloaded to the disk at "
	if strings.HasPrefix(message, prefix) {
		return strings.TrimSpace(strings.TrimPrefix(message, prefix))
	}
	return ""
}

// ChromeWebStoreExtensionDownloadURL converts a Chrome Web Store extension ID into
// the CRX download URL expected by the launcher extension creation endpoint.
func ChromeWebStoreExtensionDownloadURL(extensionID string) (string, error) {
	id := strings.ToLower(strings.TrimSpace(extensionID))
	if !isChromeWebStoreExtensionID(id) {
		return "", NewArgError("extensionID", "it must be a 32-character Chrome Web Store extension id")
	}
	values := url.Values{}
	values.Set("response", "redirect")
	values.Set("prodversion", "131.0.6778.205")
	values.Set("acceptformat", "crx2,crx3")
	values.Set("x", fmt.Sprintf("id=%s&installsource=ondemand&uc", id))
	return "https://clients2.google.com/service/update2/crx?" + values.Encode(), nil
}

func isChromeWebStoreExtensionID(extensionID string) bool {
	if len(extensionID) != 32 {
		return false
	}
	for _, ch := range extensionID {
		if ch < 'a' || ch > 'p' {
			return false
		}
	}
	return true
}
