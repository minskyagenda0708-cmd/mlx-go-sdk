package mlx

import (
	"context"
	"net/http"
)

// Client is the main entry point for the MultiloginX SDK.
type Client struct {
	httpClient  *http.Client
	baseURL     stringableURL
	launcherURL stringableURL
	cookiesURL  stringableURL
	proxyURL    stringableURL
	retry       RetryOptions
	retrySet    bool
	token       string
	userAgent   string

	Profiles  ProfilesService
	Launcher  LauncherService
	Proxies   ProxyService
	Folders   FoldersService
	Transfers TransfersService
	Archives  ArchiveManager
	Cookies   CookiesService
	Resources ResourcesService
	Workflows WorkflowService
	Tags      TagsService
}

type stringableURL interface {
	String() string
}

// New creates a new MultiloginX client.
func New(opts ...Option) (*Client, error) {
	baseURL, err := parseBaseURL(defaultBaseURL)
	if err != nil {
		return nil, err
	}
	launcherURL, err := parseBaseURL(defaultLauncherURL)
	if err != nil {
		return nil, err
	}
	cookiesURL, err := parseBaseURL(defaultCookiesURL)
	if err != nil {
		return nil, err
	}
	proxyURL, err := parseBaseURL(defaultProxyURL)
	if err != nil {
		return nil, err
	}

	c := &Client{
		httpClient:  &http.Client{},
		baseURL:     baseURL,
		launcherURL: launcherURL,
		cookiesURL:  cookiesURL,
		proxyURL:    proxyURL,
		userAgent:   defaultUserAgent,
	}

	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}

	if c.token == "" {
		return nil, ErrMissingToken
	}

	c.Profiles = &ProfilesServiceOp{client: c}
	c.Launcher = &LauncherServiceOp{client: c}
	c.Proxies = &ProxyServiceOp{client: c}
	c.Folders = &FoldersServiceOp{client: c}
	c.Transfers = &TransfersServiceOp{client: c}
	c.Archives = &ArchiveManagerOp{client: c}
	c.Cookies = &CookiesServiceOp{client: c}
	c.Resources = &ResourcesServiceOp{client: c}
	c.Workflows = &WorkflowServiceOp{client: c}
	c.Tags = &TagsServiceOp{client: c}

	return c, nil
}

// NewFromEnv creates a client using the `MLX_TOKEN` environment variable.
func NewFromEnv(opts ...Option) (*Client, error) {
	defaults := []Option{
		WithToken(tokenFromEnv()),
		WithBaseURL(envOrDefault(EnvBaseURL, defaultBaseURL)),
		WithLauncherURL(envOrDefault(EnvLauncherURL, defaultLauncherURL)),
		WithCookiesURL(envOrDefault(EnvCookiesURL, defaultCookiesURL)),
		WithProxyURL(envOrDefault(EnvProxyURL, defaultProxyURL)),
	}
	opts = append(defaults, opts...)
	return New(opts...)
}

func ensureContext(ctx context.Context) error {
	if ctx == nil {
		return ErrNilContext
	}
	return nil
}
