package services

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
)

type ScrumAnalyticsService struct {
	analyticsRepo repositories.ScrumAnalyticsRepository
	sprintRepo    repositories.SprintRepository
	cacheRepo     repositories.AnalyticsCacheRepository
}

func NewScrumAnalyticsService(analyticsRepo repositories.ScrumAnalyticsRepository, sprintRepo repositories.SprintRepository, cacheRepo repositories.AnalyticsCacheRepository) *ScrumAnalyticsService {
	return &ScrumAnalyticsService{
		analyticsRepo: analyticsRepo,
		sprintRepo:    sprintRepo,
		cacheRepo:     cacheRepo,
	}
}

// GetVelocityData возвращает данные для графика скорости.
func (s *ScrumAnalyticsService) GetVelocityData(ctx context.Context, projectID uuid.UUID) ([]domain.VelocityPoint, error) {
	const reportType = "scrum_velocity"
	params := struct{}{}

	// Пытаемся получить из кэша.
	if raw, err := s.cacheRepo.Get(ctx, projectID, reportType, params); err == nil {
		var points []domain.VelocityPoint
		if err := json.Unmarshal(raw, &points); err == nil {
			return points, nil
		}
	}

	points, err := s.analyticsRepo.GetVelocity(ctx, projectID)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(points)
	if err == nil {
		_ = s.cacheRepo.Save(ctx, projectID, reportType, params, data, time.Hour)
	}

	return points, nil
}

// GetBurndownData возвращает данные для диаграммы сгорания по спринту.
func (s *ScrumAnalyticsService) GetBurndownData(ctx context.Context, sprintID uuid.UUID) (*domain.BurndownData, error) {
	sprint, err := s.sprintRepo.GetByID(ctx, sprintID)
	if err != nil {
		return nil, err
	}

	points, err := s.analyticsRepo.GetBurndown(ctx, sprintID)
	if err != nil {
		return nil, err
	}

	total := 0
	for _, p := range points {
		if p.Date.Equal(sprint.StartDate) {
			total = p.RemainingPoints
			break
		}
	}

	// Рассчитываем идеальную линию.
	ideal := calculateIdealBurndown(total, sprint.StartDate, sprint.EndDate)
	for i := range points {
		points[i].IdealPoints = ideal[i].IdealPoints
	}

	return &domain.BurndownData{
		SprintID:    sprint.ID.String(),
		SprintName:  sprint.Name,
		StartDate:   sprint.StartDate,
		EndDate:     sprint.EndDate,
		TotalPoints: total,
		Points:      points,
	}, nil
}

func calculateIdealBurndown(totalPoints int, startDate, endDate time.Time) []domain.BurndownPoint {
	if totalPoints <= 0 {
		return nil
	}
	days := int(endDate.Sub(startDate).Hours()/24) + 1
	if days <= 0 {
		days = 1
	}
	step := float64(totalPoints) / float64(days-1)

	points := make([]domain.BurndownPoint, days)
	for i := 0; i < days; i++ {
		date := startDate.AddDate(0, 0, i)
		remaining := int(float64(totalPoints) - step*float64(i) + 0.5)
		if remaining < 0 {
			remaining = 0
		}
		points[i] = domain.BurndownPoint{
			Date:        date,
			IdealPoints: remaining,
		}
	}
	return points
}

