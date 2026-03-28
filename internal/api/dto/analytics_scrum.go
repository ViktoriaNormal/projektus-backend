package dto

type VelocitySprintDTO struct {
	SprintID        string `json:"sprint_id"`
	Name            string `json:"name"`
	CommittedPoints int    `json:"committed_points"`
	CompletedPoints int    `json:"completed_points"`
}

type ScrumVelocityReport struct {
	Sprints []VelocitySprintDTO `json:"sprints"`
}

type BurndownPointDTO struct {
	Date            string `json:"date"`
	RemainingPoints int    `json:"remaining_points"`
	IdealPoints     int    `json:"ideal_points"`
}

type BurndownReportDTO struct {
	SprintID string             `json:"sprint_id"`
	Points   []BurndownPointDTO `json:"points"`
}
