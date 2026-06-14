package rest

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"finish-line/internal/auth/service"
	"finish-line/internal/common/httpx"
)

const refreshCookieName = "refresh_token"

// refreshCookiePath scopes the refresh cookie so the browser only ever sends
// it to the auth endpoints, not to every API call.
const refreshCookiePath = "/api/v1/auth"

// AuthService is the consumer-side contract this adapter needs from the auth
// application service.
type AuthService interface {
	Login(ctx context.Context, email, password string) (*service.TokenPair, error)
	Refresh(ctx context.Context, rawRefresh string) (*service.TokenPair, error)
	Logout(ctx context.Context, rawRefresh string) error
}

type Handler struct {
	svc          AuthService
	refreshTTL   time.Duration
	secureCookie bool
	loginLimiter gin.HandlerFunc
}

func NewHandler(svc AuthService, refreshTTL time.Duration, secureCookie bool, loginLimiter gin.HandlerFunc) *Handler {
	return &Handler{svc: svc, refreshTTL: refreshTTL, secureCookie: secureCookie, loginLimiter: loginLimiter}
}

func (h *Handler) RegisterRoutes(r gin.IRouter) {
	auth := r.Group("/auth")
	auth.POST("/login", h.loginLimiter, h.login)
	auth.POST("/refresh", h.refresh)
	auth.POST("/logout", h.logout)
}

func (h *Handler) login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid request body")
		return
	}

	pair, err := h.svc.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		httpx.RespondError(c, err)
		return
	}

	h.setRefreshCookie(c, pair.RefreshToken)
	c.JSON(http.StatusOK, toTokenResponse(pair))
}

func (h *Handler) refresh(c *gin.Context) {
	raw, err := c.Cookie(refreshCookieName)
	if err != nil {
		httpx.Unauthorized(c, "missing refresh token")
		return
	}

	pair, err := h.svc.Refresh(c.Request.Context(), raw)
	if err != nil {
		// A failed refresh (expired, reused, invalid) means the cookie is
		// worthless — clear it so the client is forced to log in again.
		h.clearRefreshCookie(c)
		httpx.RespondError(c, err)
		return
	}

	h.setRefreshCookie(c, pair.RefreshToken)
	c.JSON(http.StatusOK, toTokenResponse(pair))
}

func (h *Handler) logout(c *gin.Context) {
	// Logout relies on the refresh cookie, not the access token, so it works
	// even after the access token has expired.
	if raw, err := c.Cookie(refreshCookieName); err == nil {
		if err := h.svc.Logout(c.Request.Context(), raw); err != nil {
			httpx.RespondError(c, err)
			return
		}
	}
	h.clearRefreshCookie(c)
	c.Status(http.StatusNoContent)
}

func (h *Handler) setRefreshCookie(c *gin.Context, token string) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(refreshCookieName, token, int(h.refreshTTL.Seconds()), refreshCookiePath, "", h.secureCookie, true)
}

func (h *Handler) clearRefreshCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(refreshCookieName, "", -1, refreshCookiePath, "", h.secureCookie, true)
}
