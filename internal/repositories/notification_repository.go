package repositories

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
)

type NotificationRepository interface {
	GetSettingsByUser(ctx context.Context, userID string) ([]domain.NotificationSetting, error)
	UpsertSetting(ctx context.Context, setting domain.NotificationSetting) error
	GetSetting(ctx context.Context, userID string, eventType domain.EventType) (*domain.NotificationSetting, error)

	CreateNotification(ctx context.Context, n domain.Notification) (*domain.Notification, error)
	GetUserNotifications(ctx context.Context, userID string, unreadOnly bool, limit, offset int32) ([]domain.Notification, error)
	MarkNotificationAsRead(ctx context.Context, userID, notificationID string) error
	MarkAllNotificationsAsRead(ctx context.Context, userID string) error
	GetUnreadCount(ctx context.Context, userID string) (int, error)
	GetPendingEmailNotifications(ctx context.Context) ([]domain.Notification, error)
}

type notificationRepository struct {
	q *db.Queries
}

func NewNotificationRepository(q *db.Queries) NotificationRepository {
	return &notificationRepository{q: q}
}

func (r *notificationRepository) GetSettingsByUser(ctx context.Context, userID string) ([]domain.NotificationSetting, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.GetNotificationSettingsByUser(ctx, uid)
	if err != nil {
		return nil, err
	}
	result := make([]domain.NotificationSetting, len(rows))
	for i, s := range rows {
		result[i] = mapDBNotificationSettingToDomain(s)
	}
	return result, nil
}

func (r *notificationRepository) UpsertSetting(ctx context.Context, setting domain.NotificationSetting) error {
	uid, err := uuid.Parse(setting.UserID)
	if err != nil {
		return err
	}
	var offset sql.NullInt32
	if setting.ReminderOffsetMinutes != nil {
		offset = sql.NullInt32{Int32: int32(*setting.ReminderOffsetMinutes), Valid: true}
	}
	return r.q.UpsertNotificationSetting(ctx, db.UpsertNotificationSettingParams{
		UserID:                uid,
		EventType:             string(setting.EventType),
		InSystem:              setting.InSystem,
		InEmail:               setting.InEmail,
		ReminderOffsetMinutes: offset,
	})
}

func (r *notificationRepository) GetSetting(ctx context.Context, userID string, eventType domain.EventType) (*domain.NotificationSetting, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	row, err := r.q.GetNotificationSetting(ctx, db.GetNotificationSettingParams{
		UserID:    uid,
		EventType: string(eventType),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	s := mapDBNotificationSettingToDomain(row)
	return &s, nil
}

func (r *notificationRepository) CreateNotification(ctx context.Context, n domain.Notification) (*domain.Notification, error) {
	uid, err := uuid.Parse(n.UserID)
	if err != nil {
		return nil, err
	}
	body := sql.NullString{}
	if n.Body != nil {
		body = sql.NullString{String: *n.Body, Valid: true}
	}
	payload := pqtype.NullRawMessage{}
	if len(n.PayloadJSON) > 0 {
		payload = pqtype.NullRawMessage{RawMessage: n.PayloadJSON, Valid: true}
	}
	emailStatus := sql.NullString{}
	if n.EmailStatus != nil {
		emailStatus = sql.NullString{String: *n.EmailStatus, Valid: true}
	}
	row, err := r.q.CreateNotification(ctx, db.CreateNotificationParams{
		UserID:      uid,
		EventType:   string(n.EventType),
		Channel:     string(n.Channel),
		Title:       n.Title,
		Body:        body,
		Payload:     payload,
		EmailStatus: emailStatus,
	})
	if err != nil {
		return nil, err
	}
	d := mapDBNotificationToDomain(row)
	return &d, nil
}

func (r *notificationRepository) GetUserNotifications(ctx context.Context, userID string, unreadOnly bool, limit, offset int32) ([]domain.Notification, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.GetUserNotifications(ctx, db.GetUserNotificationsParams{
		UserID:  uid,
		Column2: unreadOnly,
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		return nil, err
	}
	result := make([]domain.Notification, len(rows))
	for i, n := range rows {
		result[i] = mapDBNotificationToDomain(n)
	}
	return result, nil
}

func (r *notificationRepository) MarkNotificationAsRead(ctx context.Context, userID, notificationID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	nid, err := uuid.Parse(notificationID)
	if err != nil {
		return err
	}
	return r.q.MarkNotificationAsRead(ctx, db.MarkNotificationAsReadParams{
		ID:     nid,
		UserID: uid,
	})
}

func (r *notificationRepository) MarkAllNotificationsAsRead(ctx context.Context, userID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	return r.q.MarkAllNotificationsAsRead(ctx, uid)
}

func (r *notificationRepository) GetUnreadCount(ctx context.Context, userID string) (int, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return 0, err
	}
	n, err := r.q.GetUnreadNotificationCount(ctx, uid)
	return int(n), err
}

func (r *notificationRepository) GetPendingEmailNotifications(ctx context.Context) ([]domain.Notification, error) {
	rows, err := r.q.GetNotificationsForEmail(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]domain.Notification, len(rows))
	for i, n := range rows {
		result[i] = mapDBNotificationToDomain(n)
	}
	return result, nil
}

func mapDBNotificationSettingToDomain(s db.NotificationSetting) domain.NotificationSetting {
	var offset *int
	if s.ReminderOffsetMinutes.Valid {
		v := int(s.ReminderOffsetMinutes.Int32)
		offset = &v
	}
	return domain.NotificationSetting{
		ID:                    s.ID.String(),
		UserID:                s.UserID.String(),
		EventType:             domain.EventType(s.EventType),
		InSystem:              s.InSystem,
		InEmail:               s.InEmail,
		ReminderOffsetMinutes: offset,
		CreatedAt:             s.CreatedAt,
		UpdatedAt:             s.UpdatedAt,
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
	var emailStatus *string
	if n.EmailStatus.Valid {
		emailStatus = &n.EmailStatus.String
	}
	var readAt *time.Time
	if n.ReadAt.Valid {
		readAt = &n.ReadAt.Time
	}
	var emailSentAt *time.Time
	if n.EmailSentAt.Valid {
		emailSentAt = &n.EmailSentAt.Time
	}
	return domain.Notification{
		ID:          n.ID.String(),
		UserID:      n.UserID.String(),
		EventType:   domain.EventType(n.EventType),
		Channel:     domain.ChannelType(n.Channel),
		Title:       n.Title,
		Body:        body,
		PayloadJSON: payload,
		IsRead:      n.IsRead,
		CreatedAt:   n.CreatedAt,
		ReadAt:      readAt,
		EmailStatus: emailStatus,
		EmailSentAt: emailSentAt,
	}
}
