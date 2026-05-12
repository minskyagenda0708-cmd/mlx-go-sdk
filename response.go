package mlx

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

func (c *Client) do(req *http.Request, out any) (*Response, error) {
	opts, retryEnabled := c.retryOptions(req)
	if !retryEnabled {
		return c.doOnce(req, out)
	}
	interval := opts.InitialInterval

	for attempt := 1; attempt <= opts.MaxAttempts; attempt++ {
		attemptReq, err := cloneRequestForAttempt(req)
		if err != nil {
			return nil, err
		}

		httpResp, err := c.httpClient.Do(attemptReq)
		if err != nil {
			transportErr := &TransportError{Request: attemptReq, Err: err}
			if attempt == opts.MaxAttempts || !opts.ShouldRetry(transportErr) {
				return nil, transportErr
			}
			if err := waitWithContext(req.Context(), retryDelay(transportErr, interval, opts)); err != nil {
				return nil, err
			}
			interval = nextRetryInterval(interval, opts.Multiplier, opts.MaxInterval)
			continue
		}

		body, err := io.ReadAll(httpResp.Body)
		httpResp.Body.Close()
		if err != nil {
			return nil, err
		}

		if httpResp.StatusCode < 200 || httpResp.StatusCode > 299 {
			apiErr := &ErrorResponse{Response: httpResp, Body: body}
			_ = json.Unmarshal(body, apiErr)
			if attempt == opts.MaxAttempts || !opts.ShouldRetry(apiErr) {
				return nil, apiErr
			}
			if err := waitWithContext(req.Context(), retryDelay(apiErr, interval, opts)); err != nil {
				return nil, err
			}
			interval = nextRetryInterval(interval, opts.Multiplier, opts.MaxInterval)
			continue
		}

		resp := &Response{}
		if out == nil {
			var envelope Envelope[json.RawMessage]
			if len(body) > 0 {
				if err := json.Unmarshal(body, &envelope); err != nil {
					return nil, err
				}
				resp.Status = envelope.Status
				resp.Raw = envelope.Data
			}
			return resp, nil
		}

		if len(body) > 0 {
			if err := json.Unmarshal(body, out); err != nil {
				return nil, err
			}
			switch v := out.(type) {
			case interface{ GetStatus() Status }:
				resp.Status = v.GetStatus()
			default:
				resp.Raw = out
			}
		}

		return resp, nil
	}

	return nil, nil
}

func (c *Client) doOnce(req *http.Request, out any) (*Response, error) {
	attemptReq, err := cloneRequestForAttempt(req)
	if err != nil {
		return nil, err
	}

	httpResp, err := c.httpClient.Do(attemptReq)
	if err != nil {
		return nil, &TransportError{Request: attemptReq, Err: err}
	}

	body, err := io.ReadAll(httpResp.Body)
	httpResp.Body.Close()
	if err != nil {
		return nil, err
	}

	if httpResp.StatusCode < 200 || httpResp.StatusCode > 299 {
		apiErr := &ErrorResponse{Response: httpResp, Body: body}
		_ = json.Unmarshal(body, apiErr)
		return nil, apiErr
	}

	return decodeResponseBody(body, out)
}

func (c *Client) retryOptions(req *http.Request) (RetryOptions, bool) {
	if !c.retrySet || req == nil || !isRetryableMethod(req.Method) {
		return RetryOptions{}, false
	}
	return normalizeRetryOptions(c.retry), true
}

func isRetryableMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace, http.MethodPut, http.MethodDelete:
		return true
	default:
		return false
	}
}

func decodeResponseBody(body []byte, out any) (*Response, error) {
	resp := &Response{}
	if out == nil {
		var envelope Envelope[json.RawMessage]
		if len(body) > 0 {
			if err := json.Unmarshal(body, &envelope); err != nil {
				return nil, err
			}
			resp.Status = envelope.Status
			resp.Raw = envelope.Data
		}
		return resp, nil
	}

	if len(body) > 0 {
		if err := json.Unmarshal(body, out); err != nil {
			return nil, err
		}
		switch v := out.(type) {
		case interface{ GetStatus() Status }:
			resp.Status = v.GetStatus()
		default:
			resp.Raw = out
		}
	}

	return resp, nil
}

func cloneRequestForAttempt(req *http.Request) (*http.Request, error) {
	cloned := req.Clone(req.Context())
	if req.Body == nil {
		return cloned, nil
	}
	if req.GetBody != nil {
		body, err := req.GetBody()
		if err != nil {
			return nil, err
		}
		cloned.Body = body
		return cloned, nil
	}
	if req.Body == http.NoBody {
		cloned.Body = http.NoBody
		return cloned, nil
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	req.Body = io.NopCloser(bytes.NewReader(body))
	cloned.Body = io.NopCloser(bytes.NewReader(body))
	cloned.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	return cloned, nil
}
