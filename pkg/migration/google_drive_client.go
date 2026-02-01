package migration

import (
	"context"
	"fmt"
	"io"
	"log"

	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// GoogleDriveClient handles downloading files from Google Drive
type GoogleDriveClient struct {
	token   *oauth2.Token
	service *drive.Service
}

// NewGoogleDriveClient creates a new Google Drive API client
func NewGoogleDriveClient(ctx context.Context, token *oauth2.Token, oauthConfig *oauth2.Config) (*GoogleDriveClient, error) {
	client := oauthConfig.Client(ctx, token)

	service, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create Drive client: %w", err)
	}

	return &GoogleDriveClient{
		token:   token,
		service: service,
	}, nil
}

// DriveFile represents a file downloaded from Google Drive
type DriveFile struct {
	ID       string
	Name     string
	MimeType string
	Size     int64
	Content  io.ReadCloser
}

// DownloadAllFiles downloads all files from Google Drive
func (c *GoogleDriveClient) DownloadAllFiles(ctx context.Context, progressCallback func(fileName string, current, total int)) ([]*DriveFile, error) {
	// List all files
	files, err := c.listAllFiles(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	log.Printf("Found %d files in Google Drive", len(files))

	downloadedFiles := make([]*DriveFile, 0, len(files))
	total := len(files)

	for i, file := range files {
		if progressCallback != nil {
			progressCallback(file.Name, i+1, total)
		}

		// Skip folders
		if file.MimeType == "application/vnd.google-apps.folder" {
			continue
		}

		// Download file content
		content, size, err := c.downloadFile(ctx, file)
		if err != nil {
			log.Printf("Warning: failed to download file %s: %v", file.Name, err)
			continue
		}

		downloadedFiles = append(downloadedFiles, &DriveFile{
			ID:       file.Id,
			Name:     file.Name,
			MimeType: file.MimeType,
			Size:     size,
			Content:  content,
		})
	}

	return downloadedFiles, nil
}

func (c *GoogleDriveClient) listAllFiles(ctx context.Context) ([]*drive.File, error) {
	files := []*drive.File{}
	pageToken := ""

	for {
		query := c.service.Files.List().
			Context(ctx).
			Fields("nextPageToken, files(id, name, mimeType, size)").
			PageSize(100)

		if pageToken != "" {
			query = query.PageToken(pageToken)
		}

		result, err := query.Do()
		if err != nil {
			return nil, fmt.Errorf("failed to list files: %w", err)
		}

		files = append(files, result.Files...)

		pageToken = result.NextPageToken
		if pageToken == "" {
			break
		}
	}

	return files, nil
}

func (c *GoogleDriveClient) downloadFile(ctx context.Context, file *drive.File) (io.ReadCloser, int64, error) {
	// Handle Google Workspace files (Docs, Sheets, etc.) - export as PDF
	if isGoogleWorkspaceFile(file.MimeType) {
		return c.exportWorkspaceFile(ctx, file)
	}

	// Download regular files
	resp, err := c.service.Files.Get(file.Id).Context(ctx).Download()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to download file: %w", err)
	}

	size := file.Size
	return resp.Body, size, nil
}

func (c *GoogleDriveClient) exportWorkspaceFile(ctx context.Context, file *drive.File) (io.ReadCloser, int64, error) {
	// Export Google Workspace files as PDF
	exportMimeType := "application/pdf"

	resp, err := c.service.Files.Export(file.Id, exportMimeType).Context(ctx).Download()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to export file: %w", err)
	}

	// We don't know the exact size for exported files
	return resp.Body, 0, nil
}

func isGoogleWorkspaceFile(mimeType string) bool {
	googleMimeTypes := []string{
		"application/vnd.google-apps.document",
		"application/vnd.google-apps.spreadsheet",
		"application/vnd.google-apps.presentation",
		"application/vnd.google-apps.drawing",
		"application/vnd.google-apps.form",
		"application/vnd.google-apps.site",
	}

	for _, gmt := range googleMimeTypes {
		if mimeType == gmt {
			return true
		}
	}
	return false
}
