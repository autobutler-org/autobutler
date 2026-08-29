package v0_vault

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"

	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
)

const (
	lowerChars     = "abcdefghijklmnopqrstuvwxyz"
	upperChars     = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digitChars     = "0123456789"
	symbolChars    = "!@#$%^&*()-_=+[]{}|;:,.<>?"
	ambiguousChars = "0O1lI"
)

var generateVaultPasswordRoute = serverutil.ApiRoute(
	"POST", "/vault/generate", func(c *gin.Context) *serverutil.Response {
		var req generateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			// Defaults if no body provided.
			req = generateRequest{
				Length:    20,
				Uppercase: true,
				Lowercase: true,
				Digits:    true,
				Symbols:   true,
			}
		}

		if req.Length < 8 {
			req.Length = 8
		}
		if req.Length > 128 {
			req.Length = 128
		}
		if !req.Uppercase && !req.Lowercase && !req.Digits && !req.Symbols {
			req.Lowercase = true
			req.Digits = true
		}

		var charset strings.Builder
		if req.Lowercase {
			charset.WriteString(lowerChars)
		}
		if req.Uppercase {
			charset.WriteString(upperChars)
		}
		if req.Digits {
			charset.WriteString(digitChars)
		}
		if req.Symbols {
			charset.WriteString(symbolChars)
		}

		pool := charset.String()
		if req.AvoidAmbiguous {
			var filtered strings.Builder
			for _, ch := range pool {
				if !strings.ContainsRune(ambiguousChars, ch) {
					filtered.WriteRune(ch)
				}
			}
			pool = filtered.String()
		}

		if len(pool) == 0 {
			return serverutil.BadRequest(fmt.Errorf("no characters available with current settings"))
		}

		password, err := secureRandomString(pool, req.Length)
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("generate password: %w", err))
		}

		return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(gin.H{
			"password": password,
		})
	},
)

func secureRandomString(charset string, length int) (string, error) {
	if length < 1 || length > 128 {
		return "", fmt.Errorf("length must be between 1 and 128")
	}
	max := big.NewInt(int64(len(charset)))
	result := make([]byte, length)
	for i := range result {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		result[i] = charset[n.Int64()]
	}
	return string(result), nil
}
