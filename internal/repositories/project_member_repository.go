package repositories

import (
	"context"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
	"projektus-backend/pkg/errctx"
)

type ProjectMemberWithProject struct {
	MemberID    uuid.UUID
	ProjectID   uuid.UUID
	ProjectName string
	Roles       []domain.ProjectMemberRoleRef
	RoleIDs     []uuid.UUID
}

type ProjectMemberRepository interface {
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]domain.ProjectMember, error)
	GetByProjectAndUser(ctx context.Context, projectID, userID uuid.UUID) (*domain.ProjectMember, error)
	AddMember(ctx context.Context, projectID, userID uuid.UUID) (*domain.ProjectMember, error)
	RemoveMember(ctx context.Context, memberID uuid.UUID) error
	GetByID(ctx context.Context, memberID uuid.UUID) (*domain.ProjectMember, error)
	ReplaceMemberRoles(ctx context.Context, memberID uuid.UUID, roleIDs []uuid.UUID) error
	AddRoleToMember(ctx context.Context, memberID, roleID uuid.UUID) error
	ListByUser(ctx context.Context, userID uuid.UUID) ([]ProjectMemberWithProject, error)
}

type projectMemberRepository struct {
	q *db.Queries
}

func NewProjectMemberRepository(q *db.Queries) ProjectMemberRepository {
	return &projectMemberRepository{q: q}
}

// toMemberRoleRefs конвертирует sqlc-строки списка ролей участника в domain-тип.
func toMemberRoleRefs(rows []db.ListMemberRolesRow) []domain.ProjectMemberRoleRef {
	if len(rows) == 0 {
		return nil
	}
	out := make([]domain.ProjectMemberRoleRef, len(rows))
	for i, r := range rows {
		out[i] = domain.ProjectMemberRoleRef{ID: r.ID, Name: r.Name}
	}
	return out
}

func (r *projectMemberRepository) ListByProject(ctx context.Context, projectID uuid.UUID) ([]domain.ProjectMember, error) {
	rows, err := r.q.ListProjectMembers(ctx, projectID)
	if err != nil {
		return nil, errctx.Wrap(err, "ListProjectMembers", "projectID", projectID)
	}
	members := make([]domain.ProjectMember, 0, len(rows))
	for _, row := range rows {
		// загрузим роли участника
		roleNames, err := r.q.ListMemberRoles(ctx, row.ID)
		if err != nil {
			return nil, errctx.Wrap(err, "ListMemberRoles", "memberID", row.ID)
		}
		members = append(members, domain.ProjectMember{
			ID:        row.ID,
			ProjectID: row.ProjectID,
			UserID:    row.UserID,
			Roles:     toMemberRoleRefs(roleNames),
		})
	}
	return members, nil
}

func (r *projectMemberRepository) GetByProjectAndUser(ctx context.Context, projectID, userID uuid.UUID) (*domain.ProjectMember, error) {
	row, err := r.q.GetMemberByProjectAndUser(ctx, db.GetMemberByProjectAndUserParams{
		ProjectID: projectID,
		UserID:    userID,
	})
	if err != nil {
		return nil, errctx.Wrap(mapSQLErr(err, domain.ErrNotFound), "GetMemberByProjectAndUser", "projectID", projectID, "userID", userID)
	}
	return &domain.ProjectMember{
		ID:        row.ID,
		ProjectID: row.ProjectID,
		UserID:    row.UserID,
	}, nil
}

func (r *projectMemberRepository) AddMember(ctx context.Context, projectID, userID uuid.UUID) (*domain.ProjectMember, error) {
	row, err := r.q.AddProjectMember(ctx, db.AddProjectMemberParams{
		ProjectID: projectID,
		UserID:    userID,
	})
	if err != nil {
		return nil, errctx.Wrap(err, "AddProjectMember", "projectID", projectID, "userID", userID)
	}
	roleNames, err := r.q.ListMemberRoles(ctx, row.ID)
	if err != nil {
		return nil, errctx.Wrap(err, "ListMemberRoles", "memberID", row.ID)
	}
	return &domain.ProjectMember{
		ID:        row.ID,
		ProjectID: row.ProjectID,
		UserID:    row.UserID,
		Roles:     toMemberRoleRefs(roleNames),
	}, nil
}

func (r *projectMemberRepository) RemoveMember(ctx context.Context, memberID uuid.UUID) error {
	return errctx.Wrap(r.q.RemoveProjectMember(ctx, memberID), "RemoveProjectMember", "memberID", memberID)
}

func (r *projectMemberRepository) GetByID(ctx context.Context, memberID uuid.UUID) (*domain.ProjectMember, error) {
	row, err := r.q.GetProjectMember(ctx, memberID)
	if err != nil {
		return nil, errctx.Wrap(mapSQLErr(err, domain.ErrNotFound), "GetProjectMember", "memberID", memberID)
	}
	roleNames, err := r.q.ListMemberRoles(ctx, row.ID)
	if err != nil {
		return nil, errctx.Wrap(err, "ListMemberRoles", "memberID", row.ID)
	}
	return &domain.ProjectMember{
		ID:        row.ID,
		ProjectID: row.ProjectID,
		UserID:    row.UserID,
		Roles:     toMemberRoleRefs(roleNames),
	}, nil
}

func (r *projectMemberRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]ProjectMemberWithProject, error) {
	rows, err := r.q.ListProjectMembersByUser(ctx, userID)
	if err != nil {
		return nil, errctx.Wrap(err, "ListProjectMembersByUser", "userID", userID)
	}
	result := make([]ProjectMemberWithProject, 0, len(rows))
	for _, row := range rows {
		roleNames, err := r.q.ListMemberRoles(ctx, row.ID)
		if err != nil {
			return nil, errctx.Wrap(err, "ListMemberRoles", "memberID", row.ID)
		}
		roleIDs, err := r.q.ListMemberRoleIDs(ctx, row.ID)
		if err != nil {
			return nil, errctx.Wrap(err, "ListMemberRoleIDs", "memberID", row.ID)
		}
		result = append(result, ProjectMemberWithProject{
			MemberID:    row.ID,
			ProjectID:   row.ProjectID,
			ProjectName: row.ProjectName,
			Roles:       toMemberRoleRefs(roleNames),
			RoleIDs:     roleIDs,
		})
	}
	return result, nil
}

func (r *projectMemberRepository) ReplaceMemberRoles(ctx context.Context, memberID uuid.UUID, roleIDs []uuid.UUID) error {
	if err := r.q.DeleteMemberRoles(ctx, memberID); err != nil {
		return errctx.Wrap(err, "DeleteMemberRoles", "memberID", memberID)
	}
	for _, roleID := range roleIDs {
		if err := r.AddRoleToMember(ctx, memberID, roleID); err != nil {
			return err
		}
	}
	return nil
}

func (r *projectMemberRepository) AddRoleToMember(ctx context.Context, memberID, roleID uuid.UUID) error {
	// SQL-запрос идемпотентен (ON CONFLICT DO NOTHING), поэтому вызов для уже
	// назначенной роли не падает.
	return errctx.Wrap(r.q.AddRoleToMember(ctx, db.AddRoleToMemberParams{
		MemberID: memberID,
		RoleID:   roleID,
	}), "AddRoleToMember", "memberID", memberID, "roleID", roleID)
}
