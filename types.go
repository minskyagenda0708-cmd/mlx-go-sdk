package mlx

import "time"

// Status is the common status envelope returned by MultiloginX endpoints.
type Status struct {
	ErrorCode string `json:"error_code"`
	HTTPCode  int    `json:"http_code"`
	Message   string `json:"message"`
}

// Envelope is the common response wrapper used by MultiloginX endpoints.
type Envelope[T any] struct {
	Status Status `json:"status"`
	Data   T      `json:"data"`
}

// Response wraps an HTTP response together with the decoded status envelope.
type Response struct {
	Status Status
	Raw    any
}

// ListOptions models offset-based listing used by profile search.
type ListOptions struct {
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
}

// TimeRangeFilter is reused by search requests when needed later.
type TimeRangeFilter struct {
	From *time.Time
	To   *time.Time
}
