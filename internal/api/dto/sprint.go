package dto

import "github.com/google/uuid"

type CreateSprintRequest struct {
	Name          string  `json:"name" binding:"required"`
	Goal          *string `json:"goal"`
	StartDate     string  `json:"startDate" binding:"required"`          // YYYY-MM-DD
	DurationWeeks *int    `json:"durationWeeks" binding:"omitempty,oneof=1 2 3 4"`
	DurationDays  *int    `json:"durationDays" binding:"omitempty,min=1,max=28"`
}

type UpdateSprintRequest struct {
	Name          *string `json:"name"`
	Goal          *string `json:"goal"`
	StartDate     *string `json:"startDate"`
	DurationWeeks *int    `json:"durationWeeks" binding:"omitempty,oneof=1 2 3 4"`
	DurationDays  *int    `json:"durationDays" binding:"omitempty,min=1,max=28"`
}

type SprintResponse struct {
	ID        uuid.UUID `json:"id"`
	ProjectID uuid.UUID `json:"projectId"`
	Name      string    `json:"name"`
	Goal      *string   `json:"goal"`
	StartDate string    `json:"startDate"`
	EndDate   string    `json:"endDate"`
	Status    string    `json:"status"`
	CreatedAt string    `json:"createdAt"`
	UpdatedAt string    `json:"updatedAt"`
}

