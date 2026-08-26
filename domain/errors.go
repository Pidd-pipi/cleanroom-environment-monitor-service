package domain

import (
	"errors"
	"fmt"
)

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

// Error implements the error interface. When the domain error wraps an
// underlying cause the cause is appended so logs expose the real failure
// (e.g. "internal: flush interlock log: store: fsync tmp: ...") instead of
// only the high-level message.
func (e *Error) Error() string {
	msg := fmt.Sprintf("%s: %s", e.Code, e.Message)
	if e.Cause != nil {
		return msg + ": " + e.Cause.Error()
	}
	return msg
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

// Internal returns an internal domain error wrapping an unexpected cause.
// The cause is preserved so the request log surfaces the real failure.
func Internal(message string, cause error) *Error {
	return WrapError(CodeInternal, message, cause)
}

// AsDomainError unwraps a domain *Error from anywhere in an error chain
// (including multi-level wraps such as service -> store -> domain), returning
// nil when the error is not domain-typed. It walks the full chain via
// errors.As so wrapping a domain error in one or more fmt.Errorf("...: %w")
// layers never erases its code/message.
func AsDomainError(err error) *Error {
	var de *Error
	if errors.As(err, &de) {
		return de
	}
	return nil
}
