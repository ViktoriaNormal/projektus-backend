package services

import (
	"context"

	"github.com/google/uuid"

	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
)

type ProjectMemberService struct {
	members         ProjectMemberRepository
	users           repositories.UserRepository
	roles           repositories.RoleRepository
	projectRoleRepo repositories.ProjectRoleRepository
}

type ProjectMemberRepository interface {
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]domain.ProjectMember, error)
	AddMember(ctx context.Context, projectID, userID uuid.UUID) (*domain.ProjectMember, error)
	RemoveMember(ctx context.Context, memberID uuid.UUID) error
	GetByID(ctx context.Context, memberID uuid.UUID) (*domain.ProjectMember, error)
	ReplaceMemberRoles(ctx context.Context, memberID uuid.UUID, roleIDs []uuid.UUID) error
}

func NewProjectMemberService(members ProjectMemberRepository, users repositories.UserRepository, roles repositories.RoleRepository, projectRoleRepo repositories.ProjectRoleRepository) *ProjectMemberService {
	return &ProjectMemberService{members: members, users: users, roles: roles, projectRoleRepo: projectRoleRepo}
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
	// Check if member has admin role and is the last one
	member, err := s.members.GetByID(ctx, memberID)
	if err != nil {
		return err
	}
	if s.projectRoleRepo != nil {
		adminRoleID, err := s.projectRoleRepo.GetProjectAdminRoleID(ctx, member.ProjectID)
		if err == nil {
			count, _ := s.projectRoleRepo.CountMembersWithRole(ctx, member.ProjectID, adminRoleID)
			// Check if this member has admin role
			for _, roleName := range member.Roles {
				if roleName == ProjectAdminRoleName {
					if count <= 1 {
						return domain.ErrLastProjectAdmin
					}
					break
				}
			}
		}
	}
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

	// Check last admin protection before removing admin role
	if s.projectRoleRepo != nil {
		adminRoleID, err := s.projectRoleRepo.GetProjectAdminRoleID(ctx, projectID)
		if err == nil {
			// Check if we're removing admin role from the last admin
			member, merr := s.members.GetByID(ctx, memberID)
			if merr == nil {
				hasAdmin := false
				for _, rn := range member.Roles {
					if rn == ProjectAdminRoleName {
						hasAdmin = true
						break
					}
				}
				keepingAdmin := false
				for _, rn := range roleNames {
					if rn == ProjectAdminRoleName {
						keepingAdmin = true
						break
					}
				}
				if hasAdmin && !keepingAdmin {
					count, _ := s.projectRoleRepo.CountMembersWithRole(ctx, projectID, adminRoleID)
					if count <= 1 {
						return domain.ErrLastProjectAdmin
					}
				}
			}
		}
	}

	// Look up roles in project_roles table
	projectRoles, err := s.projectRoleRepo.List(ctx, projectID)
	if err != nil {
		return err
	}
	nameToID := make(map[string]uuid.UUID, len(projectRoles))
	for _, r := range projectRoles {
		nameToID[r.Name] = uuid.MustParse(r.ID)
	}
	var roleIDs []uuid.UUID
	for _, name := range roleNames {
		if id, ok := nameToID[name]; ok {
			roleIDs = append(roleIDs, id)
		}
	}
	return s.members.ReplaceMemberRoles(ctx, memberID, roleIDs)
}

