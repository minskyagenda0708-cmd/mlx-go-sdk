package mlx

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultExportPollInterval = 2 * time.Second
	defaultExportWaitTimeout  = 2 * time.Minute
)

// ArchiveManager provides filesystem-oriented helpers for export flows.
type ArchiveManager interface {
	OrganizeExport(string, string, string) (*OrganizedArchive, error)
	ExportProfileToFolder(context.Context, string, ExportProfileToFolderOptions) (*ManagedExportResult, error)
}

// ArchiveManagerOp is the concrete archive manager implementation.
type ArchiveManagerOp struct {
	client *Client
}

// ExportProfileToFolderOptions controls how an exported archive is organized on disk.
type ExportProfileToFolderOptions struct {
	RootDir      string
	FolderName   string
	ProfileName  string
	PollInterval time.Duration
	WaitTimeout  time.Duration
}

// OrganizedArchive describes where an exported archive ended up on disk.
type OrganizedArchive struct {
	SourcePath  string
	ArchiveDir  string
	ArchivePath string
	ZipFileName string
	FolderName  string
}

// ManagedExportResult combines export job details with filesystem placement.
type ManagedExportResult struct {
	ExportJob *ExportStatusResponse
	Archive   *OrganizedArchive
}

// DefaultArchiveFolderName returns a safe directory name for a profile archive.
func DefaultArchiveFolderName(profileName, profileID string, exportedAt time.Time) string {
	base := strings.TrimSpace(profileName)
	if base == "" {
		base = "profile"
	}
	stamp := exportedAt.UTC().Format("20060102-150405")
	name := fmt.Sprintf("%s__%s__%s", base, profileID, stamp)
	return sanitizeArchiveFolderName(name)
}

// OrganizeExport moves an exported zip into a dedicated folder without renaming the zip file itself.
func (m *ArchiveManagerOp) OrganizeExport(exportPath, rootDir, folderName string) (*OrganizedArchive, error) {
	if exportPath == "" {
		return nil, NewArgError("exportPath", "it must not be empty")
	}
	if rootDir == "" {
		return nil, NewArgError("rootDir", "it must not be empty")
	}
	if folderName == "" {
		return nil, NewArgError("folderName", "it must not be empty")
	}

	sourceInfo, err := os.Stat(exportPath)
	if err != nil {
		return nil, err
	}
	if sourceInfo.IsDir() {
		return nil, NewArgError("exportPath", "it must point to a zip file, not a directory")
	}
	if strings.ToLower(filepath.Ext(exportPath)) != ".zip" {
		return nil, NewArgError("exportPath", "it must point to a .zip file")
	}

	safeFolderName := sanitizeArchiveFolderName(folderName)
	archiveDir := filepath.Join(rootDir, safeFolderName)
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		return nil, err
	}

	zipFileName := filepath.Base(exportPath)
	archivePath := filepath.Join(archiveDir, zipFileName)
	if filepath.Base(archivePath) != zipFileName {
		return nil, NewArgError("archivePath", "zip file name must remain unchanged")
	}
	if _, err := os.Stat(archivePath); err == nil {
		return nil, fmt.Errorf("archive already exists at %s", archivePath)
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	if err := os.Rename(exportPath, archivePath); err != nil {
		if copyErr := moveFilePreservingName(exportPath, archivePath, sourceInfo.Mode()); copyErr != nil {
			return nil, copyErr
		}
	}

	return &OrganizedArchive{
		SourcePath:  exportPath,
		ArchiveDir:  archiveDir,
		ArchivePath: archivePath,
		ZipFileName: zipFileName,
		FolderName:  safeFolderName,
	}, nil
}

// ExportProfileToFolder exports a profile, waits for completion, and then organizes the resulting zip on disk.
func (m *ArchiveManagerOp) ExportProfileToFolder(ctx context.Context, profileID string, opts ExportProfileToFolderOptions) (*ManagedExportResult, error) {
	if m == nil || m.client == nil {
		return nil, NewArgError("archiveManager", "it must be attached to a client")
	}
	if profileID == "" {
		return nil, NewArgError("profileID", "it must not be empty")
	}
	if opts.RootDir == "" {
		return nil, NewArgError("RootDir", "it must not be empty")
	}
	if opts.PollInterval <= 0 {
		opts.PollInterval = defaultExportPollInterval
	}
	if opts.WaitTimeout <= 0 {
		opts.WaitTimeout = defaultExportWaitTimeout
	}

	exportResp, _, err := m.client.Transfers.Export(ctx, profileID)
	if err != nil {
		return nil, err
	}
	job, err := m.waitForExport(ctx, exportResp.Data.ExportID, opts.PollInterval, opts.WaitTimeout)
	if err != nil {
		return nil, err
	}

	folderName := opts.FolderName
	if folderName == "" {
		exportedAt := time.UnixMilli(job.Data.Timestamp)
		folderName = DefaultArchiveFolderName(opts.ProfileName, profileID, exportedAt)
	}
	archive, err := m.OrganizeExport(job.Data.ArchivePath(), opts.RootDir, folderName)
	if err != nil {
		return nil, err
	}

	return &ManagedExportResult{ExportJob: job, Archive: archive}, nil
}

func (m *ArchiveManagerOp) waitForExport(ctx context.Context, exportID string, pollInterval, waitTimeout time.Duration) (*ExportStatusResponse, error) {
	resp, _, err := m.client.Transfers.WaitForExportDone(ctx, exportID, PollOptions{
		InitialInterval: pollInterval,
		MaxInterval:     pollInterval,
		Timeout:         waitTimeout,
		Multiplier:      1,
	})
	return resp, err
}

func sanitizeArchiveFolderName(name string) string {
	replacer := strings.NewReplacer(
		"<", "_",
		">", "_",
		":", "_",
		"\"", "_",
		"/", "_",
		"\\", "_",
		"|", "_",
		"?", "_",
		"*", "_",
	)
	name = replacer.Replace(strings.TrimSpace(name))
	name = strings.Trim(name, ". ")
	if name == "" {
		return "profile"
	}
	return name
}

func moveFilePreservingName(sourcePath, targetPath string, mode os.FileMode) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()

	target, err := os.OpenFile(targetPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}

	copyErr := func() error {
		defer target.Close()
		if _, err := io.Copy(target, source); err != nil {
			return err
		}
		return nil
	}()
	if copyErr != nil {
		_ = os.Remove(targetPath)
		return copyErr
	}

	if err := os.Remove(sourcePath); err != nil {
		return err
	}
	return nil
}
