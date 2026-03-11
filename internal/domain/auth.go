package domain

import "errors"

var (
	ErrUserNotFound        = errors.New("user not found")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrUserBlocked         = errors.New("user is temporarily blocked")
	ErrIPBlocked           = errors.New("ip is temporarily blocked")
	ErrInvalidToken        = errors.New("invalid token")
	ErrTokenExpired        = errors.New("token expired")
	ErrPasswordPolicy      = errors.New("password does not meet policy requirements")
	ErrPasswordReuse       = errors.New("password was used recently")
	ErrRefreshTokenRevoked = errors.New("refresh token revoked")
)

