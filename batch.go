package mlx

import (
	"context"
	"fmt"
	"strings"
)

// BatchSummary reports how many profile items succeeded or failed.
type BatchSummary struct {
	Total     int
	Succeeded int
	Failed    int
}

// ProfileBatchFailure records one failed profile operation.
type ProfileBatchFailure struct {
	ProfileName string
	Err         error
}

// BatchItemResult stores the per-profile outcome of a batch workflow helper.
type BatchItemResult[T any] struct {
	ProfileName string
	Result      *T
	Err         error
}

// BatchResult stores ordered per-profile outcomes together with a summary.
type BatchResult[T any] struct {
	Summary BatchSummary
	Items   []BatchItemResult[T]
}

// Failures returns the failed profile entries from the batch result.
func (r *BatchResult[T]) Failures() []ProfileBatchFailure {
	if r == nil || len(r.Items) == 0 {
		return nil
	}
	failures := make([]ProfileBatchFailure, 0)
	for _, item := range r.Items {
		if item.Err != nil {
			failures = append(failures, ProfileBatchFailure{ProfileName: item.ProfileName, Err: item.Err})
		}
	}
	return failures
}

// BatchProfileOperationError aggregates failures from a multi-profile helper.
type BatchProfileOperationError struct {
	Operation string
	Failures  []ProfileBatchFailure
}

func (e *BatchProfileOperationError) Error() string {
	if e == nil {
		return "batch profile operation failed"
	}
	operation := strings.TrimSpace(e.Operation)
	if operation == "" {
		operation = "profile operation"
	}
	if len(e.Failures) == 0 {
		return fmt.Sprintf("batch %s failed", operation)
	}
	parts := make([]string, 0, minInt(len(e.Failures), 3))
	for i, failure := range e.Failures {
		if i == 3 {
			parts = append(parts, "...")
			break
		}
		label := strings.TrimSpace(failure.ProfileName)
		if label == "" {
			label = "<empty>"
		}
		parts = append(parts, fmt.Sprintf("%q: %v", label, failure.Err))
	}
	return fmt.Sprintf("batch %s failed for %d profiles: %s", operation, len(e.Failures), strings.Join(parts, "; "))
}

// Unwrap exposes all underlying item errors for errors.Is/errors.As checks.
func (e *BatchProfileOperationError) Unwrap() []error {
	if e == nil {
		return nil
	}
	errList := make([]error, 0, len(e.Failures))
	for _, failure := range e.Failures {
		if failure.Err != nil {
			errList = append(errList, failure.Err)
		}
	}
	return errList
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func batchProfilesByName[T any](profileNames []string, operation string, fn func(string) (*T, error)) (*BatchResult[T], error) {
	if len(profileNames) == 0 {
		return nil, NewArgError("profileNames", "it must not be empty")
	}

	result := &BatchResult[T]{
		Summary: BatchSummary{Total: len(profileNames)},
		Items:   make([]BatchItemResult[T], 0, len(profileNames)),
	}

	for _, rawName := range profileNames {
		profileName := strings.TrimSpace(rawName)
		item := BatchItemResult[T]{ProfileName: profileName}

		if profileName == "" {
			item.Err = NewArgError("profileName", "it must not be empty")
			result.Items = append(result.Items, item)
			continue
		}

		value, err := fn(profileName)
		item.Result = value
		item.Err = err
		result.Items = append(result.Items, item)
	}

	for _, item := range result.Items {
		if item.Err != nil {
			result.Summary.Failed++
		} else {
			result.Summary.Succeeded++
		}
	}

	failures := result.Failures()
	if len(failures) > 0 {
		return result, &BatchProfileOperationError{Operation: operation, Failures: failures}
	}

	return result, nil
}

// StartProfilesByName starts multiple profiles and aggregates per-profile failures.
func (s *WorkflowServiceOp) StartProfilesByName(ctx context.Context, profileNames []string, opts StartProfileByNameOptions) (*BatchResult[StartedProfileWorkflowResult], error) {
	return batchProfilesByName(profileNames, "start", func(profileName string) (*StartedProfileWorkflowResult, error) {
		return s.StartProfileByName(ctx, profileName, opts)
	})
}

// StopProfilesByName stops multiple profiles and aggregates per-profile failures.
func (s *WorkflowServiceOp) StopProfilesByName(ctx context.Context, profileNames []string, opts StopProfileByNameOptions) (*BatchResult[StoppedProfileWorkflowResult], error) {
	return batchProfilesByName(profileNames, "stop", func(profileName string) (*StoppedProfileWorkflowResult, error) {
		return s.StopProfileByName(ctx, profileName, opts)
	})
}

// ExportProfilesByNameToFolder exports multiple profiles and aggregates per-profile failures.
func (s *WorkflowServiceOp) ExportProfilesByNameToFolder(ctx context.Context, profileNames []string, opts ExportProfileByNameToFolderOptions) (*BatchResult[ExportedProfileWorkflowResult], error) {
	return batchProfilesByName(profileNames, "export", func(profileName string) (*ExportedProfileWorkflowResult, error) {
		return s.ExportProfileByNameToFolder(ctx, profileName, opts)
	})
}

// EnableExtensionForProfilesByName enables one extension across multiple profiles with aggregated failures.
func (s *WorkflowServiceOp) EnableExtensionForProfilesByName(ctx context.Context, profileNames []string, extensionID string, opts EnableExtensionForProfileByNameOptions) (*BatchResult[EnabledExtensionWorkflowResult], error) {
	if strings.TrimSpace(extensionID) == "" {
		return nil, NewArgError("extensionID", "it must not be empty")
	}
	return batchProfilesByName(profileNames, "enable extension", func(profileName string) (*EnabledExtensionWorkflowResult, error) {
		return s.EnableExtensionForProfileByName(ctx, profileName, extensionID, opts)
	})
}
