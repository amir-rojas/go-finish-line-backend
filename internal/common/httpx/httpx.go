// Package httpx holds shared HTTP response helpers so every module's
// adapter maps domain errors to status codes the same way.
package httpx

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"finish-line/internal/apperr"
)

type errorResponse struct {
	Error string `json:"error"`
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

func statusFor(kind apperr.Kind) int {
	switch kind {
	case apperr.KindValidation:
		return http.StatusBadRequest
	case apperr.KindConflict:
		return http.StatusConflict
	case apperr.KindNotFound:
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}
