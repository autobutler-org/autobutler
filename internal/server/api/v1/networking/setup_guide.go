package v1_networking

import (
	"net/http"
	"runtime"

	"autobutler/pkg/util/serverutil"

	"github.com/gin-gonic/gin"
)

var getSetupGuideRoute = serverutil.ApiRoute(
	"GET", "/networking/setup-guide", func(c *gin.Context) *serverutil.Response {
		guide := generateSetupGuide()

		return serverutil.NewResponse().
			WithContentType(serverutil.ContentTypeJSON).
			WithStatusCode(http.StatusOK).
			WithData(guide)
	},
)

func generateSetupGuide() gin.H {
	os := runtime.GOOS
	installCmd := getInstallCommand(os)

	return gin.H{
		"platform": os,
		"steps": []gin.H{
			{
				"number":      1,
				"title":       "Install Headscale",
				"description": "Install Headscale on this device to act as your coordination server",
				"command":     installCmd,
			},
			{
				"number":      2,
				"title":       "Start Headscale Server",
				"description": "Start the Headscale server in the background",
				"command":     "headscale serve &",
			},
			{
				"number":      3,
				"title":       "Create a User",
				"description": "Create a user/namespace for your devices",
				"command":     "headscale users create default",
			},
			{
				"number":      4,
				"title":       "Generate Pre-Auth Key",
				"description": "Create a pre-authentication key for this node",
				"command":     "headscale preauthkeys create --user default --reusable --expiration 90d",
			},
			{
				"number":      5,
				"title":       "Configure Node",
				"description": "Enter the configuration details in the form above. Use http://localhost:8080 as your Headscale URL and paste the pre-auth key.",
			},
		},
		"documentation_url": "https://headscale.net/",
		"alternative": gin.H{
			"title":       "Use Existing Headscale Server",
			"description": "If you already have a Headscale server running elsewhere, you can skip steps 1-4 and just configure this node to connect to your existing server.",
		},
	}
}

func getInstallCommand(os string) string {
	switch os {
	case "darwin":
		return "brew install headscale"
	case "linux":
		return "# For Debian/Ubuntu:\nwget -O- https://github.com/juanfont/headscale/releases/latest/download/headscale_linux_amd64 > /usr/local/bin/headscale && chmod +x /usr/local/bin/headscale"
	default:
		return "# Visit https://github.com/juanfont/headscale/releases"
	}
}
