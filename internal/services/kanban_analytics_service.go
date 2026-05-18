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
	// kanbanHistoryWindowDays — окно операционного обзора Канбана.
	// 30 дней соответствуют месячной каденции операционного обзора,
	// принятой в практике Канбана (Anderson, 2017).
	kanbanHistoryWindowDays = 30
	// cfdBottleneckCoverageDays — длительность поставки, которую должно
	// «съесть» накопление в колонке, чтобы считаться узким местом.
	// 7 дней = одна полная неделя пропускной способности команды.
	cfdBottleneckCoverageDays         = 7
	throughputTrendWindowWeeks        = 8
	throughputDistributionWindowWeeks = 12
	// throughputTrendRelativeThreshold — порог значимости тренда
	// пропускной способности как доля от средней недельной поставки.
	// Симметрично с velocityTrendRelativeThreshold: тренд считается
	// значимым начиная с 5 % средней пропускной способности.
	throughputTrendRelativeThreshold    = 0.05
	wipRiskPercentile                   = 85.0
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
	columnSystemTypes    map[string]string // column name -> system_type
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

// scatterDisplayOrder — стабильный псевдослучайный порядок задач на оси X scatter-графика.
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

func isKanbanInProgressColumn(systemType string) bool {
	return systemType == string(domain.StatusInProgress)
}

type cfdColumnGrowth struct {
	name   string
	growth int
}

func cfdGrowthWeeks(growth int, weeklyThroughput float64) float64 {
	if weeklyThroughput <= 0 {
		return 0
	}
	return float64(growth) / weeklyThroughput
}

// cfdStuckVerb: «зависла» — с «неделя» (≈1 нед.), иначе нейтральное «зависло».
func cfdStuckVerb(weeks float64) string {
	if weeks >= 0.95 && weeks < 1.15 {
		return "зависла"
	}
	return "зависло"
}

func cfdWeeksWorkPhrase(weeks float64) string {
	w := math.Round(weeks*10) / 10
	if w >= 0.95 && w < 1.15 {
		return "примерно 1 неделя работы команды"
	}
	n := int(math.Round(w))
	if math.Abs(w-float64(n)) < 0.05 && n >= 2 {
		return fmt.Sprintf("примерно %d %s работы команды", n, pluralForm(n, "недели", "недели", "недель"))
	}
	return fmt.Sprintf("примерно %s недели работы команды", formatValue(w))
}

func cfdJoinQuotedNames(names []string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return "«" + names[0] + "»"
	case 2:
		return "«" + names[0] + "» и «" + names[1] + "»"
	default:
		return "«" + strings.Join(names[:len(names)-1], "», «") + "» и «" + names[len(names)-1] + "»"
	}
}

func cfdColumnMaxJump(points []cfdDayPoint, column string) (delta int, fromDate, toDate string) {
	if len(points) < 2 {
		return 0, "", ""
	}
	bestDelta := 0
	bestFrom := ""
	bestTo := ""
	for i := 1; i < len(points); i++ {
		prev := points[i-1].Counts[column]
		cur := points[i].Counts[column]
		d := cur - prev
		if d > bestDelta {
			bestDelta = d
			bestFrom = points[i-1].Date
			bestTo = points[i].Date
		}
	}
	return bestDelta, bestFrom, bestTo
}

func (s *KanbanAnalyticsService) generateCFDInterpretation(r *CFDReport, weeklyThroughput float64) string {
	if len(r.Points) < 2 {
		return "Недостаточно данных для анализа потока."
	}
	_ = weeklyThroughput

	first, last := r.Points[0], r.Points[len(r.Points)-1]

	var wipCongestion []cfdColumnGrowth
	var wipReleased []cfdColumnGrowth
	totalWipGrowth := 0
	for _, name := range r.ColumnNames {
		if !isKanbanInProgressColumn(r.columnSystemTypes[name]) {
			continue
		}
		growth := last.Counts[name] - first.Counts[name]
		totalWipGrowth += growth
		if growth > 0 {
			wipCongestion = append(wipCongestion, cfdColumnGrowth{name: name, growth: growth})
		}
		if growth < 0 {
			wipReleased = append(wipReleased, cfdColumnGrowth{name: name, growth: growth})
		}
	}
	sort.Slice(wipCongestion, func(i, j int) bool {
		return wipCongestion[i].growth > wipCongestion[j].growth
	})
	sort.Slice(wipReleased, func(i, j int) bool {
		return wipReleased[i].growth < wipReleased[j].growth
	})

	doneGrowth := 0
	for _, name := range r.completedColumnNames {
		doneGrowth += last.Counts[name] - first.Counts[name]
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf(
		"За период %s–%s показывает, как менялся баланс между завершением и накоплением задач по стадиям.\n",
		r.Points[0].Date, r.Points[len(r.Points)-1].Date,
	))

	switch {
	case doneGrowth > 0:
		b.WriteString(fmt.Sprintf(
			"Полоса «Готово» выросла на %d %s, то есть поток завершения в целом сохранялся.\n",
			doneGrowth,
			pluralForm(doneGrowth, "задачу", "задачи", "задач"),
		))
	case doneGrowth == 0:
		b.WriteString("Полоса «Готово» почти не изменилась, поэтому завершение задач в этот период было редким.\n")
	default:
		b.WriteString("Размер «Готово» снизился, поэтому стоит проверить историю перемещений: часть задач могла вернуться из финальной стадии в работу.\n")
	}

	if len(wipCongestion) > 0 {
		main := wipCongestion[0]
		jump, fromDate, toDate := cfdColumnMaxJump(r.Points, main.name)
		b.WriteString(fmt.Sprintf(
			"На этом фоне наиболее заметно расширилась полоса «%s»: %+d %s за период (с %d до %d задач).",
			main.name,
			main.growth,
			pluralForm(main.growth, "задача", "задачи", "задач"),
			first.Counts[main.name],
			last.Counts[main.name],
		))
		if jump > 0 {
			b.WriteString(fmt.Sprintf(" Самый резкий скачок пришёлся на %s–%s: +%d.", fromDate, toDate, jump))
		}
		b.WriteString("\n")

		if len(wipCongestion) > 1 {
			otherNames := make([]string, 0, len(wipCongestion)-1)
			for _, c := range wipCongestion[1:] {
				otherNames = append(otherNames, c.name)
			}
			b.WriteString(fmt.Sprintf(
				"Дополнительно расширялись %s, поэтому затор выглядит не локальным, а распределённым по нескольким стадиям.\n",
				cfdJoinQuotedNames(otherNames),
			))
		}

		b.WriteString("Рекомендация: в ближайшем цикле полезно мягко ограничить вход новых задач в расширившиеся стадии и сфокусироваться на продвижении уже начатых элементов.")
	} else {
		b.WriteString("Полосы рабочих колонок заметно не расширялись, поэтому выраженного накопления незавершённой работы на графике не видно.\n")
	}

	if totalWipGrowth > 0 && doneGrowth <= 0 {
		b.WriteString("Важно и то, что фронт работ сместился в незавершённые стадии: WIP растёт быстрее, чем объём завершений.\n")
	}

	if len(wipReleased) > 0 {
		mainRelease := wipReleased[0]
		b.WriteString(fmt.Sprintf(
			"При этом есть позитивный сигнал: в «%s» объём сократился на %d %s, а значит стадия уже частично разгружается.\n",
			mainRelease.name,
			-mainRelease.growth,
			pluralForm(-mainRelease.growth, "задачу", "задачи", "задач"),
		))
	}

	return strings.TrimSpace(b.String())
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
	// Порядок по псевдослучайному ключу задачи — не по cycle time / дате завершения,
	// чтобы на scatter-диаграмме формировалось «облако», а не одна линия.
	sort.Slice(points, func(i, j int) bool {
		return scatterDisplayOrder(points[i].TaskKey) < scatterDisplayOrder(points[j].TaskKey)
	})
	report.Points = points

	report.Interpretation = s.generateScatterInterpretation(cycleTimes)
	return report, nil
}

func (s *KanbanAnalyticsService) generateScatterInterpretation(cycleTimes []float64) string {
	n := len(cycleTimes)
	sorted := make([]float64, n)
	copy(sorted, cycleTimes)
	sort.Float64s(sorted)

	if n == 0 {
		return "Недостаточно данных для интерпретации графика."
	}

	q1 := computePercentile(sorted, 25)
	median := computePercentile(sorted, 50)
	q3 := computePercentile(sorted, 75)
	iqr := q3 - q1
	upperFence := q3 + 1.5*iqr
	maxVal := sorted[n-1]

	outliers := 0
	for _, v := range sorted {
		if v > upperFence {
			outliers++
		}
	}
	bodyCount := n - outliers
	visibleTail := false
	if iqr > 0 && outliers >= 4 {
		share := float64(outliers) / float64(n)
		visibleTail = share >= 0.08 && maxVal > q3+2*iqr
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf(
		"На графике основная плотность точек сосредоточена в диапазоне %s–%s дн., а центр облака находится около %s дн.\n",
		formatValue(q1), formatValue(q3), formatValue(median),
	))
	b.WriteString(fmt.Sprintf(
		"Полный разброс между самыми короткими и самыми длинными задачами составляет %s–%s дн., однако большая часть точек остаётся рядом с центральной зоной.\n",
		formatValue(sorted[0]), formatValue(maxVal),
	))

	if visibleTail {
		b.WriteString(fmt.Sprintf(
			"В правой части графика выделяется группа более длинных задач: %d %s выше основного диапазона, максимум достигает %s дн.\n",
			outliers,
			pluralForm(outliers, "точка", "точки", "точек"),
			formatValue(maxVal),
		))
	} else {
		b.WriteString("Сильного отрыва точек от основной массы не наблюдается, поэтому облако выглядит ровным и однородным.\n")
	}

	if bodyCount > 0 {
		share := math.Round(float64(bodyCount)/float64(n)*1000) / 10
		b.WriteString(fmt.Sprintf(
			"Около %s%% задач находится внутри основного диапазона, а это обычно соответствует управляемому и повторяемому ритму.\n",
			formatValue(share),
		))
	}

	if visibleTail {
		b.WriteString("Рекомендация: полезно отдельно разобрать самые долгие задачи и спокойно проверить, где чаще возникают блокировки, ожидания или избыточный объём в одном элементе.")
	} else {
		b.WriteString("Рекомендация: можно сохранить текущий размер задач и периодически просматривать верхние точки, чтобы они не начали формировать устойчивую группу задержек.")
	}

	return b.String()
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

	var sum int
	maxIdx := 0
	minIdx := 0
	zigzags := 0
	lastSign := 0
	for _, w := range weeks {
		sum += w.Count
	}
	for i, w := range weeks {
		if w.Count > weeks[maxIdx].Count {
			maxIdx = i
		}
		if w.Count < weeks[minIdx].Count {
			minIdx = i
		}
		if i == 0 {
			continue
		}
		delta := weeks[i].Count - weeks[i-1].Count
		sign := 0
		if delta > 0 {
			sign = 1
		} else if delta < 0 {
			sign = -1
		}
		if sign != 0 && lastSign != 0 && sign != lastSign {
			zigzags++
		}
		if sign != 0 {
			lastSign = sign
		}
	}
	avg := float64(sum) / float64(n)
	trendThreshold := throughputTrendRelativeThreshold * avg

	var b strings.Builder
	b.WriteString(fmt.Sprintf(
		"На графике недельной пропускной способности видно, что максимум пришёлся на %s (%d задач), а минимум — на %s (%d задач).\n",
		weeks[maxIdx].Week, weeks[maxIdx].Count, weeks[minIdx].Week, weeks[minIdx].Count,
	))
	b.WriteString(fmt.Sprintf("Средний фактический уровень за период составляет %s задач в неделю, и это можно считать базовым рабочим ритмом команды.\n", formatValue(avg)))
	if slope > trendThreshold {
		b.WriteString(fmt.Sprintf("Линия тренда направлена вверх (%s задач/нед), поэтому к концу периода столбцы в среднем выше.\n", formatValue(slope)))
	} else if slope < -trendThreshold {
		b.WriteString(fmt.Sprintf("Линия тренда идёт вниз (%s задач/нед), поэтому к концу периода столбцы в среднем ниже.\n", formatValue(slope)))
	} else {
		b.WriteString("Линия тренда почти горизонтальна, а значит базовый недельный ритм держится без выраженного сдвига.\n")
	}

	if zigzags >= 3 {
		b.WriteString("При этом форма столбцов остаётся зигзагообразной: подъёмы и просадки регулярно сменяют друг друга.\n")
	} else {
		b.WriteString("Колебания между соседними неделями умеренные, без выраженной «пилы», поэтому динамика выглядит достаточно спокойной.\n")
	}

	if slope < -trendThreshold {
		b.WriteString("Рекомендация: имеет смысл на ретро последовательно разобрать недели с просадкой и уточнить, где поток терял темп — на блокировках, ожиданиях или частых переключениях контекста.")
	} else {
		b.WriteString("Рекомендация: можно зафиксировать текущий ритм как рабочий стандарт и отдельно разбирать только недели, где столбцы заметно ниже обычного диапазона.")
	}

	return b.String()
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
		// Один знак после запятой — как formatValue в интерпретациях.
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
		Interpretation: s.generateWipAgeInterpretation(points, alertCount, p85),
	}
	return report, nil
}

func (s *KanbanAnalyticsService) generateWipAgeInterpretation(points []wipAgePoint, alertCount int, p85 float64) string {
	wipCount := len(points)
	if wipCount == 0 {
		return "На доске нет задач в работе. Можно взять новую задачу."
	}

	columnCounts := make(map[string]int)
	for _, p := range points {
		columnCounts[p.ColumnName]++
	}
	topColumn := ""
	topColumnCount := 0
	for col, count := range columnCounts {
		if count > topColumnCount {
			topColumn = col
			topColumnCount = count
		}
	}

	oldestCount := 3
	if oldestCount > len(points) {
		oldestCount = len(points)
	}
	oldest := points[:oldestCount]

	var b strings.Builder
	b.WriteString(fmt.Sprintf("На диаграмме возраста в работе сейчас %d %s, и это формирует текущий активный WIP команды.\n", wipCount, pluralForm(wipCount, "задача", "задачи", "задач")))
	if topColumn != "" {
		b.WriteString(fmt.Sprintf(
			"Наибольшая концентрация задач находится в «%s» — %d %s.\n",
			topColumn,
			topColumnCount,
			pluralForm(topColumnCount, "задача", "задачи", "задач"),
		))
	}
	b.WriteString("Если смотреть на верхнюю часть диаграммы, самые «старые» элементы сейчас такие: ")
	for i, p := range oldest {
		part := fmt.Sprintf("%s (%s дн., %s)", p.TaskKey, formatValue(p.AgeDays), p.ColumnName)
		if i == 0 {
			b.WriteString(part)
		} else {
			b.WriteString("; " + part)
		}
	}
	b.WriteString(".\n")
	if len(points) > 1 {
		b.WriteString(fmt.Sprintf(
			"Разница между самой молодой и самой старой задачей составляет %s дн., и по ней можно оценить, насколько равномерно движется текущий WIP.\n",
			formatValue(points[0].AgeDays-points[len(points)-1].AgeDays),
		))
	}

	if p85 > 0 {
		b.WriteString(fmt.Sprintf("В качестве исторического ориентира верхняя граница нормального возраста сейчас около %s дн.\n", formatValue(p85)))
		if alertCount > 0 {
			b.WriteString(fmt.Sprintf(
				"%d %s уже вышли за эту границу, поэтому на графике заметны длинные полосы, которые постепенно повышают общий возраст WIP.\n",
				alertCount, pluralForm(alertCount, "задача", "задачи", "задач"),
			))
			b.WriteString("Рекомендация: на daily полезно начинать обсуждение именно с этих задач и помогать им сдвинуться; новые задачи в перегруженную стадию лучше добавлять только после заметного прогресса по «старым» элементам.")
		} else {
			b.WriteString("Пока все текущие задачи остаются в рабочем диапазоне возраста, без признаков устойчивого зависания.\n")
			b.WriteString("Рекомендация: можно удерживать текущий WIP и следить, чтобы верхние полосы регулярно продвигались по доске.")
		}
	} else {
		b.WriteString("Исторических данных пока недостаточно, поэтому ориентир здесь в первую очередь визуальный: чем дольше верхние полосы остаются без движения, тем выше риск застревания.\n")
		b.WriteString("Рекомендация: полезно удерживать фокус на 2-3 самых старых задачах до их явного продвижения или завершения.")
	}

	return b.String()
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
		return "Нет данных о незавершённой работе."
	}

	last := points[n-1]
	maxWip := 0
	maxStartIdx := 0
	maxEndIdx := 0
	exceedCount := 0
	exceedStart := -1
	longestExceedStart := -1
	longestExceedEnd := -1
	for _, p := range points {
		if p.Wip > maxWip {
			maxWip = p.Wip
		}
	}
	for i, p := range points {
		if p.Wip == maxWip {
			start := i
			end := i
			for end+1 < len(points) && points[end+1].Wip == maxWip {
				end++
			}
			if end-start > maxEndIdx-maxStartIdx {
				maxStartIdx = start
				maxEndIdx = end
			}
			i = end
		}
	}
	if limit != nil {
		for i, p := range points {
			if p.Wip > *limit {
				exceedCount++
				if exceedStart == -1 {
					exceedStart = i
				}
			} else if exceedStart != -1 {
				if longestExceedStart == -1 || i-1-exceedStart > longestExceedEnd-longestExceedStart {
					longestExceedStart = exceedStart
					longestExceedEnd = i - 1
				}
				exceedStart = -1
			}
		}
		if exceedStart != -1 && (longestExceedStart == -1 || len(points)-1-exceedStart > longestExceedEnd-longestExceedStart) {
			longestExceedStart = exceedStart
			longestExceedEnd = len(points) - 1
		}
	}

	result := fmt.Sprintf(
		"Текущий WIP составляет %d %s, а максимальная загрузка за период наблюдалась %s–%s и достигала %d %s.\n",
		last.Wip,
		pluralForm(last.Wip, "задача", "задачи", "задач"),
		points[maxStartIdx].Date,
		points[maxEndIdx].Date,
		maxWip,
		pluralForm(maxWip, "задача", "задачи", "задач"),
	)
	minWip := points[0].Wip
	for _, p := range points {
		if p.Wip < minWip {
			minWip = p.Wip
		}
	}
	result += fmt.Sprintf("В целом диапазон колебаний за период — от %d до %d задач в работе, и это показывает амплитуду рабочей нагрузки.\n", minWip, maxWip)

	if limit != nil {
		if exceedCount == 0 {
			result += fmt.Sprintf("Линия лимита (%d) не пересекалась, поэтому нагрузка удерживалась в договорённой зоне.\n", *limit)
			result += "Рекомендация: можно сохранять текущий режим и пересматривать лимит только при устойчивом дефиците пропускной способности."
		} else {
			pct := math.Round(float64(exceedCount) / float64(n) * 100)
			result += fmt.Sprintf("Лимит WIP (%d) превышался в %.0f%% дней.", *limit, pct)
			if longestExceedStart >= 0 {
				result += fmt.Sprintf(" Самый длинный непрерывный период превышения: %s–%s.", points[longestExceedStart].Date, points[longestExceedEnd].Date)
			}
			result += "\nРекомендация: в периоды пересечения лимита полезно сначала закрывать начатые задачи и только затем расширять входящий поток новыми элементами."
		}
	} else {
		result += "Лимит не задан, поэтому график фиксирует колебания, но не показывает, где начинается перегрузка.\n"
		result += "Рекомендация: стоит задать WIP-лимит хотя бы для ключевых рабочих колонок, чтобы пики воспринимались как управляемый сигнал, а не как случайные колебания."
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

	if len(sorted) == 0 {
		return "Недостаточно данных для интерпретации распределения."
	}

	buckets := buildDistribution(sorted, 0)
	median := computePercentile(sorted, 50)
	q1 := computePercentile(sorted, 25)
	q3 := computePercentile(sorted, 75)
	iqr := q3 - q1
	upperFence := q3 + 1.5*iqr
	tailCount := 0
	for _, v := range sorted {
		if v > upperFence {
			tailCount++
		}
	}
	visibleTail := false
	if iqr > 0 && tailCount >= 4 {
		share := float64(tailCount) / float64(len(sorted))
		visibleTail = share >= 0.08 && sorted[len(sorted)-1] > q3+2*iqr
	}

	modeIdx := 0
	for i := range buckets {
		if buckets[i].Count > buckets[modeIdx].Count {
			modeIdx = i
		}
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf(
		"Пик гистограммы находится в диапазоне %s, то есть этот срок выполнения встречается чаще всего. Центр распределения расположен около %s дн.\n",
		buckets[modeIdx].RangeLabel,
		formatValue(median),
	))
	b.WriteString(fmt.Sprintf(
		"Основная масса задач лежит в интервале %s–%s дн., поэтому именно этот диапазон лучше всего описывает обычный ритм выполнения.\n",
		formatValue(q1),
		formatValue(q3),
	))

	if visibleTail {
		b.WriteString(fmt.Sprintf(
			"В правой части гистограммы есть заметная группа более длинных задач: %d %s, при этом максимум достигает %s дн.\n",
			tailCount,
			pluralForm(tailCount, "задача", "задачи", "задач"),
			formatValue(sorted[len(sorted)-1]),
		))
		b.WriteString("Рекомендация: полезно разобрать задачи из этой группы по причинам задержки и заранее обозначить мягкий триггер эскалации для похожих случаев.")
	} else {
		b.WriteString("Сильного отрыва правой части не видно, поэтому распределение остаётся ровным и контролируемым.\n")
		b.WriteString("Рекомендация: можно сохранить текущий уровень декомпозиции и регулярный контроль блокировок.")
	}

	return b.String()
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
	if n == 0 {
		return "Недостаточно данных для интерпретации распределения пропускной способности."
	}

	buckets := buildDistribution(sorted, 0)
	modeIdx := 0
	for i := range buckets {
		if buckets[i].Count > buckets[modeIdx].Count {
			modeIdx = i
		}
	}

	zeroWeeks := 0
	for _, v := range sorted {
		if v == 0 {
			zeroWeeks++
		}
	}

	q1 := computePercentile(sorted, 25)
	q3 := computePercentile(sorted, 75)
	median := computePercentile(sorted, 50)

	var b strings.Builder
	b.WriteString(fmt.Sprintf(
		"На гистограмме самый частый недельный диапазон — %s, а центр распределения находится около %s задач.\n",
		buckets[modeIdx].RangeLabel,
		formatValue(median),
	))
	b.WriteString(fmt.Sprintf(
		"Основной рабочий диапазон по неделям — %s–%s задач, и именно здесь лежит типичная недельная отдача команды.\n",
		formatValue(q1), formatValue(q3),
	))
	if zeroWeeks > 0 {
		b.WriteString(fmt.Sprintf(
			"Также есть %d %s без завершений, из-за чего левая часть распределения выглядит просаженной.\n",
			zeroWeeks, pluralForm(zeroWeeks, "неделя", "недели", "недель"),
		))
		b.WriteString("Рекомендация: имеет смысл отдельно разобрать такие недели и проверить, не были ли они связаны с блокировками, ожиданиями или смещением фокуса на незавершённые задачи.")
	} else {
		b.WriteString("Нулевых недель нет, поэтому распределение стартует выше нуля и выглядит рабочим по всей шкале.\n")
		b.WriteString("Рекомендация: можно удерживать текущий ритм и разбирать только те недели, которые заметно выпадают ниже основного диапазона.")
	}

	return b.String()
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
