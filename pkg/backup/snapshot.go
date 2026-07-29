package backup

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/autobutler-org/autobutler/internal/db"
	"github.com/autobutler-org/autobutler/pkg/util/eventbus"
	"github.com/autobutler-org/autobutler/pkg/util/iosemutil"
	"github.com/autobutler-org/autobutler/pkg/util/storageutil"
)

type VaultExportParams struct {
	Queries          *db.Queries
	LiveKey          []byte
	RecoveryPassword string
}

type SnapshotBackupParams struct {
	TargetDeviceSerial string
	Job                *BackupJob
	Store              BackupJobStore
	EventBus           *eventbus.Bus
	Vault              *VaultExportParams
	IOSemaphore        *iosemutil.Semaphore // throttles file copies to yield to interactive requests
}

func SnapshotBackup(
	ctx context.Context,
	params SnapshotBackupParams,
	sources []SourceDevice,
	target *storageutil.ManagedDevice,
) error {
	job := params.Job
	now := time.Now()

	job.Status = BackupStatusScanning
	job.UpdatedAt = now
	params.Store.Update(ctx, job)

	if params.EventBus != nil {
		params.EventBus.Publish(eventbus.Event{
			Kind: eventbus.EventBackupStarted,
			Data: BackupProgressData{JobID: job.ID},
		})
	}

	// Phase 1: scan all sources to count files and bytes.
	job.SourceDevices = make([]SourceDeviceProgress, len(sources))
	for i, src := range sources {
		sdp := SourceDeviceProgress{
			DeviceSerial: src.Serial,
			DeviceName:   src.Name,
		}
		files, bytes, err := scanDir(src.CirrusDir)
		if err != nil {
			return failJob(ctx, params, fmt.Errorf("scan %s: %w", src.Name, err))
		}
		sdp.FilesTotal = files
		sdp.BytesTotal = bytes
		job.TotalFiles += files
		job.TotalBytes += bytes
		job.SourceDevices[i] = sdp
	}
	job.UpdatedAt = time.Now()
	params.Store.Update(ctx, job)

	// Phase 2: copy files from each source to the target.
	job.Status = BackupStatusCopying
	job.UpdatedAt = time.Now()
	params.Store.Update(ctx, job)

	tmpDir := filepath.Join(target.DataDir, "tmp")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return failJob(ctx, params, fmt.Errorf("create tmp dir: %w", err))
	}

	lastPublish := time.Time{}
	for i, src := range sources {
		dirName := deviceDirName(src.Name, src.Serial)
		targetBase := filepath.Join(target.CirrusDir, dirName)

		srcFS := os.DirFS(src.CirrusDir)
		err := fs.WalkDir(srcFS, ".", func(relPath string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}

			if err := ctx.Err(); err != nil {
				return err
			}

			targetPath := filepath.Join(targetBase, relPath)

			if d.IsDir() {
				return os.MkdirAll(targetPath, 0755)
			}

			srcPath := filepath.Join(src.CirrusDir, relPath)
			srcInfo, err := os.Stat(srcPath)
			if err != nil {
				return fmt.Errorf("stat source: %w", err)
			}

			// Smart skip: if target exists with matching size and mtime >= source, skip.
			if tgtInfo, err := os.Stat(targetPath); err == nil {
				if tgtInfo.Size() == srcInfo.Size() && !tgtInfo.ModTime().Before(srcInfo.ModTime()) {
					job.FilesSkipped++
					job.SourceDevices[i].FilesSkipped++
					return nil
				}
			}

			// Acquire IO semaphore before copying to yield to interactive requests.
			if params.IOSemaphore != nil {
				if !params.IOSemaphore.AcquireDefault(ctx) {
					return fmt.Errorf("snapshot: IO semaphore timeout copying %s", relPath)
				}
				defer params.IOSemaphore.Release()
			}

			// Atomic copy: write to temp file then rename.
			if err := atomicCopy(srcPath, targetPath, srcInfo, tmpDir); err != nil {
				return fmt.Errorf("copy %s: %w", relPath, err)
			}

			job.FilesCopied++
			job.BytesCopied += srcInfo.Size()
			job.SourceDevices[i].FilesCopied++
			job.SourceDevices[i].BytesCopied += srcInfo.Size()

			if job.TotalFiles > 0 {
				job.Progress = float64(job.FilesCopied+job.FilesSkipped) / float64(job.TotalFiles)
			}
			job.UpdatedAt = time.Now()
			params.Store.Update(ctx, job)

			// Throttle WebSocket events to ~2/sec.
			if params.EventBus != nil && time.Since(lastPublish) > 500*time.Millisecond {
				params.EventBus.Publish(eventbus.Event{
					Kind: eventbus.EventBackupProgress,
					Data: BackupProgressData{
						JobID:       job.ID,
						Progress:    job.Progress,
						FilesCopied: job.FilesCopied,
						TotalFiles:  job.TotalFiles,
						BytesCopied: job.BytesCopied,
						TotalBytes:  job.TotalBytes,
						CurrentFile: relPath,
					},
				})
				lastPublish = time.Now()
			}

			return nil
		})

		if err != nil {
			return failJob(ctx, params, fmt.Errorf("backup %s: %w", src.Name, err))
		}
	}

	// Phase 3: vault export (if requested).
	if params.Vault != nil {
		if _, err := ExportVault(ctx, params.Vault.Queries, params.Vault.LiveKey, params.Vault.RecoveryPassword, target.CirrusDir); err != nil {
			return failJob(ctx, params, fmt.Errorf("vault export: %w", err))
		}
	}

	// Phase 4: generate integrity manifest.
	manifest, err := GenerateManifest(target.CirrusDir)
	if err != nil {
		return failJob(ctx, params, fmt.Errorf("generate manifest: %w", err))
	}
	if err := WriteManifest(manifest, target.CirrusDir); err != nil {
		return failJob(ctx, params, fmt.Errorf("write manifest: %w", err))
	}

	// Phase 5: complete.
	completedAt := time.Now()
	job.Status = BackupStatusCompleted
	job.Progress = 1.0
	job.CompletedAt = &completedAt
	job.UpdatedAt = completedAt
	params.Store.Update(ctx, job)

	if params.EventBus != nil {
		params.EventBus.Publish(eventbus.Event{
			Kind: eventbus.EventBackupCompleted,
			Data: BackupProgressData{
				JobID:       job.ID,
				Progress:    1.0,
				FilesCopied: job.FilesCopied,
				TotalFiles:  job.TotalFiles,
				BytesCopied: job.BytesCopied,
				TotalBytes:  job.TotalBytes,
			},
		})
	}

	return nil
}

func failJob(ctx context.Context, params SnapshotBackupParams, err error) error {
	now := time.Now()
	params.Job.Status = BackupStatusFailed
	params.Job.ErrorMsg = err.Error()
	params.Job.CompletedAt = &now
	params.Job.UpdatedAt = now
	params.Store.Update(ctx, params.Job)

	if params.EventBus != nil {
		params.EventBus.Publish(eventbus.Event{
			Kind: eventbus.EventBackupFailed,
			Data: BackupProgressData{JobID: params.Job.ID},
		})
	}
	return err
}

func scanDir(root string) (files int, bytes int64, err error) {
	err = filepath.WalkDir(root, func(_ string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		files++
		bytes += info.Size()
		return nil
	})
	return
}

func atomicCopy(src, dst string, srcInfo os.FileInfo, tmpDir string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(tmpDir, "backup-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	srcFile, err := os.Open(src)
	if err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}

	_, err = io.Copy(tmp, srcFile)
	srcFile.Close()
	tmp.Close()
	if err != nil {
		os.Remove(tmpPath)
		return err
	}

	// Preserve mtime so smart-skip works on future backups.
	if err := os.Chtimes(tmpPath, srcInfo.ModTime(), srcInfo.ModTime()); err != nil {
		os.Remove(tmpPath)
		return err
	}

	if err := os.Rename(tmpPath, dst); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return nil
}

// deviceDirName produces a filesystem-safe directory name for a source device.
func deviceDirName(name, serial string) string {
	if serial == "" {
		return "internal"
	}
	suffix := serial
	if len(suffix) > 8 {
		suffix = suffix[:8]
	}
	safe := sanitizeName(name)
	if safe == "" {
		safe = "device"
	}
	return safe + "_" + suffix
}

func sanitizeName(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|', 0:
			b.WriteRune('_')
		default:
			b.WriteRune(r)
		}
	}
	result := strings.TrimSpace(b.String())
	if len(result) > 200 {
		result = result[:200]
	}
	return result
}
