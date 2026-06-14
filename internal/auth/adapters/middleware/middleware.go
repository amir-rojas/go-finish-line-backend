// Package middleware guards routes by validating the access token.
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"finish-line/internal/common/httpx"
)

// Authenticator is the consumer-side contract the middleware needs: turn an
// access token into the user it identifies.
type Authenticator interface {
	Authenticate(accessToken string) (uuid.UUID, error)
}

// RequireAuth rejects any request without a valid Bearer access token.
func RequireAuth(auth Authenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := bearerToken(c.GetHeader("Authorization"))
		if !ok {
			httpx.Unauthorized(c, "missing or malformed authorization header")
			c.Abort()
			return
		}

		userID, err := auth.Authenticate(token)
		if err != nil {
			httpx.RespondError(c, err)
			c.Abort()
			return
		}

		httpx.SetUserID(c, userID)
		c.Next()
	}
}

func bearerToken(header string) (string, bool) {
	const prefix = "Bearer "
	if len(header) < len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", false
	}
	return header[len(prefix):], true
}
