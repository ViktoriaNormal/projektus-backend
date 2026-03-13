package repositories

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
)

type SwimlaneConfigRepository interface {
	UpdateConfig(ctx context.Context, swimlaneID uuid.UUID, cfg domain.SwimlaneConfig) (*domain.SwimlaneConfig, error)
	GetConfig(ctx context.Context, swimlaneID uuid.UUID) (*domain.SwimlaneConfig, error)
	ListConfigsForBoard(ctx context.Context, boardID uuid.UUID) ([]domain.SwimlaneConfig, error)
}

type swimlaneConfigRepository struct {
	q *db.Queries
}

func NewSwimlaneConfigRepository(q *db.Queries) SwimlaneConfigRepository {
	return &swimlaneConfigRepository{q: q}
}

func (r *swimlaneConfigRepository) UpdateConfig(ctx context.Context, swimlaneID uuid.UUID, cfg domain.SwimlaneConfig) (*domain.SwimlaneConfig, error) {
	var src sql.NullString
	if cfg.SourceType != "" {
		src = sql.NullString{String: string(cfg.SourceType), Valid: true}
	}

	custom := uuid.NullUUID{}
	if cfg.CustomFieldID != nil {
		custom = uuid.NullUUID{UUID: *cfg.CustomFieldID, Valid: true}
	}

	var mappings pqtype.NullRawMessage
	if cfg.ValueMappings != nil {
		raw, err := json.Marshal(cfg.ValueMappings)
		if err != nil {
			return nil, err
		}
		mappings = pqtype.NullRawMessage{RawMessage: raw, Valid: true}
	}

	row, err := r.q.UpdateSwimlaneConfig(ctx, db.UpdateSwimlaneConfigParams{
		ID:            swimlaneID,
		SourceType:    src,
		CustomFieldID: custom,
		ValueMappings: mappings,
	})
	if err != nil {
		return nil, err
	}
	return mapDBSwimlaneToConfig(row)
}

func (r *swimlaneConfigRepository) GetConfig(ctx context.Context, swimlaneID uuid.UUID) (*domain.SwimlaneConfig, error) {
	row, err := r.q.GetSwimlaneConfig(ctx, swimlaneID)
	if err != nil {
		return nil, err
	}
	return mapDBSwimlaneToConfig(row)
}

func (r *swimlaneConfigRepository) ListConfigsForBoard(ctx context.Context, boardID uuid.UUID) ([]domain.SwimlaneConfig, error) {
	rows, err := r.q.GetSwimlanesWithConfig(ctx, boardID)
	if err != nil {
		return nil, err
	}
	result := make([]domain.SwimlaneConfig, 0, len(rows))
	for _, row := range rows {
		cfg, err := mapDBSwimlaneToConfig(row)
		if err != nil {
			return nil, err
		}
		result = append(result, *cfg)
	}
	return result, nil
}

func mapDBSwimlaneToConfig(s db.Swimlane) (*domain.SwimlaneConfig, error) {
	cfg := &domain.SwimlaneConfig{
		BoardID: s.BoardID,
	}
	if s.SourceType.Valid {
		cfg.SourceType = domain.SwimlaneSourceType(s.SourceType.String)
	}
	if s.CustomFieldID.Valid {
		id := s.CustomFieldID.UUID
		cfg.CustomFieldID = &id
	}
	if s.ValueMappings.Valid {
		m := make(map[string]string)
		if err := json.Unmarshal(s.ValueMappings.RawMessage, &m); err != nil {
			return nil, err
		}
		cfg.ValueMappings = m
	}
	return cfg, nil
}

