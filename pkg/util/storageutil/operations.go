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

// UploadFilesParams contains parameters for uploading files
type UploadFilesParams struct {
	RootDir      string
	FileHeaders  []*multipart.FileHeader
	ReturnDir    string
	DeviceSerial string
}

// UploadFilesResult contains the result of an upload operation
type UploadFilesResult struct {
	RootDir string
}

// UploadFilesChannel is a channel for uploading files
type UploadFilesChannel chan UploadFilesParams

// UploadFiles saves uploaded files to the filesystem
func UploadFiles(params UploadFilesParams) (*UploadFilesResult, error) {
	device, err := FindManagedDeviceBySerial(params.DeviceSerial)
	if err != nil {
		return nil, err // coverage: ignore - requires device detection failure
	}

	cirrusDir := device.CirrusDir
	if device != nil {
		cirrusDir = device.CirrusDir
	}
	for _, header := range params.FileHeaders {
		file, err := header.Open()
		if err != nil { // coverage: ignore - requires malformed multipart data
			return nil, fmt.Errorf("failed to open file %s: %w", header.Filename, err)
		}
		defer file.Close()

		newFilePath := filepath.Join(cirrusDir, params.RootDir, header.Filename)

		// Handle file name conflicts
		if _, err := os.Stat(newFilePath); err == nil {
			ext := filepath.Ext(header.Filename)
			name := header.Filename[:len(header.Filename)-len(ext)]
			i := 1
			for {
				newFileName := fmt.Sprintf("%s_(%d)%s", name, i, ext)
				newFilePath = filepath.Join(cirrusDir, params.RootDir, newFileName)
				if _, err := os.Stat(newFilePath); os.IsNotExist(err) {
					break
				}
				i++
			}
		}

		newFile, err := os.Create(newFilePath)
		if err != nil { // coverage: ignore - requires filesystem permission errors
			return nil, fmt.Errorf("failed to create file %s: %w", header.Filename, err)
		}
		defer newFile.Close()

		if _, err := io.Copy(newFile, file); err != nil { // coverage: ignore - requires I/O failure during copy
			return nil, fmt.Errorf("failed to write file %s: %w", header.Filename, err)
		}
	}

	returnDir := params.ReturnDir
	if returnDir == "" {
		returnDir = params.RootDir
	}

	return &UploadFilesResult{
		RootDir: returnDir,
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

			outFile, err := os.Create(destPath)
			if err != nil {
				part.Close()
				return fmt.Errorf("failed to create file: %w", err)
			}
			if _, err := io.Copy(outFile, part); err != nil {
				outFile.Close()
				part.Close()
				return fmt.Errorf("failed to write file: %w", err)
			}
			outFile.Close()
			part.Close()
		} else {
			part.Close()
		}
	}
	return nil
}
