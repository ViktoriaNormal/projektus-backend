package services

import (
	"context"
	"database/sql"
	"time"

	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
)

type MeetingService interface {
	ListUserMeetings(ctx context.Context, userID string, from, to *time.Time) ([]domain.Meeting, error)
	CreateMeeting(ctx context.Context, creatorID string, m domain.Meeting, participantIDs []string) (*domain.Meeting, []domain.MeetingParticipant, error)
	GetMeetingWithParticipants(ctx context.Context, requesterID, meetingID string) (*domain.Meeting, []domain.MeetingParticipant, error)
	UpdateMeeting(ctx context.Context, userID string, m domain.Meeting) (*domain.Meeting, error)
	CancelMeeting(ctx context.Context, userID, meetingID string) (*domain.Meeting, error)
	AddParticipants(ctx context.Context, userID, meetingID string, participantIDs []string) ([]domain.MeetingParticipant, error)
	RespondToInvitation(ctx context.Context, userID, meetingID string, status domain.ParticipantStatus) error
	CheckAndSendMeetingRemindersForUser(ctx context.Context, userID string, now time.Time, tick time.Duration) error
}

type meetingService struct {
	meetings       repositories.MeetingRepository
	notifications  NotificationService
}

func NewMeetingService(meetings repositories.MeetingRepository, notifications NotificationService) MeetingService {
	return &meetingService{
		meetings:      meetings,
		notifications: notifications,
	}
}

func (s *meetingService) ListUserMeetings(ctx context.Context, userID string, from, to *time.Time) ([]domain.Meeting, error) {
	var fromNT, toNT sql.NullTime
	if from != nil {
		fromNT = sql.NullTime{Time: *from, Valid: true}
	}
	if to != nil {
		toNT = sql.NullTime{Time: *to, Valid: true}
	}
	return s.meetings.ListUserMeetings(ctx, userID, fromNT, toNT)
}

func (s *meetingService) CreateMeeting(ctx context.Context, creatorID string, m domain.Meeting, participantIDs []string) (*domain.Meeting, []domain.MeetingParticipant, error) {
	// валидация времени
	if !m.StartTime.After(time.Now()) {
		return nil, nil, domain.ErrMeetingInPast
	}
	if !m.StartTime.Before(m.EndTime) {
		return nil, nil, domain.ErrInvalidTimeRange
	}
	m.CreatedBy = creatorID
	created, err := s.meetings.CreateMeeting(ctx, m)
	if err != nil {
		return nil, nil, err
	}

	// добавляем создателя как accepted
	participants := make([]domain.MeetingParticipant, 0, len(participantIDs)+1)
	creatorPart, err := s.meetings.AddParticipant(ctx, created.ID, creatorID, domain.ParticipantStatusAccepted)
	if err != nil {
		return nil, nil, err
	}
	participants = append(participants, *creatorPart)

	// остальные участники – в pending
	for _, id := range participantIDs {
		if id == creatorID {
			continue
		}
		p, err := s.meetings.AddParticipant(ctx, created.ID, id, domain.ParticipantStatusPending)
		if err != nil {
			return nil, nil, err
		}
		participants = append(participants, *p)
	}

	// Уведомления участникам (кроме создателя) о приглашении
	if len(participantIDs) > 0 {
		recipients := make([]string, 0, len(participantIDs))
		for _, id := range participantIDs {
			if id != creatorID {
				recipients = append(recipients, id)
			}
		}
		if len(recipients) > 0 {
			title := "Приглашение на встречу: " + created.Name
			body := "Вы приглашены на встречу."
			_ = s.notifications.SendEvent(ctx, domain.EventMeetingInvitationReceived, recipients, title, body, nil)
		}
	}

	return created, participants, nil
}

func (s *meetingService) GetMeetingWithParticipants(ctx context.Context, requesterID, meetingID string) (*domain.Meeting, []domain.MeetingParticipant, error) {
	m, err := s.meetings.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, nil, err
	}
	parts, err := s.meetings.GetMeetingParticipants(ctx, meetingID)
	if err != nil {
		return nil, nil, err
	}
	// минимальная проверка доступа: requester должен быть участником
	isParticipant := false
	for _, p := range parts {
		if p.UserID == requesterID {
			isParticipant = true
			break
		}
	}
	if !isParticipant && m.CreatedBy != requesterID {
		return nil, nil, domain.ErrAccessDenied
	}
	return m, parts, nil
}

func (s *meetingService) UpdateMeeting(ctx context.Context, userID string, m domain.Meeting) (*domain.Meeting, error) {
	existing, err := s.meetings.GetMeetingByID(ctx, m.ID)
	if err != nil {
		return nil, err
	}
	if existing.CreatedBy != userID {
		return nil, domain.ErrAccessDenied
	}
	// переносим отсутствующие поля из существующей встречи
	if m.Name == "" {
		m.Name = existing.Name
	}
	if m.Description == nil {
		m.Description = existing.Description
	}
	if m.Type == "" {
		m.Type = existing.Type
	}
	if m.StartTime.IsZero() {
		m.StartTime = existing.StartTime
	}
	if m.EndTime.IsZero() {
		m.EndTime = existing.EndTime
	}
	if m.Location == nil {
		m.Location = existing.Location
	}
	// валидация времени после мержа с существующими значениями
	if !m.StartTime.After(time.Now()) {
		return nil, domain.ErrMeetingInPast
	}
	if !m.StartTime.Before(m.EndTime) {
		return nil, domain.ErrInvalidTimeRange
	}
	if err := s.meetings.UpdateMeeting(ctx, m); err != nil {
		return nil, err
	}
	updated, err := s.meetings.GetMeetingByID(ctx, m.ID)
	if err != nil {
		return nil, err
	}

	// Уведомления участникам об изменении встречи
	parts, err := s.meetings.GetMeetingParticipants(ctx, m.ID)
	if err == nil && len(parts) > 0 {
		recipients := make([]string, 0, len(parts))
		for _, p := range parts {
			if p.UserID != userID {
				recipients = append(recipients, p.UserID)
			}
		}
		if len(recipients) > 0 {
			title := "Изменена встреча: " + updated.Name
			body := "Параметры встречи были обновлены."
			_ = s.notifications.SendEvent(ctx, domain.EventMeetingUpdated, recipients, title, body, nil)
		}
	}

	return updated, nil
}

func (s *meetingService) CancelMeeting(ctx context.Context, userID, meetingID string) (*domain.Meeting, error) {
	m, err := s.meetings.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, err
	}
	if m.CreatedBy != userID {
		return nil, domain.ErrAccessDenied
	}
	if m.Status == domain.MeetingStatusCancelled {
		return nil, domain.ErrAlreadyCancelled
	}
	cancelled, err := s.meetings.CancelMeeting(ctx, meetingID)
	if err != nil {
		return nil, err
	}

	// Уведомления участникам об отмене встречи
	parts, err := s.meetings.GetMeetingParticipants(ctx, meetingID)
	if err == nil && len(parts) > 0 {
		recipients := make([]string, 0, len(parts))
		for _, p := range parts {
			if p.UserID != userID {
				recipients = append(recipients, p.UserID)
			}
		}
		if len(recipients) > 0 {
			title := "Отменена встреча: " + m.Name
			body := "Встреча была отменена."
			_ = s.notifications.SendEvent(ctx, domain.EventMeetingCancelled, recipients, title, body, nil)
		}
	}

	return cancelled, nil
}

func (s *meetingService) AddParticipants(ctx context.Context, userID, meetingID string, participantIDs []string) ([]domain.MeetingParticipant, error) {
	m, err := s.meetings.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, err
	}
	if m.CreatedBy != userID {
		return nil, domain.ErrAccessDenied
	}
	result := make([]domain.MeetingParticipant, 0, len(participantIDs))
	for _, id := range participantIDs {
		p, err := s.meetings.AddParticipant(ctx, meetingID, id, domain.ParticipantStatusPending)
		if err != nil {
			return nil, err
		}
		result = append(result, *p)
	}
	return result, nil
}

func (s *meetingService) RespondToInvitation(ctx context.Context, userID, meetingID string, status domain.ParticipantStatus) error {
	// проверим, что встреча существует
	if _, err := s.meetings.GetMeetingByID(ctx, meetingID); err != nil {
		return err
	}
	return s.meetings.UpdateParticipantStatus(ctx, meetingID, userID, status)
}

// CheckAndSendMeetingRemindersForUser проверяет, не пора ли напомнить пользователю о предстоящих встречах.
// now – текущий момент (в UTC), tick – ширина окна (например, 5 минут).
func (s *meetingService) CheckAndSendMeetingRemindersForUser(_ context.Context, _ string, _ time.Time, _ time.Duration) error {
	// meeting reminders removed in schema redesign
	return nil
}

