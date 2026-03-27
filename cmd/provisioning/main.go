package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

var (
	headscaleURL    string
	headscaleAPIKey string
)

type provisionRequest struct {
	DeviceID string `json:"device_id"`
}

type provisionResponse struct {
	AuthKey string `json:"auth_key"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type rateLimitEntry struct {
	Timestamps []time.Time
}

var (
	rateMu    sync.Mutex
	rateStore = make(map[string]*rateLimitEntry)
)

const maxRequestsPerHour = 5

func checkRateLimit(deviceID string) bool {
	rateMu.Lock()
	defer rateMu.Unlock()

	now := time.Now()
	cutoff := now.Add(-1 * time.Hour)

	entry, ok := rateStore[deviceID]
	if !ok {
		rateStore[deviceID] = &rateLimitEntry{Timestamps: []time.Time{now}}
		return true
	}

	filtered := make([]time.Time, 0, len(entry.Timestamps))
	for _, t := range entry.Timestamps {
		if t.After(cutoff) {
			filtered = append(filtered, t)
		}
	}

	if len(filtered) >= maxRequestsPerHour {
		entry.Timestamps = filtered
		return false
	}

	entry.Timestamps = append(filtered, now)
	return true
}

type headscaleKeyRequest struct {
	User       string `json:"user"`
	Reusable   bool   `json:"reusable"`
	Ephemeral  bool   `json:"ephemeral"`
	Expiration string `json:"expiration"`
}

type headscaleKeyResponse struct {
	PreAuthKey struct {
		Key string `json:"key"`
	} `json:"preAuthKey"`
}

func createPreAuthKey() (string, error) {
	body, err := json.Marshal(headscaleKeyRequest{
		User:       "autobutler",
		Reusable:   false,
		Ephemeral:  false,
		Expiration: "2027-01-01T00:00:00Z",
	})
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, headscaleURL+"/api/v1/preauthkey", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+headscaleAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("headscale request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("headscale returned %d: %s", resp.StatusCode, string(respBody))
	}

	var keyResp headscaleKeyResponse
	if err := json.Unmarshal(respBody, &keyResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if keyResp.PreAuthKey.Key == "" {
		return "", fmt.Errorf("empty key in headscale response")
	}

	return keyResp.PreAuthKey.Key, nil
}

func handleProvision(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req provisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponse{Error: "invalid request body"})
		return
	}

	if req.DeviceID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponse{Error: "device_id is required"})
		return
	}

	if !checkRateLimit(req.DeviceID) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(errorResponse{Error: "rate limit exceeded, max 5 requests per hour per device"})
		return
	}

	key, err := createPreAuthKey()
	if err != nil {
		log.Printf("failed to create pre-auth key: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errorResponse{Error: "failed to provision key"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(provisionResponse{AuthKey: key})
}

func main() {
	headscaleURL = os.Getenv("HEADSCALE_URL")
	if headscaleURL == "" {
		headscaleURL = "http://localhost:8080"
	}

	headscaleAPIKey = os.Getenv("HEADSCALE_API_KEY")
	if headscaleAPIKey == "" {
		log.Fatal("HEADSCALE_API_KEY environment variable is required")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/provision", handleProvision)

	log.Printf("provisioning service listening on :8081")
	if err := http.ListenAndServe(":8081", mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
