package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	mlx "mlx-go-sdk"
)

const (
	// ConfigVersion is the current version of the reference CLI config schema.
	ConfigVersion = "1"

	// EnvConfigFile overrides the config file path used by the CLI.
	EnvConfigFile = "MLX_CONFIG_FILE"
	// EnvOutputFormat overrides the configured output format.
	EnvOutputFormat = "MLX_OUTPUT"
	// EnvTimeout overrides the configured HTTP timeout.
	EnvTimeout = "MLX_TIMEOUT"
	// EnvUserAgent overrides the configured user agent.
	EnvUserAgent = "MLX_USER_AGENT"

	defaultConfigSubdir   = "mlx-go-sdk"
	defaultConfigFileName = "config.json"

	outputFormatTable = "table"
	outputFormatJSON  = "json"
	outputFormatYAML  = "yaml"

	colorModeAuto   = "auto"
	colorModeAlways = "always"
	colorModeNever  = "never"

	storageTypeAll   = "all"
	storageTypeLocal = "local"
	storageTypeCloud = "cloud"
)

// Duration is a config-friendly duration wrapper.
//
// JSON values may be:
// - a string accepted by time.ParseDuration, for example "30s"
// - a number interpreted as seconds
type Duration time.Duration

func (d Duration) Duration() time.Duration { return time.Duration(d) }
func (d Duration) String() string          { return time.Duration(d).String() }
func (d Duration) IsZero() bool            { return time.Duration(d) == 0 }

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *Duration) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		return nil
	}
	if trimmed[0] == '"' {
		var raw string
		if err := json.Unmarshal(data, &raw); err != nil {
			return err
		}
		parsed, err := time.ParseDuration(strings.TrimSpace(raw))
		if err != nil {
			return fmt.Errorf("parse duration %q: %w", raw, err)
		}
		*d = Duration(parsed)
		return nil
	}

	var seconds float64
	if err := json.Unmarshal(data, &seconds); err != nil {
		return fmt.Errorf("duration must be a string or number of seconds: %w", err)
	}
	*d = Duration(time.Duration(seconds * float64(time.Second)))
	return nil
}

// Config is the top-level reference CLI configuration.
//
// Authentication is intentionally not represented here. The CLI must read only
// MLX_TOKEN from the environment and must not accept token values from config.
type Config struct {
	Version   string          `json:"version"`
	Endpoints EndpointsConfig `json:"endpoints"`
	Transport TransportConfig `json:"transport"`
	Retry     RetryConfig     `json:"retry"`
	Poll      PollConfig      `json:"poll"`
	Output    OutputConfig    `json:"output"`
	Defaults  DefaultsConfig  `json:"defaults"`
}

// EndpointsConfig controls URL overrides for the SDK client.
type EndpointsConfig struct {
	BaseURL     string `json:"base_url"`
	LauncherURL string `json:"launcher_url"`
	CookiesURL  string `json:"cookies_url"`
	ProxyURL    string `json:"proxy_url"`
}

// TransportConfig controls shared HTTP transport settings.
type TransportConfig struct {
	Timeout   Duration `json:"timeout"`
	UserAgent string   `json:"user_agent"`
}

// RetryConfig controls SDK retry behavior.
type RetryConfig struct {
	Enabled         bool     `json:"enabled"`
	MaxAttempts     int      `json:"max_attempts"`
	InitialInterval Duration `json:"initial_interval"`
	MaxInterval     Duration `json:"max_interval"`
	Multiplier      float64  `json:"multiplier"`
	Jitter          float64  `json:"jitter"`

	enabledExplicit bool
}

func (c *RetryConfig) UnmarshalJSON(data []byte) error {
	type alias RetryConfig
	current := alias(*c)
	aux := struct {
		Enabled *bool `json:"enabled"`
		*alias
	}{
		alias: &current,
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*c = RetryConfig(current)
	c.enabledExplicit = aux.Enabled != nil
	if aux.Enabled != nil {
		c.Enabled = *aux.Enabled
	}
	return nil
}

// PollConfig controls workflow polling behavior.
type PollConfig struct {
	InitialInterval Duration `json:"initial_interval"`
	MaxInterval     Duration `json:"max_interval"`
	Timeout         Duration `json:"timeout"`
	Multiplier      float64  `json:"multiplier"`
}

// OutputConfig controls renderer defaults.
type OutputConfig struct {
	Format string `json:"format"`
	Pretty bool   `json:"pretty"`
	Color  string `json:"color"`
}

// DefaultsConfig groups per-domain CLI defaults.
type DefaultsConfig struct {
	Folder    FolderDefaultsConfig    `json:"folder"`
	Profile   ProfileDefaultsConfig   `json:"profile"`
	Launcher  LauncherDefaultsConfig  `json:"launcher"`
	Export    ExportDefaultsConfig    `json:"export"`
	Import    ImportDefaultsConfig    `json:"import"`
	Extension ExtensionDefaultsConfig `json:"extension"`
	Cookies   CookiesDefaultsConfig   `json:"cookies"`
	Proxy     ProxyDefaultsConfig     `json:"proxy"`
}

type FolderDefaultsConfig struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ProfileDefaultsConfig struct {
	BrowserType string `json:"browser_type"`
	OSType      string `json:"os_type"`
	StorageType string `json:"storage_type"`
}

type LauncherDefaultsConfig struct {
	AutomationType string `json:"automation_type"`
	Headless       bool   `json:"headless"`
	StrictMode     bool   `json:"strict_mode"`
	WaitForRunning bool   `json:"wait_for_running"`
}

type ExportDefaultsConfig struct {
	RootDir            string `json:"root_dir"`
	StopBeforeExport   bool   `json:"stop_before_export"`
	IgnoreStopNotReady bool   `json:"ignore_stop_not_ready"`
}

type ImportDefaultsConfig struct {
	IsLocal bool `json:"is_local"`
	Wait    bool `json:"wait"`
}

type ExtensionDefaultsConfig struct {
	BrowserType             string `json:"browser_type"`
	StorageType             string `json:"storage_type"`
	RequireProfileUsageRead bool   `json:"require_profile_usage_read"`
}

type CookiesDefaultsConfig struct {
	TargetWebsite           string `json:"target_website"`
	AdditionalWebsite       string `json:"additional_website"`
	CreateMetadataIfMissing bool   `json:"create_metadata_if_missing"`
	ImportAdvancedCookies   bool   `json:"import_advanced_cookies"`
	StrictMode              bool   `json:"strict_mode"`
}

type ProxyDefaultsConfig struct {
	Protocol    string `json:"protocol"`
	SessionType string `json:"session_type"`
	Country     string `json:"country"`
	Region      string `json:"region"`
	City        string `json:"city"`

	PreferSOCKS5 bool `json:"prefer_socks5"`
	SaveTraffic  bool `json:"save_traffic"`
	PatchProfile bool `json:"patch_profile"`
}

// DefaultConfig returns normalized built-in CLI defaults.
func DefaultConfig() Config {
	cfg := builtinDefaultConfig()
	return cfg.Normalize()
}

func builtinDefaultConfig() Config {
	return Config{
		Version: ConfigVersion,
		Endpoints: EndpointsConfig{
			BaseURL:     "",
			LauncherURL: "",
			CookiesURL:  "",
			ProxyURL:    "",
		},
		Transport: TransportConfig{
			Timeout:   Duration(30 * time.Second),
			UserAgent: "mlx-go-sdk-cli",
		},
		Retry: RetryConfig{
			Enabled:         true,
			MaxAttempts:     4,
			InitialInterval: Duration(500 * time.Millisecond),
			MaxInterval:     Duration(3 * time.Second),
			Multiplier:      2,
			Jitter:          0.2,
		},
		Poll: PollConfig{
			InitialInterval: Duration(2 * time.Second),
			MaxInterval:     Duration(10 * time.Second),
			Timeout:         Duration(2 * time.Minute),
			Multiplier:      1.5,
		},
		Output: OutputConfig{
			Format: outputFormatTable,
			Pretty: true,
			Color:  colorModeAuto,
		},
		Defaults: DefaultsConfig{
			Folder: FolderDefaultsConfig{
				Name: "Default folder",
			},
			Profile: ProfileDefaultsConfig{
				BrowserType: "mimic",
				OSType:      "windows",
				StorageType: storageTypeAll,
			},
			Launcher: LauncherDefaultsConfig{
				AutomationType: string(mlx.AutomationPlaywright),
				Headless:       false,
				StrictMode:     false,
				WaitForRunning: false,
			},
			Export: ExportDefaultsConfig{
				StopBeforeExport:   true,
				IgnoreStopNotReady: false,
			},
			Import: ImportDefaultsConfig{
				IsLocal: false,
				Wait:    false,
			},
			Extension: ExtensionDefaultsConfig{
				BrowserType:             "mimic",
				StorageType:             storageTypeCloud,
				RequireProfileUsageRead: false,
			},
			Cookies: CookiesDefaultsConfig{
				CreateMetadataIfMissing: true,
				ImportAdvancedCookies:   false,
				StrictMode:              false,
			},
			Proxy: ProxyDefaultsConfig{
				Protocol:     string(mlx.ProxyProtocolSOCKS5),
				SessionType:  string(mlx.ProxySessionSticky),
				PreferSOCKS5: true,
				SaveTraffic:  false,
				PatchProfile: true,
			},
		},
	}
}

// DefaultConfigPath returns the default on-disk config path.
func DefaultConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(dir, defaultConfigSubdir, defaultConfigFileName), nil
}

// ResolveConfigPath resolves the config path using:
// 1. an explicit path argument
// 2. MLX_CONFIG_FILE
// 3. the default user config location
func ResolveConfigPath(explicit string) (string, error) {
	if trimmed := strings.TrimSpace(explicit); trimmed != "" {
		return filepath.Clean(trimmed), nil
	}
	if fromEnv := strings.TrimSpace(os.Getenv(EnvConfigFile)); fromEnv != "" {
		return filepath.Clean(fromEnv), nil
	}
	return DefaultConfigPath()
}

// LoadConfig loads the CLI config from disk, applies environment overrides, and
// validates the final result.
//
// If no explicit path is provided and no config file exists at the resolved
// location, LoadConfig returns the default config merged with environment
// overrides. If an explicit path or MLX_CONFIG_FILE is provided and the file does
// not exist, an error is returned.
func LoadConfig(explicitPath string) (Config, error) {
	cfg := DefaultConfig()

	path, explicit, err := resolveConfigPathWithExplicitness(explicitPath)
	if err != nil {
		return Config{}, err
	}

	if path != "" {
		file, err := os.Open(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) && !explicit {
				return ApplyEnvOverrides(cfg, os.Getenv)
			}
			return Config{}, fmt.Errorf("open cli config %q: %w", path, err)
		}
		defer file.Close()

		cfg, err = DecodeConfig(file, cfg)
		if err != nil {
			return Config{}, fmt.Errorf("load cli config %q: %w", path, err)
		}
	}

	return ApplyEnvOverrides(cfg, os.Getenv)
}

// DecodeConfig decodes JSON config content on top of the provided base config.
func DecodeConfig(r io.Reader, base Config) (Config, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()

	cfg := base
	if err := dec.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode cli config: %w", err)
	}
	if err := ensureJSONEOF(dec); err != nil {
		return Config{}, err
	}

	cfg = cfg.Normalize()
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// ApplyEnvOverrides applies supported environment variable overrides to cfg.
//
// Supported overrides:
// - MLX_BASE_URL
// - MLX_LAUNCHER_URL
// - MLX_COOKIES_URL
// - MLX_PROXY_URL
// - MLX_TIMEOUT
// - MLX_OUTPUT
// - MLX_USER_AGENT
func ApplyEnvOverrides(cfg Config, lookup func(string) string) (Config, error) {
	if lookup == nil {
		lookup = os.Getenv
	}

	if v := strings.TrimSpace(lookup(mlx.EnvBaseURL)); v != "" {
		cfg.Endpoints.BaseURL = v
	}
	if v := strings.TrimSpace(lookup(mlx.EnvLauncherURL)); v != "" {
		cfg.Endpoints.LauncherURL = v
	}
	if v := strings.TrimSpace(lookup(mlx.EnvCookiesURL)); v != "" {
		cfg.Endpoints.CookiesURL = v
	}
	if v := strings.TrimSpace(lookup(mlx.EnvProxyURL)); v != "" {
		cfg.Endpoints.ProxyURL = v
	}
	if v := strings.TrimSpace(lookup(EnvTimeout)); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return Config{}, fmt.Errorf("parse %s=%q: %w", EnvTimeout, v, err)
		}
		cfg.Transport.Timeout = Duration(d)
	}
	if v := strings.TrimSpace(lookup(EnvOutputFormat)); v != "" {
		cfg.Output.Format = v
	}
	if v := strings.TrimSpace(lookup(EnvUserAgent)); v != "" {
		cfg.Transport.UserAgent = v
	}

	cfg = cfg.Normalize()
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Runtime contains the resolved CLI configuration and shared SDK client.
type Runtime struct {
	Config     Config
	ConfigPath string
	Token      string
	Client     *mlx.Client
}

// TokenFromEnv returns the required MLX token from the environment.
func TokenFromEnv() (string, error) {
	token := strings.TrimSpace(os.Getenv(mlx.EnvToken))
	if token == "" {
		return "", fmt.Errorf("%s is required", mlx.EnvToken)
	}
	return token, nil
}

// NewClientFromConfig constructs one shared SDK client using the resolved CLI
// config and MLX_TOKEN from the environment.
func NewClientFromConfig(cfg Config) (*mlx.Client, string, error) {
	token, err := TokenFromEnv()
	if err != nil {
		return nil, "", err
	}

	opts := append([]mlx.Option{mlx.WithToken(token)}, cfg.ClientOptions()...)
	client, err := mlx.New(opts...)
	if err != nil {
		return nil, "", err
	}

	return client, token, nil
}

// LoadRuntime resolves config, token, and the shared SDK client bootstrap used
// by CLI commands.
func LoadRuntime(explicitConfigPath string) (*Runtime, error) {
	configPath, err := ResolveConfigPath(explicitConfigPath)
	if err != nil {
		return nil, err
	}

	cfg, err := LoadConfig(explicitConfigPath)
	if err != nil {
		return nil, err
	}

	client, token, err := NewClientFromConfig(cfg)
	if err != nil {
		return nil, err
	}

	return &Runtime{
		Config:     cfg,
		ConfigPath: configPath,
		Token:      token,
		Client:     client,
	}, nil
}

// Normalize fills zero-value fields with built-in defaults and normalizes enum
// casing where appropriate.
func (c Config) Normalize() Config {
	def := builtinDefaultConfig()

	if strings.TrimSpace(c.Version) == "" {
		c.Version = def.Version
	}

	if c.Transport.Timeout.IsZero() {
		c.Transport.Timeout = def.Transport.Timeout
	}
	if strings.TrimSpace(c.Transport.UserAgent) == "" {
		c.Transport.UserAgent = def.Transport.UserAgent
	}

	if c.Retry.MaxAttempts <= 0 {
		c.Retry.MaxAttempts = def.Retry.MaxAttempts
	}
	if c.Retry.InitialInterval.IsZero() {
		c.Retry.InitialInterval = def.Retry.InitialInterval
	}
	if c.Retry.MaxInterval.IsZero() {
		c.Retry.MaxInterval = def.Retry.MaxInterval
	}
	if c.Retry.Multiplier <= 1 {
		c.Retry.Multiplier = def.Retry.Multiplier
	}
	if c.Retry.Jitter < 0 {
		c.Retry.Jitter = def.Retry.Jitter
	}
	if !c.Retry.Enabled && !c.Retry.enabledExplicit && isRetryUnset(c.Retry) {
		c.Retry.Enabled = def.Retry.Enabled
	}

	if c.Poll.InitialInterval.IsZero() {
		c.Poll.InitialInterval = def.Poll.InitialInterval
	}
	if c.Poll.MaxInterval.IsZero() {
		c.Poll.MaxInterval = def.Poll.MaxInterval
	}
	if c.Poll.Timeout.IsZero() {
		c.Poll.Timeout = def.Poll.Timeout
	}
	if c.Poll.Multiplier <= 1 {
		c.Poll.Multiplier = def.Poll.Multiplier
	}

	c.Output.Format = strings.ToLower(strings.TrimSpace(c.Output.Format))
	if c.Output.Format == "" {
		c.Output.Format = def.Output.Format
	}
	c.Output.Color = strings.ToLower(strings.TrimSpace(c.Output.Color))
	if c.Output.Color == "" {
		c.Output.Color = def.Output.Color
	}

	if strings.TrimSpace(c.Defaults.Folder.Name) == "" {
		c.Defaults.Folder.Name = def.Defaults.Folder.Name
	}

	if strings.TrimSpace(c.Defaults.Profile.BrowserType) == "" {
		c.Defaults.Profile.BrowserType = def.Defaults.Profile.BrowserType
	}
	if strings.TrimSpace(c.Defaults.Profile.OSType) == "" {
		c.Defaults.Profile.OSType = def.Defaults.Profile.OSType
	}
	c.Defaults.Profile.StorageType = strings.ToLower(strings.TrimSpace(c.Defaults.Profile.StorageType))
	if c.Defaults.Profile.StorageType == "" {
		c.Defaults.Profile.StorageType = def.Defaults.Profile.StorageType
	}

	c.Defaults.Launcher.AutomationType = strings.ToLower(strings.TrimSpace(c.Defaults.Launcher.AutomationType))
	if c.Defaults.Launcher.AutomationType == "" {
		c.Defaults.Launcher.AutomationType = def.Defaults.Launcher.AutomationType
	}

	c.Defaults.Extension.BrowserType = strings.TrimSpace(c.Defaults.Extension.BrowserType)
	if c.Defaults.Extension.BrowserType == "" {
		c.Defaults.Extension.BrowserType = def.Defaults.Extension.BrowserType
	}
	c.Defaults.Extension.StorageType = strings.ToLower(strings.TrimSpace(c.Defaults.Extension.StorageType))
	if c.Defaults.Extension.StorageType == "" {
		c.Defaults.Extension.StorageType = def.Defaults.Extension.StorageType
	}

	c.Defaults.Proxy.Protocol = strings.ToLower(strings.TrimSpace(c.Defaults.Proxy.Protocol))
	if c.Defaults.Proxy.Protocol == "" {
		c.Defaults.Proxy.Protocol = def.Defaults.Proxy.Protocol
	}
	c.Defaults.Proxy.SessionType = strings.ToLower(strings.TrimSpace(c.Defaults.Proxy.SessionType))
	if c.Defaults.Proxy.SessionType == "" {
		c.Defaults.Proxy.SessionType = def.Defaults.Proxy.SessionType
	}

	return c
}

// Validate verifies that the config contains supported values.
func (c Config) Validate() error {
	if strings.TrimSpace(c.Version) == "" {
		return errors.New("config version must not be empty")
	}
	if err := validateOptionalURL("endpoints.base_url", c.Endpoints.BaseURL); err != nil {
		return err
	}
	if err := validateOptionalURL("endpoints.launcher_url", c.Endpoints.LauncherURL); err != nil {
		return err
	}
	if err := validateOptionalURL("endpoints.cookies_url", c.Endpoints.CookiesURL); err != nil {
		return err
	}
	if err := validateOptionalURL("endpoints.proxy_url", c.Endpoints.ProxyURL); err != nil {
		return err
	}

	if c.Transport.Timeout.Duration() <= 0 {
		return errors.New("transport.timeout must be greater than zero")
	}
	if strings.TrimSpace(c.Transport.UserAgent) == "" {
		return errors.New("transport.user_agent must not be empty")
	}

	if c.Retry.Enabled {
		if c.Retry.MaxAttempts <= 0 {
			return errors.New("retry.max_attempts must be greater than zero")
		}
		if c.Retry.InitialInterval.Duration() <= 0 {
			return errors.New("retry.initial_interval must be greater than zero")
		}
		if c.Retry.MaxInterval.Duration() <= 0 {
			return errors.New("retry.max_interval must be greater than zero")
		}
		if c.Retry.MaxInterval.Duration() < c.Retry.InitialInterval.Duration() {
			return errors.New("retry.max_interval must be greater than or equal to retry.initial_interval")
		}
		if c.Retry.Multiplier <= 1 {
			return errors.New("retry.multiplier must be greater than one")
		}
		if c.Retry.Jitter < 0 {
			return errors.New("retry.jitter must not be negative")
		}
	}

	if c.Poll.InitialInterval.Duration() <= 0 {
		return errors.New("poll.initial_interval must be greater than zero")
	}
	if c.Poll.MaxInterval.Duration() <= 0 {
		return errors.New("poll.max_interval must be greater than zero")
	}
	if c.Poll.MaxInterval.Duration() < c.Poll.InitialInterval.Duration() {
		return errors.New("poll.max_interval must be greater than or equal to poll.initial_interval")
	}
	if c.Poll.Timeout.Duration() <= 0 {
		return errors.New("poll.timeout must be greater than zero")
	}
	if c.Poll.Multiplier <= 1 {
		return errors.New("poll.multiplier must be greater than one")
	}

	switch c.Output.Format {
	case outputFormatTable, outputFormatJSON, outputFormatYAML:
	default:
		return fmt.Errorf("output.format must be one of %q, %q, or %q", outputFormatTable, outputFormatJSON, outputFormatYAML)
	}

	switch c.Output.Color {
	case colorModeAuto, colorModeAlways, colorModeNever:
	default:
		return fmt.Errorf("output.color must be one of %q, %q, or %q", colorModeAuto, colorModeAlways, colorModeNever)
	}

	if err := validateStorageType("defaults.profile.storage_type", c.Defaults.Profile.StorageType, true); err != nil {
		return err
	}
	if err := validateStorageType("defaults.extension.storage_type", c.Defaults.Extension.StorageType, false); err != nil {
		return err
	}
	if err := validateAutomationType(c.Defaults.Launcher.AutomationType); err != nil {
		return err
	}
	if err := validateProxyProtocol(c.Defaults.Proxy.Protocol); err != nil {
		return err
	}
	if err := validateProxySessionType(c.Defaults.Proxy.SessionType); err != nil {
		return err
	}

	return nil
}

// RetryOptions converts config retry settings into SDK retry options.
func (c Config) RetryOptions() mlx.RetryOptions {
	return mlx.RetryOptions{
		MaxAttempts:     c.Retry.MaxAttempts,
		InitialInterval: c.Retry.InitialInterval.Duration(),
		MaxInterval:     c.Retry.MaxInterval.Duration(),
		Multiplier:      c.Retry.Multiplier,
		Jitter:          c.Retry.Jitter,
	}
}

// PollOptions converts config polling settings into SDK poll options.
func (c Config) PollOptions() mlx.PollOptions {
	return mlx.PollOptions{
		InitialInterval: c.Poll.InitialInterval.Duration(),
		MaxInterval:     c.Poll.MaxInterval.Duration(),
		Timeout:         c.Poll.Timeout.Duration(),
		Multiplier:      c.Poll.Multiplier,
	}
}

// ClientOptions converts config into shared SDK client options.
//
// Authentication is intentionally excluded. Callers must provide MLX_TOKEN from
// the environment separately.
func (c Config) ClientOptions() []mlx.Option {
	opts := make([]mlx.Option, 0, 6)

	if c.Transport.Timeout.Duration() > 0 {
		opts = append(opts, mlx.WithTimeout(c.Transport.Timeout.Duration()))
	}
	if strings.TrimSpace(c.Transport.UserAgent) != "" {
		opts = append(opts, mlx.WithUserAgent(c.Transport.UserAgent))
	}
	if c.Retry.Enabled {
		opts = append(opts, mlx.WithRetry(c.RetryOptions()))
	}
	if strings.TrimSpace(c.Endpoints.BaseURL) != "" {
		opts = append(opts, mlx.WithBaseURL(c.Endpoints.BaseURL))
	}
	if strings.TrimSpace(c.Endpoints.LauncherURL) != "" {
		opts = append(opts, mlx.WithLauncherURL(c.Endpoints.LauncherURL))
	}
	if strings.TrimSpace(c.Endpoints.CookiesURL) != "" {
		opts = append(opts, mlx.WithCookiesURL(c.Endpoints.CookiesURL))
	}
	if strings.TrimSpace(c.Endpoints.ProxyURL) != "" {
		opts = append(opts, mlx.WithProxyURL(c.Endpoints.ProxyURL))
	}

	return opts
}

func resolveConfigPathWithExplicitness(explicit string) (string, bool, error) {
	if trimmed := strings.TrimSpace(explicit); trimmed != "" {
		return filepath.Clean(trimmed), true, nil
	}
	if fromEnv := strings.TrimSpace(os.Getenv(EnvConfigFile)); fromEnv != "" {
		return filepath.Clean(fromEnv), true, nil
	}
	path, err := DefaultConfigPath()
	if err != nil {
		return "", false, err
	}
	return path, false, nil
}

func ensureJSONEOF(dec *json.Decoder) error {
	var extra any
	err := dec.Decode(&extra)
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return fmt.Errorf("decode trailing cli config data: %w", err)
	}
	return errors.New("cli config contains multiple JSON values")
}

func isRetryUnset(cfg RetryConfig) bool {
	return cfg.MaxAttempts == 0 &&
		cfg.InitialInterval.IsZero() &&
		cfg.MaxInterval.IsZero() &&
		cfg.Multiplier == 0 &&
		cfg.Jitter == 0
}

func validateOptionalURL(name, raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("%s is invalid: %w", name, err)
	}
	if u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("%s must include scheme and host", name)
	}
	return nil
}

func validateStorageType(name, value string, allowAll bool) error {
	switch value {
	case storageTypeLocal, storageTypeCloud:
		return nil
	case storageTypeAll:
		if allowAll {
			return nil
		}
	}
	if allowAll {
		return fmt.Errorf("%s must be one of %q, %q, or %q", name, storageTypeAll, storageTypeLocal, storageTypeCloud)
	}
	return fmt.Errorf("%s must be one of %q or %q", name, storageTypeLocal, storageTypeCloud)
}

func validateAutomationType(value string) error {
	switch value {
	case string(mlx.AutomationSelenium),
		string(mlx.AutomationPlaywright),
		string(mlx.AutomationPuppeteer),
		string(mlx.AutomationRod):
		return nil
	default:
		return fmt.Errorf("defaults.launcher.automation_type must be one of %q, %q, %q, or %q",
			mlx.AutomationSelenium, mlx.AutomationPlaywright, mlx.AutomationPuppeteer, mlx.AutomationRod)
	}
}

func validateProxyProtocol(value string) error {
	switch value {
	case string(mlx.ProxyProtocolSOCKS5), string(mlx.ProxyProtocolHTTP):
		return nil
	default:
		return fmt.Errorf("defaults.proxy.protocol must be one of %q or %q", mlx.ProxyProtocolSOCKS5, mlx.ProxyProtocolHTTP)
	}
}

func validateProxySessionType(value string) error {
	switch value {
	case string(mlx.ProxySessionSticky), string(mlx.ProxySessionRotating):
		return nil
	default:
		return fmt.Errorf("defaults.proxy.session_type must be one of %q or %q", mlx.ProxySessionSticky, mlx.ProxySessionRotating)
	}
}
