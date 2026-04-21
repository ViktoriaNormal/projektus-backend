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
	Username                  string
	Email                     string
	FullName                  string
	Position                  *string
	Password                  string
	IsActive                  *bool
	SystemRoleIDs             []uuid.UUID
	OnVacation                *bool
	IsSick                    *bool
	AlternativeContactChannel *string
	AlternativeContactInfo    *string
}

// AdminUpdateUserRequest — запрос на обновление пользователя администратором.
type AdminUpdateUserRequest struct {
	Username                  *string
	Email                     *string
	FullName                  *string
	Position                  *string
	IsActive                  *bool
	RoleIDs                   *[]uuid.UUID
	OnVacation                *bool
	IsSick                    *bool
	AlternativeContactChannel *string
	AlternativeContactInfo    *string
}

// AdminUserWithRoles — пользователь с привязанными системными ролями.
type AdminUserWithRoles struct {
	User  domain.User
	Roles []domain.Role
}

// AdminUserService — операции с пользователями для администратора.
type AdminUserService struct {
	userRepo        repositories.UserRepository
	adminUserRepo   repositories.AdminUserRepository
	roleSvc         *RoleService
	passwordSvc     PasswordService
	policySvc       *PasswordPolicyService
	notificationSvc NotificationService
}

func NewAdminUserService(
	userRepo repositories.UserRepository,
	adminUserRepo repositories.AdminUserRepository,
	roleSvc *RoleService,
	passwordSvc PasswordService,
	policySvc *PasswordPolicyService,
	notificationSvc NotificationService,
) *AdminUserService {
	return &AdminUserService{
		userRepo:        userRepo,
		adminUserRepo:   adminUserRepo,
		roleSvc:         roleSvc,
		passwordSvc:     passwordSvc,
		policySvc:       policySvc,
		notificationSvc: notificationSvc,
	}
}

// AdminUserListResult — агрегированный ответ админского списка пользователей:
// страница + `total` под применённые фильтры + глобальные счётчики статистики,
// которые не зависят от фильтров (для карточек «Активные»/«Заблокированные»
// на UI, чтобы они не «плясали» при вводе в поиск).
type AdminUserListResult struct {
	Users         []AdminUserWithRoles
	Total         int64 // под применённые фильтры
	ActiveCount   int64 // по всему множеству
	InactiveCount int64 // по всему множеству
}

// ListUsers возвращает страницу пользователей с ролями + глобальные счётчики.
func (s *AdminUserService) ListUsers(ctx context.Context, limit, offset int32, filter repositories.AdminUserListFilter) (*AdminUserListResult, error) {
	// limit = 0 — дешёвый запрос только счётчиков, без выгрузки страницы.
	countOnly := limit == 0
	if !countOnly {
		if limit < 0 {
			limit = 20
		}
		if limit > 100 {
			limit = 100
		}
	}
	if offset < 0 {
		offset = 0
	}

	var (
		users []domain.User
		total int64
		err   error
	)
	if countOnly {
		// Ровно COUNT под фильтры, страницу не запрашиваем. Передаём page_limit=0,
		// но sqlc-запрос использует LIMIT как есть — пустой результат, дешёвый COUNT рядом.
		users, total, err = s.adminUserRepo.ListAllUsers(ctx, 0, 0, filter)
	} else {
		users, total, err = s.adminUserRepo.ListAllUsers(ctx, limit, offset, filter)
	}
	if err != nil {
		return nil, err
	}

	active, err := s.adminUserRepo.CountActive(ctx, filter.IncludeDeleted)
	if err != nil {
		return nil, err
	}
	inactive, err := s.adminUserRepo.CountInactive(ctx, filter.IncludeDeleted)
	if err != nil {
		return nil, err
	}

	withRoles := make([]AdminUserWithRoles, 0, len(users))
	for _, u := range users {
		roles := s.getUserRoles(ctx, u.ID)
		withRoles = append(withRoles, AdminUserWithRoles{User: u, Roles: roles})
	}

	return &AdminUserListResult{
		Users:         withRoles,
		Total:         total,
		ActiveCount:   active,
		InactiveCount: inactive,
	}, nil
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
// Ни одного пользователя нельзя зарегистрировать без системной роли — правило
// действует даже при прямых вызовах сервиса в обход DTO-валидации.
func (s *AdminUserService) CreateUser(ctx context.Context, req AdminCreateUserRequest) (*AdminUserWithRoles, error) {
	if len(req.SystemRoleIDs) == 0 {
		return nil, domain.ErrUserRequiresRole
	}
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

	onVacation := false
	if req.OnVacation != nil {
		onVacation = *req.OnVacation
	}
	isSick := false
	if req.IsSick != nil {
		isSick = *req.IsSick
	}
	altChannel := sql.NullString{}
	if req.AlternativeContactChannel != nil {
		altChannel = sql.NullString{String: *req.AlternativeContactChannel, Valid: true}
	}
	altInfo := sql.NullString{}
	if req.AlternativeContactInfo != nil {
		altInfo = sql.NullString{String: *req.AlternativeContactInfo, Valid: true}
	}

	user, err := s.adminUserRepo.CreateUser(ctx, db.AdminCreateUserParams{
		Username:                  req.Username,
		Email:                     req.Email,
		PasswordHash:              hash,
		FullName:                  req.FullName,
		AvatarUrl:                 sql.NullString{},
		Position:                  position,
		IsActive:                  isActive,
		OnVacation:                onVacation,
		IsSick:                    isSick,
		AltContactChannel: altChannel,
		AltContactInfo:    altInfo,
	})
	if err != nil {
		return nil, errctx.Wrap(err, "CreateUser", "email", req.Email)
	}

	_ = s.userRepo.InsertPasswordHistory(ctx, user.ID, hash)

	if err := s.notificationSvc.InitializeDefaultSettings(ctx, user.ID.String()); err != nil {
		return nil, errctx.Wrap(err, "CreateUser", "userID", user.ID)
	}

	if err := s.roleSvc.AssignSystemRolesToUser(ctx, user.ID, req.SystemRoleIDs); err != nil {
		return nil, errctx.Wrap(err, "CreateUser", "email", req.Email)
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

	setOnVacation := req.OnVacation != nil
	onVacation := false
	if setOnVacation {
		onVacation = *req.OnVacation
	}

	setIsSick := req.IsSick != nil
	isSick := false
	if setIsSick {
		isSick = *req.IsSick
	}

	setAltContactChannel := req.AlternativeContactChannel != nil
	altContactChannel := sql.NullString{}
	if setAltContactChannel {
		altContactChannel = sql.NullString{String: *req.AlternativeContactChannel, Valid: *req.AlternativeContactChannel != ""}
	}

	setAltContactInfo := req.AlternativeContactInfo != nil
	altContactInfo := sql.NullString{}
	if setAltContactInfo {
		altContactInfo = sql.NullString{String: *req.AlternativeContactInfo, Valid: *req.AlternativeContactInfo != ""}
	}

	user, err := s.adminUserRepo.UpdateUser(ctx, db.AdminUpdateUserParams{
		ID:                        userID,
		Username:                  username,
		Email:                     email,
		FullName:                  fullName,
		SetPosition:               setPosition,
		Position:                  position,
		SetIsActive:               setIsActive,
		IsActive:                  isActive,
		SetOnVacation:             setOnVacation,
		OnVacation:                onVacation,
		SetIsSick:                 setIsSick,
		IsSick:                    isSick,
		SetAltContactChannel:      setAltContactChannel,
		AltContactChannel:   altContactChannel,
		SetAltContactInfo:   setAltContactInfo,
		AltContactInfo:      altContactInfo,
	})
	if err != nil {
		return nil, errctx.Wrap(err, "UpdateUser", "userID", userID)
	}

	if req.RoleIDs != nil {
		if len(*req.RoleIDs) == 0 {
			return nil, domain.ErrUserRequiresRole
		}
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
func (s *AdminUserService) getUserRoles(ctx context.Context, userID uuid.UUID) []domain.Role {
	roles, err := s.roleSvc.GetUserSystemRoles(ctx, userID)
	if err != nil {
		return nil
	}
	return roles
}
