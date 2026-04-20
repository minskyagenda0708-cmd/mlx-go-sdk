package mlx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func (c *Client) newRequest(ctx context.Context, method string, base stringableURL, path string, body any) (*http.Request, error) {
	if err := ensureContext(ctx); err != nil {
		return nil, err
	}

	baseURL, err := url.Parse(base.String())
	if err != nil {
		return nil, err
	}
	rel, err := url.Parse(strings.TrimPrefix(path, "/"))
	if err != nil {
		return nil, err
	}
	u := baseURL.ResolveReference(rel)

	var reader io.Reader
	if body != nil {
		buf := new(bytes.Buffer)
		if err := json.NewEncoder(buf).Encode(body); err != nil {
			return nil, err
		}
		reader = buf
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), reader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	if c.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

func (c *Client) newAPIRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
	return c.newRequest(ctx, method, c.baseURL, path, body)
}

func (c *Client) newLauncherRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
	return c.newRequest(ctx, method, c.launcherURL, path, body)
}

func (c *Client) newCookiesRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
	return c.newRequest(ctx, method, c.cookiesURL, path, body)
}

func (c *Client) newProxyRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
	return c.newRequest(ctx, method, c.proxyURL, path, body)
}
