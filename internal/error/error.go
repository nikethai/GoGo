package error

import "errors"

var (
	// Common errors
	ErrInvalidRequestFormat = errors.New("invalid request format")
	ErrInternalServer       = errors.New("internal server error")
	ErrInvalidCredentials   = errors.New("invalid username or password")
	ErrUserNotFound         = errors.New("user not found")

	// Registration errors
	ErrRegistrationFailed         = errors.New("registration failed")
	ErrMissingRegistrationDetails = errors.New("username, password, and email are required")
	ErrDuplicateUsername          = errors.New("username already exists")
	ErrDuplicateEmail             = errors.New("email already exists")
	ErrDuplicateUsernameOrEmail   = errors.New("username or email already exists")

	// Profile errors
	ErrProfileRetrieval = errors.New("error retrieving user profile")
)
