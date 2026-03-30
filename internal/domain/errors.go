package domain

import (
	"errors"
	"fmt"
)

// ParamValidationError carries a user-facing message for project param validation failures.
type ParamValidationError struct {
	Message string
}

func (e *ParamValidationError) Error() string { return e.Message }

func NewParamValidationError(format string, args ...any) *ParamValidationError {
	return &ParamValidationError{Message: fmt.Sprintf(format, args...)}
}

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
	ErrRoleHasMembers        = errors.New("role has members")
	ErrSystemParam           = errors.New("cannot delete system param")
	ErrSystemField           = errors.New("cannot modify system field")
	ErrColumnHasTasks        = errors.New("column has tasks")
	ErrSwimlaneHasTasks      = errors.New("swimlane has tasks")
	ErrProjectAdminRole      = errors.New("cannot modify project admin role")
	ErrTemplateAdminRole     = errors.New("cannot modify template admin role")
	ErrSystemAdminRole       = errors.New("cannot modify system admin role")
	ErrLastProjectAdmin      = errors.New("cannot remove last project admin")
	ErrTagAlreadyExists      = errors.New("tag already exists")
	ErrInvalidFieldType      = errors.New("invalid field type for this context")
)

