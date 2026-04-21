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
	ErrScrumWipNotAllowed    = errors.New("swimlane WIP limits are not supported in Scrum")
	ErrCompletedColumnWip   = errors.New("WIP limit cannot be set for completed columns")
	ErrActiveSprintExists    = errors.New("project already has an active sprint")
	ErrSprintDatesOverlap    = errors.New("sprint dates overlap with existing sprint")
	ErrNoNextSprintForMove   = errors.New("no next planned sprint to move incomplete tasks")
	ErrInvalidEstimation     = errors.New("estimation must be a non-negative number")
	ErrUserRequiresRole      = errors.New("user must have at least one system role")
	ErrInvalidPermissionCode = errors.New("unknown or wrong-scope permission code")
	ErrRequiredCustomFieldNotAllowed = errors.New("custom params cannot be required — only system params can")
	ErrProjectAdminRoleMissing = errors.New("project admin role is missing")
)

// InvalidPermissionCodeError оборачивает ErrInvalidPermissionCode и несёт
// список конкретных кодов, которые не прошли валидацию — чтобы клиент мог
// поправить опечатки точечно. Реализует `errors.Is` через Unwrap.
type InvalidPermissionCodeError struct {
	Codes []string
}

func (e *InvalidPermissionCodeError) Error() string {
	return fmt.Sprintf("unknown or wrong-scope permission codes: %v", e.Codes)
}

func (e *InvalidPermissionCodeError) Unwrap() error { return ErrInvalidPermissionCode }

