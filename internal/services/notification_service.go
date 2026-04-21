package services

import (
	"context"

	"github.com/google/uuid"

	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
)

type NotificationService interface {
	GetSettings(ctx context.Context, userID string) ([]domain.NotificationSetting, error)
	UpdateSettings(ctx context.Context, userID string, settings []domain.NotificationSetting) error
	GetFeed(ctx context.Context, userID string, unreadOnly bool, limit, offset int32) ([]domain.Notification, int, error)
	MarkAsRead(ctx context.Context, userID, notificationID string) error
	MarkAllAsRead(ctx context.Context, userID string) error
	DeleteAll(ctx context.Context, userID string) error
	// SendEvent создает уведомления по заданному событию с учетом пользовательских настроек.
	SendEvent(ctx context.Context, eventType domain.EventType, userIDs []string, title, body string, payload []byte) error
	// GetSetting возвращает настройку уведомлений пользователя для конкретного события.
	GetSetting(ctx context.Context, userID string, eventType domain.EventType) (*domain.NotificationSetting, error)
	// InitializeDefaultSettings создаёт настройки по умолчанию для всех типов событий:
	// in_system=true, in_email=false. Вызывается однократно при создании пользователя.
	// Идемпотентно: повторный вызов не перезаписывает уже изменённые настройки
	// (UpsertSetting обновит только если строка отсутствует — см. замечание в реализации).
	InitializeDefaultSettings(ctx context.Context, userID string) error
}

type notificationService struct {
	repo repositories.NotificationRepository
}

func NewNotificationService(repo repositories.NotificationRepository) NotificationService {
	return &notificationService{repo: repo}
}

func (s *notificationService) GetSettings(ctx context.Context, userID string) ([]domain.NotificationSetting, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	return s.repo.GetSettingsByUser(ctx, uid)
}

func (s *notificationService) UpdateSettings(ctx context.Context, userID string, settings []domain.NotificationSetting) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	for _, st := range settings {
		st.UserID = uid
		if err := s.repo.UpsertSetting(ctx, st); err != nil {
			return err
		}
	}
	return nil
}

func (s *notificationService) GetFeed(ctx context.Context, userID string, unreadOnly bool, limit, offset int32) ([]domain.Notification, int, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, 0, err
	}
	items, err := s.repo.GetUserNotifications(ctx, uid, unreadOnly, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	count, err := s.repo.GetUnreadCount(ctx, uid)
	if err != nil {
		return nil, 0, err
	}
	return items, count, nil
}

func (s *notificationService) MarkAsRead(ctx context.Context, userID, notificationID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	nid, err := uuid.Parse(notificationID)
	if err != nil {
		return err
	}
	return s.repo.MarkNotificationAsRead(ctx, uid, nid)
}

func (s *notificationService) MarkAllAsRead(ctx context.Context, userID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	return s.repo.MarkAllNotificationsAsRead(ctx, uid)
}

func (s *notificationService) DeleteAll(ctx context.Context, userID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	return s.repo.DeleteAllNotifications(ctx, uid)
}

// SendEvent смотрит настройки пользователя и создает уведомления по включенным каналам.
// Если настроек нет, по умолчанию включены только внутрисистемные уведомления.
func (s *notificationService) SendEvent(ctx context.Context, eventType domain.EventType, userIDs []string, title, body string, payload []byte) error {
	for _, uidStr := range userIDs {
		uid, err := uuid.Parse(uidStr)
		if err != nil {
			return err
		}
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
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	return s.repo.GetSetting(ctx, uid, eventType)
}

// defaultEventTypes — полный перечень типов событий, для которых создаются
// настройки уведомлений при регистрации пользователя.
var defaultEventTypes = []domain.EventType{
	domain.EventTaskAssigned,
	domain.EventCommentMention,
	domain.EventTaskStatusChangeAuthor,
	domain.EventTaskStatusChangeAssignee,
	domain.EventTaskStatusChangeWatcher,
	domain.EventMeetingInvite,
	domain.EventMeetingChange,
	domain.EventMeetingCancel,
}

func (s *notificationService) InitializeDefaultSettings(ctx context.Context, userID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	for _, et := range defaultEventTypes {
		existing, err := s.repo.GetSetting(ctx, uid, et)
		if err != nil {
			return err
		}
		if existing != nil {
			continue
		}
		if err := s.repo.UpsertSetting(ctx, domain.NotificationSetting{
			UserID:    uid,
			EventType: et,
			InSystem:  true,
			InEmail:   false,
		}); err != nil {
			return err
		}
	}
	return nil
}
