package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projektus-backend/internal/api/dto"
	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/services"
)

type NotificationHandler struct {
	svc     services.NotificationService
	queries *db.Queries
}

func NewNotificationHandler(svc services.NotificationService, queries *db.Queries) *NotificationHandler {
	return &NotificationHandler{svc: svc, queries: queries}
}

// GET /api/v1/notifications
func (h *NotificationHandler) GetFeed(c *gin.Context) {
	userUUID, ok := requireUserUUID(c)
	if !ok {
		return
	}
	userID := userUUID.String()

	unreadOnly := c.Query("unreadOnly") == "true"

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if err != nil || limit <= 0 {
		limit = 20
	}
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		offset = 0
	}

	items, unreadCount, err := h.svc.GetFeed(c.Request.Context(), userID, unreadOnly, int32(limit), int32(offset))
	if err != nil {
		respondInternal(c, err, "Не удалось получить уведомления")
		return
	}

	resp := make([]dto.NotificationResponse, 0, len(items))
	for _, n := range items {
		r := mapNotificationToDTO(n)
		// Для meeting_invite подгружаем текущий статус приглашения, чтобы
		// фронту не пришлось делать отдельный запрос на каждую запись.
		if n.EventType == domain.EventMeetingInvite && r.MeetingID != nil {
			if mid, err := uuid.Parse(*r.MeetingID); err == nil {
				if status, err := h.queries.GetParticipantStatus(c.Request.Context(), db.GetParticipantStatusParams{
					MeetingID: mid, UserID: userUUID,
				}); err == nil {
					r.ParticipantStatus = &status
				}
			}
		}
		resp = append(resp, r)
	}

	writeSuccess(c, dto.NotificationFeedResponse{
		Items:       resp,
		UnreadCount: unreadCount,
	})
}

// POST /api/v1/notifications/:notificationId/read
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	userUUID, ok := requireUserUUID(c)
	if !ok {
		return
	}
	notificationID := c.Param("notificationId")

	if err := h.svc.MarkAsRead(c.Request.Context(), userUUID.String(), notificationID); err != nil {
		respondInternal(c, err, "Не удалось отметить уведомление прочитанным")
		return
	}

	c.Status(http.StatusNoContent)
}

// POST /api/v1/notifications/delete-all
func (h *NotificationHandler) DeleteAll(c *gin.Context) {
	userUUID, ok := requireUserUUID(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteAll(c.Request.Context(), userUUID.String()); err != nil {
		respondInternal(c, err, "Не удалось удалить уведомления")
		return
	}
	c.Status(http.StatusNoContent)
}

// POST /api/v1/notifications/read-all
func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	userUUID, ok := requireUserUUID(c)
	if !ok {
		return
	}
	if err := h.svc.MarkAllAsRead(c.Request.Context(), userUUID.String()); err != nil {
		respondInternal(c, err, "Не удалось отметить уведомления прочитанными")
		return
	}
	c.Status(http.StatusNoContent)
}

// GET /api/v1/notifications/settings
func (h *NotificationHandler) GetSettings(c *gin.Context) {
	userUUID, ok := requireUserUUID(c)
	if !ok {
		return
	}
	settings, err := h.svc.GetSettings(c.Request.Context(), userUUID.String())
	if err != nil {
		respondInternal(c, err, "Не удалось получить настройки уведомлений")
		return
	}

	resp := make([]dto.NotificationSettingResponse, 0, len(settings))
	for _, s := range settings {
		resp = append(resp, mapSettingToDTO(s))
	}
	writeSuccess(c, resp)
}

// PUT /api/v1/notifications/settings
func (h *NotificationHandler) UpdateSettings(c *gin.Context) {
	userUUID, ok := requireUserUUID(c)
	if !ok {
		return
	}
	userID := userUUID.String()

	req, ok := bindJSON[[]dto.UpdateNotificationSettingItem](c)
	if !ok {
		return
	}

	settings := make([]domain.NotificationSetting, 0, len(req))
	for _, item := range req {
		s := domain.NotificationSetting{
			UserID:    userUUID,
			EventType: domain.EventType(item.EventType),
			InSystem:  true,
			InEmail:   false,
		}
		if item.Enabled != nil {
			s.InSystem = *item.Enabled
		}
		settings = append(settings, s)
	}

	if err := h.svc.UpdateSettings(c.Request.Context(), userID, settings); err != nil {
		respondInternal(c, err, "Не удалось сохранить настройки уведомлений")
		return
	}

	updated, err := h.svc.GetSettings(c.Request.Context(), userID)
	if err != nil {
		respondInternal(c, err, "Не удалось получить настройки уведомлений")
		return
	}
	resp := make([]dto.NotificationSettingResponse, 0, len(updated))
	for _, s := range updated {
		resp = append(resp, mapSettingToDTO(s))
	}
	writeSuccess(c, resp)
}

func mapNotificationToDTO(n domain.Notification) dto.NotificationResponse {
	r := dto.NotificationResponse{
		ID:        n.ID.String(),
		Type:      string(n.EventType),
		Message:   n.Title,
		Read:      n.IsRead,
		CreatedAt: n.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if len(n.PayloadJSON) > 0 {
		var p domain.NotificationPayload
		if json.Unmarshal(n.PayloadJSON, &p) == nil {
			r.TaskID = p.TaskID
			r.TaskKey = p.TaskKey
			r.MeetingID = p.MeetingID
			r.MeetingName = p.MeetingName
			r.MeetingStartTime = p.MeetingStartTime
		}
	}

	return r
}

func mapSettingToDTO(s domain.NotificationSetting) dto.NotificationSettingResponse {
	return dto.NotificationSettingResponse{
		EventType: string(s.EventType),
		Enabled:   s.InSystem,
	}
}
