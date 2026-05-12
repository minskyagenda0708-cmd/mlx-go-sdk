package mlx

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"
)

const (
	defaultBaseURL     = "https://api.multilogin.com/"
	defaultLauncherURL = "https://launcher.mlx.yt:45001/"
	defaultCookiesURL  = "https://cookies.multilogin.com/"
	defaultProxyURL    = "https://profile-proxy.multilogin.com/"
	defaultUserAgent   = "mlx-go-sdk/0.1"
	EnvBaseURL         = "MLX_BASE_URL"
	EnvLauncherURL     = "MLX_LAUNCHER_URL"
	EnvCookiesURL      = "MLX_COOKIES_URL"
	EnvProxyURL        = "MLX_PROXY_URL"
	EnvRunE2E          = "MLX_RUN_E2E"
	EnvE2EFolderID     = "MLX_E2E_FOLDER_ID"
	EnvE2EProfileID    = "MLX_E2E_PROFILE_ID"
)

// Option configures a Client.
type Option func(*Client) error

// WithBaseURL overrides the REST API base URL.
func WithBaseURL(raw string) Option {
	return func(c *Client) error {
		u, err := parseBaseURL(raw)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidBaseURL, err)
		}
		c.baseURL = u
		return nil
	}
}

// WithLauncherURL overrides the local launcher base URL.
func WithLauncherURL(raw string) Option {
	return func(c *Client) error {
		u, err := parseBaseURL(raw)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidLauncherURL, err)
		}
		c.launcherURL = u
		return nil
	}
}

// WithCookiesURL overrides the pre-made cookies API base URL.
func WithCookiesURL(raw string) Option {
	return func(c *Client) error {
		u, err := parseBaseURL(raw)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidBaseURL, err)
		}
		c.cookiesURL = u
		return nil
	}
}

// WithProxyURL overrides the MLX profile proxy API base URL.
func WithProxyURL(raw string) Option {
	return func(c *Client) error {
		u, err := parseBaseURL(raw)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidBaseURL, err)
		}
		c.proxyURL = u
		return nil
	}
}

// WithHTTPClient overrides the underlying HTTP client.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) error {
		if httpClient == nil {
			return NewArgError("httpClient", "it must not be nil")
		}
		c.httpClient = httpClient
		return nil
	}
}

// WithToken sets the bearer token explicitly.
func WithToken(token string) Option {
	return func(c *Client) error {
		if token == "" {
			return ErrMissingToken
		}
		c.token = token
		return nil
	}
}

// WithUserAgent sets the user agent header.
func WithUserAgent(userAgent string) Option {
	return func(c *Client) error {
		if userAgent == "" {
			return NewArgError("userAgent", "it must not be empty")
		}
		c.userAgent = userAgent
		return nil
	}
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) error {
		if timeout <= 0 {
			return NewArgError("timeout", "it must be greater than zero")
		}
		if c.httpClient == nil {
			c.httpClient = &http.Client{}
		}
		c.httpClient.Timeout = timeout
		return nil
	}
}

// WithRetry configures automatic retries for transient transport and MLX API failures.
func WithRetry(opts RetryOptions) Option {
	return func(c *Client) error {
		c.retry = normalizeRetryOptions(opts)
		c.retrySet = true
		return nil
	}
}

func parseBaseURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("url must include scheme and host")
	}
	if u.Path == "" {
		u.Path = "/"
	}
	if u.Path[len(u.Path)-1] != '/' {
		u.Path += "/"
	}
	return u, nil
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
