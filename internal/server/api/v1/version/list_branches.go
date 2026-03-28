package v1_version

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/settingsutil"

	"github.com/gin-gonic/gin"
)

type BranchBuild struct {
	Branch     string    `json:"branch"`
	PRNumber   int       `json:"prNumber"`
	PRTitle    string    `json:"prTitle"`
	BuiltAt    time.Time `json:"builtAt"`
	ArtifactID int64     `json:"artifactId"`
}

const defaultProvisioningURL = "http://165.227.215.101:8081"

func getProvisioningURL() string {
	if url := os.Getenv("AUTOBUTLER_PROVISIONING_URL"); url != "" {
		return url
	}
	return defaultProvisioningURL
}

func getProvisioningSecret() string {
	return os.Getenv("AUTOBUTLER_PROVISIONING_SECRET")
}

// listBranches godoc
// @Summary List available branch builds
// @Description Returns available PR branch artifacts from the provisioning service. Requires dev mode.
// @Tags version
// @Produce json
// @Success 200 {array} BranchBuild
// @Failure 403 {object} serverutil.Response "Dev mode required"
// @Failure 502 {object} serverutil.Response "Provisioning service error"
// @Router /version/branches [get]
func listBranches(c *gin.Context) *serverutil.Response {
	if !settingsutil.GetDevMode() {
		return serverutil.Forbidden(errors.New("dev mode required"))
	}

	url := fmt.Sprintf("%s/artifacts", getProvisioningURL())
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, url, nil)
	if err != nil {
		return serverutil.NewResponse().WithStatusCode(http.StatusBadGateway).WithError(fmt.Errorf("failed to create request: %w", err))
	}

	secret := getProvisioningSecret()
	if secret != "" {
		req.Header.Set("X-Provisioning-Secret", secret)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return serverutil.NewResponse().WithStatusCode(http.StatusBadGateway).WithError(fmt.Errorf("provisioning service unavailable: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return serverutil.NewResponse().WithStatusCode(http.StatusBadGateway).WithError(fmt.Errorf("provisioning service returned %d: %s", resp.StatusCode, string(body)))
	}

	var builds []BranchBuild
	if err := json.NewDecoder(resp.Body).Decode(&builds); err != nil {
		return serverutil.NewResponse().WithStatusCode(http.StatusBadGateway).WithError(fmt.Errorf("failed to parse provisioning response: %w", err))
	}

	return serverutil.Ok().WithData(builds)
}

var listBranchesRoute = serverutil.ApiRoute(
	"GET", "/version/branches", listBranches,
)
