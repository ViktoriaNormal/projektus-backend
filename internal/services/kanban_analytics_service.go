package services

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"time"

	"github.com/google/uuid"

	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
)

type KanbanAnalyticsService struct {
	repo     repositories.KanbanAnalyticsRepository
	boardRepo repositories.BoardRepository
	cacheRepo repositories.AnalyticsCacheRepository
}

func NewKanbanAnalyticsService(
	repo repositories.KanbanAnalyticsRepository,
	boardRepo repositories.BoardRepository,
	cacheRepo repositories.AnalyticsCacheRepository,
) *KanbanAnalyticsService {
	return &KanbanAnalyticsService{
		repo:      repo,
		boardRepo: boardRepo,
		cacheRepo: cacheRepo,
	}
}

// KanbanAnalyticsFilter — параметры выборки отчётов Kanban.
type KanbanAnalyticsFilter struct {
	ProjectID       uuid.UUID
	BoardID         *uuid.UUID
	StartDate       time.Time
	EndDate         time.Time
	GroupBy         string  // day, week, month
	ClassOfService  *string // опциональный фильтр по классу обслуживания
}

// GetCumulativeFlowData возвращает данные для накопительной диаграммы потока (CFD).
func (s *KanbanAnalyticsService) GetCumulativeFlowData(ctx context.Context, f KanbanAnalyticsFilter) ([]domain.CumulativeFlowPoint, error) {
	boardID, err := s.resolveBoardID(ctx, f.ProjectID, f.BoardID)
	if err != nil {
		return nil, err
	}

	cacheKey := cacheKeyCfd(f.ProjectID, boardID, f.StartDate, f.EndDate)
	if raw, err := s.cacheRepo.Get(ctx, f.ProjectID, "kanban_cfd", cacheKey); err == nil {
		var out []domain.CumulativeFlowPoint
		if err := json.Unmarshal(raw, &out); err == nil {
			return out, nil
		}
	}

	points, err := s.repo.GetCfdColumnCountsByDate(ctx, f.ProjectID, boardID, f.StartDate, f.EndDate)
	if err != nil {
		return nil, err
	}
	if data, err := json.Marshal(points); err == nil {
		_ = s.cacheRepo.Save(ctx, f.ProjectID, "kanban_cfd", cacheKey, data, time.Hour)
	}
	return points, nil
}

// GetThroughputData возвращает данные для графика скорости поставки.
func (s *KanbanAnalyticsService) GetThroughputData(ctx context.Context, f KanbanAnalyticsFilter) ([]domain.ThroughputPoint, error) {
	groupBy := f.GroupBy
	if groupBy == "" {
		groupBy = "day"
	}
	key := map[string]string{
		"start_date": f.StartDate.Format("2006-01-02"),
		"end_date":    f.EndDate.Format("2006-01-02"),
		"group_by":    groupBy,
	}
	if raw, err := s.cacheRepo.Get(ctx, f.ProjectID, "kanban_throughput", key); err == nil {
		var out []domain.ThroughputPoint
		if err := json.Unmarshal(raw, &out); err == nil {
			return out, nil
		}
	}
	points, err := s.repo.GetThroughput(ctx, f.ProjectID, f.StartDate, f.EndDate, groupBy)
	if err != nil {
		return nil, err
	}
	if data, err := json.Marshal(points); err == nil {
		_ = s.cacheRepo.Save(ctx, f.ProjectID, "kanban_throughput", key, data, 30*time.Minute)
	}
	return points, nil
}

// GetWipOverTime возвращает WIP по дням без возраста.
func (s *KanbanAnalyticsService) GetWipOverTime(ctx context.Context, f KanbanAnalyticsFilter) ([]domain.WipPoint, error) {
	return s.repo.GetWipOverTime(ctx, f.ProjectID, f.StartDate, f.EndDate)
}

// GetWipAgeChart возвращает WIP по дням с средним/максимальным возрастом.
func (s *KanbanAnalyticsService) GetWipAgeChart(ctx context.Context, f KanbanAnalyticsFilter) ([]domain.WipPoint, error) {
	return s.repo.GetWipWithAge(ctx, f.ProjectID, f.StartDate, f.EndDate)
}

// GetCycleTimeScatterplot возвращает точки для диаграммы рассеяния времени производства.
func (s *KanbanAnalyticsService) GetCycleTimeScatterplot(ctx context.Context, f KanbanAnalyticsFilter) ([]domain.CycleTimePoint, error) {
	key := cacheKeyDatesClass(f.StartDate, f.EndDate, f.ClassOfService)
	if raw, err := s.cacheRepo.Get(ctx, f.ProjectID, "kanban_cycle_time_scatterplot", key); err == nil {
		var out []domain.CycleTimePoint
		if err := json.Unmarshal(raw, &out); err == nil {
			return out, nil
		}
	}
	points, err := s.repo.GetCycleTimeScatterplot(ctx, f.ProjectID, f.StartDate, f.EndDate, f.ClassOfService)
	if err != nil {
		return nil, err
	}
	if data, err := json.Marshal(points); err == nil {
		_ = s.cacheRepo.Save(ctx, f.ProjectID, "kanban_cycle_time_scatterplot", key, data, 30*time.Minute)
	}
	return points, nil
}

// GetAverageCycleTimeTrend возвращает средний cycle time по периодам (тренд).
func (s *KanbanAnalyticsService) GetAverageCycleTimeTrend(ctx context.Context, f KanbanAnalyticsFilter) ([]domain.AverageCycleTimePoint, error) {
	period := f.GroupBy
	if period == "" {
		period = "day"
	}
	key := cacheKeyDatesClass(f.StartDate, f.EndDate, f.ClassOfService)
	key["period"] = period
	if raw, err := s.cacheRepo.Get(ctx, f.ProjectID, "kanban_cycle_time_trend", key); err == nil {
		var out []domain.AverageCycleTimePoint
		if err := json.Unmarshal(raw, &out); err == nil {
			return out, nil
		}
	}
	points, err := s.repo.GetAverageCycleTimeByPeriod(ctx, f.ProjectID, f.StartDate, f.EndDate, f.ClassOfService, period)
	if err != nil {
		return nil, err
	}
	if data, err := json.Marshal(points); err == nil {
		_ = s.cacheRepo.Save(ctx, f.ProjectID, "kanban_cycle_time_trend", key, data, 30*time.Minute)
	}
	return points, nil
}

// GetCycleTimeHistogram возвращает гистограмму распределения времени производства и процентили.
// numBuckets — число интервалов (по умолчанию 20), maxDays — верхняя граница в днях (по умолчанию 30).
func (s *KanbanAnalyticsService) GetCycleTimeHistogram(ctx context.Context, f KanbanAnalyticsFilter, numBuckets int, maxDays float64) (*domain.HistogramData, error) {
	cacheKey := map[string]string{
		"start_date": f.StartDate.Format("2006-01-02"),
		"end_date":   f.EndDate.Format("2006-01-02"),
		"class":      "",
		"buckets":    strconv.Itoa(numBuckets),
		"max_days":   strconv.FormatFloat(maxDays, 'f', 1, 64),
	}
	if f.ClassOfService != nil {
		cacheKey["class"] = *f.ClassOfService
	}
	if raw, err := s.cacheRepo.Get(ctx, f.ProjectID, "kanban_cycle_time_histogram", cacheKey); err == nil {
		var out domain.HistogramData
		if err := json.Unmarshal(raw, &out); err == nil {
			return &out, nil
		}
	}

	points, err := s.repo.GetCycleTimeScatterplot(ctx, f.ProjectID, f.StartDate, f.EndDate, f.ClassOfService)
	if err != nil {
		return nil, err
	}
	values := make([]float64, 0, len(points))
	for _, p := range points {
		values = append(values, p.CycleTimeDays)
	}
	data := buildHistogramFromValues(values, numBuckets, maxDays)
	if data != nil {
		if encoded, err := json.Marshal(data); err == nil {
			_ = s.cacheRepo.Save(ctx, f.ProjectID, "kanban_cycle_time_histogram", cacheKey, encoded, time.Hour)
		}
	}
	return data, nil
}

// GetThroughputHistogram возвращает гистограмму распределения скорости поставки (сколько периодов с каким числом задач).
// period — day или week; bucketSize — ширина интервала по количеству задач (по умолчанию 1).
func (s *KanbanAnalyticsService) GetThroughputHistogram(ctx context.Context, f KanbanAnalyticsFilter, period string, bucketSize int) (*domain.HistogramData, error) {
	if period == "" {
		period = "day"
	}
	if period != "week" {
		period = "day"
	}
	cacheKey := map[string]string{
		"start_date":  f.StartDate.Format("2006-01-02"),
		"end_date":    f.EndDate.Format("2006-01-02"),
		"period":      period,
		"bucket_size": strconv.Itoa(bucketSize),
	}
	if raw, err := s.cacheRepo.Get(ctx, f.ProjectID, "kanban_throughput_histogram", cacheKey); err == nil {
		var out domain.HistogramData
		if err := json.Unmarshal(raw, &out); err == nil {
			return &out, nil
		}
	}

	throughput, err := s.repo.GetThroughput(ctx, f.ProjectID, f.StartDate, f.EndDate, period)
	if err != nil {
		return nil, err
	}
	counts := make([]float64, 0, len(throughput))
	for _, p := range throughput {
		counts = append(counts, float64(p.TaskCount))
	}
	data := buildThroughputHistogram(counts, bucketSize)
	if data != nil {
		if encoded, err := json.Marshal(data); err == nil {
			_ = s.cacheRepo.Save(ctx, f.ProjectID, "kanban_throughput_histogram", cacheKey, encoded, time.Hour)
		}
	}
	return data, nil
}

// buildHistogramFromValues строит гистограмму по значениям cycle time (в днях), считает процентили.
func buildHistogramFromValues(values []float64, numBuckets int, maxDays float64) *domain.HistogramData {
	if len(values) == 0 {
		return &domain.HistogramData{Buckets: nil, TotalTasks: 0}
	}
	if numBuckets <= 0 {
		numBuckets = 20
	}
	if maxDays <= 0 {
		maxDays = 30
	}

	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	avg := 0.0
	for _, v := range values {
		avg += v
	}
	avg /= float64(len(values))

	bucketWidth := maxDays / float64(numBuckets)
	buckets := make([]domain.HistogramBucket, 0, numBuckets)
	for i := 0; i < numBuckets; i++ {
		start := float64(i) * bucketWidth
		end := start + bucketWidth
		count := 0
		for _, v := range values {
			if v >= start {
				if i == numBuckets-1 {
					if v <= maxDays {
						count++
					}
				} else if v < end {
					count++
				}
			}
		}
		buckets = append(buckets, domain.HistogramBucket{BucketStart: start, BucketEnd: end, TaskCount: count})
	}

	return &domain.HistogramData{
		Buckets:    buckets,
		TotalTasks: len(values),
		Average:    avg,
		Median:     percentile(sorted, 50),
		P85:        percentile(sorted, 85),
		P95:        percentile(sorted, 95),
	}
}

// buildThroughputHistogram — гистограмма по количеству задач за период (counts = task_count по каждому периоду).
func buildThroughputHistogram(counts []float64, bucketSize int) *domain.HistogramData {
	if len(counts) == 0 {
		return &domain.HistogramData{Buckets: nil, TotalTasks: 0}
	}
	if bucketSize <= 0 {
		bucketSize = 1
	}

	sorted := make([]float64, len(counts))
	copy(sorted, counts)
	sort.Float64s(sorted)

	avg := 0.0
	for _, c := range counts {
		avg += c
	}
	avg /= float64(len(counts))

	maxCount := 0.0
	for _, c := range counts {
		if c > maxCount {
			maxCount = c
		}
	}
	numBuckets := int(maxCount)/bucketSize + 1
	if numBuckets > 50 {
		numBuckets = 50
	}

	buckets := make([]domain.HistogramBucket, 0, numBuckets)
	for i := 0; i < numBuckets; i++ {
		start := float64(i * bucketSize)
		end := float64((i + 1) * bucketSize)
		count := 0
		for _, c := range counts {
			if c >= start {
				if i == numBuckets-1 {
					count++
				} else if c < end {
					count++
				}
			}
		}
		buckets = append(buckets, domain.HistogramBucket{BucketStart: start, BucketEnd: end, TaskCount: count})
	}

	return &domain.HistogramData{
		Buckets:    buckets,
		TotalTasks: len(counts),
		Average:    avg,
		Median:     percentile(sorted, 50),
		P85:        percentile(sorted, 85),
		P95:        percentile(sorted, 95),
	}
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := (p / 100) * float64(len(sorted)-1)
	low := int(idx)
	high := low + 1
	if high >= len(sorted) {
		return sorted[len(sorted)-1]
	}
	frac := idx - float64(low)
	return sorted[low]*(1-frac) + sorted[high]*frac
}

func (s *KanbanAnalyticsService) resolveBoardID(ctx context.Context, projectID uuid.UUID, boardID *uuid.UUID) (uuid.UUID, error) {
	if boardID != nil {
		return *boardID, nil
	}
	boards, err := s.boardRepo.ListProjectBoards(ctx, projectID.String())
	if err != nil {
		return uuid.Nil, err
	}
	if len(boards) == 0 {
		return uuid.Nil, domain.ErrNotFound
	}
	return uuid.Parse(boards[0].ID)
}

func cacheKeyCfd(projectID, boardID uuid.UUID, start, end time.Time) map[string]string {
	return map[string]string{
		"board_id":   boardID.String(),
		"start_date": start.Format("2006-01-02"),
		"end_date":   end.Format("2006-01-02"),
	}
}

func cacheKeyDatesClass(start, end time.Time, classOfService *string) map[string]string {
	m := map[string]string{
		"start_date": start.Format("2006-01-02"),
		"end_date":   end.Format("2006-01-02"),
		"class":      "",
	}
	if classOfService != nil {
		m["class"] = *classOfService
	}
	return m
}
