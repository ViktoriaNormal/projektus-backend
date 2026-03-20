package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
	"projektus-backend/pkg/errctx"
)

type AdminUserRepository interface {
	ListAllUsers(ctx context.Context, limit, offset int32, includeDeleted bool) ([]domain.User, int64, error)
	GetUserByID(ctx context.Context, userID uuid.UUID) (*domain.User, error)
	CreateUser(ctx context.Context, params db.AdminCreateUserParams) (*domain.User, error)
	UpdateUser(ctx context.Context, params db.AdminUpdateUserParams) (*domain.User, error)
	SoftDeleteUser(ctx context.Context, userID uuid.UUID) error
}

type adminUserRepository struct {
	q *db.Queries
}

func NewAdminUserRepository(q *db.Queries) AdminUserRepository {
	return &adminUserRepository{q: q}
}

func (r *adminUserRepository) ListAllUsers(ctx context.Context, limit, offset int32, includeDeleted bool) ([]domain.User, int64, error) {
	rows, err := r.q.ListAllUsers(ctx, db.ListAllUsersParams{
		Limit:   limit,
		Offset:  offset,
		Column3: includeDeleted,
	})
	if err != nil {
		return nil, 0, err
	}
	total, err := r.q.ListAllUsersCount(ctx, includeDeleted)
	if err != nil {
		return nil, 0, err
	}
	list := make([]domain.User, 0, len(rows))
	for _, u := range rows {
		list = append(list, *mapDBUserToDomainUser(u))
	}
	return list, total, nil
}

func (r *adminUserRepository) GetUserByID(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	u, err := r.q.AdminGetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, errctx.Wrap(err, "AdminGetUserByID", "userID", userID)
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
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, errctx.Wrap(err, "AdminUpdateUser", "userID", params.ID)
	}
	return mapDBUserToDomainUser(u), nil
}

func (r *adminUserRepository) SoftDeleteUser(ctx context.Context, userID uuid.UUID) error {
	return r.q.SoftDeleteUser(ctx, userID)
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
	return &domain.User{
		ID:           u.ID.String(),
		Username:     u.Username,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		FullName:     u.FullName,
		AvatarURL:    avatarURL,
		Position:     position,
		IsActive:     u.IsActive,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
}
