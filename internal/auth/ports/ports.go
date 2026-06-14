package ports

import (
	"context"
	"time"

	"github.com/google/uuid"

	"finish-line/internal/auth/domain"
	userdomain "finish-line/internal/user/domain"
)

// UserFinder is the narrow view auth needs of the user module — only the
// lookup it actually uses, nothing more (interface segregation in practice).
type UserFinder interface {
	ByEmail(ctx context.Context, email string) (*userdomain.User, error)
}

// PasswordVerifier checks a plain password against a stored hash. Auth only
// verifies; it never hashes, so it does not depend on the full hasher.
type PasswordVerifier interface {
	Verify(plain, hash string) (bool, error)
}

// TokenService issues and parses stateless access tokens.
type TokenService interface {
	Issue(userID uuid.UUID) (token string, expiresAt time.Time, err error)
	Parse(token string) (domain.Claims, error)
}

// RefreshTokenRepository persists rotatable refresh tokens.
type RefreshTokenRepository interface {
	Create(ctx context.Context, t *domain.RefreshToken) error
	ByTokenHash(ctx context.Context, hash string) (*domain.RefreshToken, error)
	MarkRotated(ctx context.Context, id uuid.UUID) error
	RevokeFamily(ctx context.Context, familyID uuid.UUID) error
}
