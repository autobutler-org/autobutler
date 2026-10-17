// Package frame serves /frame: a small slideshow page for a dedicated screen such as a photo frame (#3021).
//
// The page is plain HTML, CSS and JavaScript with no framework, so it runs on hardware too weak for the Flutter web
// client. It signs in with the same account as the app, shows one of the signed-in account's albums, and talks only to
// the existing /api/v0 routes: nothing here reads the database or opens a file. /frame and its assets need no session,
// because the page is where the sign-in happens; everything it shows comes from routes that do.
package frame

import "github.com/gin-gonic/gin"

// Register mounts /frame and /frame/<asset> on the engine. Call it before the single-page-app fallback, which would
// otherwise answer them with the Flutter app.
func Register(engine *gin.Engine) {
	engine.GET("/frame", serveAsset)
	engine.GET("/frame/*asset", serveAsset)
}
