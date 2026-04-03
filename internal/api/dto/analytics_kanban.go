package dto

// CumulativeFlowPointDTO — точка накопительной диаграммы потока.
type CumulativeFlowPointDTO struct {
	Date         string         `json:"date"`
	StatusCounts map[string]int `json:"status_counts"`
}

// ThroughputPointDTO — точка графика скорости поставки.
type ThroughputPointDTO struct {
	Period          string  `json:"period"`
	ClassOfService  *string `json:"class_of_service,omitempty"`
	TaskCount       int     `json:"task_count"`
	CumulativeCount int     `json:"cumulative_count,omitempty"`
}

// WipPointDTO — точка графика WIP.
type WipPointDTO struct {
	Date      string  `json:"date"`
	WipCount  int     `json:"wip_count"`
	AvgWipAge float64 `json:"avg_wip_age,omitempty"`
	MaxWipAge float64 `json:"max_wip_age,omitempty"`
}

// CycleTimePointDTO — точка диаграммы рассеяния времени производства.
type CycleTimePointDTO struct {
	TaskID         string  `json:"task_id"`
	TaskKey        string  `json:"task_key"`
	ClassOfService *string `json:"class_of_service,omitempty"`
	CompletedAt    string  `json:"completed_at"`
	CycleTimeDays  float64 `json:"cycle_time_days"`
}

// AverageCycleTimePointDTO — точка графика среднего времени производства.
type AverageCycleTimePointDTO struct {
	Period           string  `json:"period"`
	ClassOfService   *string `json:"class_of_service,omitempty"`
	AvgCycleTimeDays float64 `json:"avg_cycle_time_days"`
	TaskCount        int     `json:"task_count"`
}

// HistogramBucketDTO — интервал гистограммы.
type HistogramBucketDTO struct {
	BucketStart float64 `json:"bucket_start"`
	BucketEnd   float64 `json:"bucket_end"`
	TaskCount   int     `json:"task_count"`
}

// HistogramDataDTO — гистограмма с процентилями.
type HistogramDataDTO struct {
	Buckets    []HistogramBucketDTO `json:"buckets"`
	TotalTasks int                  `json:"total_tasks"`
	Average    float64              `json:"average"`
	Median     float64              `json:"median"`
	P85        float64              `json:"p85"`
	P95        float64              `json:"p95"`
}

// === Kanban Analytics Response Types ===

// --- Summary ---
type KanbanSummaryData struct {
	AverageVelocity     float64 `json:"average_velocity"`
	AverageVelocityUnit string  `json:"average_velocity_unit"`
	VelocityTrend       float64 `json:"velocity_trend"`
	CycleTime           float64 `json:"cycle_time"`
	CycleTimeTrend      float64 `json:"cycle_time_trend"`
	Throughput          float64 `json:"throughput"`
	ThroughputTrend     float64 `json:"throughput_trend"`
	Wip                 int     `json:"wip"`
	WipChange           int     `json:"wip_change"`
}

type KanbanSummaryResponse struct {
	Data           KanbanSummaryData `json:"data"`
	Interpretation string            `json:"interpretation"`
}

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

// --- Throughput (weekly) ---
type ThroughputWeekDTO struct {
	Week  string `json:"week"`
	Count int    `json:"count"`
}

type ThroughputResponse struct {
	Data           []ThroughputWeekDTO `json:"data"`
	Interpretation string              `json:"interpretation"`
}

// --- Average Cycle Time (weekly) ---
type AvgCycleTimeWeekDTO struct {
	Week string  `json:"week"`
	Avg  float64 `json:"avg"`
	P50  float64 `json:"p50"`
	P85  float64 `json:"p85"`
}

type AvgCycleTimeResponse struct {
	Data           []AvgCycleTimeWeekDTO `json:"data"`
	Interpretation string                `json:"interpretation"`
}

// --- Throughput Trend ---
type ThroughputTrendPointDTO struct {
	Week   string  `json:"week"`
	Actual int     `json:"actual"`
	Trend  float64 `json:"trend"`
}

type ThroughputTrendResponse struct {
	Data           []ThroughputTrendPointDTO `json:"data"`
	Interpretation string                    `json:"interpretation"`
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
