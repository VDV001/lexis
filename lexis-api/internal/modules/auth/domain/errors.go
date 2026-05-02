package domain

import "errors"

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailTaken         = errors.New("email already taken")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrTokenExpired       = errors.New("token expired")
	ErrTokenRevoked       = errors.New("token revoked")
	ErrTokenNotFound      = errors.New("token not found")
	ErrAvatarURLTooLong   = errors.New("avatar_url must be at most 2048 characters")
	ErrInvalidDisplayName   = errors.New("invalid display name")
	ErrDisplayNameRequired  = errors.New("display name is required")
	ErrDisplayNameTooLong   = errors.New("display name must be at most 100 characters")
	ErrInvalidEmail         = errors.New("invalid email")
	ErrInvalidPassword      = errors.New("invalid password")
	ErrInvalidSettings      = errors.New("invalid settings")
)
