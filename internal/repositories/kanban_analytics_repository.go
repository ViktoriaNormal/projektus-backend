package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
)

type KanbanAnalyticsRepository interface {
	GetCfdColumnCountsByDate(ctx context.Context, projectID uuid.UUID, boardID uuid.UUID, startDate, endDate time.Time) ([]domain.CumulativeFlowPoint, error)
	GetThroughput(ctx context.Context, projectID uuid.UUID, startDate, endDate time.Time, groupBy string) ([]domain.ThroughputPoint, error)
	GetWipOverTime(ctx context.Context, projectID uuid.UUID, startDate, endDate time.Time) ([]domain.WipPoint, error)
	GetWipWithAge(ctx context.Context, projectID uuid.UUID, startDate, endDate time.Time) ([]domain.WipPoint, error)
	GetCycleTimeScatterplot(ctx context.Context, projectID uuid.UUID, startDate, endDate time.Time, classOfService *string) ([]domain.CycleTimePoint, error)
	GetAverageCycleTimeByPeriod(ctx context.Context, projectID uuid.UUID, startDate, endDate time.Time, classOfService *string, period string) ([]domain.AverageCycleTimePoint, error)
}

type kanbanAnalyticsRepository struct {
	q *db.Queries
}

func NewKanbanAnalyticsRepository(q *db.Queries) KanbanAnalyticsRepository {
	return &kanbanAnalyticsRepository{q: q}
}

func (r *kanbanAnalyticsRepository) GetCfdColumnCountsByDate(ctx context.Context, projectID uuid.UUID, boardID uuid.UUID, startDate, endDate time.Time) ([]domain.CumulativeFlowPoint, error) {
	rows, err := r.q.GetCfdColumnCountsByDate(ctx, db.GetCfdColumnCountsByDateParams{
		ProjectID: projectID,
		Column2:   startDate,
		Column3:   endDate,
		BoardID:   boardID,
	})
	if err != nil {
		return nil, err
	}

	// Группируем по дате, строим накопительные счётчики по порядку колонок.
	type colCount struct {
		order int16
		name  string
		count int
	}
	byDate := make(map[time.Time][]colCount)
	var datesOrdered []time.Time
	dateSeen := make(map[time.Time]bool)

	for _, row := range rows {
		var d time.Time
		if row.Date.Valid {
			d = row.Date.Time
		} else {
			continue
		}
		if !dateSeen[d] {
			datesOrdered = append(datesOrdered, d)
			dateSeen[d] = true
		}
		byDate[d] = append(byDate[d], colCount{
			order: row.ColumnOrder,
			name:  row.ColumnName,
			count: int(row.TaskCount),
		})
	}

	// Сортируем колонки внутри каждой даты по order и считаем накопительно.
	result := make([]domain.CumulativeFlowPoint, 0, len(datesOrdered))
	for _, d := range datesOrdered {
		cols := byDate[d]
		// Сортируем по column_order
		for i := 0; i < len(cols); i++ {
			for j := i + 1; j < len(cols); j++ {
				if cols[j].order < cols[i].order {
					cols[i], cols[j] = cols[j], cols[i]
				}
			}
		}
		cumulative := make(map[string]int)
		var run int
		for _, c := range cols {
			run += c.count
			cumulative[c.name] = run
		}
		result = append(result, domain.CumulativeFlowPoint{
			Date:         d,
			StatusCounts: cumulative,
		})
	}
	return result, nil
}

func (r *kanbanAnalyticsRepository) GetThroughput(ctx context.Context, projectID uuid.UUID, startDate, endDate time.Time, groupBy string) ([]domain.ThroughputPoint, error) {
	if groupBy == "week" {
		rows, err := r.q.GetThroughputByPeriod(ctx, db.GetThroughputByPeriodParams{
			ProjectID: projectID,
			Column2:   startDate,
			Column3:   endDate,
			Column4:   "week",
		})
		if err != nil {
			return nil, err
		}
		return mapThroughputByPeriod(rows), nil
	}
	rows, err := r.q.GetThroughputSimple(ctx, db.GetThroughputSimpleParams{
		ProjectID: projectID,
		Column2:   startDate,
		Column3:   endDate,
	})
	if err != nil {
		return nil, err
	}
	points := make([]domain.ThroughputPoint, 0, len(rows))
	var cum int
	for _, row := range rows {
		cum += int(row.TaskCount)
		points = append(points, domain.ThroughputPoint{
			PeriodStart:     row.DayStart,
			ClassOfService:  nil,
			TaskCount:       int(row.TaskCount),
			CumulativeCount: cum,
		})
	}
	return points, nil
}

func mapThroughputByPeriod(rows []db.GetThroughputByPeriodRow) []domain.ThroughputPoint {
	points := make([]domain.ThroughputPoint, 0, len(rows))
	for _, row := range rows {
		var cos *string
		if row.ClassOfService.Valid {
			cos = &row.ClassOfService.String
		}
		points = append(points, domain.ThroughputPoint{
			PeriodStart:     row.PeriodStart,
			ClassOfService:  cos,
			TaskCount:       int(row.TaskCount),
			CumulativeCount: 0, // при разбивке по классам накопительный итог считается на фронте при необходимости
		})
	}
	return points
}

func (r *kanbanAnalyticsRepository) GetWipOverTime(ctx context.Context, projectID uuid.UUID, startDate, endDate time.Time) ([]domain.WipPoint, error) {
	rows, err := r.q.GetWipOverTime(ctx, db.GetWipOverTimeParams{
		ProjectID: projectID,
		Column2:   startDate,
		Column3:   endDate,
	})
	if err != nil {
		return nil, err
	}
	points := make([]domain.WipPoint, 0, len(rows))
	for _, row := range rows {
		points = append(points, domain.WipPoint{
			Date:     row.Date,
			WipCount: int(row.WipCount),
		})
	}
	return points, nil
}

func (r *kanbanAnalyticsRepository) GetWipWithAge(ctx context.Context, projectID uuid.UUID, startDate, endDate time.Time) ([]domain.WipPoint, error) {
	rows, err := r.q.GetWipWithAge(ctx, db.GetWipWithAgeParams{
		ProjectID: projectID,
		Column2:   startDate,
		Column3:   endDate,
	})
	if err != nil {
		return nil, err
	}
	points := make([]domain.WipPoint, 0, len(rows))
	for _, row := range rows {
		points = append(points, domain.WipPoint{
			Date:      row.Date,
			WipCount:  int(row.WipCount),
			AvgWipAge: row.AvgWipAgeDays,
			MaxWipAge: row.MaxWipAgeDays,
		})
	}
	return points, nil
}

func (r *kanbanAnalyticsRepository) GetCycleTimeScatterplot(ctx context.Context, projectID uuid.UUID, startDate, endDate time.Time, classOfService *string) ([]domain.CycleTimePoint, error) {
	arg := db.GetCycleTimeScatterplotParams{
		ProjectID: projectID,
		Column2:   startDate,
		Column3:   endDate,
		Column4:   "",
	}
	if classOfService != nil && *classOfService != "" {
		arg.Column4 = *classOfService
	}
	rows, err := r.q.GetCycleTimeScatterplot(ctx, arg)
	if err != nil {
		return nil, err
	}
	points := make([]domain.CycleTimePoint, 0, len(rows))
	for _, row := range rows {
		completedAt := parseTime(row.CompletedAt)
		var cos *string
		if row.ClassOfService.Valid {
			cos = &row.ClassOfService.String
		}
		points = append(points, domain.CycleTimePoint{
			TaskID:         row.TaskID.String(),
			TaskKey:        row.TaskKey,
			ClassOfService: cos,
			CompletedAt:    completedAt,
			CycleTimeDays:  row.CycleTimeDays,
		})
	}
	return points, nil
}

func (r *kanbanAnalyticsRepository) GetAverageCycleTimeByPeriod(ctx context.Context, projectID uuid.UUID, startDate, endDate time.Time, classOfService *string, period string) ([]domain.AverageCycleTimePoint, error) {
	if period != "day" && period != "week" && period != "month" {
		period = "day"
	}
	arg := db.GetAverageCycleTimeByPeriodParams{
		ProjectID: projectID,
		Column2:   startDate,
		Column3:   endDate,
		Column4:   "",
		Column5:   period,
	}
	if classOfService != nil && *classOfService != "" {
		arg.Column4 = *classOfService
	}
	rows, err := r.q.GetAverageCycleTimeByPeriod(ctx, arg)
	if err != nil {
		return nil, err
	}
	points := make([]domain.AverageCycleTimePoint, 0, len(rows))
	for _, row := range rows {
		var cos *string
		if row.ClassOfService.Valid {
			cos = &row.ClassOfService.String
		}
		points = append(points, domain.AverageCycleTimePoint{
			PeriodStart:      row.PeriodStart,
			ClassOfService:   cos,
			AvgCycleTimeDays: row.AvgCycleTimeDays,
			TaskCount:        int(row.TaskCount),
		})
	}
	return points, nil
}

func parseTime(v interface{}) time.Time {
	if v == nil {
		return time.Time{}
	}
	if t, ok := v.(time.Time); ok {
		return t
	}
	return time.Time{}
}
