package domain

import (
	"errors"

	"finish-line/internal/apperr"
)

// Client-facing errors. Credentials and token failures all surface as 401
// and stay deliberately vague — never reveal whether an email exists or why
// exactly a token failed.
var (
	ErrInvalidCredentials = apperr.New(apperr.KindUnauthorized, "invalid credentials")
	ErrInvalidToken       = apperr.New(apperr.KindUnauthorized, "invalid or expired token")
	ErrTokenReuse         = apperr.New(apperr.KindUnauthorized, "token reuse detected")
)

// ErrRefreshTokenNotFound is an internal signal from the repository, always
// translated by the service into ErrInvalidToken before reaching a client.
var ErrRefreshTokenNotFound = errors.New("refresh token not found")
