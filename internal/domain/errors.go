package domain

import "errors"

var (
	ErrNotFound       = errors.New("not found")
	ErrAccessDenied   = errors.New("access denied")
	ErrInvalidInput   = errors.New("invalid input")
	ErrConflict       = errors.New("conflict")
	ErrInvalidMeeting = errors.New("invalid meeting")
)

