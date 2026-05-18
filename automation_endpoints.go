package mlx

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const automationEndpointTimeout = 3 * time.Second

// AutomationEndpointError describes an unusable launcher automation endpoint.
type AutomationEndpointError struct {
	RequestedAutomation AutomationType
	LauncherAutomation  AutomationType
	Port                string
	Message             string
	Err                 error
}

func (e *AutomationEndpointError) Error() string {
	if e == nil {
		return "launcher automation endpoint error"
	}
	if e.Message == "" {
		return fmt.Sprintf(
			"launcher automation endpoint is unusable (requested=%q launcher=%q port=%q)",
			e.RequestedAutomation,
			e.LauncherAutomation,
			e.Port,
		)
	}
	return fmt.Sprintf(
		"launcher automation endpoint is unusable: %s (requested=%q launcher=%q port=%q)",
		e.Message,
		e.RequestedAutomation,
		e.LauncherAutomation,
		e.Port,
	)
}

func (e *AutomationEndpointError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

type cdpVersionResponse struct {
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

func normalizeLauncherAutomation(requested AutomationType) AutomationType {
	if requested == AutomationRod {
		return AutomationPlaywright
	}
	return requested
}

func enrichStartedProfileData(
	data *StartedProfileData,
	requested AutomationType,
	launcher AutomationType,
) {
	if data == nil {
		return
	}
	data.RequestedAutomation = requested
	data.LauncherAutomation = launcher
	if strings.TrimSpace(data.CDPPort) == "" {
		data.CDPPort = strings.TrimSpace(data.Port)
	}
}

func (d *StartedProfileData) ResolveCDPWebSocketURL(ctx context.Context) (string, error) {
	port := d.cdpPort()
	if port == "" {
		return "", d.newAutomationEndpointError("", "empty cdp port", nil)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("http://127.0.0.1:%s/json/version", port),
		nil,
	)
	if err != nil {
		return "", d.newAutomationEndpointError(port, "build cdp version request", err)
	}

	httpClient := &http.Client{Timeout: automationEndpointTimeout}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", d.newAutomationEndpointError(port, "request cdp version endpoint", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", d.newAutomationEndpointError(
			port,
			fmt.Sprintf("unexpected http status %d", resp.StatusCode),
			nil,
		)
	}

	var payload cdpVersionResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", d.newAutomationEndpointError(port, "decode cdp version response", err)
	}

	wsURL := strings.TrimSpace(payload.WebSocketDebuggerURL)
	if wsURL == "" {
		return "", d.newAutomationEndpointError(
			port,
			"empty webSocketDebuggerUrl",
			nil,
		)
	}

	parsed, err := url.Parse(wsURL)
	if err != nil {
		return "", d.newAutomationEndpointError(port, "parse webSocketDebuggerUrl", err)
	}
	parsed.Host = fmt.Sprintf("127.0.0.1:%s", port)

	return parsed.String(), nil
}

func (d *StartedProfileData) ResolveRodControlURL(ctx context.Context) (string, error) {
	return d.ResolveCDPWebSocketURL(ctx)
}

func (d *StartedProfileData) cdpPort() string {
	if d == nil {
		return ""
	}
	if port := strings.TrimSpace(d.CDPPort); port != "" {
		return port
	}
	return strings.TrimSpace(d.Port)
}

func (d *StartedProfileData) newAutomationEndpointError(
	port string,
	message string,
	err error,
) *AutomationEndpointError {
	if d == nil {
		return &AutomationEndpointError{
			Port:    port,
			Message: message,
			Err:     err,
		}
	}
	return &AutomationEndpointError{
		RequestedAutomation: d.RequestedAutomation,
		LauncherAutomation:  d.LauncherAutomation,
		Port:                port,
		Message:             message,
		Err:                 err,
	}
}
