package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"

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
}

type meetingService struct {
	meetings      repositories.MeetingRepository
	notifications NotificationService
}

func NewMeetingService(meetings repositories.MeetingRepository, notifications NotificationService) MeetingService {
	return &meetingService{
		meetings:      meetings,
		notifications: notifications,
	}
}

func (s *meetingService) ListUserMeetings(ctx context.Context, userID string, from, to *time.Time) ([]domain.Meeting, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, domain.ErrInvalidInput
	}
	var fromNT, toNT sql.NullTime
	if from != nil {
		fromNT = sql.NullTime{Time: *from, Valid: true}
	}
	if to != nil {
		toNT = sql.NullTime{Time: *to, Valid: true}
	}
	return s.meetings.ListUserMeetings(ctx, uid, fromNT, toNT)
}

func (s *meetingService) CreateMeeting(ctx context.Context, creatorID string, m domain.Meeting, participantIDs []string) (*domain.Meeting, []domain.MeetingParticipant, error) {
	// валидация времени
	if !m.StartTime.After(time.Now()) {
		return nil, nil, domain.ErrMeetingInPast
	}
	if !m.StartTime.Before(m.EndTime) {
		return nil, nil, domain.ErrInvalidTimeRange
	}
	creatorUID, err := uuid.Parse(creatorID)
	if err != nil {
		return nil, nil, domain.ErrInvalidInput
	}
	m.CreatedBy = creatorUID
	created, err := s.meetings.CreateMeeting(ctx, m)
	if err != nil {
		return nil, nil, err
	}

	// добавляем создателя как accepted
	participants := make([]domain.MeetingParticipant, 0, len(participantIDs)+1)
	creatorPart, err := s.meetings.AddParticipant(ctx, created.ID, creatorUID, domain.ParticipantStatusAccepted)
	if err != nil {
		return nil, nil, err
	}
	participants = append(participants, *creatorPart)

	// остальные участники – в pending
	for _, id := range participantIDs {
		if id == creatorID {
			continue
		}
		pid, err := uuid.Parse(id)
		if err != nil {
			return nil, nil, domain.ErrInvalidInput
		}
		p, err := s.meetings.AddParticipant(ctx, created.ID, pid, domain.ParticipantStatusPending)
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
			_ = s.notifications.SendEvent(ctx, domain.EventMeetingInvite, recipients, title, body, meetingPayload(created.ID.String(), created.Name, created.StartTime))
		}
	}

	return created, participants, nil
}

func (s *meetingService) GetMeetingWithParticipants(ctx context.Context, requesterID, meetingID string) (*domain.Meeting, []domain.MeetingParticipant, error) {
	mid, err := uuid.Parse(meetingID)
	if err != nil {
		return nil, nil, domain.ErrInvalidInput
	}
	requesterUID, err := uuid.Parse(requesterID)
	if err != nil {
		return nil, nil, domain.ErrInvalidInput
	}
	m, err := s.meetings.GetMeetingByID(ctx, mid)
	if err != nil {
		return nil, nil, err
	}
	parts, err := s.meetings.GetMeetingParticipants(ctx, mid)
	if err != nil {
		return nil, nil, err
	}
	// минимальная проверка доступа: requester должен быть участником
	isParticipant := false
	for _, p := range parts {
		if p.UserID == requesterUID {
			isParticipant = true
			break
		}
	}
	if !isParticipant && m.CreatedBy != requesterUID {
		return nil, nil, domain.ErrAccessDenied
	}
	return m, parts, nil
}

func (s *meetingService) UpdateMeeting(ctx context.Context, userID string, m domain.Meeting) (*domain.Meeting, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, domain.ErrInvalidInput
	}
	existing, err := s.meetings.GetMeetingByID(ctx, m.ID)
	if err != nil {
		return nil, err
	}
	if existing.CreatedBy != uid {
		return nil, domain.ErrAccessDenied
	}
	// переносим отсутствующие non-nullable поля из существующей встречи
	// (nullable поля Description и Location мержатся в хендлере через NullableField)
	if m.Name == "" {
		m.Name = existing.Name
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
			if p.UserID != uid {
				recipients = append(recipients, p.UserID.String())
			}
		}
		if len(recipients) > 0 {
			title := "Изменена встреча: " + updated.Name
			body := "Параметры встречи были обновлены."
			_ = s.notifications.SendEvent(ctx, domain.EventMeetingChange, recipients, title, body, meetingPayload(updated.ID.String(), updated.Name, updated.StartTime))
		}
	}

	return updated, nil
}

func (s *meetingService) CancelMeeting(ctx context.Context, userID, meetingID string) (*domain.Meeting, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, domain.ErrInvalidInput
	}
	mid, err := uuid.Parse(meetingID)
	if err != nil {
		return nil, domain.ErrInvalidInput
	}
	m, err := s.meetings.GetMeetingByID(ctx, mid)
	if err != nil {
		return nil, err
	}
	if m.CreatedBy != uid {
		return nil, domain.ErrAccessDenied
	}
	if m.Status == domain.MeetingStatusCancelled {
		return nil, domain.ErrAlreadyCancelled
	}
	cancelled, err := s.meetings.CancelMeeting(ctx, mid)
	if err != nil {
		return nil, err
	}

	// Уведомления участникам об отмене встречи
	parts, err := s.meetings.GetMeetingParticipants(ctx, mid)
	if err == nil && len(parts) > 0 {
		recipients := make([]string, 0, len(parts))
		for _, p := range parts {
			if p.UserID != uid {
				recipients = append(recipients, p.UserID.String())
			}
		}
		if len(recipients) > 0 {
			title := "Отменена встреча: " + m.Name
			body := "Встреча была отменена."
			_ = s.notifications.SendEvent(ctx, domain.EventMeetingCancel, recipients, title, body, meetingPayload(m.ID.String(), m.Name, m.StartTime))
		}
	}

	return cancelled, nil
}

func (s *meetingService) AddParticipants(ctx context.Context, userID, meetingID string, participantIDs []string) ([]domain.MeetingParticipant, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, domain.ErrInvalidInput
	}
	mid, err := uuid.Parse(meetingID)
	if err != nil {
		return nil, domain.ErrInvalidInput
	}
	m, err := s.meetings.GetMeetingByID(ctx, mid)
	if err != nil {
		return nil, err
	}
	if m.CreatedBy != uid {
		return nil, domain.ErrAccessDenied
	}
	result := make([]domain.MeetingParticipant, 0, len(participantIDs))
	for _, id := range participantIDs {
		pid, err := uuid.Parse(id)
		if err != nil {
			return nil, domain.ErrInvalidInput
		}
		p, err := s.meetings.AddParticipant(ctx, mid, pid, domain.ParticipantStatusPending)
		if err != nil {
			return nil, err
		}
		result = append(result, *p)
	}
	return result, nil
}

func (s *meetingService) RespondToInvitation(ctx context.Context, userID, meetingID string, status domain.ParticipantStatus) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return domain.ErrInvalidInput
	}
	mid, err := uuid.Parse(meetingID)
	if err != nil {
		return domain.ErrInvalidInput
	}
	// проверим, что встреча существует
	if _, err := s.meetings.GetMeetingByID(ctx, mid); err != nil {
		return err
	}
	return s.meetings.UpdateParticipantStatus(ctx, mid, uid, status)
}

func meetingPayload(id, name string, startTime time.Time) []byte {
	st := startTime.Format(time.RFC3339)
	data, _ := json.Marshal(domain.NotificationPayload{MeetingID: &id, MeetingName: &name, MeetingStartTime: &st})
	return data
}


