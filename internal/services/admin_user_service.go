package services

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
)

// AdminCreateUserRequest — запрос на создание пользователя администратором.
type AdminCreateUserRequest struct {
	Username        string
	Email           string
	FullName        string
	InitialPassword string
	SystemRoleIDs   []uuid.UUID
}

// AdminUserService — операции с пользователями для администратора.
type AdminUserService struct {
	userRepo      repositories.UserRepository
	adminUserRepo repositories.AdminUserRepository
	roleSvc       *RoleService
	passwordSvc   PasswordService
	policySvc     *PasswordPolicyService
}

func NewAdminUserService(
	userRepo repositories.UserRepository,
	adminUserRepo repositories.AdminUserRepository,
	roleSvc *RoleService,
	passwordSvc PasswordService,
	policySvc *PasswordPolicyService,
) *AdminUserService {
	return &AdminUserService{
		userRepo:      userRepo,
		adminUserRepo: adminUserRepo,
		roleSvc:       roleSvc,
		passwordSvc:   passwordSvc,
		policySvc:     policySvc,
	}
}

// ListUsers возвращает список пользователей с пагинацией.
func (s *AdminUserService) ListUsers(ctx context.Context, limit, offset int32, includeDeleted bool) ([]domain.User, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return s.adminUserRepo.ListAllUsers(ctx, limit, offset, includeDeleted)
}

// CreateUser создаёт пользователя с начальным паролем и назначает системные роли.
func (s *AdminUserService) CreateUser(ctx context.Context, req AdminCreateUserRequest) (*domain.User, error) {
	if err := s.policySvc.ValidatePassword(ctx, req.InitialPassword); err != nil {
		return nil, err
	}
	hash, err := s.passwordSvc.HashPassword(req.InitialPassword)
	if err != nil {
		return nil, err
	}
	user, err := s.userRepo.CreateUser(ctx, req.Username, req.Email, hash, req.FullName, nil)
	if err != nil {
		return nil, err
	}
	_ = s.userRepo.InsertPasswordHistory(ctx, user.ID, hash)
	if len(req.SystemRoleIDs) > 0 {
		uid, _ := uuid.Parse(user.ID)
		if err := s.roleSvc.AssignSystemRolesToUser(ctx, uid, req.SystemRoleIDs); err != nil {
			return user, err
		}
	}
	return user, nil
}

// DeleteUser выполняет мягкое удаление пользователя. Нельзя удалить самого себя.
func (s *AdminUserService) DeleteUser(ctx context.Context, targetUserID uuid.UUID, currentUserID uuid.UUID) error {
	if targetUserID == currentUserID {
		return domain.ErrInvalidInput
	}
	_, err := s.userRepo.GetUserByID(ctx, targetUserID.String())
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return domain.ErrNotFound
		}
		return err
	}
	return s.adminUserRepo.SoftDeleteUser(ctx, targetUserID)
}
