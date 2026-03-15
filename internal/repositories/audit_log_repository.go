package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
)

type AuditLogRepository interface {
	List(ctx context.Context, userID *uuid.UUID, actionType *string, from, to *time.Time, limit, offset int32) ([]*domain.AuditLogEntry, int64, error)
	Insert(ctx context.Context, userID uuid.UUID, actionType, entityType string, entityID *uuid.UUID, metadata []byte) (*domain.AuditLogEntry, error)
}

type auditLogRepository struct {
	q *db.Queries
}

func NewAuditLogRepository(q *db.Queries) AuditLogRepository {
	return &auditLogRepository{q: q}
}

func (r *auditLogRepository) List(ctx context.Context, userID *uuid.UUID, actionType *string, from, to *time.Time, limit, offset int32) ([]*domain.AuditLogEntry, int64, error) {
	arg := db.ListAuditLogsParams{
		UserID:     any(userID),
		ActionType: any(actionType),
		FromAt:     any(from),
		ToAt:       any(to),
		Limit:      limit,
		Offset:     offset,
	}
	rows, err := r.q.ListAuditLogs(ctx, arg)
	if err != nil {
		return nil, 0, err
	}
	countArg := db.CountAuditLogsParams{
		UserID:     any(userID),
		ActionType: any(actionType),
		FromAt:     any(from),
		ToAt:       any(to),
	}
	total, err := r.q.CountAuditLogs(ctx, countArg)
	if err != nil {
		return nil, 0, err
	}
	out := make([]*domain.AuditLogEntry, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapDBAuditLogToDomain(row))
	}
	return out, total, nil
}

func (r *auditLogRepository) Insert(ctx context.Context, userID uuid.UUID, actionType, entityType string, entityID *uuid.UUID, metadata []byte) (*domain.AuditLogEntry, error) {
	arg := db.InsertAuditLogParams{
		UserID:     userID,
		ActionType: actionType,
		EntityType: sql.NullString{},
		EntityID:   uuid.NullUUID{},
		Metadata:   pqtype.NullRawMessage{},
	}
	if entityType != "" {
		arg.EntityType = sql.NullString{String: entityType, Valid: true}
	}
	if entityID != nil {
		arg.EntityID = uuid.NullUUID{UUID: *entityID, Valid: true}
	}
	if len(metadata) > 0 {
		arg.Metadata = pqtype.NullRawMessage{RawMessage: metadata, Valid: true}
	}
	row, err := r.q.InsertAuditLog(ctx, arg)
	if err != nil {
		return nil, err
	}
	return mapDBAuditLogToDomain(row), nil
}

func mapDBAuditLogToDomain(row db.AuditLog) *domain.AuditLogEntry {
	e := &domain.AuditLogEntry{
		ID:         row.ID,
		UserID:     row.UserID,
		ActionType: row.ActionType,
		CreatedAt:  row.CreatedAt,
	}
	if row.EntityType.Valid {
		e.EntityType = &row.EntityType.String
	}
	if row.EntityID.Valid {
		e.EntityID = &row.EntityID.UUID
	}
	if row.Metadata.Valid {
		e.Metadata = row.Metadata.RawMessage
	}
	return e
}
