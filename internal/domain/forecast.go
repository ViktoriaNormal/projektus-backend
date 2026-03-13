package domain

import (
	"time"

	"github.com/google/uuid"
)

type ForecastPoint struct {
	Date        time.Time `json:"date"`
	Probability float64   `json:"probability"` // 0..100
}

type ForecastRequest struct {
	ProjectID        uuid.UUID
	WorkItemCount    int
	Simulations      int
	ConfidenceLevels []float64
}

type ForecastResult struct {
	ProjectID     uuid.UUID       `json:"projectId"`
	WorkItemCount int             `json:"workItemCount"`
	Points        []ForecastPoint `json:"points"`
	GeneratedAt   time.Time       `json:"generatedAt"`
}

type CycleTimeData struct {
	TaskID         uuid.UUID
	CycleTimeHours float64
	CompletedAt    time.Time
}

