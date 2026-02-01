package storageutil

import (
	"archive/zip"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
)

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
func DeleteFiles(params DeleteFilesParams) (*DeleteFilesResult, error) {
	device, err := FindManagedDeviceBySerial(params.DeviceSerial)
	if err != nil {
		return nil, err // coverage: ignore - requires device detection failure
	}

	cirrusDir := GetCirrusDir()
	if device != nil {
		cirrusDir = device.CirrusDir
	}
	for _, filePath := range params.FilePaths {
		fullPath := filepath.Join(cirrusDir, params.RootDir, filePath)
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
func MoveFile(params MoveFileParams) (*MoveFileResult, error) {
	oldDevice, err := FindManagedDeviceBySerial(params.OldDeviceSerial)
	if err != nil {
		return nil, err // coverage: ignore - requires device detection failure
	}
	newDevice, err := FindManagedDeviceBySerial(params.NewDeviceSerial)
	if err != nil {
		return nil, err // coverage: ignore - requires device detection failure
	}

	defaultCirrusDir := GetCirrusDir()

	oldCirrusDir := defaultCirrusDir
	if oldDevice != nil {
		oldCirrusDir = oldDevice.CirrusDir
	}
	newCirrusDir := defaultCirrusDir
	if newDevice != nil {
		newCirrusDir = newDevice.CirrusDir
	}

	oldFullPath := filepath.Join(oldCirrusDir, params.OldFilePath)
	newFullPath := filepath.Join(newCirrusDir, params.NewFilePath)

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
func CreateFolder(params CreateFolderParams) (*CreateFolderResult, error) {
	device, err := FindManagedDeviceBySerial(params.DeviceSerial)
	if err != nil {
		return nil, err // coverage: ignore - requires device detection failure
	}
	rootDir := GetCirrusDir()
	if device != nil {
		rootDir = device.CirrusDir
	}
	fullPath := filepath.Join(rootDir, params.FolderDir, params.FolderName)

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
func DownloadFile(params DownloadFileParams) (*DownloadFileResult, error) {
	device, err := FindManagedDeviceBySerial(params.DeviceSerial)
	if err != nil {
		return nil, err // coverage: ignore - requires device detection failure
	}
	cirrusDir := GetCirrusDir()
	if device != nil {
		cirrusDir = device.CirrusDir
	}
	fullPath := filepath.Join(cirrusDir, params.FilePath)
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
}

// UploadFilesStreamed streams multipart file uploads directly to disk
func UploadFilesStreamed(params UploadFilesStreamedParams) error {
	device, err := FindManagedDeviceBySerial(params.DeviceSerial)
	if err != nil {
		return fmt.Errorf("device not found: %w", err)
	}
	cirrusDir := GetCirrusDir()
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
			fileName := part.FileName()
			destDir := filepath.Join(cirrusDir, params.RootDir)
			if err := os.MkdirAll(destDir, 0755); err != nil {
				part.Close()
				return fmt.Errorf("failed to create directory: %w", err)
			}
			destPath := filepath.Join(destDir, fileName)

			// Handle file name conflicts
			if _, err := os.Stat(destPath); err == nil {
				ext := filepath.Ext(fileName)
				name := fileName[:len(fileName)-len(ext)]
				i := 1
				for {
					newFileName := fmt.Sprintf("%s_(%d)%s", name, i, ext)
					destPath = filepath.Join(destDir, newFileName)
					if _, err := os.Stat(destPath); os.IsNotExist(err) {
						break
					}
					i++
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

			// Try to place the file into destDir using a hard link (will fail if dest exists).
			// If hard link is not possible (cross-device), fall back to creating the
			// destination with O_EXCL and copying from the temp file.
			ext := filepath.Ext(fileName)
			name := fileName[:len(fileName)-len(ext)]
			if name == "" {
				name = "file"
			}
			candidate := fileName
			i := 0
			for {
				destPath = filepath.Join(destDir, candidate)

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
