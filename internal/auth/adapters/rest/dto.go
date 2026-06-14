package rest

import (
	"time"

	"finish-line/internal/auth/service"
)

type loginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// tokenResponse carries only the access token. The refresh token never
// appears in the body — it lives exclusively in the httpOnly cookie.
type tokenResponse struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	ExpiresAt   time.Time `json:"expires_at"`
}

func toTokenResponse(p *service.TokenPair) tokenResponse {
	return tokenResponse{
		AccessToken: p.AccessToken,
		TokenType:   "Bearer",
		ExpiresAt:   p.AccessExpiresAt,
	}
}
