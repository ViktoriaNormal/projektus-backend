package domain

import "time"

type MeetingType string

const (
	// Scrum
	MeetingTypeSprintPlanning   MeetingType = "Планирование спринта"
	MeetingTypeDailyScrum       MeetingType = "Ежедневный Scrum"
	MeetingTypeSprintReview     MeetingType = "Обзор спринта"
	MeetingTypeSprintRetrospect MeetingType = "Ретроспектива спринта"

	// Kanban cadences
	MeetingTypeDailyMeeting          MeetingType = "ежедневная встреча"
	MeetingTypeRiskReview            MeetingType = "обзор рисков"
	MeetingTypeStrategyReview        MeetingType = "обзор стратегии"
	MeetingTypeServiceDeliveryReview MeetingType = "обзор предоставления услуг"
	MeetingTypeOperationsReview      MeetingType = "обзор операций"
	MeetingTypeReplenishment         MeetingType = "пополнение запасов"
	MeetingTypeDeliveryPlanning      MeetingType = "планирование поставок"

	// Custom
	MeetingTypeCustom MeetingType = "Пользовательское событие"
)

type MeetingStatus string

const (
	MeetingStatusActive    MeetingStatus = "active"
	MeetingStatusCancelled MeetingStatus = "cancelled"
)

type ParticipantStatus string

const (
	ParticipantStatusPending  ParticipantStatus = "pending"
	ParticipantStatusAccepted ParticipantStatus = "accepted"
	ParticipantStatusDeclined ParticipantStatus = "declined"
)

type Meeting struct {
	ID          string        `json:"id"`
	ProjectID   *string       `json:"project_id,omitempty"`
	Name        string        `json:"name"`
	Description *string       `json:"description,omitempty"`
	Type        MeetingType   `json:"meeting_type"`
	Location    *string       `json:"location,omitempty"`
	Status      MeetingStatus `json:"status"`
	StartTime   time.Time     `json:"start_time"`
	EndTime     time.Time     `json:"end_time"`
	CreatedBy   string        `json:"created_by"`
}

type MeetingParticipant struct {
	ID        string            `json:"id"`
	MeetingID string            `json:"meeting_id"`
	UserID    string            `json:"user_id"`
	Status    ParticipantStatus `json:"status"`
}
