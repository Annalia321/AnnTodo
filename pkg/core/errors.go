package core

import (
	"errors"
	"net/http"
)

// Error is a business error that maps to an HTTP status and a stable code the
// frontend can translate.
type Error struct {
	Status  int
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Message }

func newErr(status int, code, msg string) *Error {
	return &Error{Status: status, Code: code, Message: msg}
}

// BadRequest builds a validation error with a custom message.
func BadRequest(msg string) *Error { return newErr(http.StatusBadRequest, "invalid_request", msg) }

var (
	ErrNotFound           = newErr(http.StatusNotFound, "not_found", "resource not found")
	ErrUnauthorized       = newErr(http.StatusUnauthorized, "unauthorized", "login required")
	ErrForbidden          = newErr(http.StatusForbidden, "forbidden", "permission denied")
	ErrInvalidCredentials = newErr(http.StatusUnauthorized, "invalid_credentials", "invalid username or password")
	ErrUserDisabled       = newErr(http.StatusForbidden, "user_disabled", "this account has been disabled")
	ErrUsernameTaken      = newErr(http.StatusConflict, "username_taken", "username already exists")
	ErrEmailTaken         = newErr(http.StatusConflict, "email_taken", "email already in use")
	ErrAlreadyInitialized = newErr(http.StatusConflict, "already_initialized", "system already initialized")
	ErrNotInitialized     = newErr(http.StatusConflict, "not_initialized", "system not initialized")
	ErrRegistrationClosed = newErr(http.StatusForbidden, "registration_closed", "registration is disabled")
	ErrWrongPassword      = newErr(http.StatusBadRequest, "wrong_password", "current password is incorrect")
	ErrCannotModifySelf   = newErr(http.StatusBadRequest, "cannot_modify_self", "you cannot change your own role, status or account here")
	ErrInvalidImport      = newErr(http.StatusBadRequest, "invalid_import", "invalid import file")
	ErrInvalidUsername    = newErr(http.StatusBadRequest, "invalid_username", "username must be 3-32 letters, digits, '_', '-' or '.'")
	ErrInvalidPassword    = newErr(http.StatusBadRequest, "invalid_password", "password must be 8-72 characters")
	ErrInvalidEmail       = newErr(http.StatusBadRequest, "invalid_email", "invalid email address")
	ErrInvalidAvatar      = newErr(http.StatusBadRequest, "invalid_avatar", "avatar must be a PNG, JPEG, GIF or WebP image under 2 MB")
)

// AsError extracts a *Error from err.
func AsError(err error) (*Error, bool) {
	var e *Error
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}
