package dto

type MonteCarloForecastRequest struct {
	WorkItemCount    int       `json:"work_item_count" binding:"required,min=1"`
	Simulations      int       `json:"simulations" binding:"required,min=100,max=10000"`
	ConfidenceLevels []float64 `json:"confidence_levels,omitempty"`
}

type ForecastPointDTO struct {
	Date        string  `json:"date"`
	Probability float64 `json:"probability"`
}

type MonteCarloForecastResultDTO struct {
	ProjectID     string            `json:"project_id"`
	WorkItemCount int              `json:"work_item_count"`
	Points        []ForecastPointDTO `json:"points"`
	GeneratedAt   string            `json:"generated_at"`
}
