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

func NewKanbanAnalyticsService(queries *db.Queries, dbtx db.DBTX) *KanbanAnalyticsService {
	return &KanbanAnalyticsService{queries: queries, dbtx: dbtx}
}

// ========== Report structs ==========

type CFDReport struct {
	ColumnNames    []string
	Points         []cfdDayPoint
	Interpretation string
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

func buildDistribution(values []float64, bucketSize float64) []distributionBucket {
	if len(values) == 0 {
		return nil
	}
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	if bucketSize <= 0 {
		maxVal := sorted[len(sorted)-1]
		bucketSize = math.Ceil(maxVal / 8)
		if bucketSize < 1 {
			bucketSize = 1
		}
	}

	maxVal := sorted[len(sorted)-1]
	numBuckets := int(math.Ceil(maxVal/bucketSize)) + 1
	if numBuckets > 20 {
		numBuckets = 20
		bucketSize = math.Ceil(maxVal / 20)
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

	// Убираем хвостовые пустые бакеты
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
	for _, c := range columns {
		colNames = append(colNames, c.Name)
	}

	report := &CFDReport{ColumnNames: colNames}

	if len(history) == 0 {
		report.Interpretation = "Нет данных для построения накопительной диаграммы потока. Переместите задачи по колонкам доски."
		return report, nil
	}

	now := time.Now()
	startDate := now.AddDate(0, 0, -30)

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

	report.Interpretation = s.generateCFDInterpretation(report)
	return report, nil
}

func (s *KanbanAnalyticsService) generateCFDInterpretation(r *CFDReport) string {
	if len(r.Points) < 2 {
		return "Недостаточно данных для анализа потока."
	}

	first := r.Points[0]
	last := r.Points[len(r.Points)-1]

	result := "Диаграмма показывает количество задач в каждой колонке по дням — слои отражают поток работы."

	// Ищем бутылочные горлышки: колонка, где рост больше всего
	var bottleneck string
	maxGrowth := 0
	for _, name := range r.ColumnNames {
		growth := last.Counts[name] - first.Counts[name]
		if growth > maxGrowth {
			maxGrowth = growth
			bottleneck = name
		}
	}

	totalFirst := 0
	totalLast := 0
	for _, name := range r.ColumnNames {
		totalFirst += first.Counts[name]
		totalLast += last.Counts[name]
	}

	if totalLast > totalFirst {
		result += fmt.Sprintf(" Общее количество задач на доске выросло с %d до %d.", totalFirst, totalLast)
	}

	if bottleneck != "" && maxGrowth > 2 {
		result += fmt.Sprintf(" Колонка «%s» накапливает задачи (+%d за период) — возможное бутылочное горлышко.", bottleneck, maxGrowth)
	}

	// Проверяем стабильность: если слой "done" растёт равномерно — поток стабилен
	doneFirst := 0
	doneLast := 0
	for _, name := range r.ColumnNames {
		// Ищем колонку completed
		if name == r.ColumnNames[len(r.ColumnNames)-1] {
			doneFirst = first.Counts[name]
			doneLast = last.Counts[name]
		}
	}
	if doneLast > doneFirst+5 {
		result += " Поток завершённых задач растёт — команда стабильно выпускает работу."
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
	p85 := computePercentile(sorted, 85)

	result := fmt.Sprintf("Диаграмма показывает время выполнения каждой задачи. Из %d %s среднее время: %s дн., медиана: %s дн., 85-й процентиль: %s дн.",
		n, pluralForm(n, "задачи", "задач", "задач"), formatValue(avg), formatValue(median), formatValue(p85))

	// Оценка предсказуемости через CV
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

	if cv < 0.5 {
		result += " Процесс предсказуем — разброс небольшой."
	} else if cv < 1 {
		result += " Разброс умеренный — некоторые задачи занимают значительно больше времени."
	} else {
		result += " Разброс очень большой — процесс непредсказуем. Рекомендация: декомпозируйте крупные задачи."
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

	weeks := s.groupByWeeks(tasks, 8)

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

	result := fmt.Sprintf("Диаграмма показывает фактическую пропускную способность и линию тренда. Средняя: %s %s в неделю.",
		formatValue(avg), pluralForm(int(math.Round(avg)), "задача", "задачи", "задач"))

	if slope > 0.5 {
		result += fmt.Sprintf(" Тренд растущий (+%s задач в неделю) — команда наращивает темп.", formatValue(slope))
	} else if slope < -0.5 {
		result += fmt.Sprintf(" Тренд снижающийся (%s задач в неделю) — возможны проблемы с процессом или нагрузкой.", formatValue(slope))
	} else {
		result += " Тренд стабильный — пропускная способность не меняется."
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

	report := &WipAgeReport{
		Points:         points,
		Interpretation: s.generateWipAgeInterpretation(points),
	}
	return report, nil
}

func (s *KanbanAnalyticsService) generateWipAgeInterpretation(points []wipAgePoint) string {
	n := len(points)
	if n == 0 {
		return "Нет задач в колонках «в работе» и «на паузе». Когда карточки появятся в этих колонках, здесь отобразится их возраст — сколько дней они уже в активной работе."
	}

	var sum float64
	ages := make([]float64, n)
	for i, p := range points {
		sum += p.AgeDays
		ages[i] = p.AgeDays
	}
	avg := sum / float64(n)

	sorted := make([]float64, n)
	copy(sorted, ages)
	sort.Float64s(sorted)
	median := computePercentile(sorted, 50)

	result := fmt.Sprintf("Сейчас в работе (включая паузу) %d %s. Средний возраст карточек в этих колонках: %s дн., медиана: %s дн.",
		n, pluralForm(n, "задача", "задачи", "задач"),
		formatValue(avg), formatValue(median))

	outliers := make([]wipAgePoint, 0)
	if n >= 4 {
		q1 := computePercentile(sorted, 25)
		q3 := computePercentile(sorted, 75)
		iqr := q3 - q1
		threshold := q3 + math.Max(1.5*iqr, 1)
		for _, p := range points {
			if p.AgeDays >= threshold && p.AgeDays > median {
				outliers = append(outliers, p)
			}
		}
	} else if n == 2 {
		a, b := points[0].AgeDays, points[1].AgeDays // points отсортированы по убыванию возраста
		if a >= b*2+3 || (b > 0 && a/b >= 2.5) {
			outliers = append(outliers, points[0])
		}
	} else if n > 2 && n < 4 {
		for _, p := range points {
			if p.AgeDays >= median*2 && p.AgeDays-median >= 3 {
				outliers = append(outliers, p)
			}
		}
	} else if n == 1 && points[0].AgeDays >= 14 {
		outliers = append(outliers, points[0])
	}

	if len(outliers) > 0 {
		parts := make([]string, 0, len(outliers))
		for _, p := range outliers {
			parts = append(parts, fmt.Sprintf("%s (~%s дн., «%s»)", p.TaskKey, formatValue(p.AgeDays), p.ColumnName))
		}
		result += fmt.Sprintf(" **Внимание:** по сравнению с остальными выделяются «застывшие» задачи: %s. Вынесите их на ближайший дейли — обсудите блокеры, разбиение или передачу работы.",
			strings.Join(parts, "; "))
	} else {
		result += " Ярких выбросов по возрасту нет — распределение относительно ровное. При появлении долго висящих карточек отчёт подсветит их автоматически."
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
	startDate := now.AddDate(0, 0, -30)

	points := make([]wipHistoryPoint, 0, 31)
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

	last := points[n-1]
	var sum float64
	maxWip := 0
	exceedCount := 0
	for _, p := range points {
		sum += float64(p.Wip)
		if p.Wip > maxWip {
			maxWip = p.Wip
		}
		if limit != nil && p.Wip > *limit {
			exceedCount++
		}
	}
	avg := sum / float64(n)

	result := fmt.Sprintf("Диаграмма показывает количество задач в работе по дням. Сейчас в работе: %d, среднее за период: %s, максимум: %d.",
		last.Wip, formatValue(avg), maxWip)

	if limit != nil {
		if exceedCount == 0 {
			result += fmt.Sprintf(" WIP-лимит (%d) не превышался — дисциплина соблюдается.", *limit)
		} else {
			pct := math.Round(float64(exceedCount) / float64(n) * 100)
			result += fmt.Sprintf(" WIP-лимит (%d) превышался в %.0f%% дней. Рекомендация: либо снижайте WIP, либо пересмотрите лимит.", *limit, pct)
		}
	} else {
		result += " WIP-лимиты не установлены. Рекомендация: установите лимиты для контроля потока работы."
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

	report.Buckets = buildDistribution(values, 2)
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
	p85 := computePercentile(sorted, 85)

	result := fmt.Sprintf("Гистограмма показывает, за сколько дней завершается большинство задач. Из %d %s: медиана %s дн., среднее %s дн., 85-й процентиль %s дн.",
		n, pluralForm(n, "задачи", "задач", "задач"), formatValue(median), formatValue(avg), formatValue(p85))

	if p85 > avg*2 {
		result += " Есть длинный хвост — часть задач занимает значительно больше времени. Рекомендация: декомпозируйте крупные задачи."
	}

	result += fmt.Sprintf(" С вероятностью 85%% задача будет завершена за %s дней или меньше.", formatValue(p85))

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

	weeks := s.groupByWeeks(tasks, 12)
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
	median := computePercentile(sorted, 50)
	p85 := computePercentile(sorted, 85)

	result := fmt.Sprintf("Гистограмма показывает, сколько задач команда завершает за неделю. По данным %d %s: медиана %s, среднее %s, 85-й процентиль %s %s.",
		n, pluralForm(n, "недели", "недель", "недель"), formatValue(median), formatValue(avg), formatValue(p85),
		pluralForm(int(math.Round(p85)), "задача", "задачи", "задач"))

	p15 := computePercentile(sorted, 15)
	result += fmt.Sprintf(" С вероятностью 85%% за неделю будет завершено не менее %s %s.",
		formatValue(p15),
		pluralForm(int(math.Round(p15)), "задача", "задачи", "задач"))

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
