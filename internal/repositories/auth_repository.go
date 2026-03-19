package repositories

import (
	"context"
	"database/sql"
	"errors"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"

	"projektus-backend/internal/db"
)

type AuthRepository interface {
	CreateRefreshToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
	RevokeAllUserTokens(ctx context.Context, userID string) error
	IsRefreshTokenValid(ctx context.Context, tokenHash string) (string, bool, error)

	InsertLoginAttempt(ctx context.Context, username, ip string, success bool) error
	CountFailedAttemptsByUsernameSince(ctx context.Context, username string, since time.Time) (int, error)
	CountFailedAttemptsByIPSince(ctx context.Context, ip string, since time.Time) (int, error)

	GetBlockedIPUntil(ctx context.Context, ip string) (*time.Time, error)
	BlockIPUntil(ctx context.Context, ip string, until time.Time) error
	CleanupExpiredBlockedIPs(ctx context.Context) error

	GetBlockedUserUntil(ctx context.Context, userID string) (*time.Time, error)
	BlockUserUntil(ctx context.Context, userID string, until time.Time) error
	CleanupExpiredBlockedUsers(ctx context.Context) error
}

type authRepository struct {
	q *db.Queries
}

func NewAuthRepository(q *db.Queries) AuthRepository {
	return &authRepository{q: q}
}

func parseInet(ip string) pqtype.Inet {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		parsed = net.ParseIP("127.0.0.1")
	}
	if v4 := parsed.To4(); v4 != nil {
		parsed = v4
	}
	bits := len(parsed) * 8
	return pqtype.Inet{
		IPNet: net.IPNet{IP: parsed, Mask: net.CIDRMask(bits, bits)},
		Valid: true,
	}
}

func (r *authRepository) CreateRefreshToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	_, err = r.q.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		UserID:    uid,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	})
	return err
}

func (r *authRepository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	t, err := r.q.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}
	return r.q.RevokeRefreshToken(ctx, t.ID)
}

func (r *authRepository) RevokeAllUserTokens(ctx context.Context, userID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	return r.q.RevokeAllUserRefreshTokens(ctx, uid)
}

func (r *authRepository) IsRefreshTokenValid(ctx context.Context, tokenHash string) (string, bool, error) {
	t, err := r.q.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false, nil
		}
		return "", false, err
	}
	if t.RevokedAt.Valid {
		return "", false, nil
	}
	if time.Now().After(t.ExpiresAt) {
		return "", false, nil
	}
	return t.UserID.String(), true, nil
}

func (r *authRepository) InsertLoginAttempt(ctx context.Context, username, ip string, success bool) error {
	return r.q.InsertLoginAttempt(ctx, db.InsertLoginAttemptParams{
		Username: sql.NullString{
			String: username,
			Valid:  username != "",
		},
		IpAddress: parseInet(ip),
		Success:   success,
	})
}

func (r *authRepository) CountFailedAttemptsByUsernameSince(ctx context.Context, username string, since time.Time) (int, error) {
	if username == "" {
		return 0, nil
	}
	n, err := r.q.CountFailedAttemptsByUsernameSince(ctx, db.CountFailedAttemptsByUsernameSinceParams{
		Username: sql.NullString{
			String: username,
			Valid:  true,
		},
		AttemptedAt: since,
	})
	return int(n), err
}

func (r *authRepository) CountFailedAttemptsByIPSince(ctx context.Context, ip string, since time.Time) (int, error) {
	n, err := r.q.CountFailedAttemptsByIPSince(ctx, db.CountFailedAttemptsByIPSinceParams{
		IpAddress:   parseInet(ip),
		AttemptedAt: since,
	})
	return int(n), err
}

func (r *authRepository) GetBlockedIPUntil(ctx context.Context, ip string) (*time.Time, error) {
	row, err := r.q.GetBlockedIP(ctx, parseInet(ip))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &row.BlockedUntil, nil
}

func (r *authRepository) BlockIPUntil(ctx context.Context, ip string, until time.Time) error {
	return r.q.UpsertBlockedIP(ctx, db.UpsertBlockedIPParams{
		IpAddress:    parseInet(ip),
		BlockedUntil: until,
	})
}

func (r *authRepository) CleanupExpiredBlockedIPs(ctx context.Context) error {
	return r.q.DeleteExpiredBlockedIPs(ctx)
}

func (r *authRepository) GetBlockedUserUntil(ctx context.Context, userID string) (*time.Time, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	row, err := r.q.GetBlockedUser(ctx, uid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &row.BlockedUntil, nil
}

func (r *authRepository) BlockUserUntil(ctx context.Context, userID string, until time.Time) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	return r.q.UpsertBlockedUser(ctx, db.UpsertBlockedUserParams{
		UserID:       uid,
		BlockedUntil: until,
	})
}

func (r *authRepository) CleanupExpiredBlockedUsers(ctx context.Context) error {
	return r.q.DeleteExpiredBlockedUsers(ctx)
}

