package storageutil

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime/multipart"
	"os"
	"path/filepath"
)

// BackupToDeviceParams contains parameters for backing up a device
type BackupToDeviceParams struct {
	SourceDeviceSerial string `json:"sourceDeviceSerial"`
	TargetDeviceSerial string `json:"targetDeviceSerial"`
}

// BackupToDeviceResult contains the data of a backup operation
type BackupToDeviceResult struct {
	SourceDeviceSerial string
	TargetDeviceSerial string
}

// BackupToDeviceChannel is a channel for backing up devices
type BackupToDeviceChannel chan BackupToDeviceParams

func (s *StorageService) BackupToDevice(params BackupToDeviceParams) (*BackupToDeviceResult, error) {
	// NOTE: Empty string returns the first internal storage
	sourceDevice, err := s.FindManagedDeviceBySerial(params.SourceDeviceSerial)
	if err != nil {
		return nil, err // coverage: ignore - requires device detection failure
	}

	targetDevice, err := s.FindManagedDeviceBySerial(params.TargetDeviceSerial)
	if err != nil {
		return nil, err // coverage: ignore - requires device detection failure
	}

	return BackupToDeviceWithDevices(params, sourceDevice, targetDevice)
}

// BackupToDeviceWithDevices performs a device backup using pre-resolved source and target devices.
// Use this in tests to inject test devices without hitting the real filesystem detector.
func BackupToDeviceWithDevices(params BackupToDeviceParams, sourceDevice *ManagedDevice, targetDevice *ManagedDevice) (*BackupToDeviceResult, error) {
	if sourceDevice == nil {
		return nil, errors.New("source device not found")
	}
	if targetDevice == nil {
		return nil, errors.New("target device not found")
	}

	if params.SourceDeviceSerial == params.TargetDeviceSerial {
		return nil, errors.New("source and target devices cannot be the same")
	}

	sourceDirFs := os.DirFS(sourceDevice.CirrusDir)
	// Walk all contents of sourceDirFs. If the file exists in targetDevice.CirrusDir, it will be overwritten. If it doesn't exist, it will be created.
	err := fs.WalkDir(sourceDirFs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		sourcePath := filepath.Join(sourceDevice.CirrusDir, path)
		targetPath := filepath.Join(targetDevice.CirrusDir, path)

		// Backup directory
		if d.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		// Backup file
		// For files, copy the content from source to target
		srcFile, err := os.Open(sourcePath)
		if err != nil {
			return fmt.Errorf("failed to open source file: %w", err)
		}
		defer srcFile.Close()

		destDir := filepath.Dir(targetPath)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return fmt.Errorf("failed to create target directory: %w", err)
		}

		// Truncate the file if it already exists, or create it if it doesn't exist
		destFile, err := os.Create(targetPath)
		if err != nil {
			return fmt.Errorf("failed to create target file: %w", err)
		}
		defer destFile.Close()

		if _, err := io.Copy(destFile, srcFile); err != nil {
			return fmt.Errorf("failed to copy file content: %w", err)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("backup failed: %w", err)
	}

	// Placeholder implementation - in a real implementation, this would perform the backup logic
	return &BackupToDeviceResult{
		SourceDeviceSerial: params.SourceDeviceSerial,
		TargetDeviceSerial: params.TargetDeviceSerial,
	}, nil
}

// DeleteFilesParams contains parameters for deleting files
type DeleteFilesParams struct {
	RootDir      string
	FilePaths    []string
	DeviceSerial string
}

// DeleteFilesResult contains the result of a delete operation
type DeleteFilesResult struct {
	RootDir string
}

// DeleteFilesChannel is a channel for deleting files
type DeleteFilesChannel chan DeleteFilesParams

// DeleteFiles removes files from the filesystem, handling both single and multi-device scenarios
func (s *StorageService) DeleteFiles(params DeleteFilesParams) (*DeleteFilesResult, error) {
	device, err := s.FindManagedDeviceBySerial(params.DeviceSerial)
	if err != nil {
		return nil, err // coverage: ignore - requires device detection failure
	}

	defaultCirrusDir, err := GetCirrusDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get cirrus directory: %w", err)
	}

	return DeleteFilesImpl(params, device, defaultCirrusDir)
}

// DeleteFilesImpl removes files using pre-resolved device and cirrus directory.
// Use this in tests to inject test devices without hitting the real filesystem detector.
func DeleteFilesImpl(params DeleteFilesParams, device *ManagedDevice, defaultCirrusDir string) (*DeleteFilesResult, error) {
	cirrusDir := defaultCirrusDir
	if device != nil {
		cirrusDir = device.CirrusDir
	}
	for _, filePath := range params.FilePaths {
		fullPath, err := safeJoin(cirrusDir, params.RootDir, filePath)
		if err != nil {
			return nil, fmt.Errorf("invalid file path: %w", err)
		}
		if err := os.RemoveAll(fullPath); err != nil { // coverage: ignore - requires filesystem permission errors
			return nil, fmt.Errorf("failed to delete %s: %w", filePath, err)
		}
	}

	return &DeleteFilesResult{
		RootDir: params.RootDir,
	}, nil
}

// MoveFileParams contains parameters for moving a file
type MoveFileParams struct {
	OldFilePath     string
	NewFilePath     string
	OldDeviceSerial string
	NewDeviceSerial string
}

// MoveFileResult contains the result of a move operation
type MoveFileResult struct {
	NewDir string
}

// MoveFileChannel is a channel for moving files
type MoveFileChannel chan MoveFileParams

// MoveFile moves a file from one location to another
func (s *StorageService) MoveFile(params MoveFileParams) (*MoveFileResult, error) {
	oldDevice, err := s.FindManagedDeviceBySerial(params.OldDeviceSerial)
	if err != nil {
		return nil, err // coverage: ignore - requires device detection failure
	}
	newDevice, err := s.FindManagedDeviceBySerial(params.NewDeviceSerial)
	if err != nil {
		return nil, err // coverage: ignore - requires device detection failure
	}

	defaultCirrusDir, err := GetCirrusDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get cirrus directory: %w", err)
	}

	return MoveFileImpl(params, oldDevice, newDevice, defaultCirrusDir)
}

// MoveFileImpl moves a file using pre-resolved devices and cirrus directory.
// Use this in tests to inject test devices without hitting the real filesystem detector.
func MoveFileImpl(params MoveFileParams, oldDevice *ManagedDevice, newDevice *ManagedDevice, defaultCirrusDir string) (*MoveFileResult, error) {
	oldCirrusDir := defaultCirrusDir
	if oldDevice != nil {
		oldCirrusDir = oldDevice.CirrusDir
	}
	newCirrusDir := defaultCirrusDir
	if newDevice != nil {
		newCirrusDir = newDevice.CirrusDir
	}

	oldFullPath, err := safeJoin(oldCirrusDir, params.OldFilePath)
	if err != nil {
		return nil, fmt.Errorf("invalid old file path: %w", err)
	}
	newFullPath, err := safeJoin(newCirrusDir, params.NewFilePath)
	if err != nil {
		return nil, fmt.Errorf("invalid new file path: %w", err)
	}

	newFullDir := filepath.Dir(newFullPath)
	if err := os.MkdirAll(newFullDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err) // coverage: ignore - requires filesystem permission errors
	}

	if err := os.Rename(oldFullPath, newFullPath); err != nil { // coverage: ignore - requires filesystem permission errors or cross-device move
		// Check for cross-device link error (EXDEV)
		if linkErr, ok := err.(*os.LinkError); ok && linkErr.Err.Error() == "invalid cross-device link" {
			// Fallback: copy then delete
			srcFile, err := os.Open(oldFullPath)
			if err != nil {
				return nil, fmt.Errorf("failed to open source file for cross-device move: %w", err)
			}
			defer srcFile.Close()

			// Create destination file
			dstFile, err := os.Create(newFullPath)
			if err != nil {
				return nil, fmt.Errorf("failed to create destination file for cross-device move: %w", err)
			}
			defer dstFile.Close()

			if _, err := io.Copy(dstFile, srcFile); err != nil {
				return nil, fmt.Errorf("failed to copy file for cross-device move: %w", err)
			}

			// Remove the source file
			if err := os.Remove(oldFullPath); err != nil {
				return nil, fmt.Errorf("failed to remove source file after cross-device move: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to move file: %w", err)
		}
	}

	newDir := filepath.Dir(params.NewFilePath)
	if newDir == "." {
		newDir = ""
	}

	return &MoveFileResult{
		NewDir: newDir,
	}, nil
}

// CreateFolderParams contains parameters for creating a folder
type CreateFolderParams struct {
	FolderDir    string
	FolderName   string
	DeviceSerial string
}

// CreateFolderResult contains the result of a folder creation operation
type CreateFolderResult struct {
	CurrentDir string
}

// CreateFolderChannel is a channel for creating folders
type CreateFolderChannel chan CreateFolderParams

// CreateFolder creates a new folder in the filesystem
func (s *StorageService) CreateFolder(params CreateFolderParams) (*CreateFolderResult, error) {
	device, err := s.FindManagedDeviceBySerial(params.DeviceSerial)
	if err != nil {
		return nil, err // coverage: ignore - requires device detection failure
	}
	defaultCirrusDir, err := GetCirrusDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get cirrus directory: %w", err)
	}
	return CreateFolderImpl(params, device, defaultCirrusDir)
}

// CreateFolderImpl creates a folder using a pre-resolved device and cirrus directory.
// Use this in tests to inject test devices without hitting the real filesystem detector.
func CreateFolderImpl(params CreateFolderParams, device *ManagedDevice, defaultCirrusDir string) (*CreateFolderResult, error) {
	rootDir := defaultCirrusDir
	if device != nil {
		rootDir = device.CirrusDir
	}
	fullPath, err := safeJoin(rootDir, params.FolderDir, params.FolderName)
	if err != nil {
		return nil, fmt.Errorf("invalid folder path: %w", err)
	}

	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create folder: %w", err) // coverage: ignore - requires filesystem permission errors
	}

	return &CreateFolderResult{
		CurrentDir: params.FolderDir,
	}, nil
}

// DownloadFileParams contains parameters for downloading a file
type DownloadFileParams struct {
	FilePath     string
	DeviceSerial string
}

// DownloadFileResult contains the result of a download operation
type DownloadFileResult struct {
	FullPath  string
	FileType  FileType
	IsFolder  bool
	ZipWriter *zip.Writer
	MimeType  string
}

// DownloadFile prepares a file for download, handling both files and folders (as zip)
func (s *StorageService) DownloadFile(params DownloadFileParams) (*DownloadFileResult, error) {
	device, err := s.FindManagedDeviceBySerial(params.DeviceSerial)
	if err != nil {
		return nil, err // coverage: ignore - requires device detection failure
	}
	defaultCirrusDir, err := GetCirrusDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get cirrus directory: %w", err)
	}
	return DownloadFileImpl(params, device, defaultCirrusDir)
}

// DownloadFileImpl prepares a file for download using pre-resolved device and cirrus directory.
// Use this in tests to inject test devices without hitting the real filesystem detector.
func DownloadFileImpl(params DownloadFileParams, device *ManagedDevice, defaultCirrusDir string) (*DownloadFileResult, error) {
	cirrusDir := defaultCirrusDir
	if device != nil {
		cirrusDir = device.CirrusDir
	}
	fullPath, err := safeJoin(cirrusDir, params.FilePath)
	if err != nil {
		return nil, fmt.Errorf("invalid file path: %w", err)
	}
	fileType := DetermineFileTypeFromPath(fullPath)

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file not found: %s", params.FilePath)
	}

	return &DownloadFileResult{
		FullPath: fullPath,
		FileType: fileType,
		IsFolder: fileType == FileTypeFolder,
	}, nil
}

// UploadFilesStreamedParams contains parameters for streaming file uploads
type UploadFilesStreamedParams struct {
	Reader       *multipart.Reader
	RootDir      string
	DeviceSerial string
	// Overwrite, when true, replaces an existing file with the same name instead
	// of appending a numeric suffix (e.g. file_(1).abdoc).
	Overwrite bool
}

// UploadFilesStreamed streams multipart file uploads directly to disk
func (s *StorageService) UploadFilesStreamed(params UploadFilesStreamedParams) error {
	device, err := s.FindManagedDeviceBySerial(params.DeviceSerial)
	if err != nil {
		return fmt.Errorf("device not found: %w", err)
	}
	defaultCirrusDir, err := GetCirrusDir()
	if err != nil {
		return fmt.Errorf("failed to get cirrus directory: %w", err)
	}
	return UploadFilesStreamedImpl(params, device, defaultCirrusDir)
}

// UploadFilesStreamedImpl streams file uploads using pre-resolved device and cirrus directory.
// Use this in tests to inject test devices without hitting the real filesystem detector.
func UploadFilesStreamedImpl(params UploadFilesStreamedParams, device *ManagedDevice, defaultCirrusDir string) error {
	cirrusDir := defaultCirrusDir
	if device != nil {
		cirrusDir = device.CirrusDir
	}

	for {
		part, err := params.Reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		formName := part.FormName()
		if formName == "files" && part.FileName() != "" {
			// Strip any directory components from the client-supplied filename to
			// prevent path traversal via the filename itself.
			fileName := filepath.Base(part.FileName())
			destDir, err := safeJoin(cirrusDir, params.RootDir)
			if err != nil {
				part.Close()
				return fmt.Errorf("invalid upload directory: %w", err)
			}
			if err := os.MkdirAll(destDir, 0755); err != nil {
				part.Close()
				return fmt.Errorf("failed to create directory: %w", err)
			}
			destPath, err := safeJoin(destDir, fileName)
			if err != nil {
				part.Close()
				return fmt.Errorf("invalid file name: %w", err)
			}

			// Handle file name conflicts.
			// When Overwrite is true, keep destPath as-is (the atomic rename below
			// will replace the existing file). Otherwise append a numeric suffix.
			if !params.Overwrite {
				if _, err := os.Stat(destPath); err == nil {
					ext := filepath.Ext(fileName)
					name := fileName[:len(fileName)-len(ext)]
					i := 1
					for {
						newFileName := fmt.Sprintf("%s_(%d)%s", name, i, ext)
						newDestPath, joinErr := safeJoin(destDir, newFileName)
						if joinErr != nil {
							part.Close()
							return fmt.Errorf("invalid candidate path: %w", joinErr)
						}
						destPath = newDestPath
						if _, err := os.Stat(destPath); os.IsNotExist(err) {
							break
						}
						i++
					}
				}
			}

			// Write to a temp file in data/tmp, then atomically move into place.
			// Determine tmp base adjacent to the cirrus/data dir so it's on the
			// same filesystem as the final destination.
			dataDir := GetDataDir()
			if device != nil {
				// device.CirrusDir is <dataDir>/cirrus, so parent is device data dir
				dataDir = filepath.Dir(cirrusDir)
			}
			tmpBase := filepath.Join(dataDir, "tmp")
			if err := os.MkdirAll(tmpBase, 0755); err != nil {
				part.Close()
				return fmt.Errorf("failed to create tmp directory: %w", err)
			}

			// Create a temp file and write the uploaded content to it
			tmpFile, err := os.CreateTemp(tmpBase, "upload-*")
			if err != nil {
				part.Close()
				return fmt.Errorf("failed to create temp file: %w", err)
			}
			tmpPath := tmpFile.Name()
			if _, err := io.Copy(tmpFile, part); err != nil {
				tmpFile.Close()
				os.Remove(tmpPath)
				part.Close()
				return fmt.Errorf("failed to write temp file: %w", err)
			}
			tmpFile.Close()

			// Place the temp file at destPath.
			// Overwrite=true: replace atomically (os.Rename) or truncate on copy.
			// Overwrite=false: retry with numeric suffixes on conflict.
			ext := filepath.Ext(fileName)
			name := fileName[:len(fileName)-len(ext)]
			if name == "" {
				name = "file"
			}
			candidate := fileName
			i := 0
			for {
				var joinErr error
				destPath, joinErr = safeJoin(destDir, candidate)
				if joinErr != nil {
					os.Remove(tmpPath)
					part.Close()
					return fmt.Errorf("invalid candidate path: %w", joinErr)
				}

				if params.Overwrite {
					// Atomic replace: rename temp over existing file (works same-FS).
					if err := os.Rename(tmpPath, destPath); err == nil {
						break
					}
					// Cross-device: fall back to truncating copy.
					tmpR, rerr := os.Open(tmpPath)
					if rerr != nil {
						os.Remove(tmpPath)
						part.Close()
						return fmt.Errorf("failed to open temp file for fallback copy: %w", rerr)
					}
					dst, derr := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
					if derr != nil {
						tmpR.Close()
						os.Remove(tmpPath)
						part.Close()
						return fmt.Errorf("failed to open destination for overwrite: %w", derr)
					}
					if _, cerr := io.Copy(dst, tmpR); cerr != nil {
						dst.Close()
						tmpR.Close()
						os.Remove(tmpPath)
						part.Close()
						return fmt.Errorf("failed to copy temp to destination: %w", cerr)
					}
					dst.Close()
					tmpR.Close()
					os.Remove(tmpPath)
					break
				}

				// Attempt hard link first (atomic and avoids extra copy)
				if err := os.Link(tmpPath, destPath); err == nil {
					// Success: remove temp and we're done
					os.Remove(tmpPath)
					break
				} else if os.IsExist(err) {
					// Destination exists; pick next candidate name and retry
					i++
					candidate = fmt.Sprintf("%s_(%d)%s", name, i, ext)
					continue
				} else {
					// Hard link failed for another reason (e.g., EXDEV). Try exclusive create + copy.
					// Open temp for reading
					tmpR, rerr := os.Open(tmpPath)
					if rerr != nil {
						os.Remove(tmpPath)
						part.Close()
						return fmt.Errorf("failed to open temp file for fallback copy: %w", rerr)
					}
					dst, derr := os.OpenFile(destPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
					if derr == nil {
						// Copy content from temp to destination
						if _, cerr := io.Copy(dst, tmpR); cerr != nil {
							dst.Close()
							tmpR.Close()
							os.Remove(tmpPath)
							part.Close()
							return fmt.Errorf("failed to copy temp to destination: %w", cerr)
						}
						dst.Close()
						tmpR.Close()
						os.Remove(tmpPath)
						break
					}
					tmpR.Close()
					if os.IsExist(derr) {
						// Destination exists; try next candidate name
						i++
						candidate = fmt.Sprintf("%s_(%d)%s", name, i, ext)
						continue
					}
					// Unknown error creating destination
					os.Remove(tmpPath)
					part.Close()
					return fmt.Errorf("failed to move uploaded file into place: linkErr=%v createErr=%v", err, derr)
				}
			}
			part.Close()
		} else {
			part.Close()
		}
	}
	return nil
}

// StatFileParams contains parameters for statting a file or directory
type StatFileParams struct {
	FilePath     string
	DeviceSerial string
}

// StatFileResult contains the result of a stat operation
type StatFileResult struct {
	FullPath string
	IsDir    bool
	FileType FileType
	Name     string
}

// StatFile resolves a cirrus-relative path to its filesystem metadata.
func (s *StorageService) StatFile(params StatFileParams) (*StatFileResult, error) {
	device, err := s.FindManagedDeviceBySerial(params.DeviceSerial)
	if err != nil {
		return nil, err // coverage: ignore - requires device detection failure
	}
	defaultCirrusDir, err := GetCirrusDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get cirrus directory: %w", err)
	}
	return StatFileImpl(params, device, defaultCirrusDir)
}

// StatFileImpl resolves a path using pre-resolved device and cirrus directory.
// Use this in tests to inject test devices without hitting the real filesystem detector.
func StatFileImpl(params StatFileParams, device *ManagedDevice, defaultCirrusDir string) (*StatFileResult, error) {
	cirrusDir := defaultCirrusDir
	if device != nil {
		cirrusDir = device.CirrusDir
	}
	fullPath, err := safeJoin(cirrusDir, params.FilePath)
	if err != nil {
		return nil, fmt.Errorf("invalid file path: %w", err)
	}
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("path not found: %s", params.FilePath)
		}
		return nil, fmt.Errorf("failed to stat path: %w", err)
	}
	isDir := info.IsDir()
	var fileType FileType
	if isDir {
		fileType = FileTypeFolder
	} else {
		fileType = DetermineFileTypeFromPath(fullPath)
	}
	return &StatFileResult{
		FullPath: fullPath,
		IsDir:    isDir,
		FileType: fileType,
		Name:     info.Name(),
	}, nil
}

// CopyFileParams contains parameters for duplicating a file in-place.
type CopyFileParams struct {
	// RelPath is the cirrus-relative path of the source file.
	RelPath      string
	DeviceSerial string
}

// CopyFileResult contains the result of a file copy operation.
type CopyFileResult struct {
	// NewRelPath is the cirrus-relative path of the newly created copy.
	NewRelPath string
}

// CopyFile duplicates a file within the same cirrus directory, producing a
// non-conflicting name by appending "_copy" (and further numeric suffixes if
// needed). It is the service-layer entry point and resolves the device by
// serial before delegating to CopyFileImpl.
func (s *StorageService) CopyFile(params CopyFileParams) (*CopyFileResult, error) {
	device, err := s.FindManagedDeviceBySerial(params.DeviceSerial)
	if err != nil {
		return nil, err // coverage: ignore - requires device detection failure
	}
	defaultCirrusDir, err := GetCirrusDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get cirrus directory: %w", err)
	}
	return CopyFileImpl(params, device, defaultCirrusDir)
}

// CopyFileImpl duplicates a file using pre-resolved device and cirrus directory.
// Use this in tests to inject test devices without hitting the real filesystem
// detector.
func CopyFileImpl(params CopyFileParams, device *ManagedDevice, defaultCirrusDir string) (*CopyFileResult, error) {
	cirrusDir := defaultCirrusDir
	if device != nil {
		cirrusDir = device.CirrusDir
	}

	srcFull, err := safeJoin(cirrusDir, params.RelPath)
	if err != nil {
		return nil, fmt.Errorf("invalid relPath: %w", err)
	}

	if _, err := os.Stat(srcFull); os.IsNotExist(err) {
		return nil, fmt.Errorf("%w: %s", ErrPathNotFound, params.RelPath)
	} else if err != nil {
		return nil, fmt.Errorf("failed to stat source: %w", err)
	}

	ext := filepath.Ext(srcFull)
	stem := srcFull[:len(srcFull)-len(ext)]
	destFull := GetNonConflictingPath(stem + "_copy" + ext)

	if err := copyFileContents(srcFull, destFull); err != nil {
		return nil, fmt.Errorf("copy failed: %w", err)
	}

	newRelPath, err := filepath.Rel(cirrusDir, destFull)
	if err != nil {
		return nil, fmt.Errorf("failed to compute relative path: %w", err)
	}

	return &CopyFileResult{NewRelPath: newRelPath}, nil
}

// copyFileContents writes the contents of src to dst.
func copyFileContents(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := out.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	_, err = io.Copy(out, in)
	return err
}
