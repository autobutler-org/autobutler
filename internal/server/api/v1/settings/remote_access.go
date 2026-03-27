package v1_settings

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/autobutler-org/autobutler/pkg/util/remoteutil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/settingsutil"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const defaultProvisioningURL = "http://165.227.215.101:8081"

type RemoteAccessResponse struct {
	Enabled   bool   `json:"enabled"`
	RemoteURL string `json:"remoteUrl,omitempty"`
}

type provisionRequest struct {
	DeviceID string `json:"device_id"`
}

type provisionResponse struct {
	AuthKey string `json:"auth_key"`
}

func provisioningURL() string {
	if u := os.Getenv("AUTOBUTLER_PROVISIONING_URL"); u != "" {
		return u
	}
	return defaultProvisioningURL
}

func serverPort() int {
	p := os.Getenv("PORT")
	if p == "" {
		return 8080
	}
	n, err := strconv.Atoi(p)
	if err != nil {
		return 8080
	}
	return n
}

func getDeviceID() (string, error) {
	existing := settingsutil.GetDeviceID()
	if existing != "" {
		return existing, nil
	}

	hostname, _ := os.Hostname()

	var machineID string
	for _, path := range []string{"/etc/machine-id", "/var/lib/dbus/machine-id"} {
		data, err := os.ReadFile(path)
		if err == nil {
			machineID = string(bytes.TrimSpace(data))
			break
		}
	}

	if machineID != "" {
		h := sha256.Sum256([]byte(hostname + machineID))
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

func provisionAuthKey(deviceID string) (string, error) {
	body, err := json.Marshal(provisionRequest{DeviceID: deviceID})
	if err != nil {
		return "", fmt.Errorf("marshal provision request: %w", err)
	}

	resp, err := http.Post(provisioningURL()+"/provision", "application/json", bytes.NewReader(body))
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

// getRemoteAccess godoc
// @Summary Get remote access status
// @Description Returns whether Tailscale remote access is enabled and the remote URL if available
// @Tags settings
// @Produce json
// @Success 200 {object} RemoteAccessResponse
// @Router /settings/remote-access [get]
var getRemoteAccessRoute = serverutil.ApiRoute(
	"GET", "/settings/remote-access", func(c *gin.Context) *serverutil.Response {
		return serverutil.Ok().WithData(RemoteAccessResponse{
			Enabled:   remoteutil.IsRunning(),
			RemoteURL: remoteutil.RemoteURL(),
		})
	},
)

// enableRemoteAccess godoc
// @Summary Enable remote access via Tailscale
// @Description Auto-provisions a Headscale pre-auth key and starts a tsnet node proxying traffic to the local server
// @Tags settings
// @Accept json
// @Produce json
// @Success 200 {object} RemoteAccessResponse
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /settings/remote-access [post]
var enableRemoteAccessRoute = serverutil.ApiRoute(
	"POST", "/settings/remote-access", func(c *gin.Context) *serverutil.Response {
		deviceID, err := getDeviceID()
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("device id: %w", err))
		}

		authKey, err := provisionAuthKey(deviceID)
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("provision: %w", err))
		}

		if err := remoteutil.Start(authKey); err != nil {
			return serverutil.InternalServerError(err)
		}
		if err := remoteutil.StartProxy(serverPort()); err != nil {
			remoteutil.Stop()
			return serverutil.InternalServerError(err)
		}
		if err := settingsutil.SetRemoteAccess(true); err != nil {
			return serverutil.InternalServerError(err)
		}
		return serverutil.Ok().WithData(RemoteAccessResponse{
			Enabled:   true,
			RemoteURL: remoteutil.RemoteURL(),
		})
	},
)

// disableRemoteAccess godoc
// @Summary Disable remote access
// @Description Stops the Tailscale tsnet node
// @Tags settings
// @Produce json
// @Success 200 {object} RemoteAccessResponse
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /settings/remote-access [delete]
var disableRemoteAccessRoute = serverutil.ApiRoute(
	"DELETE", "/settings/remote-access", func(c *gin.Context) *serverutil.Response {
		remoteutil.Stop()
		if err := settingsutil.SetRemoteAccess(false); err != nil {
			return serverutil.InternalServerError(err)
		}
		return serverutil.Ok().WithData(RemoteAccessResponse{
			Enabled: false,
		})
	},
)
