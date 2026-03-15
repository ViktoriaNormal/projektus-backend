package dto

// CumulativeFlowPointDTO — точка накопительной диаграммы потока.
type CumulativeFlowPointDTO struct {
	Date         string         `json:"date"`
	StatusCounts map[string]int `json:"statusCounts"`
}

// ThroughputPointDTO — точка графика скорости поставки.
type ThroughputPointDTO struct {
	Period          string  `json:"period"`
	ClassOfService  *string `json:"classOfService,omitempty"`
	TaskCount       int     `json:"taskCount"`
	CumulativeCount int     `json:"cumulativeCount,omitempty"`
}

// WipPointDTO — точка графика WIP.
type WipPointDTO struct {
	Date      string  `json:"date"`
	WipCount  int     `json:"wipCount"`
	AvgWipAge float64 `json:"avgWipAge,omitempty"`
	MaxWipAge float64 `json:"maxWipAge,omitempty"`
}

// CycleTimePointDTO — точка диаграммы рассеяния времени производства.
type CycleTimePointDTO struct {
	TaskID         string  `json:"taskId"`
	TaskKey        string  `json:"taskKey"`
	ClassOfService *string `json:"classOfService,omitempty"`
	CompletedAt    string  `json:"completedAt"`
	CycleTimeDays  float64 `json:"cycleTimeDays"`
}

// AverageCycleTimePointDTO — точка графика среднего времени производства.
type AverageCycleTimePointDTO struct {
	Period           string  `json:"period"`
	ClassOfService   *string `json:"classOfService,omitempty"`
	AvgCycleTimeDays float64 `json:"avgCycleTimeDays"`
	TaskCount        int     `json:"taskCount"`
}

// HistogramBucketDTO — интервал гистограммы.
type HistogramBucketDTO struct {
	BucketStart float64 `json:"bucketStart"`
	BucketEnd   float64 `json:"bucketEnd"`
	TaskCount   int     `json:"taskCount"`
}

// HistogramDataDTO — гистограмма с процентилями.
type HistogramDataDTO struct {
	Buckets    []HistogramBucketDTO `json:"buckets"`
	TotalTasks int                  `json:"totalTasks"`
	Average    float64              `json:"average"`
	Median     float64              `json:"median"`
	P85        float64              `json:"p85"`
	P95        float64              `json:"p95"`
}
