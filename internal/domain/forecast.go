package domain

import "time"

// MonteCarloReport holds the result of a Monte Carlo throughput-based forecast.
type MonteCarloReport struct {
	Percentiles           []MonteCarloPercentile
	ChartPoints           []MonteCarloChartPoint
	TargetDateProbability *int
}

type MonteCarloPercentile struct {
	Percentile int
	Date       time.Time
}

type MonteCarloChartPoint struct {
	Date        time.Time
	Probability int // 0–100 cumulative
}
