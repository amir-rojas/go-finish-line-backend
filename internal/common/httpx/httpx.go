// Package httpx holds shared HTTP response helpers so every module's
// adapter maps domain errors to status codes the same way.
package httpx

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"finish-line/internal/apperr"
)

type errorResponse struct {
	Error string `json:"error"`
}

// contextUserIDKey is where the auth middleware stores the authenticated
// user's ID. It lives here, in a neutral shared package, so a module's
// handler can read it without importing the auth module (which would invert
// the dependency direction).
const contextUserIDKey = "userID"

// SetUserID records the authenticated user on the request context.
func SetUserID(c *gin.Context, id uuid.UUID) {
	c.Set(contextUserIDKey, id)
}

// UserID returns the authenticated user's ID, or false if the request was
// not authenticated.
func UserID(c *gin.Context) (uuid.UUID, bool) {
	v, ok := c.Get(contextUserIDKey)
	if !ok {
		return uuid.Nil, false
	}
	id, ok := v.(uuid.UUID)
	return id, ok
}

// RespondError maps a domain error to its HTTP status by category. Errors
// without a known category are logged and masked as 500 — internal details
// never reach the client.
func RespondError(c *gin.Context, err error) {
	var appErr *apperr.Error
	if errors.As(err, &appErr) {
		c.JSON(statusFor(appErr.Kind), errorResponse{Error: appErr.Error()})
		return
	}

	slog.Error("unhandled error",
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
		"error", err,
	)
	c.JSON(http.StatusInternalServerError, errorResponse{Error: "internal server error"})
}

// BadRequest reports a transport-level problem (malformed body, bad path
// param) that never reaches the domain.
func BadRequest(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, errorResponse{Error: msg})
}

// Unauthorized reports a transport-level missing or malformed credential
// before the domain is reached.
func Unauthorized(c *gin.Context, msg string) {
	c.JSON(http.StatusUnauthorized, errorResponse{Error: msg})
}

// TooManyRequests reports that the caller has been rate limited.
func TooManyRequests(c *gin.Context, msg string) {
	c.JSON(http.StatusTooManyRequests, errorResponse{Error: msg})
}

func statusFor(kind apperr.Kind) int {
	switch kind {
	case apperr.KindValidation:
		return http.StatusBadRequest
	case apperr.KindUnauthorized:
		return http.StatusUnauthorized
	case apperr.KindConflict:
		return http.StatusConflict
	case apperr.KindNotFound:
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}
