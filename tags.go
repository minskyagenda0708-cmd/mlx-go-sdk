package mlx

import (
	"context"
	"net/http"
)

// TagsService manages tags.
type TagsService interface {
	Create(context.Context, *CreateTagsRequest) (*TagsResponse, *Response, error)
	Update(context.Context, *UpdateTagsRequest) (*TagsResponse, *Response, error)
	Remove(context.Context, *RemoveTagsRequest) (*EmptyDataResponse, *Response, error)
	AssignToProfiles(context.Context, *AssignTagsRequest) (*EmptyDataResponse, *Response, error)
	Search(context.Context, *SearchTagsRequest) (*SearchTagsResponse, *Response, error)
}

// TagsServiceOp is the concrete TagsService implementation.
type TagsServiceOp struct {
	client *Client
}

// Tag describes a tag.
type Tag struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Color      string `json:"color"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
	CreatedBy  string `json:"created_by"`
	InUseCount int    `json:"in_use_count"`
}

// CreateTagsRequest creates tags.
type CreateTagsRequest struct {
	Tags []CreateTagItem `json:"tags"`
}

// CreateTagItem is a single tag to create.
type CreateTagItem struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// UpdateTagsRequest updates tags.
type UpdateTagsRequest struct {
	Tags []UpdateTagItem `json:"tags"`
}

// UpdateTagItem is a single tag to update.
type UpdateTagItem struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// RemoveTagsRequest removes tags.
type RemoveTagsRequest struct {
	IDs []string `json:"ids"`
}

// AssignTagsRequest assigns tags to profiles.
type AssignTagsRequest struct {
	TagIDs     []string `json:"tag_ids"`
	ProfileIDs []string `json:"profile_ids"`
}

// SearchTagsRequest searches tags.
type SearchTagsRequest struct {
	SearchText string `json:"search_text"`
	Limit      int    `json:"limit"`
	Offset     int    `json:"offset"`
	OrderBy    string `json:"order_by"`
	Sort       string `json:"sort"`
}

// TagsResponse contains tag results.
type TagsResponse struct {
	Status Status   `json:"status"`
	Data   TagsData `json:"data"`
}

// GetStatus implements the status getter interface.
func (r *TagsResponse) GetStatus() Status { return r.Status }

// TagsData wraps the list of tags.
type TagsData struct {
	Tags []Tag `json:"tags"`
}

// SearchTagsResponse contains tag search results.
type SearchTagsResponse struct {
	Status Status         `json:"status"`
	Data   SearchTagsData `json:"data"`
}

// GetStatus implements the status getter interface.
func (r *SearchTagsResponse) GetStatus() Status { return r.Status }

// SearchTagsData wraps the list of tags and total count.
type SearchTagsData struct {
	Tags       []Tag `json:"tags"`
	TotalCount int   `json:"total_count"`
}

func (s *TagsServiceOp) Create(ctx context.Context, reqBody *CreateTagsRequest) (*TagsResponse, *Response, error) {
	if reqBody == nil {
		return nil, nil, NewArgError("reqBody", "it must not be nil")
	}
	req, err := s.client.newAPIRequest(ctx, http.MethodPost, "/tag/create", reqBody)
	if err != nil {
		return nil, nil, err
	}
	out := new(TagsResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *TagsServiceOp) Update(ctx context.Context, reqBody *UpdateTagsRequest) (*TagsResponse, *Response, error) {
	if reqBody == nil {
		return nil, nil, NewArgError("reqBody", "it must not be nil")
	}
	req, err := s.client.newAPIRequest(ctx, http.MethodPost, "/tag/update", reqBody)
	if err != nil {
		return nil, nil, err
	}
	out := new(TagsResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *TagsServiceOp) Remove(ctx context.Context, reqBody *RemoveTagsRequest) (*EmptyDataResponse, *Response, error) {
	if reqBody == nil {
		return nil, nil, NewArgError("reqBody", "it must not be nil")
	}
	req, err := s.client.newAPIRequest(ctx, http.MethodPost, "/tag/remove", reqBody)
	if err != nil {
		return nil, nil, err
	}
	out := new(EmptyDataResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *TagsServiceOp) AssignToProfiles(ctx context.Context, reqBody *AssignTagsRequest) (*EmptyDataResponse, *Response, error) {
	if reqBody == nil {
		return nil, nil, NewArgError("reqBody", "it must not be nil")
	}
	req, err := s.client.newAPIRequest(ctx, http.MethodPost, "/tag/assign_to_profiles", reqBody)
	if err != nil {
		return nil, nil, err
	}
	out := new(EmptyDataResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

func (s *TagsServiceOp) Search(ctx context.Context, reqBody *SearchTagsRequest) (*SearchTagsResponse, *Response, error) {
	if reqBody == nil {
		return nil, nil, NewArgError("reqBody", "it must not be nil")
	}
	req, err := s.client.newAPIRequest(ctx, http.MethodPost, "/tag/search", reqBody)
	if err != nil {
		return nil, nil, err
	}
	out := new(SearchTagsResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}
