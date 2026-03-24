package domain

import "errors"

var (
	ErrNotFound       = errors.New("not found")
	ErrAccessDenied   = errors.New("access denied")
	ErrInvalidInput   = errors.New("invalid input")
	ErrConflict       = errors.New("conflict")
	ErrInvalidMeeting    = errors.New("invalid meeting")
	ErrMeetingInPast     = errors.New("meeting in past")
	ErrInvalidTimeRange  = errors.New("invalid time range")
	ErrCannotRemoveOrganizer = errors.New("cannot remove organizer from participants")
	ErrAlreadyCancelled      = errors.New("meeting already cancelled")
	ErrForbidden             = errors.New("forbidden")
)

