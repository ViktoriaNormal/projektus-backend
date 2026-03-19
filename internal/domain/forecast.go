package domain

import (
	"time"

	"github.com/google/uuid"
)

type ForecastPoint struct {
	Date        time.Time `json:"date"`
	Probability float64   `json:"probability"`
}

type ForecastRequest struct {
	ProjectID        uuid.UUID `json:"project_id"`
	WorkItemCount    int       `json:"work_item_count"`
	Simulations      int       `json:"simulations"`
	ConfidenceLevels []float64 `json:"confidence_levels"`
}

type ForecastResult struct {
	ProjectID     uuid.UUID       `json:"project_id"`
	WorkItemCount int             `json:"work_item_count"`
	Points        []ForecastPoint `json:"points"`
	GeneratedAt   time.Time       `json:"generated_at"`
}

type CycleTimeData struct {
	TaskID         uuid.UUID `json:"task_id"`
	CycleTimeHours float64  `json:"cycle_time_hours"`
	CompletedAt    time.Time `json:"completed_at"`
}
