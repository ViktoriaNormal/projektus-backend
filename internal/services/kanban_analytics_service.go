package services

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
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
	kanbanHistoryWindowDays             = 30
	cfdBottleneckCoverageDays           = 7
	throughputTrendWindowWeeks          = 8
	throughputDistributionWindowWeeks   = 12
	throughputTrendRelativeThreshold    = 0.05
	wipRiskPercentile                   = 85.0
	cycleTimePredictableCVThreshold     = 0.5
	cycleTimeUnpredictableCVThreshold   = 1.0
	throughputConservativePercentile    = 15.0
	throughputConservativeConfidencePct = 85
	throughputLowVariationPct           = 10.0
	throughputModerateVariationPct      = 20.0
	throughputHighVariationPct          = 33.0
	distributionMaxBuckets              = 20
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
	columnSystemTypes    map[string]string
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

type cfdColumnGrowth struct {
	name   string
	growth int
	isWip  bool
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
		// Приводим времена из БД (обычно UTC) к локальному времени сервера
		started := r.StartedAt.In(time.Local)
		completed := r.CompletedAt.In(time.Local)
		ct := completed.Sub(started).Hours() / 24
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
			StartedAt:     started,
			CompletedAt:   completed,
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

func scatterDisplayOrder(taskKey string) uint32 {
	var h uint32
	for i := 0; i < len(taskKey); i++ {
		h = h*31 + uint32(taskKey[i])
	}
	return h
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

func isKanbanInProgressColumn(systemType string) bool {
	return systemType == string(domain.StatusInProgress)
}

func sortBoardColumnsForAnalytics(columns []db.GetBoardColumnsForAnalyticsRow) {
	sort.Slice(columns, func(i, j int) bool {
		if columns[i].SortOrder != columns[j].SortOrder {
			return columns[i].SortOrder < columns[j].SortOrder
		}
		return columns[i].ID.String() < columns[j].ID.String()
	})
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
	sortBoardColumnsForAnalytics(columns)

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
	columnSystemTypes := make(map[string]string, len(columns))
	for _, c := range columns {
		colNames = append(colNames, c.Name)
		if c.SystemType.Valid {
			columnSystemTypes[c.Name] = c.SystemType.String
			if c.SystemType.String == string(domain.StatusCompleted) {
				completedColumnNames = append(completedColumnNames, c.Name)
			}
		}
	}

	report := &CFDReport{
		ColumnNames:          colNames,
		completedColumnNames: completedColumnNames,
		columnSystemTypes:    columnSystemTypes,
	}

	if len(history) == 0 {
		report.Interpretation = "Нет данных для построения накопительной диаграммы потока. Переместите задачи по колонкам доски."
		return report, nil
	}

	now := time.Now()
	startDate := now.AddDate(0, 0, -kanbanHistoryWindowDays)

	for d := startDate; !d.After(now); d = d.AddDate(0, 0, 1) {
		eod := time.Date(d.Year(), d.Month(), d.Day(), 23, 59, 59, 0, d.Location())
		taskCol := make(map[uuid.UUID]string)

		for _, h := range history {
			// Приводим EnteredAt и LeftAt к локальному времени для сравнения с границей дня
			entered := h.EnteredAt.In(time.Local)
			if entered.After(eod) {
				break
			}
			if !h.LeftAt.Valid || h.LeftAt.Time.In(time.Local).After(eod) {
				taskCol[h.TaskID] = h.ColumnName
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

	weeklyThroughput := s.medianWeeklyThroughput(ctx, projectID, bid, filterSet)
	report.Interpretation = s.generateCFDInterpretation(report, weeklyThroughput)
	return report, nil
}

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

// =================== ИНТЕРПРЕТАЦИЯ CFD ===================

func (s *KanbanAnalyticsService) generateCFDInterpretation(r *CFDReport, weeklyThroughput float64) string {
	if len(r.Points) < 2 {
		return "Недостаточно данных для анализа потока."
	}

	first := r.Points[0]
	last := r.Points[len(r.Points)-1]

	var changes []cfdColumnGrowth
	for _, name := range r.ColumnNames {
		g := last.Counts[name] - first.Counts[name]
		if g == 0 {
			continue
		}
		changes = append(changes, cfdColumnGrowth{
			name:   name,
			growth: g,
			isWip:  isKanbanInProgressColumn(r.columnSystemTypes[name]),
		})
	}
	sort.Slice(changes, func(i, j int) bool {
		return changes[i].growth > changes[j].growth
	})

	doneGrowth := 0
	for _, name := range r.completedColumnNames {
		doneGrowth += last.Counts[name] - first.Counts[name]
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("С %s по %s полоса готовых задач выросла на %d %s. ",
		first.Date, last.Date, doneGrowth,
		pluralForm(doneGrowth, "задачу", "задачи", "задач")))

	if doneGrowth > 0 {
		b.WriteString("✅ Поставка происходит регулярно – это хороший знак. ")
	} else {
		b.WriteString("⚠️ Поставка стоит на месте: задачи не доходят до готовности. Это тревожный сигнал, требующий немедленного внимания. ")
	}

	bottleneckThreshold := int(math.Ceil(weeklyThroughput * cfdBottleneckCoverageDays / 7))
	if bottleneckThreshold < 1 {
		bottleneckThreshold = 1
	}
	var congested []cfdColumnGrowth
	for _, ch := range changes {
		if ch.isWip && ch.growth >= bottleneckThreshold {
			congested = append(congested, ch)
		}
	}

	if len(congested) > 0 {
		b.WriteString("🚨 Обнаружено опасное накопление задач в рабочих колонках: ")
		details := make([]string, len(congested))
		for i, c := range congested {
			details[i] = fmt.Sprintf("«%s» (+%d)", c.name, c.growth)
		}
		b.WriteString(strings.Join(details, ", "))
		b.WriteString(". ")
		b.WriteString(fmt.Sprintf(
			"Это отчётливый признак затора: прирост превышает %d %s (медианную недельную производительность команды, ≈ %.0f задач). ",
			bottleneckThreshold, pluralForm(bottleneckThreshold, "задачу", "задачи", "задач"),
			weeklyThroughput,
		))
		if peakDate := findPeakDateForColumns(r, congested); peakDate != "" {
			b.WriteString(fmt.Sprintf("Пик скопления пришёлся на %s. ", peakDate))
		}
		b.WriteString("💡 Совет: временно приостановите запуск новых задач, сфокусируйтесь на расчистке этих колонок (проверьте, нет ли блокировок, примените «рой» для быстрого закрытия зависших задач).")
	} else {
		b.WriteString(fmt.Sprintf("✅ Рабочие колонки остаются стабильными, прирост не превышает порог в %d %s (ориентир – медианная недельная производительность ≈ %.0f). ",
			bottleneckThreshold, pluralForm(bottleneckThreshold, "задачи", "задач", "задач"), weeklyThroughput))
		b.WriteString("Это здоровый поток: работа не скапливается, система сбалансирована. Продолжайте в том же духе, контролируя WIP-лимиты.")
	}

	return strings.TrimSpace(b.String())
}

func findPeakDateForColumns(r *CFDReport, congested []cfdColumnGrowth) string {
	if len(r.Points) == 0 || len(congested) == 0 {
		return ""
	}
	targetCols := make(map[string]bool, len(congested))
	for _, c := range congested {
		targetCols[c.name] = true
	}
	var peakDate string
	peakVal := 0
	for _, p := range r.Points {
		sum := 0
		for col := range targetCols {
			sum += p.Counts[col]
		}
		if sum > peakVal {
			peakVal = sum
			peakDate = p.Date
		}
	}
	return peakDate
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
	sort.Slice(points, func(i, j int) bool {
		return scatterDisplayOrder(points[i].TaskKey) < scatterDisplayOrder(points[j].TaskKey)
	})
	report.Points = points

	report.Interpretation = s.generateScatterInterpretation(cycleTimes, points)
	return report, nil
}

// =================== ИНТЕРПРЕТАЦИЯ SCATTER ===================

func (s *KanbanAnalyticsService) generateScatterInterpretation(cycleTimes []float64, points []scatterPoint) string {
	if len(cycleTimes) == 0 {
		return "Нет данных."
	}
	sorted := make([]float64, len(cycleTimes))
	copy(sorted, cycleTimes)
	sort.Float64s(sorted)

	p50 := computePercentile(sorted, 50)
	p85 := computePercentile(sorted, wipRiskPercentile)

	p25 := computePercentile(sorted, 25)
	p75 := computePercentile(sorted, 75)
	iqr := p75 - p25 // межквартильный размах

	// Классификация разброса на основе отношения IQR к медиане
	var spreadDesc string
	var assessment string
	if iqr > p50 {
		spreadDesc = "большой"
		assessment = "⚠️ Это тревожный сигнал: время выполнения задач сильно варьируется, процесс непредсказуем."
	} else if iqr > p50*0.5 {
		spreadDesc = "умеренный"
		assessment = "⚠️ Вариабельность приемлема, но есть куда стремиться."
	} else {
		spreadDesc = "небольшой"
		assessment = "✅ Отличный результат: процесс стабилен, время выполнения предсказуемо."
	}

	var b strings.Builder

	b.WriteString(fmt.Sprintf(
		"Половина задач завершается в диапазоне от %.0f до %.0f дней. ",
		p25, p75,
	))

	// Добавляем пояснение межквартильного размаха и почему разброс классифицирован именно так
	b.WriteString(fmt.Sprintf(
		"Межквартильный размах (разница между 75-м и 25-м процентилями; 75-й процентиль — 75%% задач быстрее этого срока, 25-й — 25%% быстрее) составляет %.0f дн. ",
		iqr,
	))
	if iqr > p50 {
		b.WriteString(fmt.Sprintf("Это превышает медиану (%.0f дн.), поэтому разброс оценивается как большой. ", p50))
	} else if iqr > p50*0.5 {
		b.WriteString(fmt.Sprintf("Это от 50%% до 100%% медианы (%.0f дн.), поэтому разброс умеренный. ", p50))
	} else {
		b.WriteString(fmt.Sprintf("Это менее 50%% медианы (%.0f дн.), поэтому разброс небольшой. ", p50))
	}

	b.WriteString(assessment + " ")

	if spreadDesc == "большой" || spreadDesc == "умеренный" {
		b.WriteString(fmt.Sprintf("Медиана составляет %.0f дн. ", p50))
		var outliers []scatterPoint
		for _, p := range points {
			if p.CycleTimeDays > p85 {
				outliers = append(outliers, p)
			}
		}
		if len(outliers) > 0 {
			b.WriteString(fmt.Sprintf("Обратите внимание на %d %s, которые выбиваются далеко вправо: ",
				len(outliers), pluralForm(len(outliers), "задача", "задачи", "задач")))
			show := outliers
			if len(show) > 3 {
				show = show[:3]
			}
			parts := make([]string, len(show))
			for i, t := range show {
				parts[i] = fmt.Sprintf("%s (%.0f дн.)", t.TaskKey, t.CycleTimeDays)
			}
			b.WriteString(strings.Join(parts, ", "))
			if len(outliers) > 3 {
				b.WriteString(fmt.Sprintf(" и ещё %d.", len(outliers)-3))
			}
			b.WriteString(" 💡 Совет: проанализируйте эти задачи на ретроспективе (что вызвало задержки? возможно, они слишком крупные или содержали неучтённые сложности). ")
		}
		b.WriteString("Для повышения стабильности старайтесь дробить крупные задачи и своевременно выявлять блокировки.")
	} else {
		b.WriteString(fmt.Sprintf("✅ При медиане %.0f дн. команда работает как часы. Задачи проходят поток равномерно, без сюрпризов. Так держать!", p50))
	}

	b.WriteString(" (Детальное распределение приведено на гистограмме.)")

	return b.String()
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

func (s *KanbanAnalyticsService) groupByWeeks(tasks []completedTask, maxWeeks int) []weeklyThroughputBucket {
	now := time.Now()
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

// =================== ИНТЕРПРЕТАЦИЯ THROUGHPUT ===================

func (s *KanbanAnalyticsService) generateThroughputInterpretation(weeks []weeklyThroughputBucket, slope float64) string {
	if len(weeks) == 0 {
		return "Нет данных о пропускной способности."
	}

	maxWeek := weeks[0]
	minWeek := weeks[0]
	for _, w := range weeks {
		if w.Count > maxWeek.Count {
			maxWeek = w
		}
		if w.Count < minWeek.Count {
			minWeek = w
		}
	}

	// Медианная пропускная способность (устойчива к выбросам)
	values := make([]float64, len(weeks))
	for i, w := range weeks {
		values[i] = float64(w.Count)
	}
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)
	medianThroughput := computePercentile(sorted, 50)

	// Порог значимости тренда — 5% от медианной пропускной способности
	trendThreshold := throughputTrendRelativeThreshold * medianThroughput

	var b strings.Builder

	b.WriteString(fmt.Sprintf("Лучшая неделя — %s (%d %s), худшая — %s (%d %s). ",
		maxWeek.Week, maxWeek.Count, pluralForm(maxWeek.Count, "задача", "задачи", "задач"),
		minWeek.Week, minWeek.Count, pluralForm(minWeek.Count, "задача", "задачи", "задач")))

	// Процентное изменение тренда относительно медианной пропускной способности
	trendPct := 0.0
	if medianThroughput > 0 {
		trendPct = math.Abs(slope) / medianThroughput * 100
	}

	if slope > trendThreshold {
		b.WriteString(fmt.Sprintf("✅ Тренд восходящий (+%s задач в неделю, +%.0f%% от медианной). ", formatValue(slope), trendPct))
		b.WriteString(fmt.Sprintf("Рост значимый: изменение превышает порог 5%% от медианной пропускной способности (%.1f задач в неделю). ", trendThreshold))
		b.WriteString("Однако убедитесь, что ускорение не достигается ценой качества или выгорания. Если рост устойчив, можно чуть повысить план.")
	} else if slope < -trendThreshold {
		b.WriteString(fmt.Sprintf("⚠️ Тренд нисходящий (%s задач в неделю, %.0f%% от медианной). ", formatValue(slope), trendPct))
		b.WriteString(fmt.Sprintf("Снижение значимое: изменение превышает порог 5%% от медианной пропускной способности (%.1f задач в неделю). ", trendThreshold))
		b.WriteString("💡 Совет: обсудите на ретроспективе возможные причины (блокировки, технический долг, внешние помехи). Возможно, команда перегружена или цели размыты. Определите одно-два улучшения для ближайшей итерации.")
	} else {
		b.WriteString(fmt.Sprintf("✅ Тренд стабилен (изменение около %.1f задач в неделю, %.0f%% от медианной пропускной способности). ", slope, trendPct))
		b.WriteString(fmt.Sprintf("Тренд считается стабильным, так как изменение не превышает порог 5%% от медианной (%.1f задач в неделю). ", trendThreshold))
		b.WriteString("💡 Совет: проанализируйте экстремумы: если разница между лучшей и худшей неделей велика, подумайте, что мешало в худшую и что помогло в лучшую.")
	}

	return strings.TrimSpace(b.String())
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
		// Приводим дату начала работы к локальному времени
		started := r.WorkStartedAt.In(time.Local)
		age := now.Sub(started).Hours() / 24
		if age < 0 {
			age = 0
		}
		age = math.Round(age*10) / 10
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

	report := &WipAgeReport{
		Points:         points,
		Interpretation: s.generateWipAgeInterpretation(points, p85),
	}
	return report, nil
}

// =================== ИНТЕРПРЕТАЦИЯ WIP AGE ===================

func (s *KanbanAnalyticsService) generateWipAgeInterpretation(points []wipAgePoint, p85 float64) string {
	if len(points) == 0 {
		return "На доске нет задач в работе."
	}

	var oldTasks []wipAgePoint
	if p85 > 0 {
		for _, p := range points {
			if p.AgeDays > p85 {
				oldTasks = append(oldTasks, p)
			}
		}
	} else {
		if len(points) > 3 {
			oldTasks = points[:3]
		} else {
			oldTasks = points
		}
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Сейчас в работе %d %s. Возраст считается от момента попадания в первую рабочую колонку. ",
		len(points), pluralForm(len(points), "задача", "задачи", "задач")))

	if len(oldTasks) > 0 {
		b.WriteString("⚠️ Есть задачи, которые находятся в работе слишком долго: ")
		for i, t := range oldTasks {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(fmt.Sprintf("%s (%.0f дн., сейчас в «%s»)", t.TaskKey, t.AgeDays, t.ColumnName))
		}
		if p85 > 0 {
			b.WriteString(fmt.Sprintf(". Исторический ориентир (85%% задач завершались быстрее) — %.0f дн. ", p85))
		}
		b.WriteString("Это риск для поставки: чем дольше задача висит, тем выше вероятность, что она застряла. 💡 Совет: на ближайшем дейлике сфокусируйтесь на этих задачах (выясните препятствия, окажите приоритетную помощь). Возможно, стоит временно ограничить вход новых задач, пока старые не будут закрыты.")
	} else {
		b.WriteString("✅ Возраст всех задач в пределах нормы – отлично! Работа идёт ритмично, без залежей. Продолжайте следить за появлением долгожителей.")
	}

	return strings.TrimSpace(b.String())
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
			// Приводим времена истории к локальному часовому поясу
			entered := h.EnteredAt.In(time.Local)
			if entered.After(eod) {
				break
			}
			if !h.LeftAt.Valid || h.LeftAt.Time.In(time.Local).After(eod) {
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
	report.Interpretation = s.generateWipHistoryInterpretation(points)
	return report, nil
}

// =================== ИНТЕРПРЕТАЦИЯ WIP HISTORY ===================

func (s *KanbanAnalyticsService) generateWipHistoryInterpretation(points []wipHistoryPoint) string {
	if len(points) == 0 {
		return "Нет данных."
	}

	last := points[len(points)-1]
	limit := last.Limit
	var overLimitPeriods []string
	if limit != nil {
		start := -1
		for i, p := range points {
			if p.Wip > *limit {
				if start == -1 {
					start = i
				}
			} else {
				if start != -1 {
					overLimitPeriods = append(overLimitPeriods, fmt.Sprintf("%s–%s", points[start].Date, points[i-1].Date))
					start = -1
				}
			}
		}
		if start != -1 {
			overLimitPeriods = append(overLimitPeriods, fmt.Sprintf("%s–%s", points[start].Date, points[len(points)-1].Date))
		}
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("На %s в работе %d %s. ", last.Date, last.Wip,
		pluralForm(last.Wip, "задача", "задачи", "задач")))

	if limit != nil {
		b.WriteString(fmt.Sprintf("Установленный WIP-лимит — %d. ", *limit))
		if len(overLimitPeriods) > 0 {
			b.WriteString(fmt.Sprintf("⚠️ В периоды %s лимит нарушался. ", strings.Join(overLimitPeriods, ", ")))
			b.WriteString("Превышение лимита ведёт к распылению внимания, росту незавершённой работы и удлинению времени цикла. 💡 Совет: соблюдайте строже договорённости (не берите новые задачи, пока не освободится слот). Если лимит систематически нарушается, возможно, он установлен нереалистично — обсудите на ретроспективе и скорректируйте.")
		} else {
			b.WriteString("✅ Лимит соблюдается отлично – команда дисциплинированно контролирует загрузку. Это залог ровного потока.")
		}
	} else {
		b.WriteString("⚠️ Лимит WIP не задан. 💡 Совет: установите лимит (начните с комфортного значения и корректируйте по мере накопления статистики). Без лимита легко перегрузить команду и потерять прозрачность.")
	}

	maxWip := 0
	var maxDate string
	for _, p := range points {
		if p.Wip > maxWip {
			maxWip = p.Wip
			maxDate = p.Date
		}
	}
	if maxWip > 0 {
		b.WriteString(fmt.Sprintf("Максимальная загрузка достигала %d задач %s. ", maxWip, maxDate))
		if limit != nil && maxWip > *limit {
			b.WriteString("Это существенно выше лимита – проанализируйте, что привело к такому всплеску.")
		}
	}

	return strings.TrimSpace(b.String())
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
	report.Interpretation = s.generateCycleTimeDistInterpretation(report.Buckets)
	return report, nil
}

// =================== ИНТЕРПРЕТАЦИЯ CYCLE TIME DISTRIBUTION ===================

func (s *KanbanAnalyticsService) generateCycleTimeDistInterpretation(buckets []distributionBucket) string {
	if len(buckets) == 0 {
		return "Нет данных."
	}

	maxBucket := buckets[0]
	for _, b := range buckets {
		if b.Count > maxBucket.Count {
			maxBucket = b
		}
	}

	total := 0
	for _, b := range buckets {
		total += b.Count
	}

	tailStart := -1
	for i := len(buckets) - 1; i >= 0; i-- {
		if buckets[i].Count > 0 {
			tailStart = i
			break
		}
	}
	var tailDesc string
	if tailStart >= 0 && tailStart < len(buckets)-1 {
		tailBuckets := []string{}
		for i := tailStart; i < len(buckets); i++ {
			tailBuckets = append(tailBuckets, buckets[i].RangeLabel)
		}
		tailDesc = fmt.Sprintf(" ⚠️ Однако есть длинный хвост: единичные задачи уходят далеко вправо (интервалы %s дней). Это говорит о нестабильности: некоторые задачи непропорционально затягиваются. Стоит разобраться, что их тормозит.", strings.Join(tailBuckets, ", "))
	} else {
		tailDesc = " ✅ Распределение компактное, без длинного хвоста – отлично. Команда завершает задачи в предсказуемые сроки."
	}

	var modalPhrase string
	if total > 0 && float64(maxBucket.Count) > float64(total)/2.0 {
		modalPhrase = fmt.Sprintf("✅ Большинство задач (%d из %d) укладывается в %s дней – это здоровый признак. ",
			maxBucket.Count, total, maxBucket.RangeLabel)
	} else {
		modalPhrase = fmt.Sprintf("Наиболее частый интервал — %s дней (%d из %d задач). ",
			maxBucket.RangeLabel, maxBucket.Count, total)
	}

	var b strings.Builder
	b.WriteString(modalPhrase)
	b.WriteString(tailDesc)
	return strings.TrimSpace(b.String())
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
	report.Interpretation = s.generateThroughputDistInterpretation(report.Buckets, values)
	return report, nil
}

// =================== ИНТЕРПРЕТАЦИЯ THROUGHPUT DISTRIBUTION ===================

func (s *KanbanAnalyticsService) generateThroughputDistInterpretation(buckets []distributionBucket, values []float64) string {
	if len(buckets) == 0 {
		return "Нет данных."
	}

	maxBucket := buckets[0]
	for _, b := range buckets {
		if b.Count > maxBucket.Count {
			maxBucket = b
		}
	}

	n := len(values)
	sorted := make([]float64, n)
	copy(sorted, values)
	sort.Float64s(sorted)
	p10 := computePercentile(sorted, 10)
	p90 := computePercentile(sorted, 90)

	spreadRange := p90 - p10
	var spreadDesc string
	var recommendation string
	if spreadRange > 10 {
		spreadDesc = fmt.Sprintf("⚠️ широкий (разница между 90-м и 10-м процентилями более 10 задач)")
		recommendation = "Такая нестабильность мешает планированию. 💡 Совет: проанализируйте причины провальных и рекордных недель (возможно, сказываются внешние факторы или неравномерность загрузки). Подумайте о сглаживании входящего потока."
	} else if spreadRange > 4 {
		spreadDesc = fmt.Sprintf("⚠️ умеренный (разница от 4 до 10 задач)")
		recommendation = "В целом неплохо, но идеал — узкий диапазон (разница до 4 задач). 💡 Совет: проверьте, что можно улучшить для повышения равномерности поставки."
	} else {
		spreadDesc = fmt.Sprintf("✅ очень узкий (разница до 4 задач)")
		recommendation = "Превосходно! Темп поставки очень стабилен, прогнозы будут точными. Сохраняйте текущий подход."
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Чаще всего (в %d из %d недель) команда завершает %s %s. ",
		maxBucket.Count, n, maxBucket.RangeLabel, pluralForm(maxBucket.Count, "задача", "задачи", "задач")))
	b.WriteString(fmt.Sprintf("Разброс недельной пропускной способности %s: от %.0f до %.0f задач. %s",
		spreadDesc, p10, p90, recommendation))

	return strings.TrimSpace(b.String())
}

// ========== GetMonteCarlo (без изменений) ==========

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

	samples := s.weeklyThroughputSamples(tasks, weeks)
	if len(samples) < 2 {
		return &domain.MonteCarloReport{}, nil
	}

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
	if len(report.ChartPoints) > 0 && report.ChartPoints[len(report.ChartPoints)-1].Probability < 100 {
		report.ChartPoints = append(report.ChartPoints, domain.MonteCarloChartPoint{
			Date:        maxDate,
			Probability: 100,
		})
	}

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
