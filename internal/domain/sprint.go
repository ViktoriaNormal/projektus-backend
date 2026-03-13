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
	ID        uuid.UUID
	ProjectID uuid.UUID
	Name      string
	Goal      *string
	StartDate time.Time
	EndDate   time.Time
	Status    SprintStatus
	CreatedAt time.Time
	UpdatedAt time.Time
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

