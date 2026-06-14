package domain

import (
	"time"

	"github.com/google/uuid"
)

// Claims is the identity an access token carries: who the caller is and the
// window during which the token is valid. It is the domain view of a token's
// payload, independent of JWT or any encoding.
type Claims struct {
	UserID    uuid.UUID
	IssuedAt  time.Time
	ExpiresAt time.Time
}
