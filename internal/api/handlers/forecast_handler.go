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

type ForecastHandler struct {
	service *services.MonteCarloForecastService
}

func NewForecastHandler(service *services.MonteCarloForecastService) *ForecastHandler {
	return &ForecastHandler{service: service}
}

// GenerateForecast запускает прогнозирование методом Монте-Карло.
func (h *ForecastHandler) GenerateForecast(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор проекта")
		return
	}

	var req dto.MonteCarloForecastRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}

	fReq := domain.ForecastRequest{
		ProjectID:        projectID,
		WorkItemCount:    req.WorkItemCount,
		Simulations:      req.Simulations,
		ConfidenceLevels: req.ConfidenceLevels,
	}

	result, err := h.service.GenerateForecast(c.Request.Context(), fReq)
	if err != nil {
		writeError(c, http.StatusBadRequest, "FORECAST_ERROR", err.Error())
		return
	}

	writeSuccess(c, mapForecastToDTO(result))
}

func mapForecastToDTO(f *domain.ForecastResult) dto.MonteCarloForecastResultDTO {
	points := make([]dto.ForecastPointDTO, 0, len(f.Points))
	for _, p := range f.Points {
		points = append(points, dto.ForecastPointDTO{
			Date:        p.Date.Format("2006-01-02"),
			Probability: p.Probability,
		})
	}

	return dto.MonteCarloForecastResultDTO{
		ProjectID:     f.ProjectID.String(),
		WorkItemCount: f.WorkItemCount,
		Points:        points,
		GeneratedAt:   f.GeneratedAt.Format(time.RFC3339),
	}
}

