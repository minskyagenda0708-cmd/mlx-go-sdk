package mlx

import (
	"context"
	"errors"
	"strings"
)

// WorkflowService provides higher-level helpers that combine multiple SDK calls.
type WorkflowService interface {
	StartProfileByName(context.Context, string, StartProfileByNameOptions) (*StartedProfileWorkflowResult, error)
	StopProfileByName(context.Context, string, StopProfileByNameOptions) (*StoppedProfileWorkflowResult, error)
	ExportProfileByNameToFolder(context.Context, string, ExportProfileByNameToFolderOptions) (*ExportedProfileWorkflowResult, error)
	GenerateProfileProxyByName(context.Context, string, GenerateProfileProxyByNameOptions) (*GeneratedProfileProxyWorkflowResult, error)
}

// WorkflowServiceOp is the concrete high-level workflow service.
type WorkflowServiceOp struct {
	client *Client
}

// StartProfileByNameOptions controls the lookup and launcher behavior for a start workflow.
type StartProfileByNameOptions struct {
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

// StopProfileByNameOptions controls the lookup used before stopping a profile.
type StopProfileByNameOptions struct {
	FindOptions          *FindProfileOptions
	IgnoreAlreadyStopped bool
}

// StoppedProfileWorkflowResult contains the resolved profile and stop response.
type StoppedProfileWorkflowResult struct {
	Profile      *Profile
	StopResponse *EmptyDataResponse
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

// StartProfileByName resolves a profile by exact name, starts it, and optionally waits for running status.
func (s *WorkflowServiceOp) StartProfileByName(ctx context.Context, profileName string, opts StartProfileByNameOptions) (*StartedProfileWorkflowResult, error) {
	profile, _, err := s.client.Profiles.FindByName(ctx, profileName, workflowFindOptions(opts.FindOptions))
	if err != nil {
		return nil, err
	}
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

// StopProfileByName resolves a profile by exact name and stops it.
func (s *WorkflowServiceOp) StopProfileByName(ctx context.Context, profileName string, opts StopProfileByNameOptions) (*StoppedProfileWorkflowResult, error) {
	profile, _, err := s.client.Profiles.FindByName(ctx, profileName, workflowFindOptions(opts.FindOptions))
	if err != nil {
		return nil, err
	}
	stopResp, _, err := s.client.Launcher.Stop(ctx, profile.ID)
	if err != nil {
		if opts.IgnoreAlreadyStopped && isAlreadyStoppedError(err) {
			return &StoppedProfileWorkflowResult{Profile: profile}, nil
		}
		return nil, err
	}
	return &StoppedProfileWorkflowResult{
		Profile:      profile,
		StopResponse: stopResp,
	}, nil
}

// ExportProfileByNameToFolder resolves a profile by exact name and exports it into an organized folder.
func (s *WorkflowServiceOp) ExportProfileByNameToFolder(ctx context.Context, profileName string, opts ExportProfileByNameToFolderOptions) (*ExportedProfileWorkflowResult, error) {
	profile, _, err := s.client.Profiles.FindByName(ctx, profileName, workflowFindOptions(opts.FindOptions))
	if err != nil {
		return nil, err
	}
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
	profile, _, err := s.client.Profiles.FindByName(ctx, profileName, workflowFindOptions(opts.FindOptions))
	if err != nil {
		return nil, err
	}
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
