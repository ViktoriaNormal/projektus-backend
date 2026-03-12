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
	CancelMeeting(ctx context.Context, userID, meetingID string) error
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
	// базовая валидация
	if !m.StartTime.Before(m.EndTime) {
		return nil, nil, domain.ErrInvalidMeeting
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
	// базовая валидация времени
	if !m.StartTime.IsZero() && !m.EndTime.IsZero() && !m.StartTime.Before(m.EndTime) {
		return nil, domain.ErrInvalidMeeting
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

func (s *meetingService) CancelMeeting(ctx context.Context, userID, meetingID string) error {
	m, err := s.meetings.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return err
	}
	if m.CreatedBy != userID {
		return domain.ErrAccessDenied
	}
	if err := s.meetings.CancelMeeting(ctx, meetingID); err != nil {
		return err
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

	return nil
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
func (s *meetingService) CheckAndSendMeetingRemindersForUser(ctx context.Context, userID string, now time.Time, tick time.Duration) error {
	// 1. Читаем настройку для напоминаний о встречах.
	setting, err := s.notifications.GetSetting(ctx, userID, domain.EventMeetingReminder)
	if err != nil {
		return err
	}
	if setting == nil || (!setting.InSystem && !setting.InEmail) {
		return nil
	}

	// 2. Определяем offset (по умолчанию 30 минут).
	// Если фронт прислал 0, это значит "напоминание по факту начала встречи".
	offsetMinutes := 30
	if setting.ReminderOffsetMinutes != nil {
		offsetMinutes = *setting.ReminderOffsetMinutes
	}
	offset := time.Duration(offsetMinutes) * time.Minute

	from := now.Add(offset)
	to := from.Add(tick)

	// 3. Берем встречи пользователя, для которых напоминание еще не отправлялось (фильтр в SQL).
	meetings, err := s.meetings.GetUpcomingMeetingsForUser(ctx, userID, from, to)
	if err != nil {
		return err
	}

	if len(meetings) == 0 {
		return nil
	}

	for _, m := range meetings {
		// пропускаем отменённые
		if m.CanceledAt != nil {
			continue
		}

		reminderTime := m.StartTime.Add(-offset)

		// 4. Записываем факт напоминания. Если запись уже есть (ON CONFLICT DO NOTHING),
		// CreateReminder не вернет ошибку, а просто не изменит таблицу.
		if setting.InSystem {
			_ = s.meetings.CreateReminder(ctx, m.ID, userID, domain.ChannelSystem, reminderTime)
		}
		if setting.InEmail {
			_ = s.meetings.CreateReminder(ctx, m.ID, userID, domain.ChannelEmail, reminderTime)
		}

		// 5. Создаем уведомление (внутрисистемное / email в зависимости от настроек).
		title := "Напоминание о встрече: " + m.Name
		body := "Встреча начинается в " + m.StartTime.Format(time.RFC3339)

		_ = s.notifications.SendEvent(ctx, domain.EventMeetingReminder, []string{userID}, title, body, nil)
	}

	return nil
}

