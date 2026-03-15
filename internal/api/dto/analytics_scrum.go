package dto

type VelocitySprintDTO struct {
	SprintID        string `json:"sprintId"`
	Name            string `json:"name"`
	CommittedPoints int    `json:"committedPoints"`
	CompletedPoints int    `json:"completedPoints"`
}

type ScrumVelocityReport struct {
	Sprints []VelocitySprintDTO `json:"sprints"`
}

type BurndownPointDTO struct {
	Date            string `json:"date"`
	RemainingPoints int    `json:"remainingPoints"`
	IdealPoints     int    `json:"idealPoints"`
}

type BurndownReportDTO struct {
	SprintID string             `json:"sprintId"`
	Points   []BurndownPointDTO `json:"points"`
}

