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

func (s *ProjectMemberService) AddMember(ctx context.Context, projectID, userID uuid.UUID, roleIDs []string) (*domain.ProjectMember, error) {
	// Проверяем, что пользователь существует
	if _, err := s.users.GetUserByID(ctx, userID.String()); err != nil {
		return nil, err
	}
	member, err := s.members.AddMember(ctx, projectID, userID)
	if err != nil {
		return nil, err
	}
	if len(roleIDs) > 0 {
		if err := s.setMemberRoles(ctx, member.ID, projectID, roleIDs); err != nil {
			return nil, err
		}
		return s.members.GetByID(ctx, member.ID)
	}
	return member, nil
}

func (s *ProjectMemberService) RemoveMember(ctx context.Context, memberID uuid.UUID) error {
	member, err := s.members.GetByID(ctx, memberID)
	if err != nil {
		return err
	}
	if s.projectRoleRepo != nil {
		adminRoleID, err := s.projectRoleRepo.GetProjectAdminRoleID(ctx, member.ProjectID)
		if err == nil {
			for _, roleID := range member.Roles {
				if roleID == adminRoleID.String() {
					count, _ := s.projectRoleRepo.CountMembersWithRole(ctx, member.ProjectID, adminRoleID)
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

func (s *ProjectMemberService) UpdateMemberRoles(ctx context.Context, memberID, projectID uuid.UUID, roleIDs []string) (*domain.ProjectMember, error) {
	if err := s.setMemberRoles(ctx, memberID, projectID, roleIDs); err != nil {
		return nil, err
	}
	return s.members.GetByID(ctx, memberID)
}

func (s *ProjectMemberService) setMemberRoles(ctx context.Context, memberID, projectID uuid.UUID, roleIDStrs []string) error {
	if len(roleIDStrs) == 0 {
		return s.members.ReplaceMemberRoles(ctx, memberID, nil)
	}

	// Parse role ID strings to UUIDs
	roleIDs := make([]uuid.UUID, 0, len(roleIDStrs))
	for _, idStr := range roleIDStrs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			continue
		}
		roleIDs = append(roleIDs, id)
	}

	// Check last admin protection before removing admin role
	if s.projectRoleRepo != nil {
		adminRoleID, err := s.projectRoleRepo.GetProjectAdminRoleID(ctx, projectID)
		if err == nil {
			member, merr := s.members.GetByID(ctx, memberID)
			if merr == nil {
				hasAdmin := false
				for _, rid := range member.Roles {
					if rid == adminRoleID.String() {
						hasAdmin = true
						break
					}
				}
				keepingAdmin := false
				for _, rid := range roleIDs {
					if rid == adminRoleID {
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

	return s.members.ReplaceMemberRoles(ctx, memberID, roleIDs)
}

