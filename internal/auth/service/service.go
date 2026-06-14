package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"finish-line/internal/auth/domain"
	"finish-line/internal/auth/ports"
	userdomain "finish-line/internal/user/domain"
)

type Service struct {
	users      ports.UserFinder
	verifier   ports.PasswordVerifier
	tokens     ports.TokenService
	refresh    ports.RefreshTokenRepository
	refreshTTL time.Duration
}

func New(
	users ports.UserFinder,
	verifier ports.PasswordVerifier,
	tokens ports.TokenService,
	refresh ports.RefreshTokenRepository,
	refreshTTL time.Duration,
) *Service {
	return &Service{
		users:      users,
		verifier:   verifier,
		tokens:     tokens,
		refresh:    refresh,
		refreshTTL: refreshTTL,
	}
}

// TokenPair is what a successful login or refresh hands back: a short-lived
// access token (carried as a Bearer header) and a long-lived refresh token
// (carried in an httpOnly cookie).
type TokenPair struct {
	AccessToken     string
	AccessExpiresAt time.Time
	RefreshToken    string
}

// Login verifies credentials and starts a new token family. A missing user
// and a wrong password fail identically, so the response never reveals which
// emails are registered.
func (s *Service) Login(ctx context.Context, email, password string) (*TokenPair, error) {
	u, err := s.users.ByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, userdomain.ErrNotFound) {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("finding user: %w", err)
	}

	ok, err := s.verifier.Verify(password, u.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("verifying password: %w", err)
	}
	if !ok {
		return nil, domain.ErrInvalidCredentials
	}

	return s.issueTokenPair(ctx, u.ID, uuid.New())
}

// Refresh rotates a refresh token: it consumes the presented one and issues a
// fresh pair in the same family. Presenting an already-used token means it was
// likely stolen, so the whole family is revoked.
func (s *Service) Refresh(ctx context.Context, rawRefresh string) (*TokenPair, error) {
	rt, err := s.refresh.ByTokenHash(ctx, domain.HashToken(rawRefresh))
	if err != nil {
		if errors.Is(err, domain.ErrRefreshTokenNotFound) {
			return nil, domain.ErrInvalidToken
		}
		return nil, fmt.Errorf("looking up refresh token: %w", err)
	}

	if rt.IsUsed() {
		if err := s.refresh.RevokeFamily(ctx, rt.FamilyID); err != nil {
			return nil, fmt.Errorf("revoking token family: %w", err)
		}
		return nil, domain.ErrTokenReuse
	}

	if rt.IsExpired(time.Now()) {
		return nil, domain.ErrInvalidToken
	}

	if err := s.refresh.MarkRotated(ctx, rt.ID); err != nil {
		return nil, fmt.Errorf("rotating refresh token: %w", err)
	}

	return s.issueTokenPair(ctx, rt.UserID, rt.FamilyID)
}

// Authenticate validates an access token and returns who it belongs to. Any
// failure is flattened to ErrInvalidToken — the middleware only needs to know
// the caller is not authenticated.
func (s *Service) Authenticate(accessToken string) (uuid.UUID, error) {
	claims, err := s.tokens.Parse(accessToken)
	if err != nil {
		return uuid.Nil, domain.ErrInvalidToken
	}
	return claims.UserID, nil
}

func (s *Service) issueTokenPair(ctx context.Context, userID, familyID uuid.UUID) (*TokenPair, error) {
	access, accessExp, err := s.tokens.Issue(userID)
	if err != nil {
		return nil, fmt.Errorf("issuing access token: %w", err)
	}

	raw, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("generating refresh token: %w", err)
	}

	now := time.Now()
	rt := &domain.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		FamilyID:  familyID,
		TokenHash: domain.HashToken(raw),
		ExpiresAt: now.Add(s.refreshTTL),
		CreatedAt: now,
	}
	if err := s.refresh.Create(ctx, rt); err != nil {
		return nil, fmt.Errorf("storing refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:     access,
		AccessExpiresAt: accessExp,
		RefreshToken:    raw,
	}, nil
}

// generateToken returns a 256-bit, URL-safe random string for use as the raw
// refresh token value.
func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("reading random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
