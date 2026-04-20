package mlx

import (
	"context"
	"net/http"
)

// FoldersService manages workspace folders.
type FoldersService interface {
	List(context.Context) (*ListFoldersResponse, *Response, error)
	Create(context.Context, *CreateFolderRequest) (*CreateFolderResponse, *Response, error)
	Update(context.Context, *UpdateFolderRequest) (*EmptyDataResponse, *Response, error)
	Delete(context.Context, *DeleteFoldersRequest) (*EmptyDataResponse, *Response, error)
}

// FoldersServiceOp is the concrete folder service.
type FoldersServiceOp struct {
	client *Client
}

// Folder describes a workspace folder.
type Folder struct {
	FolderID      string `json:"folder_id"`
	Name          string `json:"name"`
	Comment       string `json:"comment"`
	ProfilesCount int    `json:"profiles_count"`
	CreatedAt     string `json:"created_at"`
}

// ListFoldersResponse contains folder listings.
type ListFoldersResponse struct {
	Status Status          `json:"status"`
	Data   ListFoldersData `json:"data"`
}

func (r *ListFoldersResponse) GetStatus() Status { return r.Status }

// ListFoldersData wraps the list of folders.
type ListFoldersData struct {
	Folders []Folder `json:"folders"`
}

// CreateFolderRequest creates a folder.
type CreateFolderRequest struct {
	Name    string `json:"name"`
	Comment string `json:"comment,omitempty"`
}

// CreateFolderResponse contains the created folder ID.
type CreateFolderResponse struct {
	Status Status            `json:"status"`
	Data   CreatedFolderData `json:"data"`
}

func (r *CreateFolderResponse) GetStatus() Status { return r.Status }

// CreatedFolderData wraps the created folder ID.
type CreatedFolderData struct {
	ID string `json:"id"`
}

// UpdateFolderRequest updates a folder.
type UpdateFolderRequest struct {
	FolderID string `json:"folder_id"`
	Name     string `json:"name"`
	Comment  string `json:"comment,omitempty"`
}

// DeleteFoldersRequest removes folders.
type DeleteFoldersRequest struct {
	IDs []string `json:"ids"`
}

func (s *FoldersServiceOp) List(ctx context.Context) (*ListFoldersResponse, *Response, error) {
	req, err := s.client.newAPIRequest(ctx, http.MethodGet, "/workspace/folders", nil)
	if err != nil {
		return nil, nil, err
	}
	out := new(ListFoldersResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *FoldersServiceOp) Create(ctx context.Context, reqBody *CreateFolderRequest) (*CreateFolderResponse, *Response, error) {
	if reqBody == nil {
		return nil, nil, NewArgError("reqBody", "it must not be nil")
	}
	req, err := s.client.newAPIRequest(ctx, http.MethodPost, "/workspace/folder_create", reqBody)
	if err != nil {
		return nil, nil, err
	}
	out := new(CreateFolderResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *FoldersServiceOp) Update(ctx context.Context, reqBody *UpdateFolderRequest) (*EmptyDataResponse, *Response, error) {
	if reqBody == nil {
		return nil, nil, NewArgError("reqBody", "it must not be nil")
	}
	req, err := s.client.newAPIRequest(ctx, http.MethodPost, "/workspace/folder_update", reqBody)
	if err != nil {
		return nil, nil, err
	}
	out := new(EmptyDataResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *FoldersServiceOp) Delete(ctx context.Context, reqBody *DeleteFoldersRequest) (*EmptyDataResponse, *Response, error) {
	if reqBody == nil {
		return nil, nil, NewArgError("reqBody", "it must not be nil")
	}
	req, err := s.client.newAPIRequest(ctx, http.MethodPost, "/workspace/folders_remove", reqBody)
	if err != nil {
		return nil, nil, err
	}
	out := new(EmptyDataResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}
