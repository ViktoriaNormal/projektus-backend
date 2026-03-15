package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projektus-backend/internal/api/dto"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/services"
)

type ScrumAnalyticsHandler struct {
	service *services.ScrumAnalyticsService
}

func NewScrumAnalyticsHandler(service *services.ScrumAnalyticsService) *ScrumAnalyticsHandler {
	return &ScrumAnalyticsHandler{service: service}
}

// GetVelocity возвращает данные для графика скорости команды.
func (h *ScrumAnalyticsHandler) GetVelocity(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор проекта")
		return
	}

	points, err := h.service.GetVelocityData(c.Request.Context(), projectID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить данные скорости")
		return
	}

	resp := dto.ScrumVelocityReport{
		Sprints: make([]dto.VelocitySprintDTO, 0, len(points)),
	}
	for _, p := range points {
		resp.Sprints = append(resp.Sprints, dto.VelocitySprintDTO{
			SprintID:        p.SprintID,
			Name:            p.SprintName,
			CommittedPoints: p.CommittedPoints,
			CompletedPoints: p.CompletedPoints,
		})
	}

	writeSuccess(c, resp)
}

// GetBurndown возвращает данные для диаграммы сгорания спринта.
func (h *ScrumAnalyticsHandler) GetBurndown(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	if _, err := uuid.Parse(projectIDStr); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор проекта")
		return
	}

	sprintIDStr := c.Query("sprintId")
	if sprintIDStr == "" {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Необходимо указать sprintId")
		return
	}
	sprintID, err := uuid.Parse(sprintIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор спринта")
		return
	}

	data, err := h.service.GetBurndownData(c.Request.Context(), sprintID)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Спринт не найден")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить данные burndown")
		return
	}

	resp := dto.BurndownReportDTO{
		SprintID: data.SprintID,
		Points:   make([]dto.BurndownPointDTO, 0, len(data.Points)),
	}
	for _, p := range data.Points {
		resp.Points = append(resp.Points, dto.BurndownPointDTO{
			Date:            p.Date.Format("2006-01-02"),
			RemainingPoints: p.RemainingPoints,
			IdealPoints:     p.IdealPoints,
		})
	}

	writeSuccess(c, resp)
}

