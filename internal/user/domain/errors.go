package domain

import "errors"

var (
	ErrNameRequired         = errors.New("name is required")
	ErrEmailInvalid         = errors.New("email is invalid")
	ErrPasswordTooShort     = errors.New("password must be at least 8 characters")
	ErrPasswordTooLong      = errors.New("password must be at most 72 characters")
	ErrPasswordHashRequired = errors.New("password hash is required")
	ErrEmailTaken           = errors.New("email is already registered")
	ErrNotFound             = errors.New("user not found")
)
