package services

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"time"

	"github.com/google/uuid"

	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
)

type MonteCarloForecastService struct {
	taskHistoryRepo repositories.TaskHistoryRepository
	forecastRepo    repositories.ForecastRepository
}

func NewMonteCarloForecastService(taskHistoryRepo repositories.TaskHistoryRepository, forecastRepo repositories.ForecastRepository) *MonteCarloForecastService {
	return &MonteCarloForecastService{
		taskHistoryRepo: taskHistoryRepo,
		forecastRepo:    forecastRepo,
	}
}

// GenerateForecast выполняет прогнозирование методом Монте-Карло, с использованием кэша.
func (s *MonteCarloForecastService) GenerateForecast(ctx context.Context, req domain.ForecastRequest) (*domain.ForecastResult, error) {
	if req.WorkItemCount <= 0 || req.Simulations <= 0 {
		return nil, domain.ErrInvalidInput
	}

	// Пытаемся взять из кэша.
	if cached, err := s.forecastRepo.GetCachedForecast(ctx, req.ProjectID, req.WorkItemCount); err == nil {
		return cached, nil
	}

	cycleTimes, err := s.GetHistoricalCycleTimes(ctx, req.ProjectID)
	if err != nil {
		return nil, err
	}
	if len(cycleTimes) < 10 {
		return nil, errors.New("недостаточно исторических данных для прогноза (минимум 10 завершенных задач)")
	}

	points, err := s.RunMonteCarloSimulation(cycleTimes, req.WorkItemCount, req.Simulations)
	if err != nil {
		return nil, err
	}

	result := &domain.ForecastResult{
		ProjectID:     req.ProjectID,
		WorkItemCount: req.WorkItemCount,
		Points:        points,
		GeneratedAt:   time.Now().UTC(),
	}

	// Кэшируем на час.
	_ = s.forecastRepo.SaveForecast(ctx, result, time.Hour)

	return result, nil
}

// GetHistoricalCycleTimes возвращает массив часов выполнения завершенных задач проекта.
func (s *MonteCarloForecastService) GetHistoricalCycleTimes(ctx context.Context, projectID uuid.UUID) ([]float64, error) {
	data, err := s.taskHistoryRepo.GetProjectCycleTimes(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, errors.New("нет завершенных задач для расчета cycle time")
	}
	values := make([]float64, 0, len(data))
	for _, d := range data {
		if d.CycleTimeHours > 0 {
			values = append(values, d.CycleTimeHours)
		}
	}
	if len(values) == 0 {
		return nil, errors.New("все cycle time нулевые или некорректные")
	}
	return values, nil
}

// RunMonteCarloSimulation моделирует распределение дат завершения workItemCount задач.
func (s *MonteCarloForecastService) RunMonteCarloSimulation(cycleTimes []float64, workItemCount int, simulations int) ([]domain.ForecastPoint, error) {
	if len(cycleTimes) == 0 {
		return nil, errors.New("нет данных для симуляции")
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	dateCounts := make(map[time.Time]int)

	startDate := time.Now().UTC().Truncate(24 * time.Hour)

	for i := 0; i < simulations; i++ {
		totalHours := 0.0
		for j := 0; j < workItemCount; j++ {
			idx := r.Intn(len(cycleTimes))
			totalHours += cycleTimes[idx]
		}
		days := int(math.Ceil(totalHours / 24.0))
		if days < 0 {
			days = 0
		}
		completeDate := startDate.AddDate(0, 0, days)
		dateCounts[completeDate]++
	}

	points := make([]domain.ForecastPoint, 0, len(dateCounts))
	for date, count := range dateCounts {
		prob := float64(count) * 100.0 / float64(simulations)
		points = append(points, domain.ForecastPoint{
			Date:        date,
			Probability: prob,
		})
	}

	return points, nil
}

