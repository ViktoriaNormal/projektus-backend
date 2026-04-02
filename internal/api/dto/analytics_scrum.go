package dto

type VelocitySprintData struct {
	Sprint    string `json:"sprint"`
	SprintID  string `json:"sprint_id"`
	Planned   int    `json:"planned"`
	Completed int    `json:"completed"`
}

type VelocityMetrics struct {
	AverageVelocity    float64 `json:"average_velocity"`
	VelocityTrend      float64 `json:"velocity_trend"`
	CompletionRate     float64 `json:"completion_rate"`
	AverageSprintScope float64 `json:"average_sprint_scope"`
	SprintCount        int     `json:"sprint_count"`
}

type VelocityResponse struct {
	Data           []VelocitySprintData `json:"data"`
	Metrics        VelocityMetrics      `json:"metrics"`
	Interpretation string               `json:"interpretation"`
}

type BurndownDayData struct {
	Day       string  `json:"day"`
	Remaining float64 `json:"remaining"`
	Ideal     float64 `json:"ideal"`
}

type BurndownResponse struct {
	Data           []BurndownDayData `json:"data"`
	SprintName     string            `json:"sprint_name"`
	Interpretation string            `json:"interpretation"`
}
