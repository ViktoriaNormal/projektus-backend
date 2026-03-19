package domain

import (
	"time"

	"github.com/google/uuid"
)

type SprintStatus string

const (
	SprintStatusPlanned   SprintStatus = "planned"
	SprintStatusActive    SprintStatus = "active"
	SprintStatusCompleted SprintStatus = "completed"
)

type Sprint struct {
	ID        uuid.UUID    `json:"id"`
	ProjectID uuid.UUID    `json:"project_id"`
	Name      string       `json:"name"`
	Goal      *string      `json:"goal,omitempty"`
	StartDate time.Time    `json:"start_date"`
	EndDate   time.Time    `json:"end_date"`
	Status    SprintStatus `json:"status"`
	CreatedAt time.Time    `json:"-"`
	UpdatedAt time.Time    `json:"-"`
}

func (s *Sprint) CalculateStatus(now time.Time) SprintStatus {
	if now.Before(s.StartDate) {
		return SprintStatusPlanned
	}
	if now.After(s.EndDate) {
		return SprintStatusCompleted
	}
	return SprintStatusActive
}
