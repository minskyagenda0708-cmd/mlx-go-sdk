package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	mlx "github.com/minskyagenda0708-cmd/mlx-go-sdk"
)

func TestDefaultConfigProvidesNormalizedDefaults(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Version != ConfigVersion {
		t.Fatalf("unexpected config version: %q", cfg.Version)
	}
	if got := cfg.Transport.Timeout.Duration(); got != 30*time.Second {
		t.Fatalf("unexpected transport timeout: %s", got)
	}
	if cfg.Transport.UserAgent != "mlx-go-sdk-cli" {
		t.Fatalf("unexpected user agent: %q", cfg.Transport.UserAgent)
	}
	if !cfg.Retry.Enabled {
		t.Fatal("expected retry to be enabled by default")
	}
	if cfg.Retry.MaxAttempts != 4 {
		t.Fatalf("unexpected retry max attempts: %d", cfg.Retry.MaxAttempts)
	}
	if got := cfg.Retry.InitialInterval.Duration(); got != 500*time.Millisecond {
		t.Fatalf("unexpected retry initial interval: %s", got)
	}
	if got := cfg.Retry.MaxInterval.Duration(); got != 3*time.Second {
		t.Fatalf("unexpected retry max interval: %s", got)
	}
	if cfg.Poll.Multiplier != 1.5 {
		t.Fatalf("unexpected poll multiplier: %v", cfg.Poll.Multiplier)
	}
	if cfg.Output.Format != outputFormatTable {
		t.Fatalf("unexpected default output format: %q", cfg.Output.Format)
	}
	if cfg.Output.Color != colorModeAuto {
		t.Fatalf("unexpected default color mode: %q", cfg.Output.Color)
	}
	if cfg.Defaults.Folder.Name != "Default folder" {
		t.Fatalf("unexpected default folder name: %q", cfg.Defaults.Folder.Name)
	}
	if cfg.Defaults.Profile.BrowserType != "mimic" {
		t.Fatalf("unexpected default browser type: %q", cfg.Defaults.Profile.BrowserType)
	}
	if cfg.Defaults.Profile.OSType != "windows" {
		t.Fatalf("unexpected default os type: %q", cfg.Defaults.Profile.OSType)
	}
	if cfg.Defaults.Profile.StorageType != storageTypeAll {
		t.Fatalf("unexpected default profile storage type: %q", cfg.Defaults.Profile.StorageType)
	}
	if cfg.Defaults.Launcher.AutomationType != string(mlx.AutomationPlaywright) {
		t.Fatalf("unexpected default automation type: %q", cfg.Defaults.Launcher.AutomationType)
	}
	if cfg.Defaults.Extension.StorageType != storageTypeCloud {
		t.Fatalf("unexpected default extension storage type: %q", cfg.Defaults.Extension.StorageType)
	}
	if cfg.Defaults.Proxy.Protocol != string(mlx.ProxyProtocolSOCKS5) {
		t.Fatalf("unexpected default proxy protocol: %q", cfg.Defaults.Proxy.Protocol)
	}
	if cfg.Defaults.Proxy.SessionType != string(mlx.ProxySessionSticky) {
		t.Fatalf("unexpected default proxy session type: %q", cfg.Defaults.Proxy.SessionType)
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("default config should validate: %v", err)
	}
}

func TestDecodeConfigMergesOverridesIntoBaseConfig(t *testing.T) {
	base := DefaultConfig()

	raw := `{
		"transport": {
			"timeout": "45s",
			"user_agent": "custom-agent/2.0"
		},
		"output": {
			"format": "json"
		},
		"defaults": {
			"profile": {
				"storage_type": "local"
			},
			"launcher": {
				"automation_type": "rod"
			},
			"proxy": {
				"protocol": "http",
				"session_type": "rotating"
			}
		}
	}`

	cfg, err := DecodeConfig(strings.NewReader(raw), base)
	if err != nil {
		t.Fatalf("DecodeConfig returned error: %v", err)
	}

	if got := cfg.Transport.Timeout.Duration(); got != 45*time.Second {
		t.Fatalf("unexpected transport timeout: %s", got)
	}
	if cfg.Transport.UserAgent != "custom-agent/2.0" {
		t.Fatalf("unexpected user agent: %q", cfg.Transport.UserAgent)
	}
	if cfg.Output.Format != outputFormatJSON {
		t.Fatalf("unexpected output format: %q", cfg.Output.Format)
	}
	if !cfg.Output.Pretty {
		t.Fatal("expected omitted output.pretty to preserve the base default")
	}
	if cfg.Output.Color != colorModeAuto {
		t.Fatalf("expected omitted output.color to preserve base default, got %q", cfg.Output.Color)
	}
	if cfg.Defaults.Profile.StorageType != storageTypeLocal {
		t.Fatalf("unexpected profile storage type: %q", cfg.Defaults.Profile.StorageType)
	}
	if cfg.Defaults.Profile.BrowserType != base.Defaults.Profile.BrowserType {
		t.Fatalf("expected profile browser type to stay at base default, got %q", cfg.Defaults.Profile.BrowserType)
	}
	if cfg.Defaults.Launcher.AutomationType != string(mlx.AutomationRod) {
		t.Fatalf("unexpected launcher automation type: %q", cfg.Defaults.Launcher.AutomationType)
	}
	if cfg.Defaults.Proxy.Protocol != string(mlx.ProxyProtocolHTTP) {
		t.Fatalf("unexpected proxy protocol: %q", cfg.Defaults.Proxy.Protocol)
	}
	if cfg.Defaults.Proxy.SessionType != string(mlx.ProxySessionRotating) {
		t.Fatalf("unexpected proxy session type: %q", cfg.Defaults.Proxy.SessionType)
	}
	if cfg.Retry.MaxAttempts != base.Retry.MaxAttempts {
		t.Fatalf("expected retry.max_attempts to preserve base value, got %d", cfg.Retry.MaxAttempts)
	}
}

func TestDecodeConfigPreservesExplicitRetryDisable(t *testing.T) {
	cfg, err := DecodeConfig(strings.NewReader(`{"retry":{"enabled":false}}`), DefaultConfig())
	if err != nil {
		t.Fatalf("DecodeConfig returned error: %v", err)
	}
	if cfg.Retry.Enabled {
		t.Fatal("expected explicit retry.enabled=false to remain disabled")
	}
	if len(cfg.ClientOptions()) != 2 {
		t.Fatalf("expected client options to omit WithRetry when disabled, got %d options", len(cfg.ClientOptions()))
	}
}

func TestDecodeConfigRejectsUnknownFields(t *testing.T) {
	_, err := DecodeConfig(strings.NewReader(`{"unknown_key": true}`), DefaultConfig())
	if err == nil {
		t.Fatal("expected DecodeConfig to reject unknown fields")
	}
	if !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown field error, got %v", err)
	}
}

func TestDecodeConfigRejectsMultipleJSONValues(t *testing.T) {
	_, err := DecodeConfig(strings.NewReader(`{"output":{"format":"json"}} {"output":{"format":"yaml"}}`), DefaultConfig())
	if err == nil {
		t.Fatal("expected DecodeConfig to reject multiple JSON values")
	}
	if !strings.Contains(err.Error(), "multiple JSON values") {
		t.Fatalf("expected multiple JSON values error, got %v", err)
	}
}

func TestApplyEnvOverridesUsesSupportedEnvironmentVariables(t *testing.T) {
	t.Setenv(mlx.EnvBaseURL, "https://api.example.test")
	t.Setenv(mlx.EnvLauncherURL, "https://launcher.example.test:45001")
	t.Setenv(mlx.EnvCookiesURL, "https://cookies.example.test")
	t.Setenv(mlx.EnvProxyURL, "https://proxy.example.test")
	t.Setenv(EnvTimeout, "75s")
	t.Setenv(EnvOutputFormat, "yaml")
	t.Setenv(EnvUserAgent, "env-agent/1.2.3")

	cfg, err := ApplyEnvOverrides(DefaultConfig(), os.Getenv)
	if err != nil {
		t.Fatalf("ApplyEnvOverrides returned error: %v", err)
	}

	if cfg.Endpoints.BaseURL != "https://api.example.test" {
		t.Fatalf("unexpected base url: %q", cfg.Endpoints.BaseURL)
	}
	if cfg.Endpoints.LauncherURL != "https://launcher.example.test:45001" {
		t.Fatalf("unexpected launcher url: %q", cfg.Endpoints.LauncherURL)
	}
	if cfg.Endpoints.CookiesURL != "https://cookies.example.test" {
		t.Fatalf("unexpected cookies url: %q", cfg.Endpoints.CookiesURL)
	}
	if cfg.Endpoints.ProxyURL != "https://proxy.example.test" {
		t.Fatalf("unexpected proxy url: %q", cfg.Endpoints.ProxyURL)
	}
	if got := cfg.Transport.Timeout.Duration(); got != 75*time.Second {
		t.Fatalf("unexpected timeout after env override: %s", got)
	}
	if cfg.Output.Format != outputFormatYAML {
		t.Fatalf("unexpected output format after env override: %q", cfg.Output.Format)
	}
	if cfg.Transport.UserAgent != "env-agent/1.2.3" {
		t.Fatalf("unexpected user agent after env override: %q", cfg.Transport.UserAgent)
	}
}

func TestResolveConfigPathPrefersExplicitThenEnvironment(t *testing.T) {
	explicit := filepath.Join("some", "nested", "explicit.json")
	got, err := ResolveConfigPath(explicit)
	if err != nil {
		t.Fatalf("ResolveConfigPath returned error for explicit path: %v", err)
	}
	if got != filepath.Clean(explicit) {
		t.Fatalf("unexpected explicit config path: %q", got)
	}

	envPath := filepath.Join("config", "from-env.json")
	t.Setenv(EnvConfigFile, envPath)

	got, err = ResolveConfigPath("")
	if err != nil {
		t.Fatalf("ResolveConfigPath returned error for env path: %v", err)
	}
	if got != filepath.Clean(envPath) {
		t.Fatalf("unexpected env config path: %q", got)
	}
}

func TestLoadConfigUsesFileThenEnvironmentOverrides(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	raw := `{
		"transport": {
			"timeout": "10s",
			"user_agent": "file-agent/1.0"
		},
		"output": {
			"format": "json"
		},
		"defaults": {
			"profile": {
				"storage_type": "local"
			}
		}
	}`

	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	t.Setenv(EnvTimeout, "20s")
	t.Setenv(EnvOutputFormat, "yaml")
	t.Setenv(EnvUserAgent, "env-agent/9.9.9")

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	if got := cfg.Transport.Timeout.Duration(); got != 20*time.Second {
		t.Fatalf("expected env timeout to win over file timeout, got %s", got)
	}
	if cfg.Output.Format != outputFormatYAML {
		t.Fatalf("expected env output format to win over file format, got %q", cfg.Output.Format)
	}
	if cfg.Transport.UserAgent != "env-agent/9.9.9" {
		t.Fatalf("expected env user agent to win over file user agent, got %q", cfg.Transport.UserAgent)
	}
	if cfg.Defaults.Profile.StorageType != storageTypeLocal {
		t.Fatalf("expected file value for defaults.profile.storage_type, got %q", cfg.Defaults.Profile.StorageType)
	}
}

func TestLoadConfigReturnsErrorForMissingExplicitPath(t *testing.T) {
	_, err := LoadConfig(filepath.Join(t.TempDir(), "missing.json"))
	if err == nil {
		t.Fatal("expected LoadConfig to fail for a missing explicit path")
	}
	if !strings.Contains(err.Error(), "open cli config") {
		t.Fatalf("expected open cli config error, got %v", err)
	}
}

func TestTokenFromEnvRequiresToken(t *testing.T) {
	t.Setenv(mlx.EnvToken, "")
	_, err := TokenFromEnv()
	if err == nil {
		t.Fatal("expected TokenFromEnv to require a token")
	}
	if !strings.Contains(err.Error(), mlx.EnvToken) {
		t.Fatalf("expected missing token error to mention env var, got %v", err)
	}

	t.Setenv(mlx.EnvToken, "test-token")
	token, err := TokenFromEnv()
	if err != nil {
		t.Fatalf("TokenFromEnv returned error: %v", err)
	}
	if token != "test-token" {
		t.Fatalf("unexpected token: %q", token)
	}
}

func TestLoadRuntimeBuildsClientFromResolvedConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "runtime-config.json")

	raw := `{
		"output": {
			"format": "json"
		},
		"transport": {
			"user_agent": "runtime-agent/1.0"
		}
	}`

	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	t.Setenv(mlx.EnvToken, "runtime-token")

	rt, err := LoadRuntime(path)
	if err != nil {
		t.Fatalf("LoadRuntime returned error: %v", err)
	}
	if rt == nil {
		t.Fatal("expected non-nil runtime")
	}
	if rt.ConfigPath != filepath.Clean(path) {
		t.Fatalf("unexpected runtime config path: %q", rt.ConfigPath)
	}
	if rt.Token != "runtime-token" {
		t.Fatalf("unexpected runtime token: %q", rt.Token)
	}
	if rt.Client == nil {
		t.Fatal("expected runtime client to be created")
	}
	if rt.Config.Output.Format != outputFormatJSON {
		t.Fatalf("unexpected runtime output format: %q", rt.Config.Output.Format)
	}
	if rt.Config.Transport.UserAgent != "runtime-agent/1.0" {
		t.Fatalf("unexpected runtime user agent: %q", rt.Config.Transport.UserAgent)
	}
}

func TestDefaultConfigProxyContinuity(t *testing.T) {
	cfg := DefaultConfig()
	pc := cfg.Defaults.Proxy.Continuity
	if !pc.Enabled {
		t.Fatal("expected continuity enabled by default")
	}
	if pc.LatencyThresholdMs != 2000 || pc.LatencyHardCapMs != 3000 {
		t.Fatalf("unexpected thresholds: %+v", pc)
	}
	if pc.CandidatesPerRound != 5 {
		t.Fatalf("unexpected candidates_per_round: %d", pc.CandidatesPerRound)
	}
	if len(pc.CheckTargets) == 0 {
		t.Fatal("expected default check targets")
	}
	if pc.CheckTimeout.IsZero() {
		t.Fatal("expected non-zero check timeout")
	}
}
