package repositories

import (
	"context"
	"database/sql"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
)

type TemplateRepository interface {
	List(ctx context.Context) ([]domain.ProjectTemplate, error)
	Create(ctx context.Context, t *domain.ProjectTemplate) (*domain.ProjectTemplate, error)
}

type templateRepository struct {
	q *db.Queries
}

func NewTemplateRepository(q *db.Queries) TemplateRepository {
	return &templateRepository{q: q}
}

func (r *templateRepository) List(ctx context.Context) ([]domain.ProjectTemplate, error) {
	rows, err := r.q.ListProjectTemplates(ctx)
	if err != nil {
		return nil, err
	}
	templates := make([]domain.ProjectTemplate, 0, len(rows))
	for _, row := range rows {
		var descPtr *string
		if row.Description.Valid {
			d := row.Description.String
			descPtr = &d
		}
		templates = append(templates, domain.ProjectTemplate{
			ID:          row.ID,
			Name:        row.Name,
			Description: descPtr,
			Type:        domain.ProjectType(row.ProjectType),
			CreatedAt:   row.CreatedAt,
		})
	}
	return templates, nil
}

func (r *templateRepository) Create(ctx context.Context, t *domain.ProjectTemplate) (*domain.ProjectTemplate, error) {
	var desc sql.NullString
	if t.Description != nil && *t.Description != "" {
		desc = sql.NullString{String: *t.Description, Valid: true}
	}
	row, err := r.q.CreateProjectTemplate(ctx, db.CreateProjectTemplateParams{
		Name:        t.Name,
		Description: desc,
		ProjectType: string(t.Type),
	})
	if err != nil {
		return nil, err
	}
	var descPtr *string
	if row.Description.Valid {
		d := row.Description.String
		descPtr = &d
	}
	return &domain.ProjectTemplate{
		ID:          row.ID,
		Name:        row.Name,
		Description: descPtr,
		Type:        domain.ProjectType(row.ProjectType),
		CreatedAt:   row.CreatedAt,
	}, nil
}

