package dto

type MonteCarloPercentileDTO struct {
	Percentile int    `json:"percentile"`
	Date       string `json:"date"` // YYYY-MM-DD
}

type MonteCarloChartPointDTO struct {
	Date        string `json:"date"`        // DD.MM
	Probability int    `json:"probability"` // 0–100 cumulative
}

type MonteCarloResponse struct {
	Percentiles           []MonteCarloPercentileDTO `json:"percentiles"`
	Chart                 []MonteCarloChartPointDTO  `json:"chart"`
	TargetDateProbability *int                       `json:"target_date_probability,omitempty"`
}
