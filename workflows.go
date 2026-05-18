package mlx

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// WorkflowService provides higher-level helpers that combine multiple SDK calls.
type WorkflowService interface {
	CreateProfilesAndVerify(context.Context, *CreateProfileRequest, CreateProfilesAndVerifyOptions) (*CreatedProfilesWorkflowResult, error)
	FindProfileByNameVerified(context.Context, string, FindProfileByNameVerifiedOptions) (*VerifiedProfileWorkflowResult, error)
	StartProfileAutomationByName(context.Context, string, StartProfileAutomationByNameOptions) (*StartedProfileAutomationWorkflowResult, error)
	StartProfileByName(context.Context, string, StartProfileByNameOptions) (*StartedProfileWorkflowResult, error)
	StartProfilesByName(context.Context, []string, StartProfileByNameOptions) (*BatchResult[StartedProfileWorkflowResult], error)
	StopProfileByName(context.Context, string, StopProfileByNameOptions) (*StoppedProfileWorkflowResult, error)
	StopProfilesByName(context.Context, []string, StopProfileByNameOptions) (*BatchResult[StoppedProfileWorkflowResult], error)
	ImportProfileAndVerify(context.Context, *ImportProfileRequest, ImportProfileWorkflowOptions) (*ImportedProfileWorkflowResult, error)
	EnableExtensionForProfileByName(context.Context, string, string, EnableExtensionForProfileByNameOptions) (*EnabledExtensionWorkflowResult, error)
	EnableExtensionForProfilesByName(context.Context, []string, string, EnableExtensionForProfileByNameOptions) (*BatchResult[EnabledExtensionWorkflowResult], error)
	ExportProfileByNameToFolder(context.Context, string, ExportProfileByNameToFolderOptions) (*ExportedProfileWorkflowResult, error)
	ExportProfilesByNameToFolder(context.Context, []string, ExportProfileByNameToFolderOptions) (*BatchResult[ExportedProfileWorkflowResult], error)
	GenerateProfileProxyByName(context.Context, string, GenerateProfileProxyByNameOptions) (*GeneratedProfileProxyWorkflowResult, error)
}

// WorkflowServiceOp is the concrete high-level workflow service.
type WorkflowServiceOp struct {
	client *Client
}

// CreateProfilesAndVerifyOptions controls post-create verification polling.
type CreateProfilesAndVerifyOptions struct {
	PollOptions PollOptions
}

// CreatedProfilesWorkflowResult contains created IDs and verified profile metas.
type CreatedProfilesWorkflowResult struct {
	CreateResponse *CreateProfileResponse
	Profiles       []ProfileMeta
}

// FindProfileByNameVerifiedOptions controls exact-name lookup verification.
type FindProfileByNameVerifiedOptions struct {
	FindOptions *FindProfileOptions
}

// VerifiedProfileWorkflowResult contains lightweight and meta profile views.
type VerifiedProfileWorkflowResult struct {
	Profile *Profile
	Meta    *ProfileMeta
}

// StartProfileByNameOptions controls the lookup and launcher behavior for a start workflow.
type StartProfileByNameOptions struct {
	FindOptions    *FindProfileOptions
	StartOptions   StartProfileOptions
	WaitForRunning bool
	PollOptions    PollOptions
}

// StartProfileAutomationByNameOptions controls lookup, automation normalization, and endpoint resolution.
type StartProfileAutomationByNameOptions struct {
	FindOptions    *FindProfileOptions
	StartOptions   StartProfileOptions
	WaitForRunning bool
	PollOptions    PollOptions
}

// StartedProfileWorkflowResult contains the resolved profile and launcher results.
type StartedProfileWorkflowResult struct {
	Profile       *Profile
	StartResponse *StartProfileResponse
	RuntimeStatus *ProfileRuntimeStatusResponse
}

// StartedProfileAutomationWorkflowResult contains the resolved profile, launcher results, and CDP endpoints.
type StartedProfileAutomationWorkflowResult struct {
	Profile             *Profile
	StartResponse       *StartProfileResponse
	RuntimeStatus       *ProfileRuntimeStatusResponse
	RequestedAutomation AutomationType
	LauncherAutomation  AutomationType
	CDPPort             string
	CDPWebSocketURL     string
	RodControlURL       string
}

// StopProfileByNameOptions controls the lookup used before stopping a profile.
type StopProfileByNameOptions struct {
	FindOptions          *FindProfileOptions
	IgnoreAlreadyStopped bool
	WaitForStopped       bool
	PollOptions          PollOptions
}

// StoppedProfileWorkflowResult contains the resolved profile and stop response.
type StoppedProfileWorkflowResult struct {
	Profile       *Profile
	StopResponse  *EmptyDataResponse
	RuntimeStatus *ProfileRuntimeStatusResponse
}

// ImportProfileWorkflowOptions controls import verification.
type ImportProfileWorkflowOptions struct {
	PollOptions PollOptions
}

// ImportedProfileWorkflowResult contains verified import artifacts.
type ImportedProfileWorkflowResult struct {
	ImportResponse *ImportProfileResponse
	ImportStatus   *ImportStatusResponse
	ProfileMeta    *ProfileMeta
}

// EnableExtensionForProfileByNameOptions controls lookup and verification for extension attachment.
type EnableExtensionForProfileByNameOptions struct {
	FindOptions             *FindProfileOptions
	PollOptions             PollOptions
	RequireProfileUsageRead bool
}

// EnabledExtensionWorkflowResult contains the verified extension attachment state.
type EnabledExtensionWorkflowResult struct {
	Profile         *Profile
	EnableResponse  *StringDataResponse
	ObjectUsages    *ObjectProfileUsagesResponse
	ProfileUsages   *ProfileObjectUsagesResponse
	ProfileUsageErr error
}

// ExportProfileByNameToFolderOptions controls the lookup and export workflow behavior.
type ExportProfileByNameToFolderOptions struct {
	FindOptions        *FindProfileOptions
	ExportOptions      ExportProfileToFolderOptions
	StopBeforeExport   bool
	IgnoreStopNotReady bool
}

// ExportedProfileWorkflowResult contains the resolved profile and managed export result.
type ExportedProfileWorkflowResult struct {
	Profile *Profile
	Export  *ManagedExportResult
}

// GenerateProfileProxyByNameOptions controls profile lookup, proxy generation, and profile patching.
type GenerateProfileProxyByNameOptions struct {
	FindOptions     *FindProfileOptions
	GenerateOptions GenerateProfileProxyRequest
	PatchProfile    bool
}

// GeneratedProfileProxyWorkflowResult contains the resolved profile and generated proxy artifacts.
type GeneratedProfileProxyWorkflowResult struct {
	Profile       *Profile
	Connection    *GeneratedProxyConnection
	ProfileProxy  *Proxy
	Usage         *ProxyUsageResponse
	PatchResponse *EmptyDataResponse
}

// CreateProfilesAndVerify creates profiles and waits until their metas are readable.
func (s *WorkflowServiceOp) CreateProfilesAndVerify(ctx context.Context, reqBody *CreateProfileRequest, opts CreateProfilesAndVerifyOptions) (*CreatedProfilesWorkflowResult, error) {
	createResp, _, err := s.client.Profiles.Create(ctx, reqBody)
	if err != nil {
		return nil, err
	}
	if createResp == nil || len(createResp.Data.IDs) == 0 {
		return nil, NewArgError("createResponse.ids", "it must not be empty")
	}
	metas, err := s.waitForProfileMetas(ctx, createResp.Data.IDs, opts.PollOptions)
	if err != nil {
		return nil, err
	}
	return &CreatedProfilesWorkflowResult{
		CreateResponse: createResp,
		Profiles:       metas,
	}, nil
}

// FindProfileByNameVerified resolves a profile by exact name and confirms its meta is readable.
func (s *WorkflowServiceOp) FindProfileByNameVerified(ctx context.Context, profileName string, opts FindProfileByNameVerifiedOptions) (*VerifiedProfileWorkflowResult, error) {
	profile, _, err := s.client.Profiles.FindByName(ctx, profileName, workflowFindOptions(opts.FindOptions))
	if err != nil {
		return nil, err
	}
	meta, _, err := s.client.Profiles.GetMeta(ctx, profile.ID)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(strings.TrimSpace(meta.Name), strings.TrimSpace(profile.Name)) {
		return nil, fmt.Errorf("verified profile mismatch: search name=%q meta name=%q", profile.Name, meta.Name)
	}
	return &VerifiedProfileWorkflowResult{Profile: profile, Meta: meta}, nil
}

// StartProfileByName resolves a profile by exact name, starts it, and optionally waits for running status.
func (s *WorkflowServiceOp) StartProfileByName(ctx context.Context, profileName string, opts StartProfileByNameOptions) (*StartedProfileWorkflowResult, error) {
	verified, err := s.FindProfileByNameVerified(ctx, profileName, FindProfileByNameVerifiedOptions{FindOptions: opts.FindOptions})
	if err != nil {
		return nil, err
	}
	profile := verified.Profile
	startResp, _, err := s.client.Launcher.Start(ctx, profile.FolderID, profile.ID, opts.StartOptions)
	if err != nil {
		return nil, err
	}
	result := &StartedProfileWorkflowResult{
		Profile:       profile,
		StartResponse: startResp,
	}
	if opts.WaitForRunning {
		statusResp, _, err := s.client.Launcher.WaitForRunning(ctx, profile.ID, opts.PollOptions)
		if err != nil {
			return nil, err
		}
		result.RuntimeStatus = statusResp
	}
	return result, nil
}

// StartProfileAutomationByName resolves a profile by exact name, starts it with automation normalization, and returns resolved CDP endpoints.
func (s *WorkflowServiceOp) StartProfileAutomationByName(ctx context.Context, profileName string, opts StartProfileAutomationByNameOptions) (*StartedProfileAutomationWorkflowResult, error) {
	verified, err := s.FindProfileByNameVerified(ctx, profileName, FindProfileByNameVerifiedOptions{FindOptions: opts.FindOptions})
	if err != nil {
		return nil, err
	}
	profile := verified.Profile
	startResp, _, err := s.client.Launcher.Start(ctx, profile.FolderID, profile.ID, opts.StartOptions)
	if err != nil {
		return nil, err
	}
	cdpWebSocketURL, err := startResp.Data.ResolveCDPWebSocketURL(ctx)
	if err != nil {
		return nil, err
	}
	rodControlURL, err := startResp.Data.ResolveRodControlURL(ctx)
	if err != nil {
		return nil, err
	}
	result := &StartedProfileAutomationWorkflowResult{
		Profile:             profile,
		StartResponse:       startResp,
		RequestedAutomation: startResp.Data.RequestedAutomation,
		LauncherAutomation:  startResp.Data.LauncherAutomation,
		CDPPort:             startResp.Data.CDPPort,
		CDPWebSocketURL:     cdpWebSocketURL,
		RodControlURL:       rodControlURL,
	}
	if opts.WaitForRunning {
		statusResp, _, err := s.client.Launcher.WaitForRunning(ctx, profile.ID, opts.PollOptions)
		if err != nil {
			return nil, err
		}
		result.RuntimeStatus = statusResp
	}
	return result, nil
}

// StopProfileByName resolves a profile by exact name and stops it.
func (s *WorkflowServiceOp) StopProfileByName(ctx context.Context, profileName string, opts StopProfileByNameOptions) (*StoppedProfileWorkflowResult, error) {
	verified, err := s.FindProfileByNameVerified(ctx, profileName, FindProfileByNameVerifiedOptions{FindOptions: opts.FindOptions})
	if err != nil {
		return nil, err
	}
	profile := verified.Profile
	stopResp, _, err := s.client.Launcher.Stop(ctx, profile.ID)
	if err != nil {
		if opts.IgnoreAlreadyStopped && isAlreadyStoppedError(err) {
			result := &StoppedProfileWorkflowResult{Profile: profile}
			if opts.WaitForStopped {
				statusResp, statusErr := s.waitForStopped(ctx, profile.ID, opts.PollOptions)
				if statusErr != nil {
					return nil, statusErr
				}
				result.RuntimeStatus = statusResp
			}
			return result, nil
		}
		return nil, err
	}
	result := &StoppedProfileWorkflowResult{
		Profile:      profile,
		StopResponse: stopResp,
	}
	if opts.WaitForStopped {
		statusResp, err := s.waitForStopped(ctx, profile.ID, opts.PollOptions)
		if err != nil {
			return nil, err
		}
		result.RuntimeStatus = statusResp
	}
	return result, nil
}

// ImportProfileAndVerify imports a profile archive and confirms the resulting profile meta is readable.
func (s *WorkflowServiceOp) ImportProfileAndVerify(ctx context.Context, reqBody *ImportProfileRequest, opts ImportProfileWorkflowOptions) (*ImportedProfileWorkflowResult, error) {
	importResp, _, err := s.client.Transfers.Import(ctx, reqBody)
	if err != nil {
		return nil, err
	}
	if importResp == nil || strings.TrimSpace(importResp.Data.ImportID) == "" {
		return nil, NewArgError("importResponse.import_id", "it must not be empty")
	}
	statusResp, _, err := s.client.Transfers.WaitForImportDone(ctx, importResp.Data.ImportID, opts.PollOptions)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(statusResp.Data.NewProfileID) == "" {
		return nil, NewArgError("importStatus.new_profile_id", "it must not be empty")
	}
	meta, _, err := s.client.Profiles.GetMeta(ctx, statusResp.Data.NewProfileID)
	if err != nil {
		return nil, err
	}
	return &ImportedProfileWorkflowResult{
		ImportResponse: importResp,
		ImportStatus:   statusResp,
		ProfileMeta:    meta,
	}, nil
}

// EnableExtensionForProfileByName enables an extension and verifies the object-to-profile binding.
func (s *WorkflowServiceOp) EnableExtensionForProfileByName(ctx context.Context, profileName, extensionID string, opts EnableExtensionForProfileByNameOptions) (*EnabledExtensionWorkflowResult, error) {
	if strings.TrimSpace(extensionID) == "" {
		return nil, NewArgError("extensionID", "it must not be empty")
	}
	verified, err := s.FindProfileByNameVerified(ctx, profileName, FindProfileByNameVerifiedOptions{FindOptions: opts.FindOptions})
	if err != nil {
		return nil, err
	}
	enableResp, _, err := s.client.Resources.EnableExtensionForProfiles(ctx, extensionID, &SetResourceProfilesRequest{ProfileIDs: []string{verified.Profile.ID}})
	if err != nil {
		return nil, err
	}
	objectUsages, err := s.waitForExtensionUsage(ctx, extensionID, verified.Profile.ID, opts.PollOptions)
	if err != nil {
		return nil, err
	}
	result := &EnabledExtensionWorkflowResult{
		Profile:        verified.Profile,
		EnableResponse: enableResp,
		ObjectUsages:   objectUsages,
	}
	if opts.RequireProfileUsageRead {
		profileUsages, _, err := s.client.Resources.ProfileExtensionUsages(ctx, verified.Profile.ID)
		if err != nil {
			return nil, err
		}
		result.ProfileUsages = profileUsages
	} else {
		profileUsages, _, err := s.client.Resources.ProfileExtensionUsages(ctx, verified.Profile.ID)
		if err == nil {
			result.ProfileUsages = profileUsages
		} else {
			result.ProfileUsageErr = err
		}
	}
	return result, nil
}

// ExportProfileByNameToFolder resolves a profile by exact name and exports it into an organized folder.
func (s *WorkflowServiceOp) ExportProfileByNameToFolder(ctx context.Context, profileName string, opts ExportProfileByNameToFolderOptions) (*ExportedProfileWorkflowResult, error) {
	verified, err := s.FindProfileByNameVerified(ctx, profileName, FindProfileByNameVerifiedOptions{FindOptions: opts.FindOptions})
	if err != nil {
		return nil, err
	}
	profile := verified.Profile
	if opts.StopBeforeExport {
		if _, _, err := s.client.Launcher.Stop(ctx, profile.ID); err != nil && !opts.IgnoreStopNotReady {
			return nil, err
		}
	}
	exportOpts := opts.ExportOptions
	if exportOpts.ProfileName == "" {
		exportOpts.ProfileName = profile.Name
	}
	result, err := s.client.Archives.ExportProfileToFolder(ctx, profile.ID, exportOpts)
	if err != nil {
		return nil, err
	}
	return &ExportedProfileWorkflowResult{
		Profile: profile,
		Export:  result,
	}, nil
}

// GenerateProfileProxyByName resolves a profile, generates an MLX-managed proxy, and optionally patches the profile.
func (s *WorkflowServiceOp) GenerateProfileProxyByName(ctx context.Context, profileName string, opts GenerateProfileProxyByNameOptions) (*GeneratedProfileProxyWorkflowResult, error) {
	verified, err := s.FindProfileByNameVerified(ctx, profileName, FindProfileByNameVerifiedOptions{FindOptions: opts.FindOptions})
	if err != nil {
		return nil, err
	}
	profile := verified.Profile
	proxyResult, err := s.client.Proxies.GenerateProfileProxy(ctx, &opts.GenerateOptions)
	if err != nil {
		return nil, err
	}
	result := &GeneratedProfileProxyWorkflowResult{
		Profile:      profile,
		Connection:   proxyResult.Connection,
		ProfileProxy: proxyResult.ProfileProxy,
		Usage:        proxyResult.Usage,
	}
	if opts.PatchProfile {
		patchResp, _, err := s.client.Profiles.Patch(ctx, &PatchProfileRequest{
			ProfileID: profile.ID,
			Proxy:     proxyResult.ProfileProxy,
		})
		if err != nil {
			return nil, err
		}
		result.PatchResponse = patchResp
	}
	return result, nil
}

func (s *WorkflowServiceOp) waitForProfileMetas(ctx context.Context, ids []string, opts PollOptions) ([]ProfileMeta, error) {
	resp, _, err := pollUntil(ctx, opts, "created profiles did not become readable", func(ctx context.Context) (*ProfileMetasResponse, *Response, error) {
		return s.client.Profiles.GetMetas(ctx, &ProfileMetasRequest{IDs: ids})
	}, func(resp *ProfileMetasResponse) bool {
		if resp == nil {
			return false
		}
		return len(resp.Data.Profiles) == len(ids) && containsAllProfileIDs(resp.Data.Profiles, ids)
	}, func(resp *ProfileMetasResponse) string {
		if resp == nil {
			return ""
		}
		return fmt.Sprintf("verified=%d/%d", len(resp.Data.Profiles), len(ids))
	})
	if err != nil {
		return nil, err
	}
	return resp.Data.Profiles, nil
}

func (s *WorkflowServiceOp) waitForStopped(ctx context.Context, profileID string, opts PollOptions) (*ProfileRuntimeStatusResponse, error) {
	resp, _, err := pollUntil(ctx, opts, fmt.Sprintf("profile %s did not reach stopped status", profileID), func(ctx context.Context) (*ProfileRuntimeStatusResponse, *Response, error) {
		return s.client.Launcher.Status(ctx, profileID)
	}, func(resp *ProfileRuntimeStatusResponse) bool {
		if resp == nil {
			return false
		}
		status := strings.ToLower(strings.TrimSpace(resp.Data.Status))
		return status == "stopped" || strings.Contains(status, "stopped")
	}, func(resp *ProfileRuntimeStatusResponse) string {
		if resp == nil {
			return ""
		}
		return resp.Data.Status
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *WorkflowServiceOp) waitForExtensionUsage(ctx context.Context, extensionID, profileID string, opts PollOptions) (*ObjectProfileUsagesResponse, error) {
	resp, _, err := pollUntil(ctx, opts, fmt.Sprintf("extension %s was not attached to profile %s", extensionID, profileID), func(ctx context.Context) (*ObjectProfileUsagesResponse, *Response, error) {
		return s.client.Resources.ObjectProfileUsages(ctx, extensionID)
	}, func(resp *ObjectProfileUsagesResponse) bool {
		if resp == nil {
			return false
		}
		for _, usage := range resp.Data {
			if usage.ID == profileID {
				return true
			}
		}
		return false
	}, func(resp *ObjectProfileUsagesResponse) string {
		if resp == nil {
			return ""
		}
		return fmt.Sprintf("object_usages=%d", len(resp.Data))
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func containsAllProfileIDs(metas []ProfileMeta, ids []string) bool {
	if len(ids) == 0 {
		return true
	}
	seen := make(map[string]struct{}, len(metas))
	for _, meta := range metas {
		seen[meta.ID] = struct{}{}
	}
	for _, id := range ids {
		if _, ok := seen[id]; !ok {
			return false
		}
	}
	return true
}

func isAlreadyStoppedError(err error) bool {
	var apiErr *ErrorResponse
	if errors.As(err, &apiErr) && apiErr != nil {
		message := strings.ToLower(strings.TrimSpace(apiErr.Status.Message))
		return strings.Contains(message, "already stopped")
	}
	return strings.Contains(strings.ToLower(err.Error()), "already stopped")
}

func workflowFindOptions(opts *FindProfileOptions) *FindProfileOptions {
	if opts == nil {
		return &FindProfileOptions{StorageType: "local"}
	}
	cloned := *opts
	if cloned.StorageType == "" {
		cloned.StorageType = "local"
	}
	return &cloned
}
