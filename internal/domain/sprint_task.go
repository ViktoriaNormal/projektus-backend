package domain

import "github.com/google/uuid"

type SprintTask struct {
	SprintID uuid.UUID `json:"sprint_id"`
	TaskID   uuid.UUID `json:"task_id"`
	Order    int       `json:"order"`
}
