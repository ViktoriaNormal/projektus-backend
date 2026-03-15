package domain

import "time"

// CumulativeFlowPoint — одна точка CFD: дата и накопительные счётчики по колонкам.
// StatusCounts: имя колонки -> накопительное количество (колонка 1, колонка 1+2, ...).
type CumulativeFlowPoint struct {
	Date         time.Time
	StatusCounts map[string]int
}

// ThroughputPoint — количество завершённых задач за период.
type ThroughputPoint struct {
	PeriodStart   time.Time
	ClassOfService *string
	TaskCount     int
	CumulativeCount int
}

// WipPoint — WIP на дату и опционально возраст.
type WipPoint struct {
	Date        time.Time
	WipCount    int
	AvgWipAge   float64
	MaxWipAge   float64
}

// CycleTimePoint — одна точка для scatterplot: задача с временем цикла.
type CycleTimePoint struct {
	TaskID         string
	TaskKey        string
	ClassOfService *string
	CompletedAt    time.Time
	CycleTimeDays  float64
}

// AverageCycleTimePoint — средний cycle time за период (для тренда).
type AverageCycleTimePoint struct {
	PeriodStart      time.Time
	ClassOfService   *string
	AvgCycleTimeDays float64
	TaskCount        int
}

// HistogramBucket — один интервал гистограммы.
type HistogramBucket struct {
	BucketStart float64
	BucketEnd   float64
	TaskCount   int
}

// HistogramData — гистограмма с процентилями (для cycle time или throughput).
type HistogramData struct {
	Buckets    []HistogramBucket
	TotalTasks int
	Average    float64
	Median     float64
	P85        float64
	P95        float64
}
