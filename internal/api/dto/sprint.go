package dto

import "github.com/google/uuid"

type CreateSprintRequest struct {
	Name          string  `json:"name" binding:"required"`
	Goal          *string `json:"goal"`
	StartDate     string  `json:"start_date" binding:"required"`          // YYYY-MM-DD
	DurationWeeks *int    `json:"duration_weeks" binding:"omitempty,oneof=1 2 3 4"`
	DurationDays  *int    `json:"duration_days" binding:"omitempty,min=1,max=28"`
}

type UpdateSprintRequest struct {
	Name          *string               `json:"name"`
	Goal          NullableField[string] `json:"goal"`
	StartDate     *string               `json:"start_date"`
	DurationWeeks *int                  `json:"duration_weeks" binding:"omitempty,oneof=1 2 3 4"`
	DurationDays  *int                  `json:"duration_days" binding:"omitempty,min=1,max=28"`
}

type SprintResponse struct {
	ID        uuid.UUID `json:"id"`
	ProjectID uuid.UUID `json:"project_id"`
	Name      string    `json:"name"`
	Goal      *string   `json:"goal"`
	StartDate string    `json:"start_date"`
	EndDate   string    `json:"end_date"`
	Status    string    `json:"status"`
	CreatedAt string    `json:"created_at"`
	UpdatedAt string    `json:"updated_at"`
}
