package domain

import "fmt"

// ErrorCode is a stable machine-readable error code returned to clients.
type ErrorCode string

const (
	// CodeNotFound means the requested resource does not exist.
	CodeNotFound ErrorCode = "not_found"
	// CodeInvalidInput means the request payload failed validation.
	CodeInvalidInput ErrorCode = "invalid_input"
	// CodeConflict means the request violates an invariant of the domain
	// (for example an illegal state-machine transition).
	CodeConflict ErrorCode = "conflict"
	// CodeInternal is used for unexpected failures.
	CodeInternal ErrorCode = "internal"
	// CodeUnauthorized is reserved for future operator authentication.
	CodeUnauthorized ErrorCode = "unauthorized"
)

// Error is a domain error carrying a stable code, a human-readable message
// and the underlying cause (if any).
type Error struct {
	Code    ErrorCode
	Message string
	Cause   error
}

// Error implements the error interface.
func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the wrapped cause so errors.Is/As work.
func (e *Error) Unwrap() error { return e.Cause }

// NewError builds a domain error.
func NewError(code ErrorCode, message string) *Error {
	return &Error{Code: code, Message: message}
}

// WrapError builds a domain error from an underlying cause.
func WrapError(code ErrorCode, message string, cause error) *Error {
	return &Error{Code: code, Message: message, Cause: cause}
}

// NotFound returns a not-found domain error.
func NotFound(what, id string) *Error {
	return NewError(CodeNotFound, fmt.Sprintf("%s %q not found", what, id))
}

// InvalidInput returns an invalid-input domain error.
func InvalidInput(message string) *Error {
	return NewError(CodeInvalidInput, message)
}

// Conflict returns a conflict domain error.
func Conflict(message string) *Error {
	return NewError(CodeConflict, message)
}

// AsDomainError unwraps a domain *Error from any error chain, returning nil
// when the error is not domain-typed.
func AsDomainError(err error) *Error {
	if de, ok := err.(*Error); ok {
		return de
	}
	u, ok := err.(interface{ Unwrap() error })
	if !ok {
		return nil
	}
	if de, ok := u.Unwrap().(*Error); ok {
		return de
	}
	return nil
}
