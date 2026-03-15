package repositories

import (
	"context"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
)

type ScrumAnalyticsRepository interface {
	GetVelocity(ctx context.Context, projectID uuid.UUID) ([]domain.VelocityPoint, error)
	GetBurndown(ctx context.Context, sprintID uuid.UUID) ([]domain.BurndownPoint, error)
}

type scrumAnalyticsRepository struct {
	q *db.Queries
}

func NewScrumAnalyticsRepository(q *db.Queries) ScrumAnalyticsRepository {
	return &scrumAnalyticsRepository{q: q}
}

func (r *scrumAnalyticsRepository) GetVelocity(ctx context.Context, projectID uuid.UUID) ([]domain.VelocityPoint, error) {
	rows, err := r.q.GetVelocityData(ctx, projectID)
	if err != nil {
		return nil, err
	}
	result := make([]domain.VelocityPoint, 0, len(rows))
	for _, row := range rows {
		result = append(result, domain.VelocityPoint{
			SprintID:        row.SprintID.String(),
			SprintName:      row.SprintName,
			StartDate:       row.StartDate,
			EndDate:         row.EndDate,
			CommittedPoints: int(row.CommittedPoints),
			CompletedPoints: int(row.CompletedPoints),
		})
	}
	return result, nil
}

func (r *scrumAnalyticsRepository) GetBurndown(ctx context.Context, sprintID uuid.UUID) ([]domain.BurndownPoint, error) {
	rows, err := r.q.GetBurndownData(ctx, sprintID)
	if err != nil {
		return nil, err
	}
	points := make([]domain.BurndownPoint, 0, len(rows))
	for _, row := range rows {
		points = append(points, domain.BurndownPoint{
			Date:            row.Day,
			RemainingPoints: int(row.RemainingPoints),
		})
	}
	return points, nil
}

