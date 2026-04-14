package serverutil

import (
	"os"
	"strconv"
)

// ServerPort returns the HTTP port the server listens on, read from the PORT
// environment variable. Defaults to 8080.
func ServerPort() int {
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
