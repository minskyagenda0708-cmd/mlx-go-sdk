package mlx

import (
	"encoding/json"
	"io"
	"net/http"
)

func (c *Client) do(req *http.Request, out any) (*Response, error) {
	httpResp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}

	if httpResp.StatusCode < 200 || httpResp.StatusCode > 299 {
		apiErr := &ErrorResponse{Response: httpResp}
		_ = json.Unmarshal(body, apiErr)
		return nil, apiErr
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
