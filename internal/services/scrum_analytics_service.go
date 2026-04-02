package services

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"time"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
)

type ScrumAnalyticsService struct {
	sprintRepo repositories.SprintRepository
	queries    *db.Queries
}

func NewScrumAnalyticsService(sprintRepo repositories.SprintRepository, queries *db.Queries) *ScrumAnalyticsService {
	return &ScrumAnalyticsService{sprintRepo: sprintRepo, queries: queries}
}

type MetricType string

const (
	MetricTaskCount       MetricType = "task_count"
	MetricStoryPoints     MetricType = "story_points"
	MetricEstimationHours MetricType = "estimation_hours"
)

// VelocityResult — данные velocity для одного спринта
type VelocityResult struct {
	SprintID   uuid.UUID
	SprintName string
	Planned    float64
	Completed  float64
}

// VelocityReport — полный отчёт velocity
type VelocityReport struct {
	Data           []VelocityResult
	AverageVelocity    float64
	VelocityTrend      float64
	CompletionRate     float64
	AverageSprintScope float64
	SprintCount        int
	Interpretation     string
}

func (s *ScrumAnalyticsService) GetVelocity(ctx context.Context, projectID uuid.UUID, metricType MetricType, limit int) (*VelocityReport, error) {
	sprints, err := s.sprintRepo.GetCompletedSprints(ctx, projectID)
	if err != nil {
		return nil, err
	}

	if limit > 0 && len(sprints) > limit {
		sprints = sprints[len(sprints)-limit:]
	}

	report := &VelocityReport{
		Data:        make([]VelocityResult, 0, len(sprints)),
		SprintCount: len(sprints),
	}

	if len(sprints) == 0 {
		report.Interpretation = "Нет данных для анализа velocity. Завершите хотя бы один спринт."
		return report, nil
	}

	var totalPlanned, totalCompleted float64

	for _, sprint := range sprints {
		tasks, err := s.queries.GetSprintTasksForAnalytics(ctx, sprint.ID)
		if err != nil {
			return nil, err
		}

		planned, completed := s.calculateMetrics(tasks, metricType)

		report.Data = append(report.Data, VelocityResult{
			SprintID:   sprint.ID,
			SprintName: sprint.Name,
			Planned:    planned,
			Completed:  completed,
		})

		totalPlanned += planned
		totalCompleted += completed
	}

	n := float64(len(sprints))
	report.AverageVelocity = totalCompleted / n
	report.AverageSprintScope = totalPlanned / n
	if totalPlanned > 0 {
		report.CompletionRate = math.Round(totalCompleted / totalPlanned * 100)
	}
	report.VelocityTrend = s.calculateTrend(report.Data)
	report.Interpretation = s.generateVelocityInterpretation(report)

	return report, nil
}

func (s *ScrumAnalyticsService) calculateMetrics(tasks []db.GetSprintTasksForAnalyticsRow, metricType MetricType) (planned, completed float64) {
	for _, t := range tasks {
		var value float64
		switch metricType {
		case MetricTaskCount:
			value = 1
		case MetricStoryPoints, MetricEstimationHours:
			if t.Estimation.Valid {
				value = parseNumericValue(t.Estimation.String)
			}
		}

		planned += value
		if t.ColumnSystemType.Valid && t.ColumnSystemType.String == string(domain.StatusCompleted) {
			completed += value
		}
	}
	return
}

var numericRegexp = regexp.MustCompile(`(\d+(?:[.,]\d+)?)`)

func parseNumericValue(s string) float64 {
	match := numericRegexp.FindString(s)
	if match == "" {
		return 0
	}
	// Replace comma with dot for parsing
	for i := range match {
		if match[i] == ',' {
			match = match[:i] + "." + match[i+1:]
			break
		}
	}
	val, err := strconv.ParseFloat(match, 64)
	if err != nil {
		return 0
	}
	return val
}

func (s *ScrumAnalyticsService) calculateTrend(data []VelocityResult) float64 {
	n := len(data)
	if n < 2 {
		return 0
	}

	// Линейная регрессия по completed-значениям
	var sumX, sumY, sumXY, sumX2 float64
	for i, d := range data {
		x := float64(i)
		y := d.Completed
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}
	nf := float64(n)
	denom := nf*sumX2 - sumX*sumX
	if denom == 0 {
		return 0
	}
	slope := (nf*sumXY - sumX*sumY) / denom
	return math.Round(slope*100) / 100
}

func (s *ScrumAnalyticsService) generateVelocityInterpretation(report *VelocityReport) string {
	if report.SprintCount == 0 {
		return "Нет данных для анализа velocity."
	}
	if report.SprintCount == 1 {
		return fmt.Sprintf("Данные за 1 спринт. Средняя скорость: %.0f. Процент выполнения: %.0f%%.",
			report.AverageVelocity, report.CompletionRate)
	}

	// Стандартное отклонение
	var sumSq float64
	for _, d := range report.Data {
		diff := d.Completed - report.AverageVelocity
		sumSq += diff * diff
	}
	stdDev := math.Sqrt(sumSq / float64(report.SprintCount))
	cv := float64(0)
	if report.AverageVelocity > 0 {
		cv = stdDev / report.AverageVelocity * 100
	}

	// Сравнение последних спринтов с предыдущими
	var recentTrend string
	if report.SprintCount >= 3 {
		mid := report.SprintCount / 2
		var firstHalf, secondHalf float64
		for i, d := range report.Data {
			if i < mid {
				firstHalf += d.Completed
			} else {
				secondHalf += d.Completed
			}
		}
		firstAvg := firstHalf / float64(mid)
		secondAvg := secondHalf / float64(report.SprintCount-mid)
		if secondAvg > firstAvg*1.1 {
			recentTrend = " Последние спринты показывают рост скорости."
		} else if secondAvg < firstAvg*0.9 {
			recentTrend = " Последние спринты показывают снижение скорости."
		} else {
			recentTrend = " Скорость остаётся стабильной."
		}
	}

	// Стабильность
	var stability string
	if cv < 15 {
		stability = "стабильна"
	} else if cv < 30 {
		stability = "умеренно стабильна"
	} else {
		stability = "нестабильна"
	}

	trendDir := ""
	if report.VelocityTrend > 0.5 {
		trendDir = fmt.Sprintf(" с тенденцией к росту (+%.1f за спринт)", report.VelocityTrend)
	} else if report.VelocityTrend < -0.5 {
		trendDir = fmt.Sprintf(" с тенденцией к снижению (%.1f за спринт)", report.VelocityTrend)
	}

	return fmt.Sprintf(
		"Скорость команды %s%s. Средняя velocity: %.0f (σ=%.1f, CV=%.0f%%). Процент выполнения: %.0f%%. Средний scope спринта: %.0f.%s",
		stability, trendDir, report.AverageVelocity, stdDev, cv,
		report.CompletionRate, report.AverageSprintScope, recentTrend,
	)
}

type statusEntry struct {
	enteredAt   time.Time
	leftAt      *time.Time
	isCompleted bool
}

// BurndownResult — данные burndown для одного дня
type BurndownDayResult struct {
	Day       string
	Remaining float64
	Ideal     float64
}

// BurndownReport — полный отчёт burndown
type BurndownReport struct {
	Data           []BurndownDayResult
	SprintName     string
	Interpretation string
}

func (s *ScrumAnalyticsService) GetBurndown(ctx context.Context, projectID uuid.UUID, metricType MetricType, sprintID *uuid.UUID) (*BurndownReport, error) {
	var sprint *domain.Sprint
	var err error

	if sprintID != nil {
		sprint, err = s.sprintRepo.GetByID(ctx, *sprintID)
	} else {
		sprint, err = s.sprintRepo.GetActiveSprint(ctx, projectID)
	}
	if err != nil {
		return nil, err
	}

	// Получаем задачи спринта
	tasks, err := s.queries.GetSprintTasksForAnalytics(ctx, sprint.ID)
	if err != nil {
		return nil, err
	}

	// Считаем начальный объём
	var totalWork float64
	taskValues := make(map[uuid.UUID]float64)
	for _, t := range tasks {
		var value float64
		switch metricType {
		case MetricTaskCount:
			value = 1
		case MetricStoryPoints, MetricEstimationHours:
			if t.Estimation.Valid {
				value = parseNumericValue(t.Estimation.String)
			}
		}
		totalWork += value
		taskValues[t.ID] = value
	}

	// Получаем историю перемещений
	history, err := s.queries.GetSprintTaskStatusHistory(ctx, sprint.ID)
	if err != nil {
		return nil, err
	}

	// Определяем временные рамки
	startDate := sprint.StartDate
	endDate := sprint.EndDate
	now := time.Now()
	if sprint.Status == domain.SprintStatusActive && now.Before(endDate) {
		endDate = now
	}

	totalDays := int(sprint.EndDate.Sub(startDate).Hours()/24) + 1
	if totalDays < 1 {
		totalDays = 1
	}

	// Для каждого дня спринта считаем оставшуюся работу
	report := &BurndownReport{
		SprintName: sprint.Name,
		Data:       make([]BurndownDayResult, 0),
	}

	// Собираем записи по задачам
	taskHistory := make(map[uuid.UUID][]statusEntry)
	for _, h := range history {
		isCompleted := h.ColumnSystemType.Valid && h.ColumnSystemType.String == string(domain.StatusCompleted)
		entry := statusEntry{
			enteredAt:   h.EnteredAt,
			isCompleted: isCompleted,
		}
		if h.LeftAt.Valid {
			t := h.LeftAt.Time
			entry.leftAt = &t
		}
		taskHistory[h.TaskID] = append(taskHistory[h.TaskID], entry)
	}

	dayNum := 0
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		dayNum++
		endOfDay := time.Date(d.Year(), d.Month(), d.Day(), 23, 59, 59, 0, d.Location())

		// Считаем, сколько завершено к концу этого дня
		var completedWork float64
		for taskID, value := range taskValues {
			entries, ok := taskHistory[taskID]
			if !ok {
				// Нет истории — проверяем текущий статус задачи (для задач без истории)
				for _, t := range tasks {
					if t.ID == taskID && t.ColumnSystemType.Valid && t.ColumnSystemType.String == string(domain.StatusCompleted) {
						// Если нет истории но задача completed, считаем завершённой с 1 дня
						completedWork += value
					}
				}
				continue
			}
			if isTaskCompletedAtTime(entries, endOfDay) {
				completedWork += value
			}
		}

		remaining := totalWork - completedWork
		ideal := totalWork - totalWork*float64(dayNum)/float64(totalDays)
		if ideal < 0 {
			ideal = 0
		}

		report.Data = append(report.Data, BurndownDayResult{
			Day:       fmt.Sprintf("День %d", dayNum),
			Remaining: math.Round(remaining*100) / 100,
			Ideal:     math.Round(ideal*100) / 100,
		})
	}

	report.Interpretation = s.generateBurndownInterpretation(report, totalWork)
	return report, nil
}

func isTaskCompletedAtTime(entries []statusEntry, t time.Time) bool {
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		if !e.isCompleted {
			continue
		}
		if e.enteredAt.Before(t) || e.enteredAt.Equal(t) {
			if e.leftAt == nil || e.leftAt.After(t) {
				return true
			}
		}
	}
	return false
}

func (s *ScrumAnalyticsService) generateBurndownInterpretation(report *BurndownReport, totalWork float64) string {
	if len(report.Data) == 0 {
		return "Нет данных для анализа burndown."
	}

	last := report.Data[len(report.Data)-1]
	completedPercent := float64(0)
	if totalWork > 0 {
		completedPercent = math.Round((totalWork - last.Remaining) / totalWork * 100)
	}

	var status string
	if last.Remaining <= last.Ideal {
		status = "Команда опережает идеальный график"
	} else {
		status = "Команда отстаёт от идеального графика"
	}

	// Проверяем скачки (scope changes)
	var scopeChanges int
	for i := 1; i < len(report.Data); i++ {
		diff := report.Data[i].Remaining - report.Data[i-1].Remaining
		if diff > 0 {
			scopeChanges++
		}
	}

	scopeNote := ""
	if scopeChanges > 0 {
		scopeNote = fmt.Sprintf(" Обнаружено %d увеличение(й) объёма работы — возможно, задачи добавлялись в спринт после запуска.", scopeChanges)
	}

	return fmt.Sprintf(
		"%s. Выполнено %.0f%% от общего объёма (осталось %.0f из %.0f).%s",
		status, completedPercent, last.Remaining, totalWork, scopeNote,
	)
}
