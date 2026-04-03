package handlers

import (
	"math"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projektus-backend/internal/api/dto"
	"projektus-backend/internal/services"
)

type KanbanAnalyticsHandler struct {
	analyticsSvc *services.KanbanAnalyticsService
}

func NewKanbanAnalyticsHandler(analyticsSvc *services.KanbanAnalyticsService) *KanbanAnalyticsHandler {
	return &KanbanAnalyticsHandler{analyticsSvc: analyticsSvc}
}

func parseBoardID(c *gin.Context) *uuid.UUID {
	if s := c.Query("board_id"); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			return &id
		}
	}
	return nil
}

func parseFieldFilters(c *gin.Context) map[string][]string {
	filters := make(map[string][]string)
	for key, values := range c.Request.URL.Query() {
		if !strings.HasPrefix(key, "filter_") {
			continue
		}
		fieldID := strings.TrimPrefix(key, "filter_")
		if fieldID == "" {
			continue
		}
		var all []string
		for _, v := range values {
			for _, part := range strings.Split(v, ",") {
				part = strings.TrimSpace(part)
				if part != "" {
					all = append(all, part)
				}
			}
		}
		if len(all) > 0 {
			filters[fieldID] = all
		}
	}
	return filters
}

func (h *KanbanAnalyticsHandler) GetSummary(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор проекта")
		return
	}

	report, err := h.analyticsSvc.GetSummary(c.Request.Context(), projectID, parseBoardID(c), parseFieldFilters(c))
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить сводные данные Kanban")
		return
	}

	writeSuccess(c, dto.KanbanSummaryResponse{
		Data: dto.KanbanSummaryData{
			AverageVelocity:     math.Round(report.AverageVelocity*10) / 10,
			AverageVelocityUnit: report.AverageVelocityUnit,
			VelocityTrend:       report.VelocityTrend,
			CycleTime:           report.CycleTime,
			CycleTimeTrend:      report.CycleTimeTrend,
			Throughput:          report.Throughput,
			ThroughputTrend:     report.ThroughputTrend,
			Wip:                 report.Wip,
			WipChange:           report.WipChange,
		},
		Interpretation: report.Interpretation,
	})
}

func (h *KanbanAnalyticsHandler) GetCumulativeFlow(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор проекта")
		return
	}

	report, err := h.analyticsSvc.GetCumulativeFlow(c.Request.Context(), projectID, parseBoardID(c), parseFieldFilters(c))
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить данные накопительного потока")
		return
	}

	data := make([]map[string]interface{}, 0, len(report.Points))
	for _, p := range report.Points {
		point := map[string]interface{}{"date": p.Date}
		for _, col := range report.ColumnNames {
			point[col] = p.Counts[col]
		}
		data = append(data, point)
	}

	writeSuccess(c, dto.CumulativeFlowResponse{
		Data:           data,
		Interpretation: report.Interpretation,
	})
}

func (h *KanbanAnalyticsHandler) GetCycleTimeScatter(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор проекта")
		return
	}

	report, err := h.analyticsSvc.GetCycleTimeScatter(c.Request.Context(), projectID, parseBoardID(c), parseFieldFilters(c))
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить данные cycle time scatter")
		return
	}

	data := make([]dto.CycleTimeScatterPointDTO, 0, len(report.Points))
	for _, p := range report.Points {
		data = append(data, dto.CycleTimeScatterPointDTO{
			Task: p.TaskKey,
			Time: p.CycleTimeDays,
		})
	}

	writeSuccess(c, dto.CycleTimeScatterResponse{
		Data:           data,
		Interpretation: report.Interpretation,
	})
}

func (h *KanbanAnalyticsHandler) GetThroughput(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор проекта")
		return
	}

	report, err := h.analyticsSvc.GetThroughput(c.Request.Context(), projectID, parseBoardID(c), parseFieldFilters(c))
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить данные throughput")
		return
	}

	data := make([]dto.ThroughputWeekDTO, 0, len(report.Points))
	for _, p := range report.Points {
		data = append(data, dto.ThroughputWeekDTO{
			Week:  p.Week,
			Count: p.Count,
		})
	}

	writeSuccess(c, dto.ThroughputResponse{
		Data:           data,
		Interpretation: report.Interpretation,
	})
}

func (h *KanbanAnalyticsHandler) GetAvgCycleTime(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор проекта")
		return
	}

	report, err := h.analyticsSvc.GetAvgCycleTime(c.Request.Context(), projectID, parseBoardID(c), parseFieldFilters(c))
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить данные среднего cycle time")
		return
	}

	data := make([]dto.AvgCycleTimeWeekDTO, 0, len(report.Points))
	for _, p := range report.Points {
		data = append(data, dto.AvgCycleTimeWeekDTO{
			Week: p.Week,
			Avg:  p.Avg,
			P50:  p.P50,
			P85:  p.P85,
		})
	}

	writeSuccess(c, dto.AvgCycleTimeResponse{
		Data:           data,
		Interpretation: report.Interpretation,
	})
}

func (h *KanbanAnalyticsHandler) GetThroughputTrend(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор проекта")
		return
	}

	report, err := h.analyticsSvc.GetThroughputTrend(c.Request.Context(), projectID, parseBoardID(c), parseFieldFilters(c))
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить тренд throughput")
		return
	}

	data := make([]dto.ThroughputTrendPointDTO, 0, len(report.Points))
	for _, p := range report.Points {
		data = append(data, dto.ThroughputTrendPointDTO{
			Week:   p.Week,
			Actual: p.Actual,
			Trend:  p.Trend,
		})
	}

	writeSuccess(c, dto.ThroughputTrendResponse{
		Data:           data,
		Interpretation: report.Interpretation,
	})
}

func (h *KanbanAnalyticsHandler) GetWipHistory(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор проекта")
		return
	}

	report, err := h.analyticsSvc.GetWipHistory(c.Request.Context(), projectID, parseBoardID(c), parseFieldFilters(c))
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить историю WIP")
		return
	}

	data := make([]dto.WipHistoryPointDTO, 0, len(report.Points))
	for _, p := range report.Points {
		data = append(data, dto.WipHistoryPointDTO{
			Date:  p.Date,
			Wip:   p.Wip,
			Limit: p.Limit,
		})
	}

	writeSuccess(c, dto.WipHistoryResponse{
		Data:           data,
		Interpretation: report.Interpretation,
	})
}

func (h *KanbanAnalyticsHandler) GetCycleTimeDistribution(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор проекта")
		return
	}

	report, err := h.analyticsSvc.GetCycleTimeDistribution(c.Request.Context(), projectID, parseBoardID(c), parseFieldFilters(c))
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить распределение cycle time")
		return
	}

	data := make([]dto.DistributionBucketDTO, 0, len(report.Buckets))
	for _, b := range report.Buckets {
		data = append(data, dto.DistributionBucketDTO{
			Range: b.RangeLabel,
			Count: b.Count,
		})
	}

	writeSuccess(c, dto.DistributionResponse{
		Data:           data,
		Interpretation: report.Interpretation,
	})
}

func (h *KanbanAnalyticsHandler) GetThroughputDistribution(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор проекта")
		return
	}

	report, err := h.analyticsSvc.GetThroughputDistribution(c.Request.Context(), projectID, parseBoardID(c), parseFieldFilters(c))
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить распределение throughput")
		return
	}

	data := make([]dto.DistributionBucketDTO, 0, len(report.Buckets))
	for _, b := range report.Buckets {
		data = append(data, dto.DistributionBucketDTO{
			Range: b.RangeLabel,
			Count: b.Count,
		})
	}

	writeSuccess(c, dto.DistributionResponse{
		Data:           data,
		Interpretation: report.Interpretation,
	})
}
