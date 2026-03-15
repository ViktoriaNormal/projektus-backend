package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projektus-backend/internal/api/dto"
	"projektus-backend/internal/services"
)

type AdminAuditLogHandler struct {
	auditLogSvc *services.AuditLogService
}

func NewAdminAuditLogHandler(auditLogSvc *services.AuditLogService) *AdminAuditLogHandler {
	return &AdminAuditLogHandler{auditLogSvc: auditLogSvc}
}

// GetLogs GET /admin/logs — журнал действий с фильтрами.
func (h *AdminAuditLogHandler) GetLogs(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")
	userIDStr := c.Query("userId")
	actionType := c.Query("actionType")
	fromStr := c.Query("from")
	toStr := c.Query("to")

	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset, _ := strconv.Atoi(offsetStr)
	if offset < 0 {
		offset = 0
	}

	filter := services.ListAuditLogsFilter{
		Limit:  int32(limit),
		Offset: int32(offset),
	}
	if userIDStr != "" {
		if uid, err := uuid.Parse(userIDStr); err == nil {
			filter.UserID = &uid
		}
	}
	if actionType != "" {
		filter.ActionType = &actionType
	}
	if fromStr != "" {
		if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
			filter.From = &t
		}
	}
	if toStr != "" {
		if t, err := time.Parse(time.RFC3339, toStr); err == nil {
			filter.To = &t
		}
	}

	entries, total, err := h.auditLogSvc.List(c.Request.Context(), filter)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить журнал")
		return
	}

	resp := make([]dto.AuditLogEntryResponse, 0, len(entries))
	for _, e := range entries {
		r := dto.AuditLogEntryResponse{
			ID:         e.ID.String(),
			UserID:     e.UserID.String(),
			ActionType: e.ActionType,
			CreatedAt:  e.CreatedAt.Format(time.RFC3339),
			Metadata:   e.Metadata,
		}
		if e.EntityType != nil {
			r.EntityType = e.EntityType
		}
		if e.EntityID != nil {
			s := e.EntityID.String()
			r.EntityID = &s
		}
		resp = append(resp, r)
	}

	writeSuccess(c, gin.H{
		"entries": resp,
		"total":   total,
	})
}
