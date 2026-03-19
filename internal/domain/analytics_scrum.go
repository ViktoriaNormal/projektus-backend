package domain

import "time"

type VelocityPoint struct {
	SprintID        string    `json:"sprint_id"`
	SprintName      string    `json:"sprint_name"`
	StartDate       time.Time `json:"start_date"`
	EndDate         time.Time `json:"end_date"`
	CommittedPoints int       `json:"committed_points"`
	CompletedPoints int       `json:"completed_points"`
}

type BurndownPoint struct {
	Date            time.Time `json:"date"`
	RemainingPoints int       `json:"remaining_points"`
	IdealPoints     int       `json:"ideal_points"`
}

type BurndownData struct {
	SprintID    string          `json:"sprint_id"`
	SprintName  string          `json:"sprint_name"`
	StartDate   time.Time       `json:"start_date"`
	EndDate     time.Time       `json:"end_date"`
	TotalPoints int             `json:"total_points"`
	Points      []BurndownPoint `json:"points"`
}
