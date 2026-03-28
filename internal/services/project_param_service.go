package services

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
)

type ProjectParamService struct {
	repo repositories.ProjectParamRepository
}

func NewProjectParamService(repo repositories.ProjectParamRepository) *ProjectParamService {
	return &ProjectParamService{repo: repo}
}

func (s *ProjectParamService) ListParams(ctx context.Context, projectID uuid.UUID) ([]domain.ProjectParam, error) {
	return s.repo.List(ctx, projectID)
}

func (s *ProjectParamService) CreateParam(ctx context.Context, projectID uuid.UUID, name, fieldType string, isRequired bool, options []string, value *string) (*domain.ProjectParam, error) {
	existing, _ := s.repo.List(ctx, projectID)
	order := int32(len(existing) + 1)

	var val sql.NullString
	if value != nil {
		val = sql.NullString{String: *value, Valid: true}
	}

	var opts pqtype.NullRawMessage
	if len(options) > 0 {
		opts = repositories.OptionsToJSON(options)
	}

	return s.repo.Create(ctx, db.CreateProjectParamParams{
		ProjectID:  uuid.NullUUID{UUID: projectID, Valid: true},
		Name:       name,
		FieldType:  fieldType,
		IsSystem:   false,
		IsRequired: isRequired,
		SortOrder:  order,
		Options:    opts,
		Value:      val,
	})
}

func (s *ProjectParamService) UpdateParam(ctx context.Context, projectID uuid.UUID, paramID uuid.UUID, name *string, isRequired *bool, options []string, value *string) (*domain.ProjectParam, error) {
	existing, err := s.repo.GetByID(ctx, paramID)
	if err != nil {
		return nil, err
	}
	if existing.ProjectID != projectID.String() {
		return nil, domain.ErrNotFound
	}

	params := db.UpdateProjectParamParams{ID: paramID}
	if name != nil {
		params.Name = sql.NullString{String: *name, Valid: true}
	}
	if isRequired != nil {
		params.IsRequired = sql.NullBool{Bool: *isRequired, Valid: true}
	}
	if options != nil {
		params.Options = repositories.OptionsToJSON(options)
	}
	if value != nil {
		params.Value = sql.NullString{String: *value, Valid: true}
	}

	return s.repo.Update(ctx, params)
}

func (s *ProjectParamService) DeleteParam(ctx context.Context, projectID uuid.UUID, paramID uuid.UUID) error {
	existing, err := s.repo.GetByID(ctx, paramID)
	if err != nil {
		return err
	}
	if existing.ProjectID != projectID.String() {
		return domain.ErrNotFound
	}
	if existing.IsSystem && existing.IsRequired {
		return domain.ErrSystemParam
	}
	return s.repo.Delete(ctx, paramID)
}

func (s *ProjectParamService) ReorderParams(ctx context.Context, projectID uuid.UUID, orders map[uuid.UUID]int32) error {
	for id, order := range orders {
		if err := s.repo.UpdateOrder(ctx, id, order); err != nil {
			return err
		}
	}
	return nil
}
