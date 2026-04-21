package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

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
		ID:          m.ID.String(),
		OrganizerID: m.CreatedBy.String(),
		Name:        m.Name,
		MeetingType: string(m.Type),
		Location:    m.Location,
		Status:      string(m.Status),
		StartTime:   m.StartTime.Format(time.RFC3339),
		EndTime:     m.EndTime.Format(time.RFC3339),
	}
	if m.ProjectID != nil {
		pid := m.ProjectID.String()
		resp.ProjectID = &pid
	}
	if m.Description != nil {
		resp.Description = m.Description
	}
	return resp
}

func mapParticipantToResponse(p *domain.MeetingParticipant) dto.MeetingParticipantResponse {
	return dto.MeetingParticipantResponse{
		ID:        p.ID.String(),
		MeetingID: p.MeetingID.String(),
		UserID:    p.UserID.String(),
		Status:    string(p.Status),
	}
}

// GET /api/v1/meetings
func (h *MeetingHandler) ListUserMeetings(c *gin.Context) {
	userUUID, ok := requireUserUUID(c)
	if !ok {
		return
	}
	userID := userUUID.String()

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

	// Всегда возвращаем только встречи, в которых пользователь — организатор
	// или участник. Для администраторов исключений нет.
	meetings, err := h.svc.ListUserMeetings(c.Request.Context(), userID, fromPtr, toPtr)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Внутренняя ошибка сервера")
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
	userUUID, ok := requireUserUUID(c)
	if !ok {
		return
	}
	userID := userUUID.String()

	req, ok := bindJSON[dto.CreateMeetingRequest](c)
	if !ok {
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
		pid, err := uuid.Parse(*req.ProjectID)
		if err != nil {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный project_id")
			return
		}
		m.ProjectID = &pid
	}
	if req.MeetingType != nil {
		m.Type = domain.MeetingType(*req.MeetingType)
	}
	m.Location = &req.Location

	created, participants, err := h.svc.CreateMeeting(c.Request.Context(), userID, m, req.ParticipantIDs)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Внутренняя ошибка сервера")
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
	userUUID, ok := requireUserUUID(c)
	if !ok {
		return
	}
	userID := userUUID.String()
	meetingID := c.Param("meetingId")

	m, parts, err := h.svc.GetMeetingWithParticipants(c.Request.Context(), userID, meetingID)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Внутренняя ошибка сервера")
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
	userUUID, ok := requireUserUUID(c)
	if !ok {
		return
	}
	userID := userUUID.String()
	meetingID := c.Param("meetingId")

	req, ok := bindJSON[dto.UpdateMeetingRequest](c)
	if !ok {
		return
	}

	// Fetch existing meeting to build a complete domain object.
	existing, _, err := h.svc.GetMeetingWithParticipants(c.Request.Context(), userID, meetingID)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Внутренняя ошибка сервера")
		return
	}

	m := *existing
	if req.ProjectID.Set {
		if ptr := req.ProjectID.Ptr(); ptr != nil {
			pid, err := uuid.Parse(*ptr)
			if err != nil {
				writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный project_id")
				return
			}
			m.ProjectID = &pid
		} else {
			m.ProjectID = nil
		}
	}
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
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Внутренняя ошибка сервера")
		return
	}

	writeSuccess(c, mapMeetingToResponse(updated))
}

// POST /api/v1/meetings/:meetingId/cancel
func (h *MeetingHandler) CancelMeeting(c *gin.Context) {
	userUUID, ok := requireUserUUID(c)
	if !ok {
		return
	}
	userID := userUUID.String()
	meetingID := c.Param("meetingId")

	cancelled, err := h.svc.CancelMeeting(c.Request.Context(), userID, meetingID)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Внутренняя ошибка сервера")
		return
	}

	writeSuccess(c, mapMeetingToResponse(cancelled))
}

// GET /api/v1/meetings/:meetingId/participants
func (h *MeetingHandler) ListParticipants(c *gin.Context) {
	userUUID, ok := requireUserUUID(c)
	if !ok {
		return
	}
	userID := userUUID.String()
	meetingID := c.Param("meetingId")

	m, parts, err := h.svc.GetMeetingWithParticipants(c.Request.Context(), userID, meetingID)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Внутренняя ошибка сервера")
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
	userUUID, ok := requireUserUUID(c)
	if !ok {
		return
	}
	userID := userUUID.String()
	meetingID := c.Param("meetingId")

	type addParticipantsBody struct {
		UserIDs []string `json:"user_ids"`
	}
	body, ok := bindJSON[addParticipantsBody](c)
	if !ok {
		return
	}

	parts, err := h.svc.AddParticipants(c.Request.Context(), userID, meetingID, body.UserIDs)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Внутренняя ошибка сервера")
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
	userUUID, ok := requireUserUUID(c)
	if !ok {
		return
	}
	userID := userUUID.String()
	meetingID := c.Param("meetingId")

	type respondBody struct {
		Status string `json:"status" binding:"required"`
	}
	body, ok := bindJSON[respondBody](c)
	if !ok {
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
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Внутренняя ошибка сервера")
		return
	}

	c.Status(http.StatusNoContent)
}
