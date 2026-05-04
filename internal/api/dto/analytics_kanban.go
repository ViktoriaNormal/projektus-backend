package dto

// === Kanban Analytics Response Types ===

// --- Cumulative Flow ---
type CumulativeFlowResponse struct {
	Data           []map[string]interface{} `json:"data"`
	Interpretation string                   `json:"interpretation"`
}

// --- Cycle Time Scatter ---
type CycleTimeScatterPointDTO struct {
	Task string  `json:"task"`
	Time float64 `json:"time"`
}

type CycleTimeScatterResponse struct {
	Data           []CycleTimeScatterPointDTO `json:"data"`
	Interpretation string                     `json:"interpretation"`
}

// --- Throughput (факт по неделям + линия тренда) ---
type ThroughputPointDTO struct {
	Week   string  `json:"week"`
	Actual int     `json:"actual"`
	Trend  float64 `json:"trend"`
}

type ThroughputResponse struct {
	Data           []ThroughputPointDTO `json:"data"`
	Interpretation string               `json:"interpretation"`
}

// --- WIP Age ---
type WipAgePointDTO struct {
	TaskKey    string  `json:"task_key"`
	AgeDays    float64 `json:"age_days"`
	ColumnName string  `json:"column_name"`
}

type WipAgeResponse struct {
	Data           []WipAgePointDTO `json:"data"`
	Interpretation string           `json:"interpretation"`
}

// --- WIP History ---
type WipHistoryPointDTO struct {
	Date  string `json:"date"`
	Wip   int    `json:"wip"`
	Limit *int   `json:"limit,omitempty"`
}

type WipHistoryResponse struct {
	Data           []WipHistoryPointDTO `json:"data"`
	Interpretation string               `json:"interpretation"`
}

// --- Distribution (cycle time & throughput) ---
type DistributionBucketDTO struct {
	Range string `json:"range"`
	Count int    `json:"count"`
}

type DistributionResponse struct {
	Data           []DistributionBucketDTO `json:"data"`
	Interpretation string                  `json:"interpretation"`
}
