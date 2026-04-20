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

// StartProfileByName resolves a profile by exact name, starts it, and optionally waits for running status.
func (s *WorkflowServiceOp) StartProfileByName(ctx context.Context, profileName string, opts StartProfileByNameOptions) (*StartedProfileWorkflowResult, error) {
	profile, _, err := s.client.Profiles.FindByName(ctx, profileName, opts.FindOptions)
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
	profile, _, err := s.client.Profiles.FindByName(ctx, profileName, opts.FindOptions)
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
	profile, _, err := s.client.Profiles.FindByName(ctx, profileName, opts.FindOptions)
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

func isAlreadyStoppedError(err error) bool {
	var apiErr *ErrorResponse
	if errors.As(err, &apiErr) && apiErr != nil {
		message := strings.ToLower(strings.TrimSpace(apiErr.Status.Message))
		return strings.Contains(message, "already stopped")
	}
	return strings.Contains(strings.ToLower(err.Error()), "already stopped")
}
