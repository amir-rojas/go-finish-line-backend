package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
)

// RefreshToken is a stored, rotatable credential. The raw token value is only
// ever held by the client; the database keeps its hash. Tokens are grouped
// into a family (one login session) so reuse of a rotated token can invalidate
// every sibling at once.
type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	FamilyID  uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	RotatedAt *time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

// IsUsed reports whether the token has already been rotated or revoked.
// Presenting a used token is the signal of a stolen-token replay.
func (t *RefreshToken) IsUsed() bool {
	return t.RotatedAt != nil || t.RevokedAt != nil
}

func (t *RefreshToken) IsExpired(now time.Time) bool {
	return now.After(t.ExpiresAt)
}

func (t *RefreshToken) IsActive(now time.Time) bool {
	return !t.IsUsed() && !t.IsExpired(now)
}

// HashToken hashes a raw refresh token for storage and lookup. SHA-256 is
// the right tool here: the token is high-entropy random data, so a fast hash
// is safe — unlike a password, which needs a deliberately slow hash.
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
