package services

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
)

type KanbanAnalyticsService struct {
	queries *db.Queries
	dbtx    db.DBTX
}

const (
	// kanbanHistoryWindowDays — окно операционного обзора Канбана.
	// 30 дней соответствуют месячной каденции операционного обзора,
	// принятой в практике Канбана (Anderson, 2017).
	kanbanHistoryWindowDays = 30
	// cfdBottleneckCoverageDays — длительность поставки, которую должно
	// «съесть» накопление в колонке, чтобы считаться узким местом.
	// 7 дней = одна полная неделя пропускной способности команды.
	cfdBottleneckCoverageDays = 7
	throughputTrendWindowWeeks        = 8
	throughputDistributionWindowWeeks = 12
	// throughputTrendRelativeThreshold — порог значимости тренда
	// пропускной способности как доля от средней недельной поставки.
	// Симметрично с velocityTrendRelativeThreshold: тренд считается
	// значимым начиная с 5 % средней пропускной способности.
	throughputTrendRelativeThreshold = 0.05
	wipRiskPercentile                = 85.0
	cycleTimePredictableCVThreshold     = 0.5
	cycleTimeUnpredictableCVThreshold   = 1.0
	throughputConservativePercentile    = 15.0
	throughputConservativeConfidencePct = 85
	// Пороги CV недельной пропускной способности по классической шкале
	// однородности совокупности (Loginom; Елисеева–Юзбашев). Throughput —
	// сумма по неделе, по ЦПТ распределение стремится к нормальному,
	// поэтому применима та же шкала, что и для Velocity.
	throughputLowVariationPct      = 10.0
	throughputModerateVariationPct = 20.0
	throughputHighVariationPct     = 33.0
	distributionMaxBuckets         = 20
)

func NewKanbanAnalyticsService(queries *db.Queries, dbtx db.DBTX) *KanbanAnalyticsService {
	return &KanbanAnalyticsService{queries: queries, dbtx: dbtx}
}

// ========== Report structs ==========

type CFDReport struct {
	ColumnNames          []string
	Points               []cfdDayPoint
	Interpretation       string
	completedColumnNames []string
}

type cfdDayPoint struct {
	Date   string
	Counts map[string]int
}

type CycleTimeScatterReport struct {
	Points         []scatterPoint
	Interpretation string
}

type scatterPoint struct {
	TaskKey       string
	CycleTimeDays float64
}

// weeklyThroughputBucket — внутреннее представление недели для расчёта throughput и тренда.
type weeklyThroughputBucket struct {
	Week  string
	Count int
}

type ThroughputReport struct {
	Points         []throughputPoint
	Interpretation string
}

type throughputPoint struct {
	Week   string
	Actual int
	Trend  float64
}

type WipAgeReport struct {
	Points         []wipAgePoint
	Interpretation string
}

type wipAgePoint struct {
	TaskKey    string
	AgeDays    float64
	ColumnName string
}

type WipHistoryReport struct {
	Points         []wipHistoryPoint
	Interpretation string
}

type wipHistoryPoint struct {
	Date  string
	Wip   int
	Limit *int
}

type DistributionReport struct {
	Buckets        []distributionBucket
	Interpretation string
}

type distributionBucket struct {
	RangeLabel string
	Count      int
}

// ========== Internal helpers ==========

type completedTask struct {
	TaskID        uuid.UUID
	TaskKey       string
	Estimation    float64
	StartedAt     time.Time
	CompletedAt   time.Time
	CycleTimeDays float64
}

func (s *KanbanAnalyticsService) resolveBoard(ctx context.Context, projectID uuid.UUID, boardID *uuid.UUID) (uuid.UUID, string, error) {
	if boardID != nil {
		board, err := s.queries.GetBoardByID(ctx, *boardID)
		if err != nil {
			return uuid.Nil, "", fmt.Errorf("доска не найдена: %w", err)
		}
		return board.ID, board.EstimationUnit, nil
	}
	board, err := s.queries.GetDefaultBoardForProject(ctx, uuid.NullUUID{UUID: projectID, Valid: true})
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("не удалось найти доску проекта: %w", err)
	}
	return board.ID, board.EstimationUnit, nil
}

func (s *KanbanAnalyticsService) getCompletedTasks(ctx context.Context, projectID, boardID uuid.UUID) ([]completedTask, error) {
	rows, err := s.queries.GetCompletedTasksForKanban(ctx, db.GetCompletedTasksForKanbanParams{
		ProjectID: projectID,
		BoardID:   boardID,
	})
	if err != nil {
		return nil, err
	}
	tasks := make([]completedTask, 0, len(rows))
	for _, r := range rows {
		ct := r.CompletedAt.Sub(r.StartedAt).Hours() / 24
		if ct < 0 {
			ct = 0
		}
		est := float64(0)
		if r.Estimation.Valid {
			est = parseNumericValue(r.Estimation.String)
		}
		tasks = append(tasks, completedTask{
			TaskID:        r.TaskID,
			TaskKey:       r.TaskKey,
			Estimation:    est,
			StartedAt:     r.StartedAt,
			CompletedAt:   r.CompletedAt,
			CycleTimeDays: math.Round(ct*100) / 100,
		})
	}
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].CompletedAt.Before(tasks[j].CompletedAt)
	})
	return tasks, nil
}

func weekKey(t time.Time) string {
	y, w := t.ISOWeek()
	return fmt.Sprintf("%d-W%02d", y, w)
}

func weekLabel(index int) string {
	return fmt.Sprintf("Нед %d", index+1)
}

func computePercentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := p / 100 * float64(len(sorted)-1)
	lower := int(math.Floor(idx))
	upper := int(math.Ceil(idx))
	if lower == upper || upper >= len(sorted) {
		return sorted[lower]
	}
	frac := idx - float64(lower)
	return sorted[lower]*(1-frac) + sorted[upper]*frac
}

func linearRegressionLine(values []float64) (slope float64, trendLine []float64) {
	n := len(values)
	if n < 2 {
		trendLine = make([]float64, n)
		copy(trendLine, values)
		return 0, trendLine
	}
	var sumX, sumY, sumXY, sumX2 float64
	for i, v := range values {
		x := float64(i)
		sumX += x
		sumY += v
		sumXY += x * v
		sumX2 += x * x
	}
	nf := float64(n)
	denom := nf*sumX2 - sumX*sumX
	if denom == 0 {
		trendLine = make([]float64, n)
		for i := range trendLine {
			trendLine[i] = sumY / nf
		}
		return 0, trendLine
	}
	slope = (nf*sumXY - sumX*sumY) / denom
	intercept := (sumY - slope*sumX) / nf
	trendLine = make([]float64, n)
	for i := range trendLine {
		trendLine[i] = math.Round((intercept+slope*float64(i))*100) / 100
	}
	return math.Round(slope*100) / 100, trendLine
}

// buildDistribution формирует гистограмму на основе правила Стёрджеса.
func buildDistribution(values []float64, forcedBucketSize float64) []distributionBucket {
	N := len(values)
	if N == 0 {
		return nil
	}

	sorted := make([]float64, N)
	copy(sorted, values)
	sort.Float64s(sorted)

	maxVal := sorted[N-1]

	var numBuckets int
	var bucketSize float64

	if forcedBucketSize <= 0 {
		if N < 2 {
			numBuckets = 1
		} else {
			numBuckets = 1 + int(math.Floor(math.Log2(float64(N))))
		}

		bucketSize = math.Ceil(maxVal / float64(numBuckets))
		if bucketSize < 1 {
			bucketSize = 1
		}

		numBuckets = int(math.Ceil(maxVal/bucketSize)) + 1
	} else {
		bucketSize = forcedBucketSize
		numBuckets = int(math.Ceil(maxVal/bucketSize)) + 1
	}

	if numBuckets > distributionMaxBuckets {
		numBuckets = distributionMaxBuckets
		bucketSize = math.Ceil(maxVal / float64(distributionMaxBuckets))
		if bucketSize < 1 {
			bucketSize = 1
		}
	}

	buckets := make([]distributionBucket, numBuckets)
	for i := range buckets {
		lo := float64(i) * bucketSize
		hi := lo + bucketSize
		buckets[i] = distributionBucket{
			RangeLabel: fmt.Sprintf("%.0f-%.0f", lo, hi),
		}
	}

	for _, v := range sorted {
		idx := int(v / bucketSize)
		if idx >= numBuckets {
			idx = numBuckets - 1
		}
		buckets[idx].Count++
	}

	last := len(buckets) - 1
	for last > 0 && buckets[last].Count == 0 {
		last--
	}
	return buckets[:last+1]
}

// ========== GetCumulativeFlow ==========

func (s *KanbanAnalyticsService) GetCumulativeFlow(ctx context.Context, projectID uuid.UUID, boardID *uuid.UUID, fieldFilters map[string][]string) (*CFDReport, error) {
	bid, _, err := s.resolveBoard(ctx, projectID, boardID)
	if err != nil {
		return nil, err
	}

	var filterSet map[uuid.UUID]struct{}
	if len(fieldFilters) > 0 {
		filterSet, err = BuildTaskFilter(ctx, s.dbtx, projectID, bid, fieldFilters)
		if err != nil {
			return nil, err
		}
	}

	columns, err := s.queries.GetBoardColumnsForAnalytics(ctx, bid)
	if err != nil {
		return nil, err
	}

	history, err := s.queries.GetProjectTaskHistoryForKanban(ctx, db.GetProjectTaskHistoryForKanbanParams{
		ProjectID: projectID, BoardID: bid,
	})
	if err != nil {
		return nil, err
	}
	if filterSet != nil {
		history = filterHistoryRows(history, filterSet)
	}

	colNames := make([]string, 0, len(columns))
	completedColumnNames := make([]string, 0)
	for _, c := range columns {
		colNames = append(colNames, c.Name)
		if c.SystemType.Valid && c.SystemType.String == string(domain.StatusCompleted) {
			completedColumnNames = append(completedColumnNames, c.Name)
		}
	}

	report := &CFDReport{
		ColumnNames:          colNames,
		completedColumnNames: completedColumnNames,
	}

	if len(history) == 0 {
		report.Interpretation = "Нет данных для построения накопительной диаграммы потока. Переместите задачи по колонкам доски."
		return report, nil
	}

	now := time.Now()
	startDate := now.AddDate(0, 0, -kanbanHistoryWindowDays)

	// Для каждого дня определяем, в какой колонке находится каждая задача
	for d := startDate; !d.After(now); d = d.AddDate(0, 0, 1) {
		eod := time.Date(d.Year(), d.Month(), d.Day(), 23, 59, 59, 0, d.Location())
		taskCol := make(map[uuid.UUID]string) // task -> column name

		for _, h := range history {
			if h.EnteredAt.After(eod) {
				break
			}
			if !h.LeftAt.Valid || h.LeftAt.Time.After(eod) {
				taskCol[h.TaskID] = h.ColumnName
			} else if h.LeftAt.Valid && !h.LeftAt.Time.After(eod) {
				// Задача ушла из этой колонки до конца дня
				// Не удаляем — следующая запись перезапишет
			}
		}

		counts := make(map[string]int, len(colNames))
		for _, name := range colNames {
			counts[name] = 0
		}
		for _, colName := range taskCol {
			counts[colName]++
		}

		report.Points = append(report.Points, cfdDayPoint{
			Date:   d.Format("02.01"),
			Counts: counts,
		})
	}

	// Медианная недельная пропускная способность команды используется
	// как естественный масштаб для порогов узкого места и стабильной поставки.
	weeklyThroughput := s.medianWeeklyThroughput(ctx, projectID, bid, filterSet)
	report.Interpretation = s.generateCFDInterpretation(report, weeklyThroughput)
	return report, nil
}

// medianWeeklyThroughput возвращает медианную недельную пропускную способность
// команды за окно операционного обзора (kanbanHistoryWindowDays). Используется
// как масштаб для интерпретации CFD: его задачно-временная единица отражает
// темп поставки команды и не зависит от размера команды.
func (s *KanbanAnalyticsService) medianWeeklyThroughput(
	ctx context.Context, projectID, boardID uuid.UUID, filterSet map[uuid.UUID]struct{},
) float64 {
	tasks, err := s.getCompletedTasks(ctx, projectID, boardID)
	if err != nil || len(tasks) == 0 {
		return 0
	}
	if filterSet != nil {
		tasks = filterCompletedTasks(tasks, filterSet)
	}
	weeks := kanbanHistoryWindowDays / 7
	if weeks < 1 {
		weeks = 1
	}
	samples := s.weeklyThroughputSamples(tasks, weeks)
	if len(samples) == 0 {
		return 0
	}
	values := make([]float64, len(samples))
	for i, v := range samples {
		values[i] = float64(v)
	}
	sort.Float64s(values)
	return computePercentile(values, 50)
}

func (s *KanbanAnalyticsService) generateCFDInterpretation(r *CFDReport, weeklyThroughput float64) string {
	if len(r.Points) < 2 {
		return "Недостаточно данных для анализа потока."
	}

	first, last := r.Points[0], r.Points[len(r.Points)-1]
	result := "ℹ️ **О графике:** Это «рентгеновский снимок» вашего процесса за последние 30 дней. Ширина каждого цветового слоя — это количество задач на данном этапе. Идеально — когда полосы идут параллельно (работа течет ровно). Плохо — когда какая-то полоса резко расширяется. Это означает затор (бутылочное горлышко), где задачи скапливаются быстрее, чем их успевают делать.\n\n"

	// Узкое место — колонка с наибольшим положительным накоплением,
	// чьё накопление сопоставимо с недельной пропускной способностью команды
	// (то есть «съело» хотя бы одну неделю поставки). Это инвариантно
	// к размеру команды и согласовано с законом Литтла.
	var bottleneck string
	maxGrowth := 0
	for _, name := range r.ColumnNames {
		growth := last.Counts[name] - first.Counts[name]
		if growth > maxGrowth {
			maxGrowth = growth
			bottleneck = name
		}
	}

	// Стабильная поставка — фактическое число завершённых задач за период
	// не меньше половины ожидаемой поставки (медианная недельная пропускная
	// способность × число недель в окне / 2). Половина ожидаемого защищает
	// от случайных нулевых недель.
	doneGrowth := 0
	for _, name := range r.completedColumnNames {
		doneGrowth += last.Counts[name] - first.Counts[name]
	}
	periodWeeks := float64(len(r.Points)-1) / 7.0
	stableDeliveryThreshold := weeklyThroughput * periodWeeks / 2

	if weeklyThroughput > 0 && float64(doneGrowth) >= stableDeliveryThreshold {
		result += fmt.Sprintf("Команда стабильно поставляет результат (завершено %d задач за период). ", doneGrowth)
	}

	bottleneckThreshold := int(math.Ceil(weeklyThroughput * cfdBottleneckCoverageDays / 7))
	if bottleneckThreshold < 1 {
		bottleneckThreshold = 1
	}

	if bottleneck != "" && maxGrowth >= bottleneckThreshold {
		result += fmt.Sprintf("\n🚨 Найдено узкое место: В колонке «%s» задачи накапливаются слишком быстро (+%d задач за период).\n", bottleneck, maxGrowth)
		result += "\n💡 **Что с этим делать:** Помогите разгрести завал в этой колонке. Временно перекиньте туда специалистов или ограничьте взятие новых задач в работу, пока пробка не рассосется."
	} else {
		result += "Поток выглядит сбалансированным, явных узких мест (заторов) не обнаружено. Продолжайте в том же духе!"
	}

	return result
}

// ========== GetCycleTimeScatter ==========

func (s *KanbanAnalyticsService) GetCycleTimeScatter(ctx context.Context, projectID uuid.UUID, boardID *uuid.UUID, fieldFilters map[string][]string) (*CycleTimeScatterReport, error) {
	bid, _, err := s.resolveBoard(ctx, projectID, boardID)
	if err != nil {
		return nil, err
	}

	var filterSet map[uuid.UUID]struct{}
	if len(fieldFilters) > 0 {
		filterSet, err = BuildTaskFilter(ctx, s.dbtx, projectID, bid, fieldFilters)
		if err != nil {
			return nil, err
		}
	}

	tasks, err := s.getCompletedTasks(ctx, projectID, bid)
	if err != nil {
		return nil, err
	}
	if filterSet != nil {
		tasks = filterCompletedTasks(tasks, filterSet)
	}

	report := &CycleTimeScatterReport{}

	if len(tasks) == 0 {
		report.Interpretation = "Нет завершённых задач для анализа времени выполнения."
		return report, nil
	}

	points := make([]scatterPoint, 0, len(tasks))
	cycleTimes := make([]float64, 0, len(tasks))
	for _, t := range tasks {
		points = append(points, scatterPoint{
			TaskKey:       t.TaskKey,
			CycleTimeDays: t.CycleTimeDays,
		})
		cycleTimes = append(cycleTimes, t.CycleTimeDays)
	}
	report.Points = points

	report.Interpretation = s.generateScatterInterpretation(cycleTimes)
	return report, nil
}

func (s *KanbanAnalyticsService) generateScatterInterpretation(cycleTimes []float64) string {
	n := len(cycleTimes)
	sorted := make([]float64, n)
	copy(sorted, cycleTimes)
	sort.Float64s(sorted)

	var sum float64
	for _, v := range sorted {
		sum += v
	}
	avg := sum / float64(n)
	median := computePercentile(sorted, 50)
	p85 := computePercentile(sorted, wipRiskPercentile)

	var sumSq float64
	for _, v := range sorted {
		diff := v - avg
		sumSq += diff * diff
	}
	stdDev := math.Sqrt(sumSq / float64(n))
	cv := float64(0)
	if avg > 0 {
		cv = stdDev / avg
	}

	result := "ℹ️ **О графике:** Показывает время выполнения каждой из исторических задач (каждая точка — это задача). Нужен для того, чтобы давать обещания клиентам по срокам. Для этого используется 85-й процентиль: линия, ниже которой лежит 85% всех задач. Отлично — когда точки лежат кучно. Плохо — когда есть высокие одинокие выбросы («висяки»).\n\n"

	result += fmt.Sprintf("Из %d завершённых задач типичная задача (Медиана) закрывалась за %s дн. А наш безопасный 85-й процентиль составляет %s дн. ",
		n, formatValue(median), formatValue(p85))

	if cv < cycleTimePredictableCVThreshold {
		result += "Процесс высоко предсказуем, разброс времени небольшой!\n\n"
	} else if cv < cycleTimeUnpredictableCVThreshold {
		result += "Разброс умеренный — некоторые задачи занимают значительно больше времени.\n\n"
	} else {
		result += "Разброс времени очень большой и хаотичный — процесс непредсказуем.\n\n"
	}

	result += "💡 **Что с этим делать:** "
	result += fmt.Sprintf("Если клиент спрашивает сроки, всегда называйте цифру **%s дн.** ", formatValue(p85))
	if cv >= cycleTimeUnpredictableCVThreshold {
		result += "А из-за сильного хаоса в сроках вам жизненно необходимо ввести правило: принудительно разбивать крупные задачи на более мелкие (декомпозировать)."
	}

	return result
}

func (s *KanbanAnalyticsService) groupByWeeks(tasks []completedTask, maxWeeks int) []weeklyThroughputBucket {
	now := time.Now()

	// Генерируем все недели в диапазоне, включая пустые
	weekCounts := make(map[string]int)
	for _, t := range tasks {
		weekCounts[weekKey(t.CompletedAt)]++
	}

	seen := make(map[string]bool)
	result := make([]weeklyThroughputBucket, 0, maxWeeks)
	for i := 0; i < maxWeeks; i++ {
		d := now.AddDate(0, 0, -(maxWeeks-1-i)*7)
		key := weekKey(d)
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, weeklyThroughputBucket{
			Week:  weekLabel(len(result)),
			Count: weekCounts[key],
		})
	}
	return result
}

// ========== GetThroughput ==========

func (s *KanbanAnalyticsService) GetThroughput(ctx context.Context, projectID uuid.UUID, boardID *uuid.UUID, fieldFilters map[string][]string) (*ThroughputReport, error) {
	bid, _, err := s.resolveBoard(ctx, projectID, boardID)
	if err != nil {
		return nil, err
	}

	var filterSet map[uuid.UUID]struct{}
	if len(fieldFilters) > 0 {
		filterSet, err = BuildTaskFilter(ctx, s.dbtx, projectID, bid, fieldFilters)
		if err != nil {
			return nil, err
		}
	}

	tasks, err := s.getCompletedTasks(ctx, projectID, bid)
	if err != nil {
		return nil, err
	}
	if filterSet != nil {
		tasks = filterCompletedTasks(tasks, filterSet)
	}

	report := &ThroughputReport{}

	if len(tasks) == 0 {
		report.Interpretation = "Нет завершённых задач для анализа пропускной способности."
		return report, nil
	}

	weeks := s.groupByWeeks(tasks, throughputTrendWindowWeeks)

	values := make([]float64, len(weeks))
	for i, w := range weeks {
		values[i] = float64(w.Count)
	}

	slope, trendLine := linearRegressionLine(values)

	points := make([]throughputPoint, len(weeks))
	for i, w := range weeks {
		points[i] = throughputPoint{
			Week:   w.Week,
			Actual: w.Count,
			Trend:  trendLine[i],
		}
	}

	report.Points = points
	report.Interpretation = s.generateThroughputInterpretation(weeks, slope)
	return report, nil
}

func (s *KanbanAnalyticsService) generateThroughputInterpretation(weeks []weeklyThroughputBucket, slope float64) string {
	n := len(weeks)
	if n == 0 {
		return "Нет данных о пропускной способности."
	}

	var sum float64
	for _, w := range weeks {
		sum += float64(w.Count)
	}
	avg := sum / float64(n)

	result := "ℹ️ **О графике:** Оценивает ритмичность команды. Показывает количество закрытых задач с наложенной линией тренда (прямая линия, сглаживающая случайные скачки). Хорошо — когда тренд растет или стабильно идет ровно. Плохо — когда тренд упрямо ползет вниз.\n\n"

	result += fmt.Sprintf("В среднем мы закрываем %s %s в неделю. ",
		formatValue(avg), pluralForm(int(math.Round(avg)), "задача", "задачи", "задач"))

	// Относительный порог значимости тренда — доля от средней пропускной способности.
	// Делает порог инвариантным к размеру команды.
	trendThreshold := throughputTrendRelativeThreshold * avg

	if slope > trendThreshold {
		result += fmt.Sprintf("Тренд растущий (+%s задач/нед) — команда отлично разгоняется!\n\n", formatValue(slope))
	} else if slope < -trendThreshold {
		result += fmt.Sprintf("Тренд снижающийся (%s задач/нед) — темп падает.\n\n", formatValue(slope))
	} else {
		result += "Тренд стабильный — пропускная способность не меняется.\n\n"
	}

	result += "💡 **Что с этим делать:** "
	if slope < -trendThreshold {
		result += "Падение скорости поставки — тревожный сигнал. Убедитесь, что задачи не блокируются, требования понятны, а команда не перегружена посторонней работой."
	} else {
		result += "Сохраняйте этот стабильный ритм, он отлично подходит для долгосрочного планирования."
	}

	return result
}

// ========== GetWipAge ==========

func (s *KanbanAnalyticsService) GetWipAge(ctx context.Context, projectID uuid.UUID, boardID *uuid.UUID, fieldFilters map[string][]string) (*WipAgeReport, error) {
	bid, _, err := s.resolveBoard(ctx, projectID, boardID)
	if err != nil {
		return nil, err
	}

	var filterSet map[uuid.UUID]struct{}
	if len(fieldFilters) > 0 {
		filterSet, err = BuildTaskFilter(ctx, s.dbtx, projectID, bid, fieldFilters)
		if err != nil {
			return nil, err
		}
	}

	rows, err := s.queries.GetWipAgeTasksForKanban(ctx, db.GetWipAgeTasksForKanbanParams{
		ProjectID: projectID,
		BoardID:   bid,
	})
	if err != nil {
		return nil, err
	}

	if filterSet != nil {
		filtered := rows[:0]
		for _, r := range rows {
			if _, ok := filterSet[r.TaskID]; ok {
				filtered = append(filtered, r)
			}
		}
		rows = filtered
	}

	now := time.Now()
	points := make([]wipAgePoint, 0, len(rows))
	for _, r := range rows {
		age := now.Sub(r.WorkStartedAt).Hours() / 24
		if age < 0 {
			age = 0
		}
		age = math.Round(age*100) / 100
		points = append(points, wipAgePoint{
			TaskKey:    r.TaskKey,
			AgeDays:    age,
			ColumnName: r.ColumnName,
		})
	}

	sort.Slice(points, func(i, j int) bool {
		return points[i].AgeDays > points[j].AgeDays
	})

	completedTasks, err := s.getCompletedTasks(ctx, projectID, bid)
	if err != nil {
		return nil, err
	}
	if filterSet != nil {
		completedTasks = filterCompletedTasks(completedTasks, filterSet)
	}

	var p85 float64
	if len(completedTasks) > 0 {
		cycleTimes := make([]float64, len(completedTasks))
		for i, task := range completedTasks {
			cycleTimes[i] = task.CycleTimeDays
		}
		sort.Float64s(cycleTimes)
		p85 = computePercentile(cycleTimes, wipRiskPercentile)
	}

	alertCount := 0
	if p85 > 0 {
		for _, point := range points {
			if point.AgeDays > p85 {
				alertCount++
			}
		}
	}

	report := &WipAgeReport{
		Points:         points,
		Interpretation: s.generateWipAgeInterpretation(len(points), alertCount, p85),
	}
	return report, nil
}

func (s *KanbanAnalyticsService) generateWipAgeInterpretation(wipCount, alertCount int, p85 float64) string {
	if wipCount == 0 {
		return "На доске нет задач в работе. Отличный повод взять новую!"
	}

	result := "ℹ️ **О графике:** Это радар текущих рисков. Показывает возраст задач, находящихся в работе *прямо сейчас*. Норма — когда возраст текущих задач не превышает ваш исторический срок выполнения (85-й процентиль). Плохо — когда задачи стареют и переходят эту красную линию.\n\n"

	result += fmt.Sprintf("Сейчас в работе находится %d задач. ", wipCount)

	if p85 > 0 {
		result += fmt.Sprintf("Наша историческая норма (85-й процентиль) составляет %s дн.\n", formatValue(p85))

		if alertCount > 0 {
			result += fmt.Sprintf("\n🚨 ВНИМАНИЕ: Возраст %d задач уже превысил нашу норму!\n", alertCount)
			result += "\n💡 **Что делать прямо сейчас:** На ближайшем собрании (Daily) перестаньте обсуждать новые задачи. Откройте эти «старые» карточки и решите всей командой, как их разблокировать и довести до конца."
		} else {
			result += "\n✅ Возраст всех текущих задач в норме (никто не превысил границу риска).\n"
			result += "\n💡 Отличная работа! Продолжайте следить за тем, чтобы задачи не застревали."
		}
	} else {
		result += "\nПока недостаточно исторических данных для расчета вашей нормы времени выполнения."
	}

	return result
}

// ========== GetWipHistory ==========

func (s *KanbanAnalyticsService) GetWipHistory(ctx context.Context, projectID uuid.UUID, boardID *uuid.UUID, fieldFilters map[string][]string) (*WipHistoryReport, error) {
	bid, _, err := s.resolveBoard(ctx, projectID, boardID)
	if err != nil {
		return nil, err
	}

	var filterSet map[uuid.UUID]struct{}
	if len(fieldFilters) > 0 {
		filterSet, err = BuildTaskFilter(ctx, s.dbtx, projectID, bid, fieldFilters)
		if err != nil {
			return nil, err
		}
	}

	columns, err := s.queries.GetBoardColumnsForAnalytics(ctx, bid)
	if err != nil {
		return nil, err
	}

	history, err := s.queries.GetProjectTaskHistoryForKanban(ctx, db.GetProjectTaskHistoryForKanbanParams{
		ProjectID: projectID, BoardID: bid,
	})
	if err != nil {
		return nil, err
	}
	if filterSet != nil {
		history = filterHistoryRows(history, filterSet)
	}

	report := &WipHistoryReport{}

	if len(history) == 0 {
		report.Interpretation = "Нет данных о незавершённой работе. Переместите задачи в рабочие колонки."
		return report, nil
	}

	// Суммарный WIP-лимит по in_progress/paused колонкам
	var wipLimitSum int
	hasLimit := false
	wipColumns := make(map[uuid.UUID]bool)
	for _, c := range columns {
		if c.SystemType.Valid && (c.SystemType.String == "in_progress" || c.SystemType.String == "paused") {
			wipColumns[c.ID] = true
			if c.WipLimit.Valid {
				wipLimitSum += int(c.WipLimit.Int16)
				hasLimit = true
			}
		}
	}

	var wipLimit *int
	if hasLimit {
		wipLimit = &wipLimitSum
	}

	now := time.Now()
	startDate := now.AddDate(0, 0, -kanbanHistoryWindowDays)

	points := make([]wipHistoryPoint, 0, kanbanHistoryWindowDays+1)
	for d := startDate; !d.After(now); d = d.AddDate(0, 0, 1) {
		eod := time.Date(d.Year(), d.Month(), d.Day(), 23, 59, 59, 0, d.Location())

		taskCol := make(map[uuid.UUID]uuid.UUID)
		for _, h := range history {
			if h.EnteredAt.After(eod) {
				break
			}
			if !h.LeftAt.Valid || h.LeftAt.Time.After(eod) {
				taskCol[h.TaskID] = h.ColumnID
			}
		}

		wipCount := 0
		for _, colID := range taskCol {
			if wipColumns[colID] {
				wipCount++
			}
		}

		points = append(points, wipHistoryPoint{
			Date:  d.Format("02.01"),
			Wip:   wipCount,
			Limit: wipLimit,
		})
	}

	report.Points = points
	report.Interpretation = s.generateWipInterpretation(points, wipLimit)
	return report, nil
}

func (s *KanbanAnalyticsService) generateWipInterpretation(points []wipHistoryPoint, limit *int) string {
	n := len(points)
	if n == 0 {
		return "Нет данных о WIP."
	}

	result := "ℹ️ **О графике:** Показывает динамику количества задач в работе по дням за последний месяц в сравнении с установленными WIP-лимитами. Хорошо — когда график живет под линией лимита. Плохо — частые пробития лимита, ведущие к перегрузке команды и замедлению работы.\n\n"

	last := points[n-1]
	maxWip := 0
	exceedCount := 0
	for _, p := range points {
		if p.Wip > maxWip {
			maxWip = p.Wip
		}
		if limit != nil && p.Wip > *limit {
			exceedCount++
		}
	}

	result += fmt.Sprintf("Сейчас в работе: %d задач (Максимум за месяц достигал %d).\n", last.Wip, maxWip)

	if limit != nil {
		if exceedCount == 0 {
			result += fmt.Sprintf("✅ WIP-лимит (%d) ни разу не превышался — отличная дисциплина потока!\n", *limit)
		} else {
			pct := math.Round(float64(exceedCount) / float64(n) * 100)
			result += fmt.Sprintf("⚠️ Внимание: WIP-лимит (%d) был пробит в %.0f%% дней.\n\n", *limit, pct)
			result += "💡 **Что с этим делать:** Команда систематически берет на себя больше, чем может «переварить». Либо начните строже соблюдать лимит (не берите новые задачи, пока не закончите старые), либо честно пересмотрите сам лимит в сторону увеличения."
		}
	} else {
		result += "\n💡 **Что с этим делать:** У вас не установлены WIP-лимиты! Обязательно задайте ограничения в настройках колонок, иначе вы не сможете контролировать скорость работы."
	}

	return result
}

// ========== GetCycleTimeDistribution ==========

func (s *KanbanAnalyticsService) GetCycleTimeDistribution(ctx context.Context, projectID uuid.UUID, boardID *uuid.UUID, fieldFilters map[string][]string) (*DistributionReport, error) {
	bid, _, err := s.resolveBoard(ctx, projectID, boardID)
	if err != nil {
		return nil, err
	}

	var filterSet map[uuid.UUID]struct{}
	if len(fieldFilters) > 0 {
		filterSet, err = BuildTaskFilter(ctx, s.dbtx, projectID, bid, fieldFilters)
		if err != nil {
			return nil, err
		}
	}

	tasks, err := s.getCompletedTasks(ctx, projectID, bid)
	if err != nil {
		return nil, err
	}
	if filterSet != nil {
		tasks = filterCompletedTasks(tasks, filterSet)
	}

	report := &DistributionReport{}

	if len(tasks) == 0 {
		report.Interpretation = "Нет завершённых задач для анализа распределения времени выполнения."
		return report, nil
	}

	values := make([]float64, len(tasks))
	for i, t := range tasks {
		values[i] = t.CycleTimeDays
	}

	report.Buckets = buildDistribution(values, 0)
	report.Interpretation = s.generateCycleTimeDistInterpretation(values)
	return report, nil
}

func (s *KanbanAnalyticsService) generateCycleTimeDistInterpretation(values []float64) string {
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	n := len(sorted)
	var sum float64
	for _, v := range sorted {
		sum += v
	}
	avg := sum / float64(n)
	median := computePercentile(sorted, 50)
	p85 := computePercentile(sorted, wipRiskPercentile)

	var cv float64
	if avg > 0 && n > 1 {
		var sqSum float64
		for _, v := range sorted {
			d := v - avg
			sqSum += d * d
		}
		stddev := math.Sqrt(sqSum / float64(n))
		cv = stddev / avg
	}

	result := "ℹ️ **О графике:** Показывает, какие сроки выполнения встречаются чаще всего среди завершенных задач. Хорошо — когда график похож на узкую гору слева. Плохо — когда есть «длинный правый хвост» (гора слева, а вправо тянутся единичные, но экстремально долгие задачи, ломающие планирование).\n\n"

	result += fmt.Sprintf("Большинство ваших задач (Медиана) закрывается за %s дн. При этом 85-й процентиль (наша граница безопасности) составляет %s дн.\n\n", formatValue(median), formatValue(p85))

	if cv >= cycleTimeUnpredictableCVThreshold {
		result += "⚠️ Диагностирован длинный правый хвост! Основная масса задач делается быстро, но регулярно попадаются «задачи-монстры», которые делаются аномально долго.\n"
		result += "\n💡 **Что с этим делать:** Внедрите жесткое правило: если задача при взятии в работу кажется большой, принудительно разбивайте ее на две-три маленькие."
	} else {
		result += "✅ Распределение выглядит здоровым. Аномально долгих задач (хвоста) не наблюдается, молодцы!"
	}

	return result
}

// ========== GetThroughputDistribution ==========

func (s *KanbanAnalyticsService) GetThroughputDistribution(ctx context.Context, projectID uuid.UUID, boardID *uuid.UUID, fieldFilters map[string][]string) (*DistributionReport, error) {
	bid, _, err := s.resolveBoard(ctx, projectID, boardID)
	if err != nil {
		return nil, err
	}

	var filterSet map[uuid.UUID]struct{}
	if len(fieldFilters) > 0 {
		filterSet, err = BuildTaskFilter(ctx, s.dbtx, projectID, bid, fieldFilters)
		if err != nil {
			return nil, err
		}
	}

	tasks, err := s.getCompletedTasks(ctx, projectID, bid)
	if err != nil {
		return nil, err
	}
	if filterSet != nil {
		tasks = filterCompletedTasks(tasks, filterSet)
	}

	report := &DistributionReport{}

	if len(tasks) == 0 {
		report.Interpretation = "Нет завершённых задач для анализа распределения пропускной способности."
		return report, nil
	}

	weeks := s.groupByWeeks(tasks, throughputDistributionWindowWeeks)
	if len(weeks) < 2 {
		report.Interpretation = "Недостаточно данных — нужно минимум 2 недели с завершёнными задачами."
		return report, nil
	}

	values := make([]float64, len(weeks))
	for i, w := range weeks {
		values[i] = float64(w.Count)
	}

	report.Buckets = buildDistribution(values, 0)
	report.Interpretation = s.generateThroughputDistInterpretation(values)
	return report, nil
}

func (s *KanbanAnalyticsService) generateThroughputDistInterpretation(values []float64) string {
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	n := len(sorted)
	var sum float64
	for _, v := range sorted {
		sum += v
	}
	avg := sum / float64(n)
	p15 := computePercentile(sorted, throughputConservativePercentile)

	var cvPct float64
	if avg > 0 && n > 1 {
		var sqSum float64
		for _, v := range sorted {
			d := v - avg
			sqSum += d * d
		}
		stddev := math.Sqrt(sqSum / float64(n))
		cvPct = (stddev / avg) * 100
	}

	result := "ℹ️ **О графике:** Показывает, насколько равномерно команда выдает результат по неделям. Хорошо — когда гистограмма узкая (вы работаете стабильно). Плохо — когда гистограмма размазана (вы работаете рывками: то пусто, то густо).\n\n"

	result += fmt.Sprintf("Наш гарантированный минимум (15-й процентиль) составляет %s задач в неделю.\n\n", formatValue(p15))

	switch {
	case cvPct == 0:
		result += "Стабильность темпа оценить нельзя: либо средняя пропускная способность нулевая, либо в выборке всего одно ненулевое значение.\n\n"
	case cvPct < throughputLowVariationPct:
		result += fmt.Sprintf("✅ Темп поставки очень стабилен (коэффициент вариации %.1f%% — незначительный разброс): команда выдает примерно одинаковое число задач каждую неделю.\n\n", cvPct)
	case cvPct < throughputModerateVariationPct:
		result += fmt.Sprintf("✅ Темп поставки стабилен (коэффициент вариации %.1f%% — умеренный разброс): отдельные недели колеблются вокруг среднего, но без срывов.\n\n", cvPct)
	case cvPct <= throughputHighVariationPct:
		result += fmt.Sprintf("⚠️ Темп поставки заметно колеблется (коэффициент вариации %.1f%% — выраженный разброс при сохранении однородности выборки): отдельные недели существенно отклоняются от среднего, имеет смысл проверить причины «провальных» недель.\n\n", cvPct)
	default:
		result += fmt.Sprintf("⚠️ Темп поставки нестабилен (коэффициент вариации %.1f%% — выборка неоднородна): команда работает рывками, среднее значение перестаёт быть надёжной оценкой; при планировании опирайтесь на консервативный 15-й процентиль.\n\n", cvPct)
	}

	result += "💡 **Что с этим делать:** "
	result += fmt.Sprintf("Для надежного и консервативного планирования используйте показатель **%s задач в неделю** — с вероятностью %d%% команда сделает не меньше этого объема.", formatValue(p15), throughputConservativeConfidencePct)

	return result
}

// ========== GetMonteCarlo ==========

const monteCarloSimulations = 10000
const monteCarloDefaultWeeks = 12

func (s *KanbanAnalyticsService) GetMonteCarlo(
	ctx context.Context, projectID uuid.UUID, boardID *uuid.UUID,
	fieldFilters map[string][]string,
	taskCount int, weeks int, targetDate *time.Time,
) (*domain.MonteCarloReport, error) {
	bid, _, err := s.resolveBoard(ctx, projectID, boardID)
	if err != nil {
		return nil, err
	}

	var filterSet map[uuid.UUID]struct{}
	if len(fieldFilters) > 0 {
		filterSet, err = BuildTaskFilter(ctx, s.dbtx, projectID, bid, fieldFilters)
		if err != nil {
			return nil, err
		}
	}

	tasks, err := s.getCompletedTasks(ctx, projectID, bid)
	if err != nil {
		return nil, err
	}
	if filterSet != nil {
		tasks = filterCompletedTasks(tasks, filterSet)
	}

	if weeks < 2 {
		weeks = monteCarloDefaultWeeks
	}

	// Build weekly throughput samples (last N weeks, including zero-weeks).
	samples := s.weeklyThroughputSamples(tasks, weeks)
	if len(samples) < 2 {
		return &domain.MonteCarloReport{}, nil
	}

	// Check that at least one week has non-zero throughput,
	// otherwise simulation would loop forever.
	hasNonZero := false
	for _, v := range samples {
		if v > 0 {
			hasNonZero = true
			break
		}
	}
	if !hasNonZero {
		return &domain.MonteCarloReport{}, nil
	}

	// Run simulation.
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	completionDates := make([]time.Time, monteCarloSimulations)

	for i := 0; i < monteCarloSimulations; i++ {
		remaining := taskCount
		current := today
		for remaining > 0 {
			tp := samples[rand.Intn(len(samples))]
			if tp <= 0 {
				current = current.AddDate(0, 0, 7)
				continue
			}
			if tp >= remaining {
				// Interpolate partial week.
				days := int(math.Ceil(float64(remaining) / float64(tp) * 7))
				current = current.AddDate(0, 0, days)
				remaining = 0
			} else {
				remaining -= tp
				current = current.AddDate(0, 0, 7)
			}
		}
		completionDates[i] = current
	}

	sort.Slice(completionDates, func(i, j int) bool {
		return completionDates[i].Before(completionDates[j])
	})

	// Extract percentiles.
	percentiles := []int{50, 75, 85, 90, 95}
	report := &domain.MonteCarloReport{}
	for _, p := range percentiles {
		idx := p * monteCarloSimulations / 100
		if idx >= monteCarloSimulations {
			idx = monteCarloSimulations - 1
		}
		report.Percentiles = append(report.Percentiles, domain.MonteCarloPercentile{
			Percentile: p,
			Date:       completionDates[idx],
		})
	}

	// Build chart: step through weekly from min to max date.
	minDate := completionDates[0]
	maxDate := completionDates[monteCarloSimulations-1]
	for d := minDate; !d.After(maxDate); d = d.AddDate(0, 0, 7) {
		count := sort.Search(monteCarloSimulations, func(i int) bool {
			return completionDates[i].After(d)
		})
		prob := count * 100 / monteCarloSimulations
		report.ChartPoints = append(report.ChartPoints, domain.MonteCarloChartPoint{
			Date:        d,
			Probability: prob,
		})
	}
	// Ensure last point reaches the max.
	if len(report.ChartPoints) > 0 && report.ChartPoints[len(report.ChartPoints)-1].Probability < 100 {
		report.ChartPoints = append(report.ChartPoints, domain.MonteCarloChartPoint{
			Date:        maxDate,
			Probability: 100,
		})
	}

	// Target date probability.
	if targetDate != nil {
		td := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 23, 59, 59, 0, targetDate.Location())
		count := sort.Search(monteCarloSimulations, func(i int) bool {
			return completionDates[i].After(td)
		})
		prob := count * 100 / monteCarloSimulations
		report.TargetDateProbability = &prob
	}

	return report, nil
}

// weeklyThroughputSamples returns the number of completed tasks per ISO week
// for the last maxWeeks weeks, including zero-count weeks.
func (s *KanbanAnalyticsService) weeklyThroughputSamples(tasks []completedTask, maxWeeks int) []int {
	now := time.Now()
	weekCounts := make(map[string]int)
	for _, t := range tasks {
		weekCounts[weekKey(t.CompletedAt)]++
	}

	seen := make(map[string]bool)
	samples := make([]int, 0, maxWeeks)
	for i := 0; i < maxWeeks; i++ {
		d := now.AddDate(0, 0, -(maxWeeks-1-i)*7)
		key := weekKey(d)
		if seen[key] {
			continue
		}
		seen[key] = true
		samples = append(samples, weekCounts[key])
	}
	return samples
}
