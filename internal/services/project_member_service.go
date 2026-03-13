package services

import (
	"context"

	"github.com/google/uuid"

	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
)

type ProjectMemberService struct {
	members ProjectMemberRepository
	users   repositories.UserRepository
	roles   repositories.RoleRepository
}

type ProjectMemberRepository interface {
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]domain.ProjectMember, error)
	AddMember(ctx context.Context, projectID, userID uuid.UUID) (*domain.ProjectMember, error)
	RemoveMember(ctx context.Context, memberID uuid.UUID) error
	GetByID(ctx context.Context, memberID uuid.UUID) (*domain.ProjectMember, error)
	ReplaceMemberRoles(ctx context.Context, memberID uuid.UUID, roleIDs []uuid.UUID) error
}

func NewProjectMemberService(members ProjectMemberRepository, users repositories.UserRepository, roles repositories.RoleRepository) *ProjectMemberService {
	return &ProjectMemberService{members: members, users: users, roles: roles}
}

func (s *ProjectMemberService) ListMembers(ctx context.Context, projectID uuid.UUID) ([]domain.ProjectMember, error) {
	return s.members.ListByProject(ctx, projectID)
}

func (s *ProjectMemberService) AddMember(ctx context.Context, projectID, userID uuid.UUID, roleNames []string) (*domain.ProjectMember, error) {
	// Проверяем, что пользователь существует
	if _, err := s.users.GetUserByID(ctx, userID.String()); err != nil {
		return nil, err
	}
	member, err := s.members.AddMember(ctx, projectID, userID)
	if err != nil {
		return nil, err
	}
	if len(roleNames) > 0 {
		if err := s.setMemberRolesByNames(ctx, member.ID, projectID, roleNames); err != nil {
			return nil, err
		}
		// перечитаем участника с ролями
		return s.members.GetByID(ctx, member.ID)
	}
	return member, nil
}

func (s *ProjectMemberService) RemoveMember(ctx context.Context, memberID uuid.UUID) error {
	return s.members.RemoveMember(ctx, memberID)
}

func (s *ProjectMemberService) UpdateMemberRoles(ctx context.Context, memberID, projectID uuid.UUID, roleNames []string) (*domain.ProjectMember, error) {
	if err := s.setMemberRolesByNames(ctx, memberID, projectID, roleNames); err != nil {
		return nil, err
	}
	return s.members.GetByID(ctx, memberID)
}

func (s *ProjectMemberService) setMemberRolesByNames(ctx context.Context, memberID, projectID uuid.UUID, roleNames []string) error {
	if len(roleNames) == 0 {
		return s.members.ReplaceMemberRoles(ctx, memberID, nil)
	}
	projectRoles, err := s.roles.ListProjectRoles(ctx, projectID)
	if err != nil {
		return err
	}
	// мапа имя роли -> id
	nameToID := make(map[string]uuid.UUID, len(projectRoles))
	for _, r := range projectRoles {
		nameToID[r.Name] = r.ID
	}
	var roleIDs []uuid.UUID
	for _, name := range roleNames {
		if id, ok := nameToID[name]; ok {
			roleIDs = append(roleIDs, id)
		}
	}
	return s.members.ReplaceMemberRoles(ctx, memberID, roleIDs)
}

