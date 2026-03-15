package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AuditLogEntry — запись журнала действий пользователя.
type AuditLogEntry struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	ActionType string
	EntityType *string
	EntityID   *uuid.UUID
	CreatedAt  time.Time
	Metadata   json.RawMessage
}
