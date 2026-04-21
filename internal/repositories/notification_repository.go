package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
	"projektus-backend/pkg/errctx"
)

type NotificationRepository interface {
	GetSettingsByUser(ctx context.Context, userID uuid.UUID) ([]domain.NotificationSetting, error)
	UpsertSetting(ctx context.Context, setting domain.NotificationSetting) error
	GetSetting(ctx context.Context, userID uuid.UUID, eventType domain.EventType) (*domain.NotificationSetting, error)

	CreateNotification(ctx context.Context, n domain.Notification) (*domain.Notification, error)
	GetUserNotifications(ctx context.Context, userID uuid.UUID, unreadOnly bool, limit, offset int32) ([]domain.Notification, error)
	MarkNotificationAsRead(ctx context.Context, userID, notificationID uuid.UUID) error
	MarkAllNotificationsAsRead(ctx context.Context, userID uuid.UUID) error
	DeleteAllNotifications(ctx context.Context, userID uuid.UUID) error
	GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error)
	GetPendingEmailNotifications(ctx context.Context) ([]domain.Notification, error)
}

type notificationRepository struct {
	q *db.Queries
}

func NewNotificationRepository(q *db.Queries) NotificationRepository {
	return &notificationRepository{q: q}
}

func (r *notificationRepository) GetSettingsByUser(ctx context.Context, userID uuid.UUID) ([]domain.NotificationSetting, error) {
	rows, err := r.q.GetNotificationSettingsByUser(ctx, userID)
	if err != nil {
		return nil, errctx.Wrap(err, "GetNotificationSettingsByUser", "userID", userID)
	}
	result := make([]domain.NotificationSetting, len(rows))
	for i, s := range rows {
		result[i] = mapDBNotificationSettingToDomain(s)
	}
	return result, nil
}

func (r *notificationRepository) UpsertSetting(ctx context.Context, setting domain.NotificationSetting) error {
	return errctx.Wrap(r.q.UpsertNotificationSetting(ctx, db.UpsertNotificationSettingParams{
		UserID:    setting.UserID,
		EventType: string(setting.EventType),
		InSystem:  setting.InSystem,
		InEmail:   setting.InEmail,
	}), "UpsertNotificationSetting", "userID", setting.UserID, "eventType", setting.EventType)
}

func (r *notificationRepository) GetSetting(ctx context.Context, userID uuid.UUID, eventType domain.EventType) (*domain.NotificationSetting, error) {
	row, err := r.q.GetNotificationSetting(ctx, db.GetNotificationSettingParams{
		UserID:    userID,
		EventType: string(eventType),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, errctx.Wrap(err, "GetNotificationSetting", "userID", userID, "eventType", eventType)
	}
	s := mapDBNotificationSettingToDomain(row)
	return &s, nil
}

func (r *notificationRepository) CreateNotification(ctx context.Context, n domain.Notification) (*domain.Notification, error) {
	body := sql.NullString{}
	if n.Body != nil {
		body = sql.NullString{String: *n.Body, Valid: true}
	}
	payload := pqtype.NullRawMessage{}
	if len(n.PayloadJSON) > 0 {
		payload = pqtype.NullRawMessage{RawMessage: n.PayloadJSON, Valid: true}
	}
	row, err := r.q.CreateNotification(ctx, db.CreateNotificationParams{
		UserID:    n.UserID,
		EventType: string(n.EventType),
		Title:     n.Title,
		Body:      body,
		Payload:   payload,
	})
	if err != nil {
		return nil, errctx.Wrap(err, "CreateNotification", "userID", n.UserID, "eventType", n.EventType)
	}
	d := mapDBNotificationToDomain(row)
	return &d, nil
}

func (r *notificationRepository) GetUserNotifications(ctx context.Context, userID uuid.UUID, unreadOnly bool, limit, offset int32) ([]domain.Notification, error) {
	rows, err := r.q.GetUserNotifications(ctx, db.GetUserNotificationsParams{
		UserID:  userID,
		Column2: unreadOnly,
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		return nil, errctx.Wrap(err, "GetUserNotifications", "userID", userID)
	}
	result := make([]domain.Notification, len(rows))
	for i, n := range rows {
		result[i] = mapDBNotificationToDomain(n)
	}
	return result, nil
}

func (r *notificationRepository) MarkNotificationAsRead(ctx context.Context, userID, notificationID uuid.UUID) error {
	return errctx.Wrap(r.q.MarkNotificationAsRead(ctx, db.MarkNotificationAsReadParams{
		ID:     notificationID,
		UserID: userID,
	}), "MarkNotificationAsRead", "userID", userID, "notificationID", notificationID)
}

func (r *notificationRepository) MarkAllNotificationsAsRead(ctx context.Context, userID uuid.UUID) error {
	return errctx.Wrap(r.q.MarkAllNotificationsAsRead(ctx, userID), "MarkAllNotificationsAsRead", "userID", userID)
}

func (r *notificationRepository) DeleteAllNotifications(ctx context.Context, userID uuid.UUID) error {
	return errctx.Wrap(r.q.DeleteAllNotifications(ctx, userID), "DeleteAllNotifications", "userID", userID)
}

func (r *notificationRepository) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	n, err := r.q.GetUnreadNotificationCount(ctx, userID)
	if err != nil {
		return 0, errctx.Wrap(err, "GetUnreadNotificationCount", "userID", userID)
	}
	return int(n), nil
}

func (r *notificationRepository) GetPendingEmailNotifications(_ context.Context) ([]domain.Notification, error) {
	// email notification queries removed in schema redesign
	return nil, nil
}

func mapDBNotificationSettingToDomain(s db.NotificationSetting) domain.NotificationSetting {
	return domain.NotificationSetting{
		ID:        s.ID,
		UserID:    s.UserID,
		EventType: domain.EventType(s.EventType),
		InSystem:  s.InSystem,
		InEmail:   s.InEmail,
	}
}

func mapDBNotificationToDomain(n db.Notification) domain.Notification {
	var body *string
	if n.Body.Valid {
		body = &n.Body.String
	}
	var payload []byte
	if n.Payload.Valid {
		payload = n.Payload.RawMessage
	}
	return domain.Notification{
		ID:          n.ID,
		UserID:      n.UserID,
		EventType:   domain.EventType(n.EventType),
		Title:       n.Title,
		Body:        body,
		PayloadJSON: payload,
		IsRead:      n.IsRead,
		CreatedAt:   n.CreatedAt,
	}
}
