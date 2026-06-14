package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"finish-line/internal/auth/domain"
	"finish-line/internal/auth/ports"
)

// Service signs and verifies stateless access tokens with HS256. It is the
// concrete implementation of the TokenService port; the core only ever sees
// the interface.
type Service struct {
	secret []byte
	ttl    time.Duration
}

var _ ports.TokenService = (*Service)(nil)

func New(secret string, ttl time.Duration) *Service {
	return &Service{secret: []byte(secret), ttl: ttl}
}

func (s *Service) Issue(userID uuid.UUID) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(s.ttl)

	claims := jwt.RegisteredClaims{
		Subject:   userID.String(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("signing token: %w", err)
	}
	return signed, expiresAt, nil
}

func (s *Service) Parse(raw string) (domain.Claims, error) {
	var claims jwt.RegisteredClaims
	token, err := jwt.ParseWithClaims(raw, &claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		return domain.Claims{}, fmt.Errorf("parsing token: %w", err)
	}
	if !token.Valid {
		return domain.Claims{}, errors.New("token is not valid")
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return domain.Claims{}, fmt.Errorf("invalid subject: %w", err)
	}

	c := domain.Claims{UserID: userID}
	if claims.IssuedAt != nil {
		c.IssuedAt = claims.IssuedAt.Time
	}
	if claims.ExpiresAt != nil {
		c.ExpiresAt = claims.ExpiresAt.Time
	}
	return c, nil
}
