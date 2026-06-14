package domain

import "finish-line/internal/apperr"

var (
	ErrNameRequired         = apperr.New(apperr.KindValidation, "name is required")
	ErrEmailInvalid         = apperr.New(apperr.KindValidation, "email is invalid")
	ErrPasswordTooShort     = apperr.New(apperr.KindValidation, "password must be at least 8 characters")
	ErrPasswordTooLong      = apperr.New(apperr.KindValidation, "password must be at most 72 characters")
	ErrPasswordHashRequired = apperr.New(apperr.KindValidation, "password hash is required")
	ErrIncorrectPassword    = apperr.New(apperr.KindUnauthorized, "current password is incorrect")
	ErrEmailTaken           = apperr.New(apperr.KindConflict, "email is already registered")
	ErrNotFound             = apperr.New(apperr.KindNotFound, "user not found")
)
