package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
)

type ProjectMemberRepository interface {
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]domain.ProjectMember, error)
	AddMember(ctx context.Context, projectID, userID uuid.UUID) (*domain.ProjectMember, error)
	RemoveMember(ctx context.Context, memberID uuid.UUID) error
	GetByID(ctx context.Context, memberID uuid.UUID) (*domain.ProjectMember, error)
	ReplaceMemberRoles(ctx context.Context, memberID uuid.UUID, roleIDs []uuid.UUID) error
}

type projectMemberRepository struct {
	q *db.Queries
}

func NewProjectMemberRepository(q *db.Queries) ProjectMemberRepository {
	return &projectMemberRepository{q: q}
}

func (r *projectMemberRepository) ListByProject(ctx context.Context, projectID uuid.UUID) ([]domain.ProjectMember, error) {
	rows, err := r.q.ListProjectMembers(ctx, projectID)
	if err != nil {
		return nil, err
	}
	members := make([]domain.ProjectMember, 0, len(rows))
	for _, row := range rows {
		// загрузим роли участника
		roleNames, err := r.q.ListMemberRoles(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		members = append(members, domain.ProjectMember{
			ID:        row.ID,
			ProjectID: row.ProjectID,
			UserID:    row.UserID,
			Roles:     roleNames,
		})
	}
	return members, nil
}

func (r *projectMemberRepository) AddMember(ctx context.Context, projectID, userID uuid.UUID) (*domain.ProjectMember, error) {
	row, err := r.q.AddProjectMember(ctx, db.AddProjectMemberParams{
		ProjectID: projectID,
		UserID:    userID,
	})
	if err != nil {
		return nil, err
	}
	roleNames, err := r.q.ListMemberRoles(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	return &domain.ProjectMember{
		ID:        row.ID,
		ProjectID: row.ProjectID,
		UserID:    row.UserID,
		Roles:     roleNames,
	}, nil
}

func (r *projectMemberRepository) RemoveMember(ctx context.Context, memberID uuid.UUID) error {
	return r.q.RemoveProjectMember(ctx, memberID)
}

func (r *projectMemberRepository) GetByID(ctx context.Context, memberID uuid.UUID) (*domain.ProjectMember, error) {
	row, err := r.q.GetProjectMember(ctx, memberID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	roleNames, err := r.q.ListMemberRoles(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	return &domain.ProjectMember{
		ID:        row.ID,
		ProjectID: row.ProjectID,
		UserID:    row.UserID,
		Roles:     roleNames,
	}, nil
}

func (r *projectMemberRepository) ReplaceMemberRoles(ctx context.Context, memberID uuid.UUID, roleIDs []uuid.UUID) error {
	if err := r.q.DeleteMemberRoles(ctx, memberID); err != nil {
		return err
	}
	for _, roleID := range roleIDs {
		if err := r.q.AddRoleToMember(ctx, db.AddRoleToMemberParams{
			ProjectMemberID: memberID,
			RoleID:          roleID,
		}); err != nil {
			return err
		}
	}
	return nil
}
