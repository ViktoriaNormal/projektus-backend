package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

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

func (h *KanbanAnalyticsHandler) GetCumulativeFlow(c *gin.Context) {
	projectID, ok := paramUUID(c, "projectId")
	if !ok {
		return
	}

	report, err := h.analyticsSvc.GetCumulativeFlow(c.Request.Context(), projectID, parseBoardID(c), parseFieldFilters(c))
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить данные накопительного потока")
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
	projectID, ok := paramUUID(c, "projectId")
	if !ok {
		return
	}

	report, err := h.analyticsSvc.GetCycleTimeScatter(c.Request.Context(), projectID, parseBoardID(c), parseFieldFilters(c))
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить данные cycle time scatter")
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
	projectID, ok := paramUUID(c, "projectId")
	if !ok {
		return
	}

	report, err := h.analyticsSvc.GetThroughput(c.Request.Context(), projectID, parseBoardID(c), parseFieldFilters(c))
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить данные throughput")
		return
	}

	data := make([]dto.ThroughputPointDTO, 0, len(report.Points))
	for _, p := range report.Points {
		data = append(data, dto.ThroughputPointDTO{
			Week:   p.Week,
			Actual: p.Actual,
			Trend:  p.Trend,
		})
	}

	writeSuccess(c, dto.ThroughputResponse{
		Data:           data,
		Interpretation: report.Interpretation,
	})
}

func (h *KanbanAnalyticsHandler) GetWipAge(c *gin.Context) {
	projectID, ok := paramUUID(c, "projectId")
	if !ok {
		return
	}

	report, err := h.analyticsSvc.GetWipAge(c.Request.Context(), projectID, parseBoardID(c), parseFieldFilters(c))
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить данные возраста WIP")
		return
	}

	data := make([]dto.WipAgePointDTO, 0, len(report.Points))
	for _, p := range report.Points {
		data = append(data, dto.WipAgePointDTO{
			TaskKey:    p.TaskKey,
			AgeDays:    p.AgeDays,
			ColumnName: p.ColumnName,
		})
	}

	writeSuccess(c, dto.WipAgeResponse{
		Data:           data,
		Interpretation: report.Interpretation,
	})
}

func (h *KanbanAnalyticsHandler) GetWipHistory(c *gin.Context) {
	projectID, ok := paramUUID(c, "projectId")
	if !ok {
		return
	}

	report, err := h.analyticsSvc.GetWipHistory(c.Request.Context(), projectID, parseBoardID(c), parseFieldFilters(c))
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить историю WIP")
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
	projectID, ok := paramUUID(c, "projectId")
	if !ok {
		return
	}

	report, err := h.analyticsSvc.GetCycleTimeDistribution(c.Request.Context(), projectID, parseBoardID(c), parseFieldFilters(c))
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить распределение cycle time")
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
	projectID, ok := paramUUID(c, "projectId")
	if !ok {
		return
	}

	report, err := h.analyticsSvc.GetThroughputDistribution(c.Request.Context(), projectID, parseBoardID(c), parseFieldFilters(c))
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить распределение throughput")
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

func (h *KanbanAnalyticsHandler) GetMonteCarlo(c *gin.Context) {
	projectID, ok := paramUUID(c, "projectId")
	if !ok {
		return
	}

	taskCountStr := c.Query("task_count")
	if taskCountStr == "" {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Параметр task_count обязателен")
		return
	}
	taskCount, err := strconv.Atoi(taskCountStr)
	if err != nil || taskCount < 1 {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "task_count должен быть положительным целым числом")
		return
	}

	weeks := 0
	if w := c.Query("weeks"); w != "" {
		if parsed, err := strconv.Atoi(w); err == nil && parsed >= 2 {
			weeks = parsed
		}
	}

	var targetDate *time.Time
	if s := c.Query("target_date"); s != "" {
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "target_date должен быть в формате YYYY-MM-DD")
			return
		}
		targetDate = &t
	}

	report, err := h.analyticsSvc.GetMonteCarlo(
		c.Request.Context(), projectID, parseBoardID(c), parseFieldFilters(c),
		taskCount, weeks, targetDate,
	)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось выполнить прогнозирование Монте-Карло")
		return
	}

	resp := dto.MonteCarloResponse{
		Percentiles:           make([]dto.MonteCarloPercentileDTO, 0, len(report.Percentiles)),
		Chart:                 make([]dto.MonteCarloChartPointDTO, 0, len(report.ChartPoints)),
		TargetDateProbability: report.TargetDateProbability,
	}
	for _, p := range report.Percentiles {
		resp.Percentiles = append(resp.Percentiles, dto.MonteCarloPercentileDTO{
			Percentile: p.Percentile,
			Date:       p.Date.Format("2006-01-02"),
		})
	}
	for _, cp := range report.ChartPoints {
		resp.Chart = append(resp.Chart, dto.MonteCarloChartPointDTO{
			Date:        cp.Date.Format("02.01"),
			Probability: cp.Probability,
		})
	}

	writeSuccess(c, resp)
}
