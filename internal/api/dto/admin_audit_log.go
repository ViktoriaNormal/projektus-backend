package dto

import "encoding/json"

// AuditLogEntryResponse — запись журнала (GET /admin/logs).
type AuditLogEntryResponse struct {
	ID         string          `json:"id"`
	UserID     string          `json:"userId"`
	ActionType string          `json:"actionType"`
	EntityType *string         `json:"entityType,omitempty"`
	EntityID   *string         `json:"entityId,omitempty"`
	CreatedAt  string          `json:"createdAt"`
	Metadata   json.RawMessage  `json:"metadata,omitempty"`
}
