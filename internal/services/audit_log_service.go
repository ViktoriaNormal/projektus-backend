package services

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
)

// ListAuditLogsFilter — фильтры для списка записей журнала.
type ListAuditLogsFilter struct {
	UserID     *uuid.UUID
	ActionType *string
	From       *time.Time
	To         *time.Time
	Limit      int32
	Offset     int32
}

// AuditLogService — чтение журнала и запись событий.
type AuditLogService struct {
	repo repositories.AuditLogRepository
}

func NewAuditLogService(repo repositories.AuditLogRepository) *AuditLogService {
	return &AuditLogService{repo: repo}
}

// List возвращает записи журнала с пагинацией и общее количество.
func (s *AuditLogService) List(ctx context.Context, filter ListAuditLogsFilter) ([]*domain.AuditLogEntry, int64, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	return s.repo.List(ctx, filter.UserID, filter.ActionType, filter.From, filter.To, limit, filter.Offset)
}

// Log записывает событие в журнал (entityType и entityID опциональны, metadata — произвольный JSON).
func (s *AuditLogService) Log(ctx context.Context, userID uuid.UUID, actionType, entityType string, entityID *uuid.UUID, metadata interface{}) error {
	var raw []byte
	if metadata != nil {
		var err error
		raw, err = json.Marshal(metadata)
		if err != nil {
			return err
		}
	}
	_, err := s.repo.Insert(ctx, userID, actionType, entityType, entityID, raw)
	return err
}
