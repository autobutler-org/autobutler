package v1_auth

import (
	"autobutler/pkg/util/serverutil"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	googleOAuthConfig *oauth2.Config
	oauthStateStore   = make(map[string]time.Time) // state -> timestamp
)

func init() {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	redirectURL := os.Getenv("GOOGLE_REDIRECT_URL")

	if clientID == "" || clientSecret == "" || redirectURL == "" {
		fmt.Println("WARNING: Google OAuth environment variables not set")
		fmt.Println("Required: GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET, GOOGLE_REDIRECT_URL")
	}

	googleOAuthConfig = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/photoslibrary.readonly",
			"https://www.googleapis.com/auth/drive.readonly",
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	fmt.Printf("Google OAuth configured with redirect URL: %s\n", redirectURL)
}

var googleAuthorizeRoute = serverutil.ApiRoute(
	"GET", "/auth/google/authorize", func(c *gin.Context) *serverutil.Response {
		// Generate a random state token for CSRF protection
		state, err := generateStateToken()
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("failed to generate state token: %w", err))
		}

		// Store state token with timestamp for validation (expires in 10 minutes)
		oauthStateStore[state] = time.Now().Add(10 * time.Minute)

		// Clean up expired states
		cleanExpiredStates()

		// Generate OAuth URL with additional parameters
		authURL := googleOAuthConfig.AuthCodeURL(
			state,
			oauth2.AccessTypeOffline,
			oauth2.ApprovalForce, // Force consent screen to show
		)

		return serverutil.Ok().WithData(gin.H{
			"authUrl": authURL,
			"state":   state,
		})
	},
)

var googleCallbackRoute = serverutil.ApiRoute(
	"GET", "/auth/google/callback", func(c *gin.Context) *serverutil.Response {
		state := c.Query("state")
		code := c.Query("code")

		// Validate state token
		expiry, exists := oauthStateStore[state]
		if !exists || time.Now().After(expiry) {
			return serverutil.BadRequest(fmt.Errorf("invalid or expired state token"))
		}

		// Remove used state token
		delete(oauthStateStore, state)

		// Exchange code for token
		token, err := googleOAuthConfig.Exchange(context.Background(), code)
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("failed to exchange token: %w", err))
		}

		// Get user info
		client := googleOAuthConfig.Client(context.Background(), token)
		resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("failed to get user info: %w", err))
		}
		defer resp.Body.Close()

		var userInfo struct {
			Email string `json:"email"`
			Name  string `json:"name"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
			return serverutil.InternalServerError(fmt.Errorf("failed to decode user info: %w", err))
		}

		// Store token (in production, save to database associated with user)
		// For now, we'll return a simple HTML page that posts message to opener
		html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<title>Authorization Successful</title>
	<style>
		body {
			font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
			display: flex;
			align-items: center;
			justify-content: center;
			height: 100vh;
			margin: 0;
			background: #f3f4f6;
		}
		.container {
			text-align: center;
			background: white;
			padding: 2rem;
			border-radius: 8px;
			box-shadow: 0 2px 4px rgba(0,0,0,0.1);
		}
		.success { color: #22c55e; font-size: 48px; margin-bottom: 1rem; }
		h1 { font-size: 1.5rem; margin: 0 0 0.5rem 0; }
		p { color: #6b7280; }
	</style>
</head>
<body>
	<div class="container">
		<div class="success">✓</div>
		<h1>Authorization Successful</h1>
		<p>You can close this window and return to AutoButler.</p>
	</div>
	<script>
		if (window.opener) {
			window.opener.postMessage({
				type: 'google-auth-success',
				email: '%s',
				name: '%s',
				token: '%s'
			}, window.location.origin);
			setTimeout(() => window.close(), 2000);
		}
	</script>
</body>
</html>
`, userInfo.Email, userInfo.Name, token.AccessToken)

		c.Header("Content-Type", "text/html")
		c.String(200, html)
		return nil // Already handled response
	},
)

var googleDisconnectRoute = serverutil.ApiRoute(
	"POST", "/auth/google/disconnect", func(c *gin.Context) *serverutil.Response {
		// In production: Remove stored token from database
		// For now, just return success
		return serverutil.Ok().WithData(gin.H{
			"message": "Disconnected successfully",
		})
	},
)

func generateStateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func cleanExpiredStates() {
	now := time.Now()
	for state, expiry := range oauthStateStore {
		if now.After(expiry) {
			delete(oauthStateStore, state)
		}
	}
}
