package migration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"golang.org/x/oauth2"
)

// GooglePhotosClient handles downloading media from Google Photos
type GooglePhotosClient struct {
	token      *oauth2.Token
	httpClient *http.Client
	baseURL    string
}

// NewGooglePhotosClient creates a new Google Photos Library API client
func NewGooglePhotosClient(ctx context.Context, token *oauth2.Token, oauthConfig *oauth2.Config) (*GooglePhotosClient, error) {
	client := oauthConfig.Client(ctx, token)

	return &GooglePhotosClient{
		token:      token,
		httpClient: client,
		baseURL:    "https://photoslibrary.googleapis.com/v1",
	}, nil
}

// PhotoItem represents a photo or video from Google Photos
type PhotoItem struct {
	ID           string
	Filename     string
	MimeType     string
	Width        int64
	Height       int64
	CreationTime string
	BaseURL      string // Download URL
	Content      io.ReadCloser
}

// MediaItemsResponse represents the API response for listing media items
type MediaItemsResponse struct {
	MediaItems    []MediaItem `json:"mediaItems"`
	NextPageToken string      `json:"nextPageToken"`
}

// MediaItem represents a single media item from the API
type MediaItem struct {
	ID            string        `json:"id"`
	Filename      string        `json:"filename"`
	MimeType      string        `json:"mimeType"`
	BaseURL       string        `json:"baseUrl"`
	MediaMetadata MediaMetadata `json:"mediaMetadata"`
}

// MediaMetadata contains metadata about the media item
type MediaMetadata struct {
	CreationTime string `json:"creationTime"`
	Width        string `json:"width"`
	Height       string `json:"height"`
}

// DownloadAllPhotos downloads all photos and videos from Google Photos
func (c *GooglePhotosClient) DownloadAllPhotos(ctx context.Context, progressCallback func(fileName string, current, total int)) ([]*PhotoItem, error) {
	// List all media items
	mediaItems, err := c.listAllMediaItems(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list media items: %w", err)
	}

	log.Printf("Found %d media items in Google Photos", len(mediaItems))

	downloadedPhotos := make([]*PhotoItem, 0, len(mediaItems))
	total := len(mediaItems)

	for i, item := range mediaItems {
		if progressCallback != nil {
			progressCallback(item.Filename, i+1, total)
		}

		// Download media content
		content, err := c.downloadMediaItem(ctx, item)
		if err != nil {
			log.Printf("Warning: failed to download media %s: %v", item.Filename, err)
			continue
		}

		downloadedPhotos = append(downloadedPhotos, &PhotoItem{
			ID:           item.ID,
			Filename:     item.Filename,
			MimeType:     item.MimeType,
			CreationTime: item.MediaMetadata.CreationTime,
			BaseURL:      item.BaseURL,
			Content:      content,
		})
	}

	return downloadedPhotos, nil
}

func (c *GooglePhotosClient) listAllMediaItems(ctx context.Context) ([]MediaItem, error) {
	mediaItems := []MediaItem{}
	pageToken := ""

	for {
		url := fmt.Sprintf("%s/mediaItems?pageSize=100", c.baseURL)
		if pageToken != "" {
			url = fmt.Sprintf("%s&pageToken=%s", url, pageToken)
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to list media items: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(body))
		}

		var result MediaItemsResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		mediaItems = append(mediaItems, result.MediaItems...)

		pageToken = result.NextPageToken
		if pageToken == "" {
			break
		}
	}

	return mediaItems, nil
}

func (c *GooglePhotosClient) downloadMediaItem(ctx context.Context, item MediaItem) (io.ReadCloser, error) {
	// Google Photos baseUrl must be appended with =d to download
	// For videos, append =dv
	downloadURL := item.BaseURL + "=d"
	if isVideo(item.MimeType) {
		downloadURL = item.BaseURL + "=dv"
	}

	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create download request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download media: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("download failed: %s", resp.Status)
	}

	return resp.Body, nil
}

func isVideo(mimeType string) bool {
	videoTypes := []string{
		"video/mp4",
		"video/quicktime",
		"video/mpeg",
		"video/x-msvideo",
		"video/x-matroska",
	}

	for _, vt := range videoTypes {
		if mimeType == vt {
			return true
		}
	}
	return false
}
