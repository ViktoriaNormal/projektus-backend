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
