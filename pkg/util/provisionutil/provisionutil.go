package provisionutil

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/autobutler-org/quark/pkg/util/settingsutil"
	"github.com/google/uuid"
)

const defaultProvisioningURL = "https://network.quark.org:8081"

type provisionRequest struct {
	DeviceID string `json:"device_id"`
}

type provisionResponse struct {
	AuthKey string `json:"auth_key"`
}

func ProvisioningURL() string {
	if u := os.Getenv("QUARK_PROVISIONING_URL"); u != "" {
		return u
	}
	return defaultProvisioningURL
}

func ProvisioningSecret() string {
	return os.Getenv("QUARK_PROVISIONING_SECRET")
}

func GetDeviceID() (string, error) {
	existing := settingsutil.GetDeviceID()
	if existing != "" {
		return existing, nil
	}

	hostname, _ := os.Hostname()

	var machineID string
	for _, path := range []string{"/etc/machine-id", "/var/lib/dbus/machine-id"} {
		data, err := os.ReadFile(path)
		if err == nil {
			b := bytes.TrimSpace(data)
			machineID = string(b)
			break
		}
	}

	if machineID != "" {
		h := sha256.Sum256([]byte(hostname + ":" + machineID))
		id := hex.EncodeToString(h[:])
		if err := settingsutil.SetDeviceID(id); err != nil {
			return "", fmt.Errorf("save device id: %w", err)
		}
		return id, nil
	}

	id := uuid.New().String()
	if err := settingsutil.SetDeviceID(id); err != nil {
		return "", fmt.Errorf("save device id: %w", err)
	}
	return id, nil
}

func ProvisionAuthKey(deviceID string) (string, error) {
	body, err := json.Marshal(provisionRequest{DeviceID: deviceID})
	if err != nil {
		return "", fmt.Errorf("marshal provision request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, ProvisioningURL()+"/provision", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create provision request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	secret := ProvisioningSecret()
	if secret == "" {
		return "", fmt.Errorf("QUARK_PROVISIONING_SECRET is not set; cannot authenticate with provisioning service")
	}
	req.Header.Set("X-Provisioning-Secret", secret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("provision request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read provision response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("provisioning service returned %d: %s", resp.StatusCode, string(respBody))
	}

	var pResp provisionResponse
	if err := json.Unmarshal(respBody, &pResp); err != nil {
		return "", fmt.Errorf("unmarshal provision response: %w", err)
	}

	if pResp.AuthKey == "" {
		return "", fmt.Errorf("empty auth key from provisioning service")
	}

	return pResp.AuthKey, nil
}
