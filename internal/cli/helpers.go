package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	mlx "github.com/minskyagenda0708-cmd/mlx-go-sdk"
)

func readJSONFile(path string, out any) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()

	dec := json.NewDecoder(file)
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}

	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("%s contains multiple JSON values", path)
		}
		return fmt.Errorf("decode trailing content in %s: %w", path, err)
	}

	return nil
}

func readCookiesFile(path string) ([]mlx.BrowserCookie, error) {
	var cookies []mlx.BrowserCookie
	if err := readJSONFile(path, &cookies); err != nil {
		return nil, err
	}

	return cookies, nil
}

type profileTemplateDocument struct {
	Name       string                   `json:"name"`
	MainParams mlx.CreateProfileRequest `json:"mainParams"`
}

func loadProfileTemplate(metaInfo, path string) (*profileTemplateDocument, error) {
	var pathErr error

	if strings.TrimSpace(path) != "" {
		body, err := os.ReadFile(path)
		if err != nil {
			pathErr = fmt.Errorf("read template %s: %w", path, err)
		} else {
			var doc profileTemplateDocument
			if err := json.Unmarshal(body, &doc); err != nil {
				pathErr = fmt.Errorf("decode template %s: %w", path, err)
			} else if profileTemplateHasUsableMainParams(&doc) ||
				strings.TrimSpace(doc.Name) != "" {
				return &doc, nil
			}
		}
	}

	if strings.TrimSpace(metaInfo) != "" {
		var doc profileTemplateDocument
		if err := json.Unmarshal([]byte(metaInfo), &doc); err != nil {
			return nil, fmt.Errorf("decode template meta_info: %w", err)
		}
		if profileTemplateHasUsableMainParams(&doc) ||
			strings.TrimSpace(doc.Name) != "" {
			return &doc, nil
		}

		return nil, errors.New(
			"template meta_info does not contain usable mainParams",
		)
	}

	if pathErr != nil {
		return nil, pathErr
	}

	return nil, errors.New("template content is empty")
}

func profileTemplateHasUsableMainParams(doc *profileTemplateDocument) bool {
	if doc == nil {
		return false
	}

	main := doc.MainParams
	return strings.TrimSpace(main.Name) != "" ||
		strings.TrimSpace(main.BrowserType) != "" ||
		strings.TrimSpace(main.FolderID) != "" ||
		strings.TrimSpace(main.OSType) != "" ||
		main.CoreVersion != 0 ||
		main.CoreMinorVersion != 0 ||
		main.AutoUpdateCore != nil ||
		main.Times != 0 ||
		strings.TrimSpace(main.Notes) != "" ||
		main.Parameters != nil ||
		len(main.Tags) != 0
}

func buildCreateProfileRequestFromTemplate(
	doc *profileTemplateDocument,
	name string,
	folderID string,
	localOverride *bool,
) (*mlx.CreateProfileRequest, error) {
	if doc == nil {
		return nil, errors.New("template document is required")
	}

	req := doc.MainParams
	if strings.TrimSpace(name) != "" {
		req.Name = name
	} else if strings.TrimSpace(req.Name) == "" {
		req.Name = strings.TrimSpace(doc.Name)
	}
	if strings.TrimSpace(req.Name) == "" {
		return nil, errors.New(
			"template does not contain a usable profile name; pass --name",
		)
	}

	if strings.TrimSpace(folderID) != "" {
		req.FolderID = folderID
	}
	if strings.TrimSpace(req.FolderID) == "" {
		return nil, errors.New(
			"template does not contain a folder id and no --folder-id/default folder id was resolved",
		)
	}

	if localOverride != nil {
		if req.Parameters == nil {
			req.Parameters = &mlx.ProfileParameters{}
		}
		if req.Parameters.Storage == nil {
			req.Parameters.Storage = &mlx.Storage{}
		}
		req.Parameters.Storage.IsLocal = *localOverride
	}

	return &req, nil
}

func resolveFolderID(rt *Runtime, explicit string) (string, error) {
	if trimmed := strings.TrimSpace(explicit); trimmed != "" {
		return trimmed, nil
	}
	if trimmed := strings.TrimSpace(rt.Config.Defaults.Folder.ID); trimmed != "" {
		return trimmed, nil
	}

	defaultName := strings.TrimSpace(rt.Config.Defaults.Folder.Name)
	if defaultName == "" {
		return "", nil
	}

	resp, _, err := rt.Client.Folders.List(context.Background())
	if err != nil {
		return "", err
	}
	for _, folder := range resp.Data.Folders {
		if strings.EqualFold(strings.TrimSpace(folder.Name), defaultName) {
			return folder.FolderID, nil
		}
	}

	return "", nil
}

func resolveProfile(
	rt *Runtime,
	profileID string,
	profileName string,
	folderID string,
) (*resolvedProfile, error) {
	if strings.TrimSpace(profileID) != "" {
		meta, _, err := rt.Client.Profiles.GetMeta(context.Background(), profileID)
		if err != nil {
			return nil, err
		}
		return &resolvedProfile{
			ID:       meta.ID,
			Name:     meta.Name,
			FolderID: meta.FolderID,
			Meta:     meta,
		}, nil
	}

	verified, err := rt.Client.Workflows.FindProfileByNameVerified(
		context.Background(),
		profileName,
		mlx.FindProfileByNameVerifiedOptions{
			FindOptions: buildFindOptions(rt.Config, folderID),
		},
	)
	if err != nil {
		return nil, err
	}

	return &resolvedProfile{
		ID:       verified.Profile.ID,
		Name:     verified.Profile.Name,
		FolderID: verified.Profile.FolderID,
		Profile:  verified.Profile,
		Meta:     verified.Meta,
	}, nil
}

func buildFindOptions(cfg Config, folderID string) *mlx.FindProfileOptions {
	find := &mlx.FindProfileOptions{
		StorageType: cfg.Defaults.Profile.StorageType,
		FolderID:    firstNonEmpty(folderID, cfg.Defaults.Folder.ID),
	}
	if strings.TrimSpace(find.StorageType) == "" {
		find.StorageType = storageTypeAll
	}
	if find.StorageType == storageTypeAll &&
		strings.TrimSpace(find.FolderID) == "" {
		return nil
	}

	return find
}

func buildGenerateProxyRequest(
	cfg Config,
	country string,
	region string,
	city string,
	protocol string,
	sessionType string,
	ipTTL int,
	count int,
	strict bool,
) *mlx.GenerateProxyRequest {
	effectiveCountry := strings.TrimSpace(country)
	if effectiveCountry == "" {
		effectiveCountry = strings.TrimSpace(cfg.Defaults.Proxy.Country)
	}

	effectiveRegion := strings.TrimSpace(region)
	if effectiveRegion == "" {
		effectiveRegion = strings.TrimSpace(cfg.Defaults.Proxy.Region)
	}

	effectiveCity := strings.TrimSpace(city)
	if effectiveCity == "" {
		effectiveCity = strings.TrimSpace(cfg.Defaults.Proxy.City)
	}

	effectiveProtocol := strings.TrimSpace(protocol)
	if effectiveProtocol == "" {
		effectiveProtocol = cfg.Defaults.Proxy.Protocol
	}

	effectiveSessionType := strings.TrimSpace(sessionType)
	if effectiveSessionType == "" {
		effectiveSessionType = cfg.Defaults.Proxy.SessionType
	}

	if count <= 0 {
		count = 1
	}

	return &mlx.GenerateProxyRequest{
		Country:     effectiveCountry,
		Region:      effectiveRegion,
		City:        effectiveCity,
		Protocol:    mlx.ProxyProtocol(effectiveProtocol),
		SessionType: mlx.ProxySessionType(effectiveSessionType),
		IPTTL:       ipTTL,
		Count:       count,
		StrictMode:  strict,
	}
}

func waitForStoppedStatus(
	ctx context.Context,
	rt *Runtime,
	profileID string,
) (*mlx.ProfileRuntimeStatusResponse, error) {
	opts := rt.Config.PollOptions()
	deadline := time.Now().Add(opts.Timeout)
	interval := opts.InitialInterval
	if interval <= 0 {
		interval = 2 * time.Second
	}
	if opts.MaxInterval <= 0 {
		opts.MaxInterval = interval
	}
	if opts.Multiplier <= 1 {
		opts.Multiplier = 1.5
	}

	var last *mlx.ProfileRuntimeStatusResponse
	for {
		resp, _, err := rt.Client.Launcher.Status(ctx, profileID)
		if err == nil {
			last = resp
			status := strings.ToLower(strings.TrimSpace(resp.Data.Status))
			if status == "stopped" || strings.Contains(status, "stopped") {
				return resp, nil
			}
		}
		if !time.Now().Before(deadline) {
			if last != nil {
				return nil, fmt.Errorf(
					"profile %s did not reach stopped status before timeout, last status=%s",
					profileID,
					last.Data.Status,
				)
			}
			return nil, fmt.Errorf(
				"profile %s did not reach stopped status before timeout",
				profileID,
			)
		}
		if err := sleepContext(ctx, interval); err != nil {
			return nil, err
		}
		interval = nextInterval(interval, opts.Multiplier, opts.MaxInterval)
	}
}

func waitForObjectUsageAttached(
	ctx context.Context,
	rt *Runtime,
	resourceID string,
	profileID string,
) (*mlx.ObjectProfileUsagesResponse, error) {
	opts := rt.Config.PollOptions()
	deadline := time.Now().Add(opts.Timeout)
	interval := opts.InitialInterval
	if interval <= 0 {
		interval = 2 * time.Second
	}
	if opts.MaxInterval <= 0 {
		opts.MaxInterval = interval
	}
	if opts.Multiplier <= 1 {
		opts.Multiplier = 1.5
	}

	var last *mlx.ObjectProfileUsagesResponse
	for {
		resp, _, err := rt.Client.Resources.ObjectProfileUsages(ctx, resourceID)
		if err == nil {
			last = resp
			if objectUsageContainsProfile(resp, profileID) {
				return resp, nil
			}
		} else if !time.Now().Before(deadline) {
			return nil, err
		}
		if !time.Now().Before(deadline) {
			if last != nil {
				return nil, fmt.Errorf(
					"extension %s was not attached to profile %s before timeout, object_usages=%d",
					resourceID,
					profileID,
					len(last.Data),
				)
			}
			return nil, fmt.Errorf(
				"extension %s was not attached to profile %s before timeout",
				resourceID,
				profileID,
			)
		}
		if err := sleepContext(ctx, interval); err != nil {
			return nil, err
		}
		interval = nextInterval(interval, opts.Multiplier, opts.MaxInterval)
	}
}

func objectUsageContainsProfile(
	resp *mlx.ObjectProfileUsagesResponse,
	profileID string,
) bool {
	if resp == nil {
		return false
	}
	for _, usage := range resp.Data {
		if usage.ID == profileID {
			return true
		}
	}

	return false
}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func nextInterval(
	current time.Duration,
	multiplier float64,
	max time.Duration,
) time.Duration {
	next := time.Duration(float64(current) * multiplier)
	if next < current {
		next = current
	}
	if max > 0 && next > max {
		return max
	}

	return next
}

func validateSelector(id, name, idFlag, nameFlag string) error {
	if strings.TrimSpace(id) == "" && strings.TrimSpace(name) == "" {
		return fmt.Errorf("one of %s or %s is required", idFlag, nameFlag)
	}
	if strings.TrimSpace(id) != "" && strings.TrimSpace(name) != "" {
		return fmt.Errorf("%s and %s are mutually exclusive", idFlag, nameFlag)
	}

	return nil
}

func requireNoExtraArgs(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(args, " "))
	}

	return nil
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}

	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}

	return ""
}

func isAlreadyStoppedError(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(strings.ToLower(err.Error()), "already stopped")
}

func errorString(err error) string {
	if err == nil {
		return ""
	}

	return err.Error()
}
