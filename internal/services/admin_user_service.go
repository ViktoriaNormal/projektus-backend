package services

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
	"projektus-backend/pkg/errctx"
)

// AdminCreateUserRequest — запрос на создание пользователя администратором.
type AdminCreateUserRequest struct {
	Username      string
	Email         string
	FullName      string
	Position      *string
	Password      string
	IsActive      *bool
	SystemRoleIDs []uuid.UUID
}

// AdminUpdateUserRequest — запрос на обновление пользователя администратором.
type AdminUpdateUserRequest struct {
	Username *string
	Email    *string
	FullName *string
	Position *string
	IsActive *bool
	RoleIDs  *[]uuid.UUID
}

// AdminUserWithRoles — пользователь с привязанными системными ролями.
type AdminUserWithRoles struct {
	User  domain.User
	Roles []domain.Role
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

// ListUsers возвращает список пользователей с ролями.
func (s *AdminUserService) ListUsers(ctx context.Context, limit, offset int32, includeDeleted bool) ([]AdminUserWithRoles, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	users, total, err := s.adminUserRepo.ListAllUsers(ctx, limit, offset, includeDeleted)
	if err != nil {
		return nil, 0, err
	}
	result := make([]AdminUserWithRoles, 0, len(users))
	for _, u := range users {
		roles := s.getUserRoles(ctx, u.ID)
		result = append(result, AdminUserWithRoles{User: u, Roles: roles})
	}
	return result, total, nil
}

// GetUser возвращает пользователя по ID с ролями.
func (s *AdminUserService) GetUser(ctx context.Context, userID uuid.UUID) (*AdminUserWithRoles, error) {
	user, err := s.adminUserRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	roles := s.getUserRoles(ctx, user.ID)
	return &AdminUserWithRoles{User: *user, Roles: roles}, nil
}

// CreateUser создаёт пользователя с начальным паролем и назначает системные роли.
func (s *AdminUserService) CreateUser(ctx context.Context, req AdminCreateUserRequest) (*AdminUserWithRoles, error) {
	if err := s.policySvc.ValidatePassword(ctx, req.Password); err != nil {
		return nil, errctx.Wrap(err, "CreateUser", "email", req.Email)
	}
	hash, err := s.passwordSvc.HashPassword(req.Password)
	if err != nil {
		return nil, errctx.Wrap(err, "CreateUser", "email", req.Email)
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	position := sql.NullString{}
	if req.Position != nil {
		position = sql.NullString{String: *req.Position, Valid: true}
	}

	user, err := s.adminUserRepo.CreateUser(ctx, db.AdminCreateUserParams{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hash,
		FullName:     req.FullName,
		AvatarUrl:    sql.NullString{},
		Position:     position,
		IsActive:     isActive,
	})
	if err != nil {
		return nil, errctx.Wrap(err, "CreateUser", "email", req.Email)
	}

	_ = s.userRepo.InsertPasswordHistory(ctx, user.ID, hash)

	if len(req.SystemRoleIDs) > 0 {
		uid, _ := uuid.Parse(user.ID)
		if err := s.roleSvc.AssignSystemRolesToUser(ctx, uid, req.SystemRoleIDs); err != nil {
			return nil, errctx.Wrap(err, "CreateUser", "email", req.Email)
		}
	}

	roles := s.getUserRoles(ctx, user.ID)
	return &AdminUserWithRoles{User: *user, Roles: roles}, nil
}

// UpdateUser обновляет данные пользователя.
func (s *AdminUserService) UpdateUser(ctx context.Context, userID uuid.UUID, req AdminUpdateUserRequest) (*AdminUserWithRoles, error) {
	username := ""
	if req.Username != nil {
		username = *req.Username
	}
	email := ""
	if req.Email != nil {
		email = *req.Email
	}
	fullName := ""
	if req.FullName != nil {
		fullName = *req.FullName
	}

	setPosition := req.Position != nil
	position := sql.NullString{}
	if setPosition {
		position = sql.NullString{String: *req.Position, Valid: *req.Position != ""}
	}

	setIsActive := req.IsActive != nil
	isActive := false
	if setIsActive {
		isActive = *req.IsActive
	}

	user, err := s.adminUserRepo.UpdateUser(ctx, db.AdminUpdateUserParams{
		ID:          userID,
		Username:    username,
		Email:       email,
		FullName:    fullName,
		SetPosition: setPosition,
		Position:    position,
		SetIsActive: setIsActive,
		IsActive:    isActive,
	})
	if err != nil {
		return nil, errctx.Wrap(err, "UpdateUser", "userID", userID)
	}

	if req.RoleIDs != nil {
		if err := s.roleSvc.AssignSystemRolesToUser(ctx, userID, *req.RoleIDs); err != nil {
			return nil, errctx.Wrap(err, "UpdateUser", "userID", userID)
		}
	}

	roles := s.getUserRoles(ctx, user.ID)
	return &AdminUserWithRoles{User: *user, Roles: roles}, nil
}

// DeleteUser выполняет мягкое удаление пользователя. Нельзя удалить самого себя.
func (s *AdminUserService) DeleteUser(ctx context.Context, targetUserID uuid.UUID, currentUserID uuid.UUID) error {
	if targetUserID == currentUserID {
		return domain.ErrInvalidInput
	}
	_, err := s.adminUserRepo.GetUserByID(ctx, targetUserID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrNotFound
		}
		return errctx.Wrap(err, "DeleteUser", "targetUserID", targetUserID)
	}
	err = s.adminUserRepo.SoftDeleteUser(ctx, targetUserID)
	return errctx.Wrap(err, "DeleteUser", "targetUserID", targetUserID)
}

// getUserRoles возвращает системные роли пользователя (не возвращает ошибку, при сбое — пустой список).
func (s *AdminUserService) getUserRoles(ctx context.Context, userID string) []domain.Role {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil
	}
	roles, err := s.roleSvc.GetUserSystemRoles(ctx, uid)
	if err != nil {
		return nil
	}
	return roles
}
