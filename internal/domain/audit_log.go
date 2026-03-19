package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AuditLogEntry — запись журнала действий пользователя.
type AuditLogEntry struct {
	ID         uuid.UUID       `json:"id"`
	UserID     uuid.UUID       `json:"user_id"`
	ActionType string          `json:"action_type"`
	EntityType *string         `json:"entity_type,omitempty"`
	EntityID   *uuid.UUID      `json:"entity_id,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
}
