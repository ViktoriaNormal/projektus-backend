package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
)

type SprintRepository interface {
	Create(ctx context.Context, s *domain.Sprint) (*domain.Sprint, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Sprint, error)
	GetProjectSprints(ctx context.Context, projectID uuid.UUID) ([]domain.Sprint, error)
	Update(ctx context.Context, s *domain.Sprint) (*domain.Sprint, error)
	Delete(ctx context.Context, id uuid.UUID) error
	GetActiveSprint(ctx context.Context, projectID uuid.UUID) (*domain.Sprint, error)
	UpdateStatuses(ctx context.Context) error
}

type sprintRepository struct {
	q *db.Queries
}

func NewSprintRepository(q *db.Queries) SprintRepository {
	return &sprintRepository{q: q}
}

func (r *sprintRepository) Create(ctx context.Context, s *domain.Sprint) (*domain.Sprint, error) {
	row, err := r.q.CreateSprint(ctx, db.CreateSprintParams{
		ProjectID: s.ProjectID,
		Name:      s.Name,
		Goal:      stringPtrToNullString(s.Goal),
		StartDate: s.StartDate,
		EndDate:   s.EndDate,
		Status:    string(s.Status),
	})
	if err != nil {
		return nil, err
	}
	return mapDBSprint(row), nil
}

func (r *sprintRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Sprint, error) {
	row, err := r.q.GetSprintByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return mapDBSprint(row), nil
}

func (r *sprintRepository) GetProjectSprints(ctx context.Context, projectID uuid.UUID) ([]domain.Sprint, error) {
	rows, err := r.q.GetProjectSprints(ctx, projectID)
	if err != nil {
		return nil, err
	}
	result := make([]domain.Sprint, 0, len(rows))
	for _, row := range rows {
		result = append(result, *mapDBSprint(row))
	}
	return result, nil
}

func (r *sprintRepository) Update(ctx context.Context, s *domain.Sprint) (*domain.Sprint, error) {
	row, err := r.q.UpdateSprint(ctx, db.UpdateSprintParams{
		ID:        s.ID,
		Name:      stringToNullString(s.Name),
		Goal:      stringPtrToNullString(s.Goal),
		StartDate: sql.NullTime{Time: s.StartDate, Valid: true},
		EndDate:   sql.NullTime{Time: s.EndDate, Valid: true},
		Status:    stringToNullString(string(s.Status)),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return mapDBSprint(row), nil
}

func (r *sprintRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteSprint(ctx, id)
}

func (r *sprintRepository) GetActiveSprint(ctx context.Context, projectID uuid.UUID) (*domain.Sprint, error) {
	row, err := r.q.GetActiveSprint(ctx, projectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return mapDBSprint(row), nil
}

func (r *sprintRepository) UpdateStatuses(ctx context.Context) error {
	return r.q.UpdateSprintStatuses(ctx)
}

func mapDBSprint(row db.Sprint) *domain.Sprint {
	var goal *string
	if row.Goal.Valid {
		g := row.Goal.String
		goal = &g
	}
	return &domain.Sprint{
		ID:        row.ID,
		ProjectID: row.ProjectID,
		Name:      row.Name,
		Goal:      goal,
		StartDate: row.StartDate,
		EndDate:   row.EndDate,
		Status:    domain.SprintStatus(row.Status),
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

