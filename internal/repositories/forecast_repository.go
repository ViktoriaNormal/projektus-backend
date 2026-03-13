package repositories

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
)

type ForecastRepository interface {
	GetCachedForecast(ctx context.Context, projectID uuid.UUID, workItemCount int) (*domain.ForecastResult, error)
	SaveForecast(ctx context.Context, forecast *domain.ForecastResult, ttl time.Duration) error
	CleanExpired(ctx context.Context) error
}

type forecastRepository struct {
	q *db.Queries
}

func NewForecastRepository(q *db.Queries) ForecastRepository {
	return &forecastRepository{q: q}
}

func (r *forecastRepository) GetCachedForecast(ctx context.Context, projectID uuid.UUID, workItemCount int) (*domain.ForecastResult, error) {
	row, err := r.q.GetForecastCache(ctx, db.GetForecastCacheParams{
		ProjectID:     projectID,
		WorkItemCount: int32(workItemCount),
	})
	if err != nil {
		return nil, err
	}
	var result domain.ForecastResult
	if err := json.Unmarshal(row.ForecastData, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *forecastRepository) SaveForecast(ctx context.Context, forecast *domain.ForecastResult, ttl time.Duration) error {
	data, err := json.Marshal(forecast)
	if err != nil {
		return err
	}
	_, err = r.q.SaveForecastCache(ctx, db.SaveForecastCacheParams{
		ProjectID:     forecast.ProjectID,
		WorkItemCount: int32(forecast.WorkItemCount),
		ForecastData:  data,
		ExpiresAt:     time.Now().Add(ttl),
	})
	return err
}

func (r *forecastRepository) CleanExpired(ctx context.Context) error {
	return r.q.CleanExpiredForecastCache(ctx)
}

