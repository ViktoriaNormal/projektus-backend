package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"projektus-backend/internal/api/dto"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/services"
)

type NotificationHandler struct {
	svc services.NotificationService
}

func NewNotificationHandler(svc services.NotificationService) *NotificationHandler {
	return &NotificationHandler{svc: svc}
}

// GET /api/v1/notifications?unreadOnly=false&limit=20&offset=0
func (h *NotificationHandler) GetFeed(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Требуется аутентификация")
		return
	}

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
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить уведомления")
		return
	}

	resp := make([]dto.NotificationResponse, 0, len(items))
	for _, n := range items {
		resp = append(resp, mapNotificationToDTO(n))
	}

	writeSuccess(c, dto.NotificationFeedResponse{
		Items:       resp,
		UnreadCount: unreadCount,
	})
}

// PATCH /api/v1/notifications/:notificationId/read
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Требуется аутентификация")
		return
	}

	notificationID := c.Param("notificationId")

	if err := h.svc.MarkAsRead(c.Request.Context(), userID, notificationID); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось отметить уведомление прочитанным")
		return
	}

	writeSuccess(c, gin.H{"message": "Уведомление отмечено прочитанным"})
}

// POST /api/v1/notifications/read-all
func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Требуется аутентификация")
		return
	}

	if err := h.svc.MarkAllAsRead(c.Request.Context(), userID); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось отметить уведомления прочитанными")
		return
	}

	writeSuccess(c, gin.H{"message": "Все уведомления отмечены прочитанными"})
}

// GET /api/v1/notifications/settings
func (h *NotificationHandler) GetSettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Требуется аутентификация")
		return
	}

	settings, err := h.svc.GetSettings(c.Request.Context(), userID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить настройки уведомлений")
		return
	}

	resp := make([]dto.NotificationSettingResponse, 0, len(settings))
	for _, s := range settings {
		resp = append(resp, dto.NotificationSettingResponse{
			ID:                    s.ID,
			UserID:                s.UserID,
			EventType:             string(s.EventType),
			InSystem:              s.InSystem,
			InEmail:               s.InEmail,
			ReminderOffsetMinutes: s.ReminderOffsetMinutes,
		})
	}

	writeSuccess(c, resp)
}

// PUT /api/v1/notifications/settings
func (h *NotificationHandler) UpdateSettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Требуется аутентификация")
		return
	}

	var req []dto.UpdateNotificationSettingItem
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}

	settings := make([]domain.NotificationSetting, 0, len(req))
	for _, item := range req {
		s := domain.NotificationSetting{
			UserID:                userID,
			EventType:             domain.EventType(item.EventType),
			InSystem:              true,
			InEmail:               false,
			ReminderOffsetMinutes: item.ReminderOffsetMinutes,
		}
		if item.InSystem != nil {
			s.InSystem = *item.InSystem
		}
		if item.InEmail != nil {
			s.InEmail = *item.InEmail
		}
		settings = append(settings, s)
	}

	if err := h.svc.UpdateSettings(c.Request.Context(), userID, settings); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось сохранить настройки уведомлений")
		return
	}

	writeSuccess(c, gin.H{"message": "Настройки сохранены"})
}

func mapNotificationToDTO(n domain.Notification) dto.NotificationResponse {
	return dto.NotificationResponse{
		ID:        n.ID,
		UserID:    n.UserID,
		EventType: string(n.EventType),
		Channel:   string(n.Channel),
		Title:     n.Title,
		Body:      n.Body,
		IsRead:    n.IsRead,
		CreatedAt: n.CreatedAt,
	}
}
