package handlers

import (
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projektus-backend/internal/api/dto"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/services"
)

type ScrumAnalyticsHandler struct {
	analyticsSvc *services.ScrumAnalyticsService
}

func NewScrumAnalyticsHandler(analyticsSvc *services.ScrumAnalyticsService) *ScrumAnalyticsHandler {
	return &ScrumAnalyticsHandler{analyticsSvc: analyticsSvc}
}

func (h *ScrumAnalyticsHandler) GetVelocity(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор проекта")
		return
	}

	metricType := parseMetricType(c.Query("metricType"))
	limit := 0
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	report, err := h.analyticsSvc.GetVelocity(c.Request.Context(), projectID, metricType, limit)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить данные velocity")
		return
	}

	data := make([]dto.VelocitySprintData, 0, len(report.Data))
	for _, d := range report.Data {
		data = append(data, dto.VelocitySprintData{
			Sprint:    d.SprintName,
			SprintID:  d.SprintID.String(),
			Planned:   int(math.Round(d.Planned)),
			Completed: int(math.Round(d.Completed)),
		})
	}

	writeSuccess(c, dto.VelocityResponse{
		Data: data,
		Metrics: dto.VelocityMetrics{
			AverageVelocity:    math.Round(report.AverageVelocity*100) / 100,
			VelocityTrend:      report.VelocityTrend,
			CompletionRate:     report.CompletionRate,
			AverageSprintScope: math.Round(report.AverageSprintScope*100) / 100,
			SprintCount:        report.SprintCount,
		},
		Interpretation: report.Interpretation,
	})
}

func (h *ScrumAnalyticsHandler) GetBurndown(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор проекта")
		return
	}

	metricType := parseMetricType(c.Query("metricType"))

	var sprintIDPtr *uuid.UUID
	if sprintIDStr := c.Query("sprintId"); sprintIDStr != "" {
		sprintID, err := uuid.Parse(sprintIDStr)
		if err != nil {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор спринта")
			return
		}
		sprintIDPtr = &sprintID
	}

	report, err := h.analyticsSvc.GetBurndown(c.Request.Context(), projectID, metricType, sprintIDPtr)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Активный спринт не найден")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить данные burndown")
		return
	}

	data := make([]dto.BurndownDayData, 0, len(report.Data))
	for _, d := range report.Data {
		data = append(data, dto.BurndownDayData{
			Day:       d.Day,
			Remaining: d.Remaining,
			Ideal:     d.Ideal,
		})
	}

	writeSuccess(c, dto.BurndownResponse{
		Data:           data,
		SprintName:     report.SprintName,
		Interpretation: report.Interpretation,
	})
}

func parseMetricType(s string) services.MetricType {
	switch s {
	case "story_points":
		return services.MetricStoryPoints
	case "estimation_hours":
		return services.MetricEstimationHours
	default:
		return services.MetricTaskCount
	}
}
