package domain

import "github.com/google/uuid"

type ProductBacklogItem struct {
	ProjectID uuid.UUID `json:"project_id"`
	TaskID    uuid.UUID `json:"task_id"`
	Order     int       `json:"order"`
}
