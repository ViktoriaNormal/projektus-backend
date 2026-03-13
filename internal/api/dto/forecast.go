package dto

type MonteCarloForecastRequest struct {
	WorkItemCount    int       `json:"workItemCount" binding:"required,min=1"`
	Simulations      int       `json:"simulations" binding:"required,min=100,max=10000"`
	ConfidenceLevels []float64 `json:"confidenceLevels,omitempty"`
}

type ForecastPointDTO struct {
	Date        string  `json:"date"`
	Probability float64 `json:"probability"`
}

type MonteCarloForecastResultDTO struct {
	ProjectID     string            `json:"projectId"`
	WorkItemCount int              `json:"workItemCount"`
	Points        []ForecastPointDTO `json:"points"`
	GeneratedAt   string            `json:"generatedAt"`
}

