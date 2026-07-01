package mlx

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ProxyCheckResult is the outcome of a single proxy health check.
type ProxyCheckResult struct {
	Alive     bool   // true if at least one target responded within the timeout
	LatencyMs int    // best (minimum) time-to-first-byte across targets, in ms
	Target    string // the target URL that produced the best measurement
	Err       error  // last error encountered, if Alive is false
}

// ProxyChecker measures whether a proxy is alive and how fast it is.
//
// The default implementation is HTTPProxyChecker. The interface exists so a
// JA3/TLS-impersonating checker can be substituted later without changing callers
// such as EnsureHealthyProxy.
type ProxyChecker interface {
	Check(ctx context.Context, p *Proxy) ProxyCheckResult
}

// proxyURL converts a *Proxy into a URL usable by http.Transport.Proxy.
func proxyURL(p *Proxy) (*url.URL, error) {
	if p == nil || strings.TrimSpace(p.Host) == "" || p.Port <= 0 {
		return nil, fmt.Errorf("proxy host/port missing")
	}
	scheme := strings.ToLower(strings.TrimSpace(p.Type))
	switch scheme {
	case "socks5", "http", "https":
	case "":
		scheme = "http"
	default:
		return nil, fmt.Errorf("unsupported proxy type %q", p.Type)
	}
	u := &url.URL{
		Scheme: scheme,
		Host:   p.Host + ":" + strconv.Itoa(p.Port),
	}
	if p.Username != "" || p.Password != "" {
		u.User = url.UserPassword(p.Username, p.Password)
	}
	return u, nil
}

const defaultCheckUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

// DefaultCheckTargets are browser-common sites used to measure proxy health.
var DefaultCheckTargets = []string{
	"https://www.google.com",
	"https://www.facebook.com",
	"https://medium.com",
}

// HTTPProxyCheckerConfig configures HTTPProxyChecker.
type HTTPProxyCheckerConfig struct {
	Targets          []string      // defaults to DefaultCheckTargets
	PerTargetTimeout time.Duration // defaults to 10s
	UserAgent        string        // defaults to a Chrome-like UA
}

// HTTPProxyChecker measures proxy liveness/latency via stdlib net/http + httptrace.
type HTTPProxyChecker struct {
	targets   []string
	timeout   time.Duration
	userAgent string
}

// NewHTTPProxyChecker builds a checker, applying defaults for empty fields.
func NewHTTPProxyChecker(cfg HTTPProxyCheckerConfig) *HTTPProxyChecker {
	c := &HTTPProxyChecker{
		targets:   cfg.Targets,
		timeout:   cfg.PerTargetTimeout,
		userAgent: cfg.UserAgent,
	}
	if len(c.targets) == 0 {
		c.targets = DefaultCheckTargets
	}
	if c.timeout <= 0 {
		c.timeout = 10 * time.Second
	}
	if c.userAgent == "" {
		c.userAgent = defaultCheckUserAgent
	}
	return c
}

// Check dials each target through the proxy, records TTFB, and returns the best.
func (c *HTTPProxyChecker) Check(ctx context.Context, p *Proxy) ProxyCheckResult {
	transport := &http.Transport{
		DisableKeepAlives: true,
	}
	// Test-only direct sentinel: skip proxy wiring.
	if !(p != nil && p.Type == "direct" && p.Host == "direct") {
		pu, err := proxyURL(p)
		if err != nil {
			return ProxyCheckResult{Alive: false, Err: err}
		}
		transport.Proxy = http.ProxyURL(pu)
	}
	client := &http.Client{Transport: transport}

	best := ProxyCheckResult{Alive: false, LatencyMs: -1}
	for _, target := range c.targets {
		res := c.probe(ctx, client, target)
		if res.Alive && (best.LatencyMs < 0 || res.LatencyMs < best.LatencyMs) {
			best = res
		} else if !best.Alive {
			best.Err = res.Err // remember last error while none alive
		}
	}
	if !best.Alive && best.LatencyMs < 0 {
		best.LatencyMs = 0
	}
	return best
}

func (c *HTTPProxyChecker) probe(ctx context.Context, client *http.Client, target string) ProxyCheckResult {
	tctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	var firstByte time.Time
	trace := &httptrace.ClientTrace{
		GotFirstResponseByte: func() { firstByte = time.Now() },
	}
	tctx = httptrace.WithClientTrace(tctx, trace)

	req, err := http.NewRequestWithContext(tctx, http.MethodGet, target, nil)
	if err != nil {
		return ProxyCheckResult{Alive: false, Target: target, Err: err}
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return ProxyCheckResult{Alive: false, Target: target, Err: err}
	}
	defer resp.Body.Close()

	ttfb := firstByte
	if ttfb.IsZero() {
		ttfb = time.Now()
	}
	latency := ttfb.Sub(start)
	if latency < 0 {
		latency = 0
	}
	return ProxyCheckResult{
		Alive:     true,
		LatencyMs: int(latency.Milliseconds()),
		Target:    target,
	}
}
