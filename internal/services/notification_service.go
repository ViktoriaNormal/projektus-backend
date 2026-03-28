package services

import (
	"context"

	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
)

type NotificationService interface {
	GetSettings(ctx context.Context, userID string) ([]domain.NotificationSetting, error)
	UpdateSettings(ctx context.Context, userID string, settings []domain.NotificationSetting) error
	GetFeed(ctx context.Context, userID string, unreadOnly bool, limit, offset int32) ([]domain.Notification, int, error)
	MarkAsRead(ctx context.Context, userID, notificationID string) error
	MarkAllAsRead(ctx context.Context, userID string) error
	// SendEvent создает уведомления по заданному событию с учетом пользовательских настроек.
	SendEvent(ctx context.Context, eventType domain.EventType, userIDs []string, title, body string, payload []byte) error
	// GetSetting возвращает настройку уведомлений пользователя для конкретного события.
	GetSetting(ctx context.Context, userID string, eventType domain.EventType) (*domain.NotificationSetting, error)
}

type notificationService struct {
	repo repositories.NotificationRepository
}

func NewNotificationService(repo repositories.NotificationRepository) NotificationService {
	return &notificationService{repo: repo}
}

func (s *notificationService) GetSettings(ctx context.Context, userID string) ([]domain.NotificationSetting, error) {
	return s.repo.GetSettingsByUser(ctx, userID)
}

func (s *notificationService) UpdateSettings(ctx context.Context, userID string, settings []domain.NotificationSetting) error {
	for _, st := range settings {
		st.UserID = userID
		// Дефолтное значение для напоминаний о встречах — 30 минут,
		// если пользователь не указал свой offset.
		if st.EventType == domain.EventMeetingReminder && st.ReminderOffsetMinutes == nil {
			def := 30
			st.ReminderOffsetMinutes = &def
		}
		if err := s.repo.UpsertSetting(ctx, st); err != nil {
			return err
		}
	}
	return nil
}

func (s *notificationService) GetFeed(ctx context.Context, userID string, unreadOnly bool, limit, offset int32) ([]domain.Notification, int, error) {
	items, err := s.repo.GetUserNotifications(ctx, userID, unreadOnly, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	count, err := s.repo.GetUnreadCount(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	return items, count, nil
}

func (s *notificationService) MarkAsRead(ctx context.Context, userID, notificationID string) error {
	return s.repo.MarkNotificationAsRead(ctx, userID, notificationID)
}

func (s *notificationService) MarkAllAsRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllNotificationsAsRead(ctx, userID)
}

// SendEvent смотрит настройки пользователя и создает уведомления по включенным каналам.
// Если настроек нет, по умолчанию включены только внутрисистемные уведомления.
func (s *notificationService) SendEvent(ctx context.Context, eventType domain.EventType, userIDs []string, title, body string, payload []byte) error {
	for _, uid := range userIDs {
		setting, err := s.repo.GetSetting(ctx, uid, eventType)
		if err != nil {
			return err
		}

		inSystem := true
		inEmail := false
		if setting != nil {
			inSystem = setting.InSystem
			inEmail = setting.InEmail
		}

		var bodyPtr *string
		if body != "" {
			bodyPtr = &body
		}

		if inSystem {
			if _, err := s.repo.CreateNotification(ctx, domain.Notification{
				UserID:      uid,
				EventType:   eventType,
				Channel:     domain.ChannelSystem,
				Title:       title,
				Body:        bodyPtr,
				PayloadJSON: payload,
			}); err != nil {
				return err
			}
		}

		if inEmail {
			if _, err := s.repo.CreateNotification(ctx, domain.Notification{
				UserID:      uid,
				EventType:   eventType,
				Channel:     domain.ChannelEmail,
				Title:       title,
				Body:        bodyPtr,
				PayloadJSON: payload,
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *notificationService) GetSetting(ctx context.Context, userID string, eventType domain.EventType) (*domain.NotificationSetting, error) {
	return s.repo.GetSetting(ctx, userID, eventType)
}


