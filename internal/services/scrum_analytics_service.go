package services

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
)

type ScrumAnalyticsService struct {
	sprintRepo repositories.SprintRepository
	queries    *db.Queries
	dbtx       db.DBTX
}

func NewScrumAnalyticsService(sprintRepo repositories.SprintRepository, queries *db.Queries, dbtx db.DBTX) *ScrumAnalyticsService {
	return &ScrumAnalyticsService{sprintRepo: sprintRepo, queries: queries, dbtx: dbtx}
}

type MetricType string

const (
	MetricTaskCount       MetricType = "task_count"
	MetricStoryPoints     MetricType = "story_points"
	MetricEstimationHours MetricType = "estimation_hours"
)

const (
	// velocityTrendRelativeThreshold — порог значимости тренда скорости как
	// доля от средней скорости команды. Тренд |slope| < 5 % средней скорости
	// считается практически нулевым (стабильным).
	velocityTrendRelativeThreshold = 0.05
	// velocityLowVariationPct, velocityModerateVariationPct,
	// velocityHighVariationPct — границы зон коэффициента вариации согласно
	// классической классификации однородности совокупности: до 10 % —
	// незначительный разброс, 10–20 % — умеренный, 20–33 % — значительный,
	// свыше 33 % — совокупность неоднородна.
	velocityLowVariationPct      = 10.0
	velocityModerateVariationPct = 20.0
	velocityHighVariationPct     = 33.0
	// velocityLowCompletionPct — нижняя граница корректного коридора
	// планирования по практикам Atlassian (75–85 %): процент выполнения
	// ниже этого значения интерпретируется как систематическое
	// перепланирование команды.
	velocityLowCompletionPct = 75.0
	// burndownCriticalLagPct — пороговое отставание реального остатка
	// работы от идеальной линии в долях от общего объёма спринта.
	// Симметрично 25-процентному отклонению от плана (1 − 0.75) в
	// коридоре корректного планирования Atlassian: команда «потеряла»
	// более четверти спринта по графику.
	burndownCriticalLagPct = 25.0
	// burndownCriticalProgressMinDays — короткое плечо в начале
	// спринта, в течение которого вывод о критическом отставании
	// не делается из-за недостаточной информативности первых дней.
	burndownCriticalProgressMinDays = 2
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
	Data               []VelocityResult
	AverageVelocity    float64
	VelocityTrend      float64
	CompletionRate     float64
	AverageSprintScope float64
	SprintCount        int
	Interpretation     string
}

func (s *ScrumAnalyticsService) GetVelocity(ctx context.Context, projectID uuid.UUID, metricType MetricType, limit int, boardID *uuid.UUID, fieldFilters map[string][]string) (*VelocityReport, error) {
	filterSet, err := s.buildScrumFilter(ctx, projectID, boardID, fieldFilters)
	if err != nil {
		return nil, err
	}

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
		report.Interpretation = "Нет данных для анализа скорости команды. Завершите хотя бы один спринт."
		return report, nil
	}

	var totalPlanned, totalCompleted float64

	for _, sprint := range sprints {
		tasks, err := s.queries.GetSprintTasksForAnalytics(ctx, sprint.ID)
		if err != nil {
			return nil, err
		}
		if filterSet != nil {
			tasks = filterSprintTaskRows(tasks, filterSet)
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
	report.Interpretation = s.generateVelocityInterpretation(report, metricType)

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

// formatValue форматирует число: целое без дробной части, дробное с 1 знаком
func formatValue(v float64) string {
	if v == math.Trunc(v) {
		return fmt.Sprintf("%.0f", v)
	}
	return fmt.Sprintf("%.1f", v)
}

// metricUnitName возвращает название единицы измерения для отображения в интерпретациях
func metricUnitName(mt MetricType) string {
	switch mt {
	case MetricStoryPoints:
		return "SP"
	case MetricEstimationHours:
		return "ч."
	default:
		return "задач"
	}
}

// pluralForm выбирает правильную форму слова по числу (русская грамматика)
func pluralForm(n int, form1, form2, form5 string) string {
	mod10 := n % 10
	mod100 := n % 100
	if mod100 >= 11 && mod100 <= 19 {
		return form5
	}
	if mod10 == 1 {
		return form1
	}
	if mod10 >= 2 && mod10 <= 4 {
		return form2
	}
	return form5
}

func (s *ScrumAnalyticsService) generateVelocityInterpretation(report *VelocityReport, metricType MetricType) string {
	if report.SprintCount == 0 {
		return "Нет данных для анализа скорости команды. Завершите хотя бы один спринт."
	}

	unit := metricUnitName(metricType)
	sprintWord := pluralForm(report.SprintCount, "спринт", "спринта", "спринтов")

	formatWithUnit := func(value float64, metricType MetricType) string {
		rounded := int(math.Round(value))
		if metricType == MetricTaskCount {
			return formatValue(value) + " " + pluralForm(rounded, "задача", "задачи", "задач")
		}
		return formatValue(value) + " " + unit
	}

	if report.SprintCount == 1 {
		// Для одного спринта не делаем строгих выводов, только эмодзи
		startMsg := ""
		if report.CompletionRate >= velocityLowCompletionPct {
			startMsg = "✅ Отличный старт! "
		} else {
			startMsg = "⚠️ "
		}
		return fmt.Sprintf(
			"%sЗа 1 спринт команда выполнила %s из %s запланированных (%.0f%%). Для выявления тенденции нужно завершить ещё 2–3 спринта.",
			startMsg,
			formatWithUnit(report.AverageVelocity, metricType),
			formatWithUnit(report.AverageSprintScope, metricType),
			report.CompletionRate,
		)
	}

	var sumSq float64
	for _, d := range report.Data {
		diff := d.Completed - report.AverageVelocity
		sumSq += diff * diff
	}
	cv := 0.0
	if report.AverageVelocity > 0 {
		cv = (math.Sqrt(sumSq/float64(report.SprintCount)) / report.AverageVelocity) * 100
	}
	trendThreshold := velocityTrendRelativeThreshold * report.AverageVelocity

	var builder strings.Builder

	builder.WriteString(fmt.Sprintf(
		"За %d %s команда в среднем выполняет %s из %s запланированных (%.0f%%).\n",
		report.SprintCount, sprintWord,
		formatWithUnit(report.AverageVelocity, metricType),
		formatWithUnit(report.AverageSprintScope, metricType),
		report.CompletionRate,
	))

	if report.VelocityTrend > trendThreshold {
		trendUnit := unit
		if metricType == MetricTaskCount {
			trendUnit = pluralForm(int(math.Round(report.VelocityTrend)), "задача", "задачи", "задач")
		}
		builder.WriteString(fmt.Sprintf("📈 Скорость стабильно растёт (+%s %s за спринт, тренд выше 5%% от средней).\n", formatValue(report.VelocityTrend), trendUnit))
	} else if report.VelocityTrend < -trendThreshold {
		trendUnit := unit
		if metricType == MetricTaskCount {
			trendUnit = pluralForm(int(math.Round(-report.VelocityTrend)), "задача", "задачи", "задач")
		}
		builder.WriteString(fmt.Sprintf("📉 Скорость снижается (%s %s за спринт, тренд ниже -5%% от средней).\n", formatValue(report.VelocityTrend), trendUnit))
	} else {
		builder.WriteString("➡️ Тренд стабильный (изменение менее 5% от средней скорости).\n")
	}

	builder.WriteString("\n")
	if cv >= velocityHighVariationPct {
		builder.WriteString(fmt.Sprintf("⚠️ Результаты нестабильны: высокий разброс (коэффициент вариации ≥%.0f%%). Среднее значение мало надёжно для прогнозирования.\n", velocityHighVariationPct))
	} else if cv >= velocityModerateVariationPct {
		builder.WriteString(fmt.Sprintf("📊 Разброс выраженный (коэффициент вариации %.0f–%.0f%%), но среднее всё ещё пригодно для планирования.\n", velocityModerateVariationPct, velocityHighVariationPct))
	} else if cv >= velocityLowVariationPct {
		builder.WriteString(fmt.Sprintf("✅ Разброс умеренный (коэффициент вариации %.0f–%.0f%%) — результаты достаточно стабильны.\n", velocityLowVariationPct, velocityModerateVariationPct))
	} else {
		builder.WriteString(fmt.Sprintf("✅ Разброс минимальный (коэффициент вариации <%.0f%%) — результаты очень стабильны.\n", velocityLowVariationPct))
	}

	builder.WriteString("\nРекомендация: ")
	if report.CompletionRate < velocityLowCompletionPct {
		builder.WriteString("команда систематически переоценивает объём; сократите планирование следующего спринта на 20–30% и постепенно наращивайте.")
	} else if cv >= velocityHighVariationPct {
		builder.WriteString("скорость нестабильна; проверьте, не слишком ли различаются задачи по сложности, и старайтесь декомпозировать их равномернее.")
	} else if report.VelocityTrend < -trendThreshold {
		builder.WriteString("скорость падает — возможны технический долг, неясные требования или перегрузка; обсудите это на ретроспективе.")
	} else {
		builder.WriteString("команда работает стабильно и предсказуемо; используйте среднюю скорость для планирования будущих спринтов.")
	}

	return builder.String()
}

type statusEntry struct {
	enteredAt   time.Time
	leftAt      *time.Time
	isCompleted bool
}

// BurndownResult — данные burndown для одного дня
type BurndownDayResult struct {
	Day       string
	Remaining *float64 // nil для будущих дней спринта (факт ещё неизвестен)
	Ideal     float64
}

func burndownDateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func burndownLastActualDay(data []BurndownDayResult) *BurndownDayResult {
	for i := len(data) - 1; i >= 0; i-- {
		if data[i].Remaining != nil {
			return &data[i]
		}
	}
	return nil
}

// BurndownReport — полный отчёт burndown
type BurndownReport struct {
	Data           []BurndownDayResult
	SprintName     string
	Interpretation string
}

func (s *ScrumAnalyticsService) GetBurndown(ctx context.Context, projectID uuid.UUID, metricType MetricType, sprintID *uuid.UUID, boardID *uuid.UUID, fieldFilters map[string][]string) (*BurndownReport, error) {
	filterSet, err := s.buildScrumFilter(ctx, projectID, boardID, fieldFilters)
	if err != nil {
		return nil, err
	}

	var sprint *domain.Sprint

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
	if filterSet != nil {
		tasks = filterSprintTaskRows(tasks, filterSet)
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
	if filterSet != nil {
		history = filterSprintHistoryRows(history, filterSet)
	}

	// Полный календарный период спринта: от start_date до end_date (включительно),
	// приведённый к локальному времени сервера, чтобы избежать расхождений с time.Now().
	startDate := burndownDateOnly(sprint.StartDate.In(time.Local))
	sprintEndDate := burndownDateOnly(sprint.EndDate.In(time.Local))
	today := burndownDateOnly(time.Now().In(time.Local))

	totalDays := int(sprintEndDate.Sub(startDate).Hours()/24) + 1
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
	for d := startDate; !d.After(sprintEndDate); d = d.AddDate(0, 0, 1) {
		dayNum++
		endOfDay := time.Date(d.Year(), d.Month(), d.Day(), 23, 59, 59, 0, d.Location())

		ideal := totalWork - totalWork*float64(dayNum)/float64(totalDays)
		if ideal < 0 {
			ideal = 0
		}

		var remainingPtr *float64
		if !d.After(today) {
			var completedWork float64
			for taskID, value := range taskValues {
				entries, ok := taskHistory[taskID]
				if !ok {
					for _, t := range tasks {
						if t.ID == taskID && t.ColumnSystemType.Valid && t.ColumnSystemType.String == string(domain.StatusCompleted) {
							completedWork += value
						}
					}
					continue
				}
				if isTaskCompletedAtTime(entries, endOfDay) {
					completedWork += value
				}
			}
			remaining := math.Round((totalWork-completedWork)*100) / 100
			remainingPtr = &remaining
		}

		report.Data = append(report.Data, BurndownDayResult{
			Day:       fmt.Sprintf("День %d", dayNum),
			Remaining: remainingPtr,
			Ideal:     math.Round(ideal*100) / 100,
		})
	}

	report.Interpretation = s.generateBurndownInterpretation(report, totalWork, metricType)
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

func (s *ScrumAnalyticsService) generateBurndownInterpretation(report *BurndownReport, totalWork float64, metricType MetricType) string {
	if len(report.Data) == 0 {
		return "Нет данных для анализа."
	}

	unit := metricUnitName(metricType)

	formatWithUnit := func(value float64, metricType MetricType) string {
		rounded := int(math.Round(value))
		if metricType == MetricTaskCount {
			return formatValue(value) + " " + pluralForm(rounded, "задача", "задачи", "задач")
		}
		return formatValue(value) + " " + unit
	}

	last := burndownLastActualDay(report.Data)
	if last == nil {
		return fmt.Sprintf("В спринте «%s» пока нет фактических данных — спринт ещё не начался или нет завершённой работы на сегодня.", report.SprintName)
	}

	completed := totalWork - *last.Remaining
	completedPercent := 0.0
	if totalWork > 0 {
		completedPercent = math.Round((totalWork - *last.Remaining) / totalWork * 100)
	}

	var builder strings.Builder

	builder.WriteString(fmt.Sprintf(
		"В спринте «%s» выполнено %s из %s (%.0f%%).\n",
		report.SprintName,
		formatWithUnit(completed, metricType),
		formatWithUnit(totalWork, metricType),
		completedPercent,
	))

	if *last.Remaining <= last.Ideal {
		builder.WriteString("✅ Команда идёт по графику или с опережением.\n")
	} else {
		diff := *last.Remaining - last.Ideal
		builder.WriteString(fmt.Sprintf("📉 Отставание от идеального графика на текущий день: %s.\n", formatWithUnit(diff, metricType)))
	}

	scopeChanges := 0
	var prevRemaining *float64
	for _, p := range report.Data {
		if p.Remaining == nil {
			continue
		}
		if prevRemaining != nil && *p.Remaining > *prevRemaining {
			scopeChanges++
		}
		prevRemaining = p.Remaining
	}
	if scopeChanges > 0 {
		scopeNote := pluralForm(scopeChanges, "раз", "раза", "раз")
		builder.WriteString(fmt.Sprintf("\n⚠️ Объём работы спринта увеличивался %d %s (добавлялись новые задачи).\n", scopeChanges, scopeNote))
	}

	criticalLag := totalWork * burndownCriticalLagPct / 100
	lag := *last.Remaining - last.Ideal
	actualDays := 0
	for _, p := range report.Data {
		if p.Remaining != nil {
			actualDays++
		}
	}
	builder.WriteString("\nРекомендация: ")
	if *last.Remaining <= last.Ideal {
		builder.WriteString("продолжайте в том же темпе; спринт, скорее всего, будет завершён в срок.")
	} else if lag > criticalLag && actualDays > burndownCriticalProgressMinDays {
		builder.WriteString(fmt.Sprintf("прогресс критически низкий (отставание >%.0f%% от объёма спринта); на ежедневной встрече выявите блокеры и рассмотрите возможность исключения второстепенных задач из спринта.", burndownCriticalLagPct))
	} else {
		builder.WriteString("сфокусируйтесь на завершении начатых задач; не берите новые, пока не закончите текущие.")
	}

	if scopeChanges > 1 {
		builder.WriteString(" по возможности избегайте добавления новых задач в середине спринта — это снижает предсказуемость.")
	}

	return builder.String()
}
