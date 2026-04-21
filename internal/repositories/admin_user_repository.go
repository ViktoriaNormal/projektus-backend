package repositories

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
	"projektus-backend/pkg/errctx"
)

// AdminUserListFilter собирает опциональные фильтры серверной выборки
// пользователей в админке. Любое поле-nil = «без фильтра».
type AdminUserListFilter struct {
	Query          *string    // ILIKE по username/email/full_name/position
	IsActive       *bool      // ровно true или false
	RoleID         *uuid.UUID // пользователи с назначенной системной ролью
	IncludeDeleted bool       // включать soft-deleted
}

type AdminUserRepository interface {
	ListAllUsers(ctx context.Context, limit, offset int32, filter AdminUserListFilter) ([]domain.User, int64, error)
	GetUserByID(ctx context.Context, userID uuid.UUID) (*domain.User, error)
	CreateUser(ctx context.Context, params db.AdminCreateUserParams) (*domain.User, error)
	UpdateUser(ctx context.Context, params db.AdminUpdateUserParams) (*domain.User, error)
	SoftDeleteUser(ctx context.Context, userID uuid.UUID) error
	// CountActive/Inactive считают по всему множеству (с учётом includeDeleted),
	// независимо от фильтров — для карточек статистики на UI.
	CountActive(ctx context.Context, includeDeleted bool) (int64, error)
	CountInactive(ctx context.Context, includeDeleted bool) (int64, error)
}

type adminUserRepository struct {
	q *db.Queries
}

func NewAdminUserRepository(q *db.Queries) AdminUserRepository {
	return &adminUserRepository{q: q}
}

func (r *adminUserRepository) ListAllUsers(ctx context.Context, limit, offset int32, filter AdminUserListFilter) ([]domain.User, int64, error) {
	qArg := sql.NullString{}
	if filter.Query != nil && *filter.Query != "" {
		qArg = sql.NullString{String: *filter.Query, Valid: true}
	}
	activeArg := sql.NullBool{}
	if filter.IsActive != nil {
		activeArg = sql.NullBool{Bool: *filter.IsActive, Valid: true}
	}
	roleArg := uuid.NullUUID{}
	if filter.RoleID != nil {
		roleArg = uuid.NullUUID{UUID: *filter.RoleID, Valid: true}
	}

	rows, err := r.q.ListAllUsers(ctx, db.ListAllUsersParams{
		IncludeDeleted: filter.IncludeDeleted,
		Q:              qArg,
		IsActiveFilter: activeArg,
		RoleIDFilter:   roleArg,
		PageLimit:      limit,
		PageOffset:     offset,
	})
	if err != nil {
		return nil, 0, errctx.Wrap(err, "ListAllUsers", "limit", limit, "offset", offset)
	}
	total, err := r.q.ListAllUsersCount(ctx, db.ListAllUsersCountParams{
		IncludeDeleted: filter.IncludeDeleted,
		Q:              qArg,
		IsActiveFilter: activeArg,
		RoleIDFilter:   roleArg,
	})
	if err != nil {
		return nil, 0, errctx.Wrap(err, "ListAllUsersCount")
	}
	list := make([]domain.User, 0, len(rows))
	for _, u := range rows {
		list = append(list, *mapDBUserToDomainUser(u))
	}
	return list, total, nil
}

func (r *adminUserRepository) CountActive(ctx context.Context, includeDeleted bool) (int64, error) {
	n, err := r.q.CountActiveUsers(ctx, includeDeleted)
	if err != nil {
		return 0, errctx.Wrap(err, "CountActiveUsers")
	}
	return n, nil
}

func (r *adminUserRepository) CountInactive(ctx context.Context, includeDeleted bool) (int64, error) {
	n, err := r.q.CountInactiveUsers(ctx, includeDeleted)
	if err != nil {
		return 0, errctx.Wrap(err, "CountInactiveUsers")
	}
	return n, nil
}

func (r *adminUserRepository) GetUserByID(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	u, err := r.q.AdminGetUserByID(ctx, userID)
	if err != nil {
		return nil, errctx.Wrap(mapSQLErr(err, domain.ErrNotFound), "AdminGetUserByID", "userID", userID)
	}
	return mapDBUserToDomainUser(u), nil
}

func (r *adminUserRepository) CreateUser(ctx context.Context, params db.AdminCreateUserParams) (*domain.User, error) {
	u, err := r.q.AdminCreateUser(ctx, params)
	if err != nil {
		return nil, errctx.Wrap(err, "AdminCreateUser", "email", params.Email)
	}
	return mapDBUserToDomainUser(u), nil
}

func (r *adminUserRepository) UpdateUser(ctx context.Context, params db.AdminUpdateUserParams) (*domain.User, error) {
	u, err := r.q.AdminUpdateUser(ctx, params)
	if err != nil {
		return nil, errctx.Wrap(mapSQLErr(err, domain.ErrNotFound), "AdminUpdateUser", "userID", params.ID)
	}
	return mapDBUserToDomainUser(u), nil
}

func (r *adminUserRepository) SoftDeleteUser(ctx context.Context, userID uuid.UUID) error {
	return errctx.Wrap(r.q.SoftDeleteUser(ctx, userID), "SoftDeleteUser", "userID", userID)
}

func mapDBUserToDomainUser(u db.User) *domain.User {
	var avatarURL *string
	if u.AvatarUrl.Valid {
		avatarURL = &u.AvatarUrl.String
	}
	var position *string
	if u.Position.Valid {
		position = &u.Position.String
	}
	var altContactChannel *string
	if u.AltContactChannel.Valid {
		altContactChannel = &u.AltContactChannel.String
	}
	var altContactInfo *string
	if u.AltContactInfo.Valid {
		altContactInfo = &u.AltContactInfo.String
	}
	return &domain.User{
		ID:                        u.ID,
		Username:                  u.Username,
		Email:                     u.Email,
		PasswordHash:              u.PasswordHash,
		FullName:                  u.FullName,
		AvatarURL:                 avatarURL,
		Position:                  position,
		OnVacation:                u.OnVacation,
		IsSick:                    u.IsSick,
		AlternativeContactChannel: altContactChannel,
		AlternativeContactInfo:    altContactInfo,
		IsActive:                  u.IsActive,
	}
}
