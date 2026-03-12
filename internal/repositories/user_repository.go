package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
)

type UserRepository interface {
	CreateUser(ctx context.Context, username, email, passwordHash, fullName string, avatarURL *string) (*domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
	GetUserByID(ctx context.Context, id string) (*domain.User, error)
	UpdatePassword(ctx context.Context, userID, newHash string) error
	InsertPasswordHistory(ctx context.Context, userID, hash string) error
	GetLastPasswordHashes(ctx context.Context, userID string, limit int32) ([]string, error)
	UpdateProfile(ctx context.Context, userID, fullName, email string) error
	UpdateAvatar(ctx context.Context, userID, avatarURL string) error
	SearchUsers(ctx context.Context, query string, limit, offset int32) ([]domain.User, error)
	ListAllUserIDs(ctx context.Context) ([]string, error)
}

type userRepository struct {
	q *db.Queries
}

func NewUserRepository(q *db.Queries) UserRepository {
	return &userRepository{q: q}
}

func (r *userRepository) CreateUser(ctx context.Context, username, email, passwordHash, fullName string, avatarURL *string) (*domain.User, error) {
	avatar := sql.NullString{}
	if avatarURL != nil {
		avatar = sql.NullString{String: *avatarURL, Valid: true}
	}
	u, err := r.q.CreateUser(ctx, db.CreateUserParams{
		Username:     username,
		Email:        email,
		PasswordHash: passwordHash,
		FullName:     fullName,
		AvatarUrl:    avatar,
	})
	if err != nil {
		return nil, err
	}
	return mapDBUserToDomain(u), nil
}

func (r *userRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	u, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return mapDBUserToDomain(u), nil
}

func (r *userRepository) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, domain.ErrUserNotFound
	}
	u, err := r.q.GetUserByID(ctx, uid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return mapDBUserToDomain(u), nil
}

func (r *userRepository) UpdatePassword(ctx context.Context, userID, newHash string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	return r.q.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
		ID:           uid,
		PasswordHash: newHash,
	})
}

func (r *userRepository) InsertPasswordHistory(ctx context.Context, userID, hash string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	return r.q.InsertPasswordHistory(ctx, db.InsertPasswordHistoryParams{
		UserID:       uid,
		PasswordHash: hash,
	})
}

func (r *userRepository) GetLastPasswordHashes(ctx context.Context, userID string, limit int32) ([]string, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	return r.q.GetLastNPasswordHashes(ctx, db.GetLastNPasswordHashesParams{
		UserID: uid,
		Limit:  limit,
	})
}

func (r *userRepository) UpdateProfile(ctx context.Context, userID, fullName, email string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	return r.q.UpdateUserProfile(ctx, db.UpdateUserProfileParams{
		ID:       uid,
		FullName: fullName,
		Email:    email,
	})
}

func (r *userRepository) UpdateAvatar(ctx context.Context, userID, avatarURL string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	return r.q.UpdateUserAvatar(ctx, db.UpdateUserAvatarParams{
		ID:        uid,
		AvatarUrl: sql.NullString{String: avatarURL, Valid: true},
	})
}

func (r *userRepository) SearchUsers(ctx context.Context, query string, limit, offset int32) ([]domain.User, error) {
	param := sql.NullString{String: query, Valid: query != ""}
	rows, err := r.q.SearchUsers(ctx, db.SearchUsersParams{
		Column1: param,
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		return nil, err
	}
	result := make([]domain.User, len(rows))
	for i, u := range rows {
		d := mapDBUserToDomain(u)
		result[i] = *d
	}
	return result, nil
}

func (r *userRepository) ListAllUserIDs(ctx context.Context) ([]string, error) {
	rows, err := r.q.ListAllUserIDs(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]string, len(rows))
	for i, id := range rows {
		result[i] = id.String()
	}
	return result, nil
}

func mapDBUserToDomain(u db.User) *domain.User {
	var avatarURL *string
	if u.AvatarUrl.Valid {
		avatarURL = &u.AvatarUrl.String
	}
	return &domain.User{
		ID:           u.ID.String(),
		Username:     u.Username,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		FullName:     u.FullName,
		AvatarURL:    avatarURL,
		IsActive:     u.IsActive,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
}
