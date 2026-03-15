package domain

import "time"

type VelocityPoint struct {
	SprintID        string
	SprintName      string
	StartDate       time.Time
	EndDate         time.Time
	CommittedPoints int
	CompletedPoints int
}

type BurndownPoint struct {
	Date            time.Time
	RemainingPoints int
	IdealPoints     int
}

type BurndownData struct {
	SprintID    string
	SprintName  string
	StartDate   time.Time
	EndDate     time.Time
	TotalPoints int
	Points      []BurndownPoint
}

