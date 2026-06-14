// Package apperr is the shared domain kernel for categorized errors.
//
// Domain errors declare WHAT went wrong and WHICH category they belong to,
// without knowing anything about HTTP. The transport layer maps categories
// to status codes. This keeps the enumeration of "which errors are 400s" out
// of every handler.
package apperr

import "errors"

type Kind int

const (
	KindValidation   Kind = iota // bad input — maps to 400
	KindUnauthorized              // bad credentials or token — maps to 401
	KindConflict                  // state collision — maps to 409
	KindNotFound                  // missing resource — maps to 404
)

// Error is a domain error tagged with a category. Its message is clean and
// safe to surface to clients; wrapping context added with %w never leaks
// because callers read Error() on the matched *Error node, not the chain.
type Error struct {
	Kind Kind
	msg  string
}

func New(kind Kind, msg string) *Error {
	return &Error{Kind: kind, msg: msg}
}

func (e *Error) Error() string { return e.msg }

// KindOf reports the category of the first *Error in err's chain.
func KindOf(err error) (Kind, bool) {
	var e *Error
	if errors.As(err, &e) {
		return e.Kind, true
	}
	return 0, false
}
