package main

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

var (
	headscaleURL    string
	headscaleAPIKey string
	sharedSecret    string
	keyExpiryHours  time.Duration
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

// rateLimitEntry tracks timestamps for both device ID and IP rate limiting.
type rateLimitEntry struct {
	Timestamps []time.Time
}

var (
	rateMu    sync.Mutex
	rateStore = make(map[string]*rateLimitEntry)
	// lastPrune tracks when we last swept stale entries from rateStore.
	lastPrune time.Time
)

const (
	maxRequestsPerHour = 5
	pruneInterval      = 10 * time.Minute
)

// pruneRateStore removes entries that have had no activity in the past hour.
// Must be called with rateMu held.
func pruneRateStore(now time.Time) {
	if now.Sub(lastPrune) < pruneInterval {
		return
	}
	cutoff := now.Add(-1 * time.Hour)
	for key, entry := range rateStore {
		active := false
		for _, t := range entry.Timestamps {
			if t.After(cutoff) {
				active = true
				break
			}
		}
		if !active {
			delete(rateStore, key)
		}
	}
	lastPrune = now
}

// checkRateLimit returns true if the request should be allowed, false if rate-limited.
// It rate-limits by both source IP and device_id, taking the stricter result.
// Both keys are checked before any timestamps are recorded to avoid partial updates
// when one key is over limit and the other is not.
func checkRateLimit(sourceIP, deviceID string) bool {
	rateMu.Lock()
	defer rateMu.Unlock()

	now := time.Now()
	cutoff := now.Add(-1 * time.Hour)

	pruneRateStore(now)

	keys := []string{"ip:" + sourceIP, "device:" + deviceID}
	filtered := make([][]time.Time, len(keys))

	for i, key := range keys {
		entry, ok := rateStore[key]
		if !ok {
			filtered[i] = []time.Time{}
			continue
		}
		f := make([]time.Time, 0, len(entry.Timestamps))
		for _, t := range entry.Timestamps {
			if t.After(cutoff) {
				f = append(f, t)
			}
		}
		filtered[i] = f
		if len(f) >= maxRequestsPerHour {
			entry.Timestamps = f
			return false
		}
	}

	for i, key := range keys {
		entry, ok := rateStore[key]
		if !ok {
			rateStore[key] = &rateLimitEntry{Timestamps: append(filtered[i], now)}
			continue
		}
		entry.Timestamps = append(filtered[i], now)
	}

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
	expiration := time.Now().UTC().Add(keyExpiryHours).Format(time.RFC3339)
	body, err := json.Marshal(headscaleKeyRequest{
		User:       "autobutler",
		Reusable:   false,
		Ephemeral:  false,
		Expiration: expiration,
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

	// Authenticate caller via shared secret (constant-time to prevent timing attacks).
	provided := r.Header.Get("X-Provisioning-Secret")
	if subtle.ConstantTimeCompare([]byte(provided), []byte(sharedSecret)) != 1 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(errorResponse{Error: "unauthorized"})
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

	// Extract source IP for rate limiting (strip port if present).
	sourceIP := r.RemoteAddr
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		sourceIP = host
	}

	if !checkRateLimit(sourceIP, req.DeviceID) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(errorResponse{Error: "rate limit exceeded, max 5 requests per hour"})
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

	sharedSecret = os.Getenv("PROVISIONING_SECRET")
	if sharedSecret == "" {
		log.Fatal("PROVISIONING_SECRET environment variable is required")
	}

	keyExpiryHours = time.Hour
	if exp := os.Getenv("PROVISIONING_KEY_EXPIRY_HOURS"); exp != "" {
		if d, err := time.ParseDuration(exp); err == nil {
			keyExpiryHours = d
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/provision", handleProvision)

	log.Printf("provisioning service listening on :8081")
	if err := http.ListenAndServe(":8081", mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
