package v0_auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/autobutler-org/autobutler/pkg/util/serverutil"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	googleOAuthConfig *oauth2.Config
	oauthStateStore   = make(map[string]time.Time)     // state -> timestamp
	tokenStore        = make(map[string]*oauth2.Token) // email -> token
)

func init() {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	redirectURL := os.Getenv("GOOGLE_REDIRECT_URL")

	if clientID == "" || clientSecret == "" || redirectURL == "" {
		log.Println("[oauth] WARNING: Google OAuth environment variables not set")
		log.Println("[oauth] Required: GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET, GOOGLE_REDIRECT_URL")
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

	log.Printf("[oauth] Google OAuth configured with redirect URL: %s", redirectURL)
}

// googleAuthorize godoc
// @Summary Get Google authorization URL
// @Description Returns an authorization URL and state token for starting OAuth flow
// @Tags auth
// @Produce json
// @Success 200 {object} object
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /auth/google/authorize [get]
func googleAuthorize(c *gin.Context) *serverutil.Response {
	// Validate OAuth config is set up
	if googleOAuthConfig.ClientID == "" || googleOAuthConfig.ClientSecret == "" {
		return serverutil.InternalServerError(fmt.Errorf("Google OAuth not configured. Please set GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET, and GOOGLE_REDIRECT_URL environment variables"))
	}

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
}

var googleAuthorizeRoute = serverutil.ApiRoute(
	"GET", "/auth/google/authorize", googleAuthorize,
)

// googleCallback godoc
// @Summary Google OAuth callback
// @Description Callback endpoint used by Google OAuth. Exchanges code for token and returns HTML.
// @Tags auth
// @Param state query string true "OAuth state token"
// @Param code query string true "Authorization code"
// @Success 200 {string} string "HTML page"
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /auth/google/callback [get]
func googleCallback(c *gin.Context) *serverutil.Response {
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

	// Store token for later use (in production, save to database)
	tokenStore[userInfo.Email] = token

	// Store token (in production, save to database associated with user)
	// For now, we'll return a simple HTML page that posts message to opener
	html := fmt.Sprintf(`<!DOCTYPE html>
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
		<p>Redirecting to AutoButler...</p>
	</div>
	<script>
		console.log('OAuth callback page loaded');
		console.log('Email: %s, Name: %s');

		// Store in localStorage on port 8080
		try {
			localStorage.setItem('google_auth_email', '%s');
			localStorage.setItem('google_auth_token', '%s');
			localStorage.setItem('google_auth_name', '%s');
			localStorage.setItem('google_auth_timestamp', Date.now().toString());
			console.log('Stored credentials in localStorage (port 8080)');
		} catch (e) {
			console.error('Failed to store in localStorage:', e);
		}

		// Redirect to frontend with credentials in URL hash (will be parsed by frontend)
		console.log('Redirecting to frontend...');
		setTimeout(() => {
			const params = new URLSearchParams({
				email: '%s',
				name: '%s',
				token: '%s'
			});
			window.location.href = 'http://localhost:5173/migrationservice#' + params.toString();
		}, 1000);
	</script>
</body>
</html>`,
		userInfo.Email, userInfo.Name,
		userInfo.Email, token.AccessToken, userInfo.Name,
		userInfo.Email, userInfo.Name, token.AccessToken)

	// Return HTML response
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(200, html)
	return nil
}

var googleCallbackRoute = serverutil.ApiRoute(
	"GET", "/auth/google/callback", googleCallback,
)

// googleDisconnect godoc
// @Summary Disconnect Google account
// @Description Removes stored Google OAuth token for a user (or all tokens if no email provided)
// @Tags auth
// @Accept json
// @Produce json
// @Param body body object false "{email: string}"
// @Success 200 {object} object
// @Router /auth/google/disconnect [post]
func googleDisconnect(c *gin.Context) *serverutil.Response {
	// Get email from request body
	var req struct {
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Email == "" {
		// If no email provided, clear all tokens (for simplicity)
		tokenStore = make(map[string]*oauth2.Token)
	} else {
		// Remove specific user's token
		delete(tokenStore, req.Email)
	}

	return serverutil.Ok().WithData(gin.H{
		"message": "Disconnected successfully",
	})
}

var googleDisconnectRoute = serverutil.ApiRoute(
	"POST", "/auth/google/disconnect", googleDisconnect,
)

// GetGoogleToken retrieves a stored OAuth token by email
func GetGoogleToken(email string) (*oauth2.Token, bool) {
	token, exists := tokenStore[email]
	return token, exists
}

// GetGoogleOAuthConfig returns the configured OAuth config
func GetGoogleOAuthConfig() *oauth2.Config {
	return googleOAuthConfig
}

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
