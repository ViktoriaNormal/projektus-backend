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

type ParticipantStatus string

const (
	ParticipantStatusPending  ParticipantStatus = "pending"
	ParticipantStatusAccepted ParticipantStatus = "accepted"
	ParticipantStatusDeclined ParticipantStatus = "declined"
)

type Meeting struct {
	ID          string
	ProjectID   *string
	Name        string
	Description *string
	Type        MeetingType
	StartTime   time.Time
	EndTime     time.Time
	CreatedBy   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CanceledAt  *time.Time
}

type MeetingParticipant struct {
	ID         string
	MeetingID  string
	UserID     string
	Status     ParticipantStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

