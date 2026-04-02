package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"projektus-backend/internal/api/dto"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/services"
)

type MeetingHandler struct {
	svc services.MeetingService
}

func NewMeetingHandler(svc services.MeetingService) *MeetingHandler {
	return &MeetingHandler{svc: svc}
}

func mapMeetingToResponse(m *domain.Meeting) dto.MeetingResponse {
	resp := dto.MeetingResponse{
		ID:          m.ID,
		OrganizerID: m.CreatedBy,
		Name:        m.Name,
		MeetingType: string(m.Type),
		Location:    m.Location,
		Status:      string(m.Status),
		StartTime:   m.StartTime.Format(time.RFC3339),
		EndTime:     m.EndTime.Format(time.RFC3339),
	}
	if m.ProjectID != nil {
		resp.ProjectID = m.ProjectID
	}
	if m.Description != nil {
		resp.Description = m.Description
	}
	return resp
}

func mapParticipantToResponse(p *domain.MeetingParticipant) dto.MeetingParticipantResponse {
	return dto.MeetingParticipantResponse{
		ID:        p.ID,
		MeetingID: p.MeetingID,
		UserID:    p.UserID,
		Status:    string(p.Status),
	}
}

// GET /api/v1/meetings
func (h *MeetingHandler) ListUserMeetings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Требуется аутентификация")
		return
	}

	var fromPtr, toPtr *time.Time
	if fromStr := c.Query("from"); fromStr != "" {
		t, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный параметр from")
			return
		}
		fromPtr = &t
	}
	if toStr := c.Query("to"); toStr != "" {
		t, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный параметр to")
			return
		}
		toPtr = &t
	}

	meetings, err := h.svc.ListUserMeetings(c.Request.Context(), userID, fromPtr, toPtr)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка сервера")
		return
	}

	resp := make([]dto.MeetingResponse, len(meetings))
	for i := range meetings {
		resp[i] = mapMeetingToResponse(&meetings[i])
	}

	writeSuccess(c, resp)
}

// POST /api/v1/meetings
func (h *MeetingHandler) CreateMeeting(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Требуется аутентификация")
		return
	}

	var req dto.CreateMeetingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}

	start, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректное время начала")
		return
	}
	end, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректное время окончания")
		return
	}

	m := domain.Meeting{
		Name:      req.Name,
		StartTime: start,
		EndTime:   end,
	}
	if req.Description != nil {
		m.Description = req.Description
	}
	if req.ProjectID != nil {
		m.ProjectID = req.ProjectID
	}
	if req.MeetingType != nil {
		m.Type = domain.MeetingType(*req.MeetingType)
	}
	m.Location = &req.Location

	created, participants, err := h.svc.CreateMeeting(c.Request.Context(), userID, m, req.ParticipantIDs)
	if err != nil {
		switch err {
		case domain.ErrMeetingInPast:
			writeError(c, http.StatusBadRequest, "MEETING_IN_PAST", "Нельзя создать встречу на прошедшее время")
		case domain.ErrInvalidTimeRange:
			writeError(c, http.StatusBadRequest, "INVALID_TIME_RANGE", "Время окончания должно быть позже времени начала")
		case domain.ErrInvalidMeeting:
			writeError(c, http.StatusBadRequest, "INVALID_MEETING", "Некорректные параметры встречи")
		default:
			writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка сервера")
		}
		return
	}

	partResp := make([]dto.MeetingParticipantResponse, len(participants))
	for i := range participants {
		partResp[i] = mapParticipantToResponse(&participants[i])
	}

	writeSuccess(c, dto.MeetingDetailsResponse{
		MeetingResponse: mapMeetingToResponse(created),
		Participants:    partResp,
	})
}

// GET /api/v1/meetings/:meetingId
func (h *MeetingHandler) GetMeeting(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Требуется аутентификация")
		return
	}
	meetingID := c.Param("meetingId")

	m, parts, err := h.svc.GetMeetingWithParticipants(c.Request.Context(), userID, meetingID)
	if err != nil {
		switch err {
		case domain.ErrNotFound:
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Встреча не найдена")
		case domain.ErrAccessDenied:
			writeError(c, http.StatusForbidden, "ACCESS_DENIED", "Нет доступа к этой встрече")
		default:
			writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка сервера")
		}
		return
	}

	partResp := make([]dto.MeetingParticipantResponse, len(parts))
	for i := range parts {
		partResp[i] = mapParticipantToResponse(&parts[i])
	}

	writeSuccess(c, dto.MeetingDetailsResponse{
		MeetingResponse: mapMeetingToResponse(m),
		Participants:    partResp,
	})
}

// PATCH /api/v1/meetings/:meetingId
func (h *MeetingHandler) UpdateMeeting(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Требуется аутентификация")
		return
	}
	meetingID := c.Param("meetingId")

	var req dto.UpdateMeetingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}

	// Fetch existing meeting to build a complete domain object.
	existing, _, err := h.svc.GetMeetingWithParticipants(c.Request.Context(), userID, meetingID)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Встреча не найдена")
			return
		}
		if err == domain.ErrAccessDenied {
			writeError(c, http.StatusForbidden, "ACCESS_DENIED", "Нет прав на изменение встречи")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка сервера")
		return
	}

	m := *existing
	if req.Name != nil {
		m.Name = *req.Name
	}
	if req.Description.Set {
		m.Description = req.Description.Ptr()
	}
	if req.MeetingType != nil {
		m.Type = domain.MeetingType(*req.MeetingType)
	}
	if req.Location.Set {
		m.Location = req.Location.Ptr()
	}
	if req.StartTime != nil {
		t, err := time.Parse(time.RFC3339, *req.StartTime)
		if err != nil {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректное время начала")
			return
		}
		m.StartTime = t
	}
	if req.EndTime != nil {
		t, err := time.Parse(time.RFC3339, *req.EndTime)
		if err != nil {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректное время окончания")
			return
		}
		m.EndTime = t
	}

	updated, err := h.svc.UpdateMeeting(c.Request.Context(), userID, m)
	if err != nil {
		switch err {
		case domain.ErrNotFound:
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Встреча не найдена")
		case domain.ErrAccessDenied:
			writeError(c, http.StatusForbidden, "ACCESS_DENIED", "Нет прав на изменение встречи")
		case domain.ErrMeetingInPast:
			writeError(c, http.StatusBadRequest, "MEETING_IN_PAST", "Нельзя назначить встречу на прошедшее время")
		case domain.ErrInvalidTimeRange:
			writeError(c, http.StatusBadRequest, "INVALID_TIME_RANGE", "Время окончания должно быть позже времени начала")
		case domain.ErrInvalidMeeting:
			writeError(c, http.StatusBadRequest, "INVALID_MEETING", "Некорректные параметры встречи")
		default:
			writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка сервера")
		}
		return
	}

	writeSuccess(c, mapMeetingToResponse(updated))
}

// POST /api/v1/meetings/:meetingId/cancel
func (h *MeetingHandler) CancelMeeting(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Требуется аутентификация")
		return
	}
	meetingID := c.Param("meetingId")

	cancelled, err := h.svc.CancelMeeting(c.Request.Context(), userID, meetingID)
	if err != nil {
		switch err {
		case domain.ErrNotFound:
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Встреча не найдена")
		case domain.ErrAccessDenied:
			writeError(c, http.StatusForbidden, "FORBIDDEN", "Пользователь не является организатором")
		case domain.ErrAlreadyCancelled:
			writeError(c, http.StatusBadRequest, "ALREADY_CANCELLED", "Встреча уже отменена")
		default:
			writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка сервера")
		}
		return
	}

	writeSuccess(c, mapMeetingToResponse(cancelled))
}

// GET /api/v1/meetings/:meetingId/participants
func (h *MeetingHandler) ListParticipants(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Требуется аутентификация")
		return
	}
	meetingID := c.Param("meetingId")

	m, parts, err := h.svc.GetMeetingWithParticipants(c.Request.Context(), userID, meetingID)
	if err != nil {
		switch err {
		case domain.ErrNotFound:
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Встреча не найдена")
		case domain.ErrAccessDenied:
			writeError(c, http.StatusForbidden, "ACCESS_DENIED", "Нет доступа к этой встрече")
		default:
			writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка сервера")
		}
		return
	}

	_ = m // данные встречи здесь не нужны, только проверка прав

	resp := make([]dto.MeetingParticipantResponse, len(parts))
	for i := range parts {
		resp[i] = mapParticipantToResponse(&parts[i])
	}
	writeSuccess(c, gin.H{
		"participants": resp,
	})
}

// POST /api/v1/meetings/:meetingId/participants
func (h *MeetingHandler) AddParticipants(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Требуется аутентификация")
		return
	}
	meetingID := c.Param("meetingId")

	var body struct {
		UserIDs []string `json:"user_ids"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}

	parts, err := h.svc.AddParticipants(c.Request.Context(), userID, meetingID, body.UserIDs)
	if err != nil {
		switch err {
		case domain.ErrNotFound:
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Встреча не найдена")
		case domain.ErrAccessDenied:
			writeError(c, http.StatusForbidden, "ACCESS_DENIED", "Нет прав на добавление участников")
		default:
			writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка сервера")
		}
		return
	}

	resp := make([]dto.MeetingParticipantResponse, len(parts))
	for i := range parts {
		resp[i] = mapParticipantToResponse(&parts[i])
	}

	writeSuccess(c, gin.H{
		"participants": resp,
	})
}

// POST /api/v1/meetings/:meetingId/response
func (h *MeetingHandler) RespondToInvitation(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Требуется аутентификация")
		return
	}
	meetingID := c.Param("meetingId")

	var body struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}

	var st domain.ParticipantStatus
	switch body.Status {
	case "accepted":
		st = domain.ParticipantStatusAccepted
	case "declined":
		st = domain.ParticipantStatusDeclined
	default:
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Недопустимый статус ответа")
		return
	}

	if err := h.svc.RespondToInvitation(c.Request.Context(), userID, meetingID, st); err != nil {
		switch err {
		case domain.ErrNotFound:
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Встреча не найдена")
		default:
			writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка сервера")
		}
		return
	}

	c.Status(http.StatusNoContent)
}

