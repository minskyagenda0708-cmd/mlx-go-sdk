package mlx

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// ProxyProtocol identifies the upstream proxy protocol.
type ProxyProtocol string

const (
	ProxyProtocolSOCKS5 ProxyProtocol = "socks5"
	ProxyProtocolHTTP   ProxyProtocol = "http"
)

// ProxySessionType identifies whether generated proxy credentials are sticky or rotating.
type ProxySessionType string

const (
	ProxySessionSticky   ProxySessionType = "sticky"
	ProxySessionRotating ProxySessionType = "rotating"
)

// ProxyService manages MLX profile-proxy workflows.
type ProxyService interface {
	Generate(context.Context, *GenerateProxyRequest) (*GenerateProxyResponse, *Response, error)
	GetUsage(context.Context) (*ProxyUsageResponse, *Response, error)
	ParseConnectionString(string, ProxyProtocol) (*GeneratedProxyConnection, error)
	BuildProfileProxy(*GeneratedProxyConnection) *Proxy
	GenerateProfileProxy(context.Context, *GenerateProfileProxyRequest) (*GenerateProfileProxyResult, error)
}

// ProxyServiceOp is the concrete proxy service implementation.
type ProxyServiceOp struct {
	client *Client
}

// GenerateProxyRequest requests one or more MLX-managed proxy endpoints.
type GenerateProxyRequest struct {
	Country     string           `json:"country,omitempty"`
	SessionType ProxySessionType `json:"sessionType,omitempty"`
	Protocol    ProxyProtocol    `json:"protocol,omitempty"`
	Region      string           `json:"region,omitempty"`
	City        string           `json:"city,omitempty"`
	IPTTL       int              `json:"IPTTL,omitempty"`
	Count       int              `json:"count,omitempty"`
	StrictMode  bool             `json:"-"`
}

// GeneratedProxyConnection is the parsed form of the returned MLX connection string.
type GeneratedProxyConnection struct {
	Raw             string
	Protocol        ProxyProtocol
	Host            string
	Port            int
	Username        string
	Password        string
	Country         string
	Region          string
	City            string
	SessionID       string
	BillingID       string
	Filter          string
	RetentionKey    string
	RetentionSecret string
}

// GenerateProxyResponse contains generated connection strings and parsed variants.
type GenerateProxyResponse struct {
	Status int                         `json:"status"`
	Data   []string                    `json:"data"`
	Parsed []*GeneratedProxyConnection `json:"-"`
}

// UnmarshalJSON accepts both the documented single-string payload and the live array payload.
func (r *GenerateProxyResponse) UnmarshalJSON(data []byte) error {
	type rawGenerateProxyResponse struct {
		Status int             `json:"status"`
		Data   json.RawMessage `json:"data"`
	}
	var raw rawGenerateProxyResponse
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	r.Status = raw.Status
	if len(raw.Data) == 0 || string(raw.Data) == "null" {
		r.Data = nil
		return nil
	}
	var single string
	if err := json.Unmarshal(raw.Data, &single); err == nil {
		r.Data = []string{single}
		return nil
	}
	var many []string
	if err := json.Unmarshal(raw.Data, &many); err != nil {
		return err
	}
	r.Data = many
	return nil
}

// ProxyUsageResponse reports proxy traffic usage for the authenticated account.
type ProxyUsageResponse struct {
	Traffic   int64  `json:"traffic"`
	BillingID string `json:"billingId"`
}

// GenerateProfileProxyRequest generates and adapts an MLX proxy for profile APIs.
type GenerateProfileProxyRequest struct {
	GenerateProxyRequest
	PreferSOCKS5 bool
	SaveTraffic  bool
}

// GenerateProfileProxyResult returns both the parsed connection and profile payload.
type GenerateProfileProxyResult struct {
	Connection   *GeneratedProxyConnection
	ProfileProxy *Proxy
	Usage        *ProxyUsageResponse
}

// Generate requests MLX-managed proxies and parses the returned connection strings.
func (s *ProxyServiceOp) Generate(ctx context.Context, reqBody *GenerateProxyRequest) (*GenerateProxyResponse, *Response, error) {
	if reqBody == nil {
		return nil, nil, NewArgError("reqBody", "it must not be nil")
	}
	normalized, err := normalizeGenerateProxyRequest(reqBody)
	if err != nil {
		return nil, nil, err
	}
	req, err := s.client.newProxyRequest(ctx, http.MethodPost, "/v1/proxy/connection_url", normalized)
	if err != nil {
		return nil, nil, err
	}
	if normalized.StrictMode {
		req.Header.Set("X-Strict-Mode", "true")
	}
	out := new(GenerateProxyResponse)
	resp, err := s.client.do(req, out)
	if err != nil {
		return nil, resp, err
	}
	out.Parsed = make([]*GeneratedProxyConnection, 0, len(out.Data))
	for _, raw := range out.Data {
		parsed, parseErr := s.ParseConnectionString(raw, normalized.Protocol)
		if parseErr != nil {
			return nil, resp, parseErr
		}
		out.Parsed = append(out.Parsed, parsed)
	}
	return out, resp, nil
}

// GetUsage returns proxy traffic data for the authenticated account.
func (s *ProxyServiceOp) GetUsage(ctx context.Context) (*ProxyUsageResponse, *Response, error) {
	req, err := s.client.newProxyRequest(ctx, http.MethodGet, "/v1/user", nil)
	if err != nil {
		return nil, nil, err
	}
	out := new(ProxyUsageResponse)
	resp, err := s.client.do(req, out)
	return out, resp, err
}

// ParseConnectionString parses the MLX connection string into a typed proxy model.
func (s *ProxyServiceOp) ParseConnectionString(raw string, protocol ProxyProtocol) (*GeneratedProxyConnection, error) {
	return ParseGeneratedProxyConnection(raw, protocol)
}

// BuildProfileProxy converts a parsed generated connection into the profile-bound proxy payload.
func (s *ProxyServiceOp) BuildProfileProxy(conn *GeneratedProxyConnection) *Proxy {
	return BuildProfileProxyFromGenerated(conn)
}

// GenerateProfileProxy generates one proxy connection and converts it into a profile payload.
func (s *ProxyServiceOp) GenerateProfileProxy(ctx context.Context, reqBody *GenerateProfileProxyRequest) (*GenerateProfileProxyResult, error) {
	if reqBody == nil {
		return nil, NewArgError("reqBody", "it must not be nil")
	}
	genReq := reqBody.GenerateProxyRequest
	if reqBody.PreferSOCKS5 && genReq.Protocol == "" {
		genReq.Protocol = ProxyProtocolSOCKS5
	}
	usage, _, err := s.GetUsage(ctx)
	if err != nil {
		return nil, err
	}
	genResp, _, err := s.Generate(ctx, &genReq)
	if err != nil {
		return nil, err
	}
	if len(genResp.Parsed) == 0 || genResp.Parsed[0] == nil {
		return nil, fmt.Errorf("mlx proxy api returned no proxy connections")
	}
	proxy := BuildProfileProxyFromGenerated(genResp.Parsed[0])
	proxy.SaveTraffic = reqBody.SaveTraffic
	return &GenerateProfileProxyResult{
		Connection:   genResp.Parsed[0],
		ProfileProxy: proxy,
		Usage:        usage,
	}, nil
}

// ParseGeneratedProxyConnection parses one MLX proxy connection string.
func ParseGeneratedProxyConnection(raw string, protocol ProxyProtocol) (*GeneratedProxyConnection, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, NewArgError("raw", "it must not be empty")
	}
	parts := strings.Split(trimmed, ":")
	if len(parts) < 4 {
		return nil, fmt.Errorf("invalid MLX proxy connection string: %q", raw)
	}
	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid proxy port in connection string %q: %w", raw, err)
	}
	username := parts[2]
	password := strings.Join(parts[3:], ":")
	conn := &GeneratedProxyConnection{
		Raw:      trimmed,
		Protocol: normalizeProxyProtocol(protocol),
		Host:     parts[0],
		Port:     port,
		Username: username,
		Password: password,
	}
	applyGeneratedProxyAffinity(conn)
	return conn, nil
}

// BuildProfileProxyFromGenerated converts a parsed connection into a profile proxy payload.
func BuildProfileProxyFromGenerated(conn *GeneratedProxyConnection) *Proxy {
	if conn == nil {
		return nil
	}
	return &Proxy{
		Host:             conn.Host,
		Type:             string(normalizeProxyProtocol(conn.Protocol)),
		Port:             conn.Port,
		Username:         conn.Username,
		Password:         conn.Password,
		Country:          conn.Country,
		Region:           conn.Region,
		City:             conn.City,
		SessionID:        conn.SessionID,
		Provider:         "multilogin",
		ConnectionString: conn.Raw,
		RetentionKey:     conn.RetentionKey,
		RetentionSecret:  conn.RetentionSecret,
	}
}

func normalizeGenerateProxyRequest(reqBody *GenerateProxyRequest) (*GenerateProxyRequest, error) {
	cloned := *reqBody
	if strings.TrimSpace(cloned.Country) == "" {
		cloned.Country = "any"
	}
	if cloned.SessionType == "" {
		cloned.SessionType = ProxySessionSticky
	}
	cloned.Protocol = normalizeProxyProtocol(cloned.Protocol)
	if cloned.Count <= 0 {
		cloned.Count = 1
	}
	if cloned.SessionType == ProxySessionRotating && cloned.IPTTL <= 0 {
		cloned.IPTTL = 86400
	}
	if cloned.Protocol != ProxyProtocolSOCKS5 && cloned.Protocol != ProxyProtocolHTTP {
		return nil, NewArgError("reqBody.Protocol", "it must be "+string(ProxyProtocolSOCKS5)+" or "+string(ProxyProtocolHTTP))
	}
	if cloned.SessionType != ProxySessionSticky && cloned.SessionType != ProxySessionRotating {
		return nil, NewArgError("reqBody.SessionType", "it must be sticky or rotating")
	}
	return &cloned, nil
}

func normalizeProxyProtocol(protocol ProxyProtocol) ProxyProtocol {
	if strings.EqualFold(string(protocol), string(ProxyProtocolHTTP)) {
		return ProxyProtocolHTTP
	}
	return ProxyProtocolSOCKS5
}

func applyGeneratedProxyAffinity(conn *GeneratedProxyConnection) {
	if conn == nil {
		return
	}
	usernameParts := strings.Split(conn.Username, "-")
	for i := 0; i < len(usernameParts)-1; i++ {
		switch usernameParts[i] {
		case "country":
			conn.Country = usernameParts[i+1]
		case "region":
			conn.Region = usernameParts[i+1]
		case "city":
			conn.City = usernameParts[i+1]
		case "sid":
			conn.SessionID = usernameParts[i+1]
		case "filter":
			conn.Filter = usernameParts[i+1]
		}
	}
	userParts := strings.SplitN(conn.Username, "_", 3)
	if len(userParts) > 0 {
		conn.BillingID = userParts[0]
	}
	if len(userParts) >= 2 {
		conn.RetentionKey = userParts[0] + "_" + userParts[1]
	}
	conn.RetentionSecret = conn.Password
}
