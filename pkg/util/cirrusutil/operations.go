package cirrusutil

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
	RootDir    string
	FilePaths  []string
	DeviceName string
}

// DeleteFilesResult contains the result of a delete operation
type DeleteFilesResult struct {
	RootDir string
}

// DeleteFilesChannel is a channel for deleting files
type DeleteFilesChannel chan DeleteFilesParams

// DeleteFiles removes files from the filesystem, handling both single and multi-device scenarios
func DeleteFiles(params DeleteFilesParams) (*DeleteFilesResult, error) {
	managedDevices, err := GetManagedDevices()
	if err != nil {
		return nil, err // coverage: ignore - requires device detection failure
	}
	var device *ManagedDevice
	if params.DeviceName != "" {
		for _, d := range managedDevices {
			if d.Name == params.DeviceName {
				device = &d
				break
			}
		}
	}

	if device != nil || len(managedDevices) == 0 {
		// Single device or specified device
		for _, filePath := range params.FilePaths {
			fullPath := filepath.Join(device.CirrusDir, params.RootDir, filePath)
			if err := os.RemoveAll(fullPath); err != nil { // coverage: ignore - requires filesystem permission errors
				return nil, fmt.Errorf("failed to delete %s: %w", filePath, err)
			}
		}
	}

	return &DeleteFilesResult{
		RootDir: params.RootDir,
	}, nil
}

// MoveFileParams contains parameters for moving a file
type MoveFileParams struct {
	FilePath      string
	NewFilePath   string
	OldDeviceName string
	NewDeviceName string
}

// MoveFileResult contains the result of a move operation
type MoveFileResult struct {
	NewDir string
}

// MoveFileChannel is a channel for moving files
type MoveFileChannel chan MoveFileParams

// MoveFile moves a file from one location to another
func MoveFile(params MoveFileParams) (*MoveFileResult, error) {
	filesDir := GetCirrusDir()
	oldFullPath := filepath.Join(filesDir, params.FilePath)
	newFullPath := filepath.Join(filesDir, params.NewFilePath)

	newFullDir := filepath.Dir(newFullPath)
	if err := os.MkdirAll(newFullDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err) // coverage: ignore - requires filesystem permission errors
	}

	if err := os.Rename(oldFullPath, newFullPath); err != nil { // coverage: ignore - requires filesystem permission errors or cross-device move
		return nil, fmt.Errorf("failed to move file: %w", err)
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
	RootDir     string
	FileHeaders []*multipart.FileHeader
	ReturnDir   string
	DeviceName  string
}

// UploadFilesResult contains the result of an upload operation
type UploadFilesResult struct {
	RootDir string
}

// UploadFilesChannel is a channel for uploading files
type UploadFilesChannel chan UploadFilesParams

// UploadFiles saves uploaded files to the filesystem
func UploadFiles(params UploadFilesParams) (*UploadFilesResult, error) {
	for _, header := range params.FileHeaders {
		file, err := header.Open()
		if err != nil { // coverage: ignore - requires malformed multipart data
			return nil, fmt.Errorf("failed to open file %s: %w", header.Filename, err)
		}
		defer file.Close()

		fileDir := GetCirrusDir()
		newFilePath := filepath.Join(fileDir, params.RootDir, header.Filename)

		// Handle file name conflicts
		if _, err := os.Stat(newFilePath); err == nil {
			ext := filepath.Ext(header.Filename)
			name := header.Filename[:len(header.Filename)-len(ext)]
			i := 1
			for {
				newFileName := fmt.Sprintf("%s_(%d)%s", name, i, ext)
				newFilePath = filepath.Join(fileDir, params.RootDir, newFileName)
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
	FolderDir  string
	FolderName string
	DeviceName string
}

// CreateFolderResult contains the result of a folder creation operation
type CreateFolderResult struct {
	CurrentDir string
}

// CreateFolderChannel is a channel for creating folders
type CreateFolderChannel chan CreateFolderParams

// CreateFolder creates a new folder in the filesystem
func CreateFolder(params CreateFolderParams) (*CreateFolderResult, error) {
	rootDir := GetCirrusDir()
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
	FilePath       string
	ManagedDevices []ManagedDevice
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
	var fullPath string
	var err error

	if len(params.ManagedDevices) == 0 {
		// Fallback to single device
		rootDir := GetCirrusDir()
		fullPath = filepath.Join(rootDir, params.FilePath)
		if !DoesFileExist(fullPath) {
			return nil, fmt.Errorf("file not found: %s", fullPath)
		}
	} else {
		// Search for file across all managed devices
		var dirsToSearch []DirWithDevice
		for _, device := range params.ManagedDevices {
			dirsToSearch = append(dirsToSearch, DirWithDevice{
				Dir:        device.CirrusDir,
				DeviceName: device.Name,
				DevicePath: device.MountPoint,
			})
		}

		fullPath, err = FindFileAcrossDevices(dirsToSearch, params.FilePath)
		if err != nil {
			return nil, fmt.Errorf("file not found: %w", err)
		}
	}

	fileType := DetermineFileTypeFromPath(fullPath)

	return &DownloadFileResult{
		FullPath: fullPath,
		FileType: fileType,
		IsFolder: fileType == FileTypeFolder,
	}, nil
}
