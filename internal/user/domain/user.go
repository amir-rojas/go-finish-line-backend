package domain

import (
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID
	Name         string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

// New builds a valid User or reports why it can't. It is the only way to
// construct a User, so an instance that exists is an instance that is valid.
func New(name, email, passwordHash string) (*User, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrNameRequired
	}

	normalized, err := NormalizeEmail(email)
	if err != nil {
		return nil, err
	}

	if passwordHash == "" {
		return nil, ErrPasswordHashRequired
	}

	return &User{
		ID:           uuid.New(),
		Name:         name,
		Email:        normalized,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now().UTC(),
	}, nil
}

// NormalizeEmail lowercases, trims and validates an email address so the
// same identity always compares equal regardless of how it was typed.
func NormalizeEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if _, err := mail.ParseAddress(email); err != nil {
		return "", ErrEmailInvalid
	}
	return email, nil
}

// ValidatePassword enforces the plain-text password policy. It lives in the
// domain because the policy is a business rule, even though hashing is not.
// The 72-byte cap matches bcrypt's input limit: anything longer would be
// silently truncated or rejected by the hasher.
func ValidatePassword(plain string) error {
	switch {
	case len(plain) < 8:
		return ErrPasswordTooShort
	case len(plain) > 72:
		return ErrPasswordTooLong
	}
	return nil
}
