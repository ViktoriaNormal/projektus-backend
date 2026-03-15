package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projektus-backend/internal/api/dto"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/services"
)

type KanbanAnalyticsHandler struct {
	service *services.KanbanAnalyticsService
}

func NewKanbanAnalyticsHandler(service *services.KanbanAnalyticsService) *KanbanAnalyticsHandler {
	return &KanbanAnalyticsHandler{service: service}
}

// parseDateRange читает from/to из query, по умолчанию последние 30 дней.
func parseDateRange(c *gin.Context) (start, end time.Time, err error) {
	now := time.Now().UTC()
	end = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, time.UTC)
	start = end.AddDate(0, 0, -30)

	if from := c.Query("from"); from != "" {
		if t, e := time.Parse("2006-01-02", from); e == nil {
			start = t
		}
	}
	if to := c.Query("to"); to != "" {
		if t, e := time.Parse("2006-01-02", to); e == nil {
			end = time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, time.UTC)
		}
	}
	if end.Before(start) {
		return time.Time{}, time.Time{}, nil
	}
	return start, end, nil
}

// GetCumulativeFlow возвращает данные для накопительной диаграммы потока (CFD).
// GET /projects/:projectId/analytics/kanban/cumulative-flow?boardId=&from=&to=
func (h *KanbanAnalyticsHandler) GetCumulativeFlow(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор проекта")
		return
	}

	start, end, err := parseDateRange(c)
	if err != nil || end.Before(start) {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный диапазон дат (from, to)")
		return
	}

	var boardID *uuid.UUID
	if s := c.Query("boardId"); s != "" {
		if id, e := uuid.Parse(s); e == nil {
			boardID = &id
		}
	}

	filter := services.KanbanAnalyticsFilter{
		ProjectID: projectID,
		BoardID:   boardID,
		StartDate: start,
		EndDate:   end,
	}
	points, err := h.service.GetCumulativeFlowData(c.Request.Context(), filter)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Доска не найдена")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить данные CFD")
		return
	}

	resp := make([]dto.CumulativeFlowPointDTO, 0, len(points))
	for _, p := range points {
		resp = append(resp, dto.CumulativeFlowPointDTO{
			Date:         p.Date.Format("2006-01-02"),
			StatusCounts: p.StatusCounts,
		})
	}
	writeSuccess(c, resp)
}

// GetThroughput возвращает данные для графика скорости поставки.
// GET /projects/:projectId/analytics/kanban/throughput?from=&to=&groupBy=day|week
func (h *KanbanAnalyticsHandler) GetThroughput(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор проекта")
		return
	}

	start, end, _ := parseDateRange(c)
	if end.Before(start) {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный диапазон дат (from, to)")
		return
	}

	groupBy := c.DefaultQuery("groupBy", "day")
	if groupBy != "day" && groupBy != "week" {
		groupBy = "day"
	}

	filter := services.KanbanAnalyticsFilter{
		ProjectID: projectID,
		StartDate: start,
		EndDate:   end,
		GroupBy:   groupBy,
	}
	points, err := h.service.GetThroughputData(c.Request.Context(), filter)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить данные throughput")
		return
	}

	resp := make([]dto.ThroughputPointDTO, 0, len(points))
	for _, p := range points {
		period := p.PeriodStart.Format("2006-01-02")
		resp = append(resp, dto.ThroughputPointDTO{
			Period:          period,
			ClassOfService:  p.ClassOfService,
			TaskCount:       p.TaskCount,
			CumulativeCount: p.CumulativeCount,
		})
	}
	writeSuccess(c, resp)
}

// GetWipOverTime возвращает WIP по дням.
// GET /projects/:projectId/analytics/kanban/wip/over-time?from=&to=
func (h *KanbanAnalyticsHandler) GetWipOverTime(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор проекта")
		return
	}

	start, end, _ := parseDateRange(c)
	if end.Before(start) {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный диапазон дат (from, to)")
		return
	}

	filter := services.KanbanAnalyticsFilter{
		ProjectID: projectID,
		StartDate: start,
		EndDate:   end,
	}
	points, err := h.service.GetWipOverTime(c.Request.Context(), filter)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить данные WIP")
		return
	}

	resp := make([]dto.WipPointDTO, 0, len(points))
	for _, p := range points {
		resp = append(resp, dto.WipPointDTO{
			Date:     p.Date.Format("2006-01-02"),
			WipCount: p.WipCount,
		})
	}
	writeSuccess(c, resp)
}

// GetWipAge возвращает WIP по дням с возрастом (средний/максимальный).
// GET /projects/:projectId/analytics/kanban/wip/age?from=&to=
func (h *KanbanAnalyticsHandler) GetWipAge(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор проекта")
		return
	}

	start, end, _ := parseDateRange(c)
	if end.Before(start) {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный диапазон дат (from, to)")
		return
	}

	filter := services.KanbanAnalyticsFilter{
		ProjectID: projectID,
		StartDate: start,
		EndDate:   end,
	}
	points, err := h.service.GetWipAgeChart(c.Request.Context(), filter)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить данные возраста WIP")
		return
	}

	resp := make([]dto.WipPointDTO, 0, len(points))
	for _, p := range points {
		resp = append(resp, dto.WipPointDTO{
			Date:      p.Date.Format("2006-01-02"),
			WipCount:  p.WipCount,
			AvgWipAge: p.AvgWipAge,
			MaxWipAge: p.MaxWipAge,
		})
	}
	writeSuccess(c, resp)
}

// GetCycleTimeScatterplot возвращает данные для диаграммы рассеяния времени производства.
// GET /projects/:projectId/analytics/kanban/cycle-time/scatterplot?from=&to=&classOfService=
func (h *KanbanAnalyticsHandler) GetCycleTimeScatterplot(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор проекта")
		return
	}

	start, end, _ := parseDateRange(c)
	if end.Before(start) {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный диапазон дат (from, to)")
		return
	}

	var classOfService *string
	if s := c.Query("classOfService"); s != "" {
		classOfService = &s
	}

	filter := services.KanbanAnalyticsFilter{
		ProjectID:      projectID,
		StartDate:      start,
		EndDate:        end,
		ClassOfService: classOfService,
	}
	points, err := h.service.GetCycleTimeScatterplot(c.Request.Context(), filter)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить данные scatterplot")
		return
	}

	resp := make([]dto.CycleTimePointDTO, 0, len(points))
	for _, p := range points {
		resp = append(resp, dto.CycleTimePointDTO{
			TaskID:         p.TaskID,
			TaskKey:        p.TaskKey,
			ClassOfService: p.ClassOfService,
			CompletedAt:    p.CompletedAt.Format("2006-01-02"),
			CycleTimeDays:  p.CycleTimeDays,
		})
	}
	writeSuccess(c, resp)
}

// GetCycleTimeTrend возвращает средний cycle time по периодам (тренд).
// GET /projects/:projectId/analytics/kanban/cycle-time/trend?from=&to=&groupBy=day|week|month&classOfService=
func (h *KanbanAnalyticsHandler) GetCycleTimeTrend(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор проекта")
		return
	}

	start, end, _ := parseDateRange(c)
	if end.Before(start) {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный диапазон дат (from, to)")
		return
	}

	groupBy := c.DefaultQuery("groupBy", "day")
	if groupBy != "day" && groupBy != "week" && groupBy != "month" {
		groupBy = "day"
	}

	var classOfService *string
	if s := c.Query("classOfService"); s != "" {
		classOfService = &s
	}

	filter := services.KanbanAnalyticsFilter{
		ProjectID:      projectID,
		StartDate:      start,
		EndDate:        end,
		GroupBy:        groupBy,
		ClassOfService: classOfService,
	}
	points, err := h.service.GetAverageCycleTimeTrend(c.Request.Context(), filter)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить данные тренда cycle time")
		return
	}

	resp := make([]dto.AverageCycleTimePointDTO, 0, len(points))
	for _, p := range points {
		resp = append(resp, dto.AverageCycleTimePointDTO{
			Period:           p.PeriodStart.Format("2006-01-02"),
			ClassOfService:   p.ClassOfService,
			AvgCycleTimeDays: p.AvgCycleTimeDays,
			TaskCount:        p.TaskCount,
		})
	}
	writeSuccess(c, resp)
}

// GetCycleTimeHistogram возвращает гистограмму распределения времени производства и процентили.
// GET /projects/:projectId/analytics/kanban/cycle-time/histogram?from=&to=&classOfService=&buckets=20&maxDays=30
func (h *KanbanAnalyticsHandler) GetCycleTimeHistogram(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор проекта")
		return
	}

	start, end, _ := parseDateRange(c)
	if end.Before(start) {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный диапазон дат (from, to)")
		return
	}

	numBuckets := 20
	if s := c.Query("buckets"); s != "" {
		if n, e := strconv.Atoi(s); e == nil && n > 0 && n <= 100 {
			numBuckets = n
		}
	}
	maxDays := 30.0
	if s := c.Query("maxDays"); s != "" {
		if f, e := strconv.ParseFloat(s, 64); e == nil && f > 0 {
			maxDays = f
		}
	}

	var classOfService *string
	if s := c.Query("classOfService"); s != "" {
		classOfService = &s
	}

	filter := services.KanbanAnalyticsFilter{
		ProjectID:      projectID,
		StartDate:      start,
		EndDate:        end,
		ClassOfService: classOfService,
	}
	data, err := h.service.GetCycleTimeHistogram(c.Request.Context(), filter, numBuckets, maxDays)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить гистограмму cycle time")
		return
	}

	resp := dto.HistogramDataDTO{
		Buckets:    make([]dto.HistogramBucketDTO, 0, len(data.Buckets)),
		TotalTasks: data.TotalTasks,
		Average:    data.Average,
		Median:     data.Median,
		P85:        data.P85,
		P95:        data.P95,
	}
	for _, b := range data.Buckets {
		resp.Buckets = append(resp.Buckets, dto.HistogramBucketDTO{
			BucketStart: b.BucketStart,
			BucketEnd:   b.BucketEnd,
			TaskCount:   b.TaskCount,
		})
	}
	writeSuccess(c, resp)
}

// GetThroughputHistogram возвращает гистограмму распределения скорости поставки.
// GET /projects/:projectId/analytics/kanban/throughput/histogram?from=&to=&period=day|week&bucketSize=1
func (h *KanbanAnalyticsHandler) GetThroughputHistogram(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор проекта")
		return
	}

	start, end, _ := parseDateRange(c)
	if end.Before(start) {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный диапазон дат (from, to)")
		return
	}

	period := c.DefaultQuery("period", "day")
	if period != "week" {
		period = "day"
	}
	bucketSize := 1
	if s := c.Query("bucketSize"); s != "" {
		if n, e := strconv.Atoi(s); e == nil && n > 0 && n <= 100 {
			bucketSize = n
		}
	}

	filter := services.KanbanAnalyticsFilter{
		ProjectID: projectID,
		StartDate: start,
		EndDate:   end,
	}
	data, err := h.service.GetThroughputHistogram(c.Request.Context(), filter, period, bucketSize)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить гистограмму throughput")
		return
	}

	resp := dto.HistogramDataDTO{
		Buckets:    make([]dto.HistogramBucketDTO, 0, len(data.Buckets)),
		TotalTasks: data.TotalTasks,
		Average:    data.Average,
		Median:     data.Median,
		P85:        data.P85,
		P95:        data.P95,
	}
	for _, b := range data.Buckets {
		resp.Buckets = append(resp.Buckets, dto.HistogramBucketDTO{
			BucketStart: b.BucketStart,
			BucketEnd:   b.BucketEnd,
			TaskCount:   b.TaskCount,
		})
	}
	writeSuccess(c, resp)
}
