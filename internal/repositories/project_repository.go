package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
)

type ProjectRepository interface {
	Create(ctx context.Context, p *domain.Project) (*domain.Project, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Project, error)
	GetByKey(ctx context.Context, key string) (*domain.Project, error)
	ListUserProjects(ctx context.Context, userID uuid.UUID, query *string, status *string, projectType *string) ([]domain.Project, error)
	Update(ctx context.Context, p *domain.Project) (*domain.Project, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type projectRepository struct {
	q *db.Queries
}

func NewProjectRepository(q *db.Queries) ProjectRepository {
	return &projectRepository{q: q}
}

func (r *projectRepository) Create(ctx context.Context, p *domain.Project) (*domain.Project, error) {
	row, err := r.q.CreateProject(ctx, db.CreateProjectParams{
		Key:         p.Key,
		Name:        p.Name,
		Description: stringPtrToNullString(p.Description),
		ProjectType: string(p.Type),
		OwnerID:     p.OwnerID,
		Status:      string(p.Status),
	})
	if err != nil {
		return nil, err
	}
	return mapDBProject(row), nil
}

func (r *projectRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
	row, err := r.q.GetProjectByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return mapDBProject(row), nil
}

func (r *projectRepository) GetByKey(ctx context.Context, key string) (*domain.Project, error) {
	row, err := r.q.GetProjectByKey(ctx, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return mapDBProject(row), nil
}

func (r *projectRepository) ListUserProjects(ctx context.Context, userID uuid.UUID, query *string, status *string, projectType *string) ([]domain.Project, error) {
	var statusArg, typeArg, queryArg sql.NullString
	if status != nil && *status != "" {
		statusArg = sql.NullString{String: *status, Valid: true}
	}
	if projectType != nil && *projectType != "" {
		typeArg = sql.NullString{String: *projectType, Valid: true}
	}
	if query != nil && *query != "" {
		queryArg = sql.NullString{String: *query, Valid: true}
	}

	rows, err := r.q.ListUserProjects(ctx, db.ListUserProjectsParams{
		UserID:       userID,
		StatusFilter: statusArg,
		TypeFilter:   typeArg,
		SearchQuery:  queryArg,
	})
	if err != nil {
		return nil, err
	}

	projects := make([]domain.Project, 0, len(rows))
	for _, row := range rows {
		p := &domain.Project{
			ID:          row.ID,
			Key:         row.Key,
			Name:        row.Name,
			Description: nullStringToStringPtr(row.Description),
			Type:        domain.ProjectType(row.ProjectType),
			OwnerID:     row.OwnerID,
			Status:      domain.ProjectStatus(row.Status),
			CreatedAt:   row.CreatedAt,
		}
		var avatarURL *string
		if row.OwnerAvatarUrl.Valid {
			avatarURL = &row.OwnerAvatarUrl.String
		}
		p.Owner = &domain.ProjectOwner{
			ID:        row.OwnerID.String(),
			FullName:  row.OwnerFullName,
			AvatarURL: avatarURL,
			Email:     row.OwnerEmail,
		}
		projects = append(projects, *p)
	}
	return projects, nil
}

func (r *projectRepository) Update(ctx context.Context, p *domain.Project) (*domain.Project, error) {
	row, err := r.q.UpdateProject(ctx, db.UpdateProjectParams{
		Name:        stringToNullString(p.Name),
		Description: stringPtrToNullString(p.Description),
		Status:      stringToNullString(string(p.Status)),
		ID:          p.ID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return mapDBProject(row), nil
}

func (r *projectRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteProject(ctx, id)
}

func mapDBProject(row db.Project) *domain.Project {
	return &domain.Project{
		ID:          row.ID,
		Key:         row.Key,
		Name:        row.Name,
		Description: nullStringToStringPtr(row.Description),
		Type:        domain.ProjectType(row.ProjectType),
		OwnerID:     row.OwnerID,
		Status:      domain.ProjectStatus(row.Status),
		CreatedAt:   row.CreatedAt,
	}
}

func nullStringToStringPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	s := ns.String
	return &s
}

func stringPtrToNullString(s *string) sql.NullString {
	if s == nil || *s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: *s, Valid: true}
}
