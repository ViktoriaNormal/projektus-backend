package repositories

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
	"projektus-backend/pkg/errctx"
)

type UserRepository interface {
	CreateUser(ctx context.Context, username, email, passwordHash, fullName string, avatarURL *string) (*domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
	GetUserByUsername(ctx context.Context, username string) (*domain.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	UpdatePassword(ctx context.Context, userID uuid.UUID, newHash string) error
	InsertPasswordHistory(ctx context.Context, userID uuid.UUID, hash string) error
	GetLastPasswordHashes(ctx context.Context, userID uuid.UUID, limit int32) ([]string, error)
	UpdateProfile(ctx context.Context, userID uuid.UUID, fullName, email string, position *string, onVacation bool, isSick bool, altContactChannel *string, altContactInfo *string) error
	UpdateAvatar(ctx context.Context, userID uuid.UUID, avatarURL string) error
	SearchUsers(ctx context.Context, query string, limit, offset int32) ([]domain.User, error)
	CountSearchUsers(ctx context.Context, query string) (int64, error)
	ListAllUserIDs(ctx context.Context) ([]uuid.UUID, error)
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
		return nil, errctx.Wrap(err, "CreateUser", "email", email)
	}
	return mapDBUserToDomain(u), nil
}

func (r *userRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	u, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, errctx.Wrap(mapSQLErr(err, domain.ErrUserNotFound), "GetUserByEmail", "email", email)
	}
	return mapDBUserToDomain(u), nil
}

func (r *userRepository) GetUserByUsername(ctx context.Context, username string) (*domain.User, error) {
	u, err := r.q.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, errctx.Wrap(mapSQLErr(err, domain.ErrUserNotFound), "GetUserByUsername", "username", username)
	}
	return mapDBUserToDomain(u), nil
}

func (r *userRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	u, err := r.q.GetUserByID(ctx, id)
	if err != nil {
		return nil, errctx.Wrap(mapSQLErr(err, domain.ErrUserNotFound), "GetUserByID", "id", id)
	}
	return mapDBUserToDomain(u), nil
}

func (r *userRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, newHash string) error {
	err := r.q.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
		ID:           userID,
		PasswordHash: newHash,
	})
	return errctx.Wrap(err, "UpdatePassword", "userID", userID)
}

func (r *userRepository) InsertPasswordHistory(ctx context.Context, userID uuid.UUID, hash string) error {
	err := r.q.InsertPasswordHistory(ctx, db.InsertPasswordHistoryParams{
		UserID:       userID,
		PasswordHash: hash,
	})
	return errctx.Wrap(err, "InsertPasswordHistory", "userID", userID)
}

func (r *userRepository) GetLastPasswordHashes(ctx context.Context, userID uuid.UUID, limit int32) ([]string, error) {
	hashes, err := r.q.GetLastNPasswordHashes(ctx, db.GetLastNPasswordHashesParams{
		UserID: userID,
		Limit:  limit,
	})
	return hashes, errctx.Wrap(err, "GetLastPasswordHashes", "userID", userID)
}

func (r *userRepository) UpdateProfile(ctx context.Context, userID uuid.UUID, fullName, email string, position *string, onVacation bool, isSick bool, altContactChannel *string, altContactInfo *string) error {
	pos := sql.NullString{}
	if position != nil {
		pos = sql.NullString{String: *position, Valid: true}
	}
	altChannel := sql.NullString{}
	if altContactChannel != nil {
		altChannel = sql.NullString{String: *altContactChannel, Valid: true}
	}
	altInfo := sql.NullString{}
	if altContactInfo != nil {
		altInfo = sql.NullString{String: *altContactInfo, Valid: true}
	}
	err := r.q.UpdateUserProfile(ctx, db.UpdateUserProfileParams{
		ID:                userID,
		FullName:          fullName,
		Email:             email,
		Position:          pos,
		OnVacation:        onVacation,
		IsSick:            isSick,
		AltContactChannel: altChannel,
		AltContactInfo:    altInfo,
	})
	return errctx.Wrap(err, "UpdateProfile", "userID", userID)
}

func (r *userRepository) UpdateAvatar(ctx context.Context, userID uuid.UUID, avatarURL string) error {
	err := r.q.UpdateUserAvatar(ctx, db.UpdateUserAvatarParams{
		ID:        userID,
		AvatarUrl: sql.NullString{String: avatarURL, Valid: true},
	})
	return errctx.Wrap(err, "UpdateAvatar", "userID", userID)
}

func (r *userRepository) SearchUsers(ctx context.Context, query string, limit, offset int32) ([]domain.User, error) {
	rows, err := r.q.SearchUsers(ctx, db.SearchUsersParams{
		Column1: query,
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		return nil, errctx.Wrap(err, "SearchUsers", "query", query, "limit", limit, "offset", offset)
	}
	result := make([]domain.User, len(rows))
	for i, u := range rows {
		d := mapDBUserToDomain(u)
		result[i] = *d
	}
	return result, nil
}

func (r *userRepository) CountSearchUsers(ctx context.Context, query string) (int64, error) {
	n, err := r.q.CountSearchUsers(ctx, query)
	if err != nil {
		return 0, errctx.Wrap(err, "CountSearchUsers", "query", query)
	}
	return n, nil
}

func (r *userRepository) ListAllUserIDs(ctx context.Context) ([]uuid.UUID, error) {
	rows, err := r.q.ListAllUserIDs(ctx)
	if err != nil {
		return nil, errctx.Wrap(err, "ListAllUserIDs")
	}
	return rows, nil
}

func mapDBUserToDomain(u db.User) *domain.User {
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
