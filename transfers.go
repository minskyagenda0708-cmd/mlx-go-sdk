package mlx

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
)

// TransfersService manages import/export operations.
type TransfersService interface {
	Export(context.Context, string) (*ExportProfileResponse, *Response, error)
	ExportStatus(context.Context, string) (*ExportStatusResponse, *Response, error)
	WaitForExportDone(context.Context, string, PollOptions) (*ExportStatusResponse, *Response, error)
	ExportStatuses(context.Context) (*ExportStatusesResponse, *Response, error)
	Import(context.Context, *ImportProfileRequest) (*ImportProfileResponse, *Response, error)
	ImportStatus(context.Context, string) (*ImportStatusResponse, *Response, error)
	WaitForImportDone(context.Context, string, PollOptions) (*ImportStatusResponse, *Response, error)
	ImportStatuses(context.Context) (*ImportStatusesResponse, *Response, error)
}

// TransfersServiceOp is the concrete import/export service.
type TransfersServiceOp struct {
	client *Client
}

// ExportProfileResponse contains export job details.
type ExportProfileResponse struct {
	Status Status         `json:"status"`
	Data   ExportJobState `json:"data"`
}

func (r *ExportProfileResponse) GetStatus() Status { return r.Status }

// ExportStatusResponse contains a single export job status.
type ExportStatusResponse struct {
	Status Status         `json:"status"`
	Data   ExportJobState `json:"data"`
}

func (r *ExportStatusResponse) GetStatus() Status { return r.Status }

// ExportStatusesResponse contains all export jobs.
type ExportStatusesResponse struct {
	Status Status             `json:"status"`
	Data   ExportStatusesData `json:"data"`
}

func (r *ExportStatusesResponse) GetStatus() Status { return r.Status }

// ExportStatusesData wraps export jobs.
type ExportStatusesData struct {
	Statuses []ExportJobState `json:"statuses"`
}

// ExportJobState describes an export job.
type ExportJobState struct {
	ExportID   string `json:"export_id"`
	ExportPath string `json:"export_path"`
	ProfileID  string `json:"profile_id"`
	Status     string `json:"status"`
	Message    string `json:"message"`
	Timestamp  int64  `json:"timestamp"`
}

// ArchivePath returns the best importable archive path for an export job.
//
// Live Multilogin X responses are inconsistent:
// - export start responses often return a `.zip` path
// - export done responses may return the same path without the `.zip` suffix
//
// Import expects the concrete archive file path, so when the launcher returns an
// extensionless export path this method normalizes it to the expected `.zip` path.
func (j ExportJobState) ArchivePath() string {
	if j.ExportPath == "" {
		return ""
	}
	ext := strings.ToLower(filepath.Ext(j.ExportPath))
	if ext == ".zip" {
		return j.ExportPath
	}
	if ext != "" {
		return j.ExportPath
	}
	return j.ExportPath + ".zip"
}

// ImportProfileRequest imports a profile archive.
type ImportProfileRequest struct {
	ImportPath string `json:"import_path"`
	IsLocal    bool   `json:"is_local"`
}

// ImportProfileResponse contains import job details.
type ImportProfileResponse struct {
	Status Status         `json:"status"`
	Data   ImportJobState `json:"data"`
}

func (r *ImportProfileResponse) GetStatus() Status { return r.Status }

// ImportStatusResponse contains a single import status.
type ImportStatusResponse struct {
	Status Status         `json:"status"`
	Data   ImportJobState `json:"data"`
}

func (r *ImportStatusResponse) GetStatus() Status { return r.Status }

// ImportStatusesResponse contains all import jobs.
type ImportStatusesResponse struct {
	Status Status             `json:"status"`
	Data   ImportStatusesData `json:"data"`
}

func (r *ImportStatusesResponse) GetStatus() Status { return r.Status }

// ImportStatusesData wraps all import jobs.
type ImportStatusesData struct {
	Statuses []ImportJobState `json:"statuses"`
}

// ImportJobState describes an import job.
type ImportJobState struct {
	ExportID      string `json:"export_id"`
	ImportID      string `json:"import_id"`
	ImportPath    string `json:"import_path"`
	ExtractedPath string `json:"extracted_path"`
	NewProfileID  string `json:"new_profile_id"`
	Status        string `json:"status"`
	Message       string `json:"message"`
	Timestamp     int64  `json:"timestamp"`
}

func (s *TransfersServiceOp) Export(ctx context.Context, profileID string) (*ExportProfileResponse, *Response, error) {
	if profileID == "" {
		return nil, nil, NewArgError("profileID", "it must not be empty")
	}
	path := fmt.Sprintf("/api/v1/profile/%s/export", url.PathEscape(profileID))
	req, err := s.client.newLauncherRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return nil, nil, err
	}
	out := new(ExportProfileResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *TransfersServiceOp) ExportStatus(ctx context.Context, exportID string) (*ExportStatusResponse, *Response, error) {
	if exportID == "" {
		return nil, nil, NewArgError("exportID", "it must not be empty")
	}
	path := fmt.Sprintf("/api/v1/profile/exports/%s/status", url.PathEscape(exportID))
	req, err := s.client.newLauncherRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	out := new(ExportStatusResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *TransfersServiceOp) WaitForExportDone(ctx context.Context, exportID string, opts PollOptions) (*ExportStatusResponse, *Response, error) {
	if exportID == "" {
		return nil, nil, NewArgError("exportID", "it must not be empty")
	}
	return pollUntil(ctx, opts, fmt.Sprintf("export %s did not reach done status", exportID), func(ctx context.Context) (*ExportStatusResponse, *Response, error) {
		return s.ExportStatus(ctx, exportID)
	}, func(resp *ExportStatusResponse) bool {
		return resp != nil && strings.EqualFold(resp.Data.Status, "done")
	}, func(resp *ExportStatusResponse) string {
		if resp == nil {
			return ""
		}
		return resp.Data.Status
	})
}

func (s *TransfersServiceOp) ExportStatuses(ctx context.Context) (*ExportStatusesResponse, *Response, error) {
	req, err := s.client.newLauncherRequest(ctx, http.MethodGet, "/api/v1/profile/exports/statuses", nil)
	if err != nil {
		return nil, nil, err
	}
	out := new(ExportStatusesResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *TransfersServiceOp) Import(ctx context.Context, reqBody *ImportProfileRequest) (*ImportProfileResponse, *Response, error) {
	if reqBody == nil {
		return nil, nil, NewArgError("reqBody", "it must not be nil")
	}
	req, err := s.client.newLauncherRequest(ctx, http.MethodPost, "/api/v1/profile/import", reqBody)
	if err != nil {
		return nil, nil, err
	}
	out := new(ImportProfileResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *TransfersServiceOp) ImportStatus(ctx context.Context, importID string) (*ImportStatusResponse, *Response, error) {
	if importID == "" {
		return nil, nil, NewArgError("importID", "it must not be empty")
	}
	path := fmt.Sprintf("/api/v1/profile/imports/%s/status", url.PathEscape(importID))
	req, err := s.client.newLauncherRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	out := new(ImportStatusResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *TransfersServiceOp) WaitForImportDone(ctx context.Context, importID string, opts PollOptions) (*ImportStatusResponse, *Response, error) {
	if importID == "" {
		return nil, nil, NewArgError("importID", "it must not be empty")
	}
	return pollUntil(ctx, opts, fmt.Sprintf("import %s did not reach done status", importID), func(ctx context.Context) (*ImportStatusResponse, *Response, error) {
		return s.ImportStatus(ctx, importID)
	}, func(resp *ImportStatusResponse) bool {
		return resp != nil && strings.EqualFold(resp.Data.Status, "done")
	}, func(resp *ImportStatusResponse) string {
		if resp == nil {
			return ""
		}
		return resp.Data.Status
	})
}

func (s *TransfersServiceOp) ImportStatuses(ctx context.Context) (*ImportStatusesResponse, *Response, error) {
	req, err := s.client.newLauncherRequest(ctx, http.MethodGet, "/api/v1/profile/imports/statuses", nil)
	if err != nil {
		return nil, nil, err
	}
	out := new(ImportStatusesResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}
