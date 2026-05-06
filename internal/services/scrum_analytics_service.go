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
		report.Interpretation = "Нет данных для анализа velocity. Завершите хотя бы один спринт."
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
		return "Нет данных для анализа. Завершите хотя бы один спринт."
	}

	unit := metricUnitName(metricType)
	sprintWord := pluralForm(report.SprintCount, "спринт", "спринта", "спринтов")

	result := "ℹ️ **О графике:** Показывает способность команды прогнозировать объём работы. Сравнивает обещания (план) и реальность (факт). Обычно анализируется за несколько последних спринтов. Хороший показатель — когда команда выполняет 75–85% плана, а линия факта ровная или плавно растёт. Плохо — когда скорость сильно скачет или мы систематически выполняем меньше 75%.\n\n"

	if report.SprintCount == 1 {
		result += fmt.Sprintf("За 1 спринт команда выполнила %s %s из %s запланированных (%.0f%%). ",
			formatValue(report.AverageVelocity), unit, formatValue(report.AverageSprintScope), report.CompletionRate)
		if report.CompletionRate >= velocityLowCompletionPct {
			result += "Отличный старт! "
		}
		result += "Для выявления долгосрочных тенденций нужно завершить ещё 2–3 спринта."
		return result
	}

	var sumSq float64
	for _, d := range report.Data {
		diff := d.Completed - report.AverageVelocity
		sumSq += diff * diff
	}
	cv := float64(0)
	if report.AverageVelocity > 0 {
		cv = (math.Sqrt(sumSq/float64(report.SprintCount)) / report.AverageVelocity) * 100
	}

	// Порог значимости тренда — относительный, как доля от средней скорости.
	// Это делает порог корректным для любой метрики (задачи, story points, часы).
	trendThreshold := velocityTrendRelativeThreshold * report.AverageVelocity

	result += fmt.Sprintf("За %d %s команда в среднем выполняет %s %s из %s запланированных (%.0f%%). ",
		report.SprintCount, sprintWord, formatValue(report.AverageVelocity), unit, formatValue(report.AverageSprintScope), report.CompletionRate)

	if report.VelocityTrend > trendThreshold {
		result += fmt.Sprintf("Скорость стабильно растёт (+%s %s за спринт). Молодцы! ", formatValue(report.VelocityTrend), unit)
	} else if report.VelocityTrend < -trendThreshold {
		result += fmt.Sprintf("Скорость снижается (%s %s за спринт). ", formatValue(report.VelocityTrend), unit)
	}

	if cv >= velocityHighVariationPct {
		result += "При этом результаты очень нестабильны: совокупность спринтов неоднородна, среднее значение теряет прогностическую ценность.\n\n"
	} else if cv >= velocityModerateVariationPct {
		result += "Разброс выраженный, но в пределах однородности — среднее значение ещё пригодно для планирования.\n\n"
	} else if cv >= velocityLowVariationPct {
		result += "Разброс умеренный — результаты достаточно стабильны.\n\n"
	} else {
		result += "Результаты отлично стабилизировались — разброс минимальный.\n\n"
	}

	result += "💡 **Что с этим делать:** "
	if report.CompletionRate < velocityLowCompletionPct {
		result += "Команда систематически переоценивает свои силы и берет слишком много работы. Сократите объём следующего спринта на 20–30% и наращивайте его постепенно."
	} else if cv >= velocityHighVariationPct {
		result += "Планирование хорошее, но скорость нестабильна. Проверьте, не различаются ли задачи слишком сильно по сложности. Старайтесь декомпозировать их равномерно."
	} else if report.VelocityTrend < -trendThreshold {
		result += "Скорость падает — возможен рост технического долга, нечеткие требования или усталость команды. Обязательно обсудите это на ретроспективе!"
	} else {
		result += "Команда работает как часы стабильно и предсказуемо! Смело используйте среднюю скорость для планирования будущих спринтов."
	}

	return result
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

	last := report.Data[len(report.Data)-1]
	completedPercent := float64(0)
	if totalWork > 0 {
		completedPercent = math.Round((totalWork - last.Remaining) / totalWork * 100)
	}
	completed := totalWork - last.Remaining

	result := "ℹ️ **О графике:** Показывает сгорание работы день за днем внутри одного текущего спринта. Пунктир — это идеальный темп, сплошная линия — реальность. Отлично — когда реальная линия идет ровно по идеальной или чуть ниже. Плохо — когда линия долго идет горизонтально (задачи висят) или вдруг уходит вверх (кто-то вкинул новые задачи посреди спринта).\n\n"

	result += fmt.Sprintf("В спринте «%s» на данный момент выполнено %s из %s %s (%.0f%%). ",
		report.SprintName, formatValue(completed), formatValue(totalWork), unit, completedPercent)

	if last.Remaining <= last.Ideal {
		result += "Команда молодец, идём с опережением или точно по графику! "
	} else {
		diff := last.Remaining - last.Ideal
		result += fmt.Sprintf("Сейчас мы отстаём от идеального графика на %s %s. ", formatValue(diff), unit)
	}

	var scopeChanges int
	for i := 1; i < len(report.Data); i++ {
		if report.Data[i].Remaining > report.Data[i-1].Remaining {
			scopeChanges++
		}
	}
	if scopeChanges > 0 {
		scopeNote := pluralForm(scopeChanges, "раз", "раза", "раз")
		result += fmt.Sprintf("\n⚠️ Внимание: Объём работы увеличивался %d %s — в активный спринт добавлялись новые задачи! ", scopeChanges, scopeNote)
	}

	// Критическим считается отставание реального остатка от идеального
	// более чем на burndownCriticalLagPct процентов общего объёма работы,
	// при условии что прошло достаточно дней для информативного вывода.
	criticalLag := totalWork * burndownCriticalLagPct / 100
	lag := last.Remaining - last.Ideal
	result += "\n\n💡 **Что делать прямо сейчас:** "
	if last.Remaining <= last.Ideal {
		result += "Продолжайте в том же духе. Если темп сохранится, спринт будет завершён в срок."
	} else if lag > criticalLag && len(report.Data) > burndownCriticalProgressMinDays {
		result += "Прогресс критически низкий. Стоит обсудить на ближайшем Daily, есть ли блокеры, и при необходимости безжалостно выкинуть из спринта второстепенные задачи."
	} else {
		result += "Сфокусируйтесь на доведении начатых задач до конца (Stop starting, start finishing), чтобы вернуться в график."
	}

	if scopeChanges > 1 {
		result += " И перестаньте добавлять новые задачи в спринт на лету — это полностью убивает предсказуемость команды."
	}

	return result
}
