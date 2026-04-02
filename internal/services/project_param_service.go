package services

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
)

type ProjectParamService struct {
	repo  repositories.ProjectParamRepository
	users repositories.UserRepository
}

func NewProjectParamService(repo repositories.ProjectParamRepository, users repositories.UserRepository) *ProjectParamService {
	return &ProjectParamService{repo: repo, users: users}
}

func (s *ProjectParamService) ListParams(ctx context.Context, projectID uuid.UUID) ([]domain.ProjectParam, error) {
	return s.repo.List(ctx, projectID)
}

func (s *ProjectParamService) CreateParam(ctx context.Context, projectID uuid.UUID, name, fieldType string, isRequired bool, options []string, value *string) (*domain.ProjectParam, error) {
	// Validate value against fieldType if provided.
	if value != nil && *value != "" {
		if err := s.validateParamValue(ctx, name, fieldType, *value, options); err != nil {
			return nil, err
		}
	}

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
		IsRequired: isRequired,
		Options:    opts,
		Value:      val,
	})
}

func (s *ProjectParamService) UpdateParam(ctx context.Context, projectID uuid.UUID, paramID uuid.UUID, name *string, isRequired *bool, options []string, value *string, clearValue bool) (*domain.ProjectParam, error) {
	existing, err := s.repo.GetByID(ctx, paramID)
	if err != nil {
		return nil, err
	}
	if existing.ProjectID != projectID.String() {
		return nil, domain.ErrNotFound
	}

	// Determine the effective state after this update.
	effectiveName := existing.Name
	if name != nil {
		effectiveName = *name
	}
	effectiveRequired := existing.IsRequired
	if isRequired != nil {
		effectiveRequired = *isRequired
	}
	effectiveOptions := existing.Options
	if options != nil {
		effectiveOptions = options
	}

	// Determine the effective value after this update.
	var effectiveValue *string
	if clearValue {
		effectiveValue = nil
	} else if value != nil {
		effectiveValue = value
	} else {
		effectiveValue = existing.Value
	}

	// Check: required param must have a non-empty value.
	if effectiveRequired && (effectiveValue == nil || *effectiveValue == "") {
		return nil, domain.NewParamValidationError(
			"Обязательный параметр «%s» не может быть пустым", effectiveName)
	}

	// Validate value against fieldType if a new value is being set.
	if value != nil && *value != "" {
		if err := s.validateParamValue(ctx, effectiveName, existing.FieldType, *value, effectiveOptions); err != nil {
			return nil, err
		}
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
	// value field no longer uses COALESCE in SQL, so we must always pass a value.
	if clearValue {
		params.Value = sql.NullString{Valid: false}
	} else if value != nil {
		params.Value = sql.NullString{String: *value, Valid: true}
	} else {
		// Not provided — keep existing value.
		if existing.Value != nil {
			params.Value = sql.NullString{String: *existing.Value, Valid: true}
		}
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

// validateParamValue checks that value conforms to fieldType rules.
func (s *ProjectParamService) validateParamValue(ctx context.Context, paramName, fieldType, value string, options []string) error {
	switch fieldType {
	case "number":
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			return domain.NewParamValidationError(
				"Значение параметра «%s» должно быть числом", paramName)
		}

	case "datetime":
		t, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return domain.NewParamValidationError(
				"Значение параметра «%s»: некорректная дата", paramName)
		}
		if y := t.Year(); y < 2000 || y > 2100 {
			return domain.NewParamValidationError(
				"Значение параметра «%s»: год должен быть в диапазоне 2000–2100", paramName)
		}

	case "checkbox":
		if value != "true" && value != "false" {
			return domain.NewParamValidationError(
				"Значение параметра «%s» должно быть \"true\" или \"false\"", paramName)
		}

	case "select":
		if !stringInSlice(value, options) {
			return domain.NewParamValidationError(
				"Значение параметра «%s» должно быть одним из допустимых вариантов", paramName)
		}

	case "multiselect":
		parts := strings.Split(value, ",")
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed == "" {
				continue
			}
			if !stringInSlice(trimmed, options) {
				return domain.NewParamValidationError(
					"Значение «%s» в параметре «%s» не входит в список допустимых вариантов", trimmed, paramName)
			}
		}

	case "user":
		if _, err := uuid.Parse(value); err != nil {
			return domain.NewParamValidationError(
				"Значение параметра «%s» должно быть валидным UUID пользователя", paramName)
		}
		if _, err := s.users.GetUserByID(ctx, value); err != nil {
			return domain.NewParamValidationError(
				"Пользователь, указанный в параметре «%s», не найден", paramName)
		}

	case "user_list":
		ids := strings.Split(value, ",")
		for _, raw := range ids {
			id := strings.TrimSpace(raw)
			if id == "" {
				continue
			}
			if _, err := uuid.Parse(id); err != nil {
				return domain.NewParamValidationError(
					"Значение «%s» в параметре «%s» не является валидным UUID", id, paramName)
			}
			if _, err := s.users.GetUserByID(ctx, id); err != nil {
				return domain.NewParamValidationError(
					"Пользователь «%s» в параметре «%s» не найден", id, paramName)
			}
		}

	case "text":
		// Any string is valid.
	}

	return nil
}

func stringInSlice(s string, slice []string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
