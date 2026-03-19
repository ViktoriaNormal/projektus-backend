package domain

import "time"

// CumulativeFlowPoint — одна точка CFD: дата и накопительные счётчики по колонкам.
// StatusCounts: имя колонки -> накопительное количество (колонка 1, колонка 1+2, ...).
type CumulativeFlowPoint struct {
	Date         time.Time      `json:"date"`
	StatusCounts map[string]int `json:"status_counts"`
}

// ThroughputPoint — количество завершённых задач за период.
type ThroughputPoint struct {
	PeriodStart     time.Time `json:"period_start"`
	ClassOfService  *string   `json:"class_of_service,omitempty"`
	TaskCount       int       `json:"task_count"`
	CumulativeCount int       `json:"cumulative_count"`
}

// WipPoint — WIP на дату и опционально возраст.
type WipPoint struct {
	Date      time.Time `json:"date"`
	WipCount  int       `json:"wip_count"`
	AvgWipAge float64   `json:"avg_wip_age"`
	MaxWipAge float64   `json:"max_wip_age"`
}

// CycleTimePoint — одна точка для scatterplot: задача с временем цикла.
type CycleTimePoint struct {
	TaskID         string    `json:"task_id"`
	TaskKey        string    `json:"task_key"`
	ClassOfService *string   `json:"class_of_service,omitempty"`
	CompletedAt    time.Time `json:"completed_at"`
	CycleTimeDays  float64   `json:"cycle_time_days"`
}

// AverageCycleTimePoint — средний cycle time за период (для тренда).
type AverageCycleTimePoint struct {
	PeriodStart      time.Time `json:"period_start"`
	ClassOfService   *string   `json:"class_of_service,omitempty"`
	AvgCycleTimeDays float64   `json:"avg_cycle_time_days"`
	TaskCount        int       `json:"task_count"`
}

// HistogramBucket — один интервал гистограммы.
type HistogramBucket struct {
	BucketStart float64 `json:"bucket_start"`
	BucketEnd   float64 `json:"bucket_end"`
	TaskCount   int     `json:"task_count"`
}

// HistogramData — гистограмма с процентилями (для cycle time или throughput).
type HistogramData struct {
	Buckets    []HistogramBucket `json:"buckets"`
	TotalTasks int               `json:"total_tasks"`
	Average    float64           `json:"average"`
	Median     float64           `json:"median"`
	P85        float64           `json:"p85"`
	P95        float64           `json:"p95"`
}
