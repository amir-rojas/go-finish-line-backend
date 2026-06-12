package ports

import (
	"context"

	"github.com/google/uuid"

	"finish-line/internal/user/domain"
)

// UserRepository is the driven port for user persistence. Implementations
// must translate storage errors into domain errors: unique email violations
// become domain.ErrEmailTaken and missing rows become domain.ErrNotFound.
type UserRepository interface {
	Create(ctx context.Context, u *domain.User) error
	ByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	ByEmail(ctx context.Context, email string) (*domain.User, error)
	List(ctx context.Context) ([]domain.User, error)
}

// PasswordHasher abstracts the hashing algorithm so the service never
// depends on a concrete crypto implementation.
type PasswordHasher interface {
	Hash(plain string) (string, error)
	Verify(plain, hash string) (bool, error)
}
