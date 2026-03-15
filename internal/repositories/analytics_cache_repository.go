package repositories

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
)

type AnalyticsCacheRepository interface {
	Get(ctx context.Context, projectID uuid.UUID, reportType string, params any) ([]byte, error)
	Save(ctx context.Context, projectID uuid.UUID, reportType string, params any, result []byte, ttl time.Duration) error
	CleanExpired(ctx context.Context) error
}

type analyticsCacheRepository struct {
	q *db.Queries
}

func NewAnalyticsCacheRepository(q *db.Queries) AnalyticsCacheRepository {
	return &analyticsCacheRepository{q: q}
}

func (r *analyticsCacheRepository) Get(ctx context.Context, projectID uuid.UUID, reportType string, params any) ([]byte, error) {
	rawParams, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	row, err := r.q.GetAnalyticsCache(ctx, db.GetAnalyticsCacheParams{
		ProjectID:  projectID,
		ReportType: reportType,
		Parameters: rawParams,
	})
	if err != nil {
		return nil, err
	}
	return row.ResultData, nil
}

func (r *analyticsCacheRepository) Save(ctx context.Context, projectID uuid.UUID, reportType string, params any, result []byte, ttl time.Duration) error {
	rawParams, err := json.Marshal(params)
	if err != nil {
		return err
	}
	_, err = r.q.SaveAnalyticsCache(ctx, db.SaveAnalyticsCacheParams{
		ProjectID:  projectID,
		ReportType: reportType,
		Parameters: rawParams,
		ResultData: result,
		ExpiresAt:  time.Now().Add(ttl),
	})
	return err
}

func (r *analyticsCacheRepository) CleanExpired(ctx context.Context) error {
	return r.q.CleanExpiredAnalyticsCache(ctx)
}

