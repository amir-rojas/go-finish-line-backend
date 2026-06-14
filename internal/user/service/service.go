package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"finish-line/internal/user/domain"
	"finish-line/internal/user/ports"
)

type Service struct {
	repo     ports.UserRepository
	hasher   ports.PasswordHasher
	sessions ports.SessionRevoker
}

func New(repo ports.UserRepository, hasher ports.PasswordHasher, sessions ports.SessionRevoker) *Service {
	return &Service{repo: repo, hasher: hasher, sessions: sessions}
}

// Register creates a new admin user with a hashed password.
func (s *Service) Register(ctx context.Context, name, email, password string) (*domain.User, error) {
	if err := domain.ValidatePassword(password); err != nil {
		return nil, err
	}

	hash, err := s.hasher.Hash(password)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	u, err := domain.New(name, email, hash)
	if err != nil {
		return nil, err
	}

	if err := s.repo.Create(ctx, u); err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}

	return u, nil
}

// EnsureAdmin creates the given admin if it does not already exist. It is
// idempotent, so it is safe to call on every dev startup as a bootstrap seed.
func (s *Service) EnsureAdmin(ctx context.Context, name, email, password string) error {
	_, err := s.Register(ctx, name, email, password)
	if err == nil || errors.Is(err, domain.ErrEmailTaken) {
		return nil
	}
	return fmt.Errorf("ensuring admin user: %w", err)
}

func (s *Service) ByEmail(ctx context.Context, email string) (*domain.User, error) {
	normalized, err := domain.NormalizeEmail(email)
	if err != nil {
		return nil, err
	}

	u, err := s.repo.ByEmail(ctx, normalized)
	if err != nil {
		return nil, fmt.Errorf("getting user by email: %w", err)
	}
	return u, nil
}

// ChangePassword updates a user's password after verifying the current one.
// Requiring the current password means a leaked access token alone is not
// enough to take over the account's credential.
func (s *Service) ChangePassword(ctx context.Context, id uuid.UUID, currentPassword, newPassword string) error {
	u, err := s.repo.ByID(ctx, id)
	if err != nil {
		return fmt.Errorf("getting user: %w", err)
	}

	ok, err := s.hasher.Verify(currentPassword, u.PasswordHash)
	if err != nil {
		return fmt.Errorf("verifying current password: %w", err)
	}
	if !ok {
		return domain.ErrIncorrectPassword
	}

	if err := domain.ValidatePassword(newPassword); err != nil {
		return err
	}

	hash, err := s.hasher.Hash(newPassword)
	if err != nil {
		return fmt.Errorf("hashing new password: %w", err)
	}

	if err := s.repo.UpdatePassword(ctx, id, hash); err != nil {
		return fmt.Errorf("updating password: %w", err)
	}

	// A changed password ends every session, including the one that made this
	// request — the user must log in again everywhere with the new password.
	if err := s.sessions.RevokeAllSessions(ctx, id); err != nil {
		return fmt.Errorf("revoking sessions after password change: %w", err)
	}
	return nil
}

func (s *Service) ByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	u, err := s.repo.ByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting user by id: %w", err)
	}
	return u, nil
}

func (s *Service) List(ctx context.Context) ([]domain.User, error) {
	users, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}
	return users, nil
}
