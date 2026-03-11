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

	InsertLoginAttempt(ctx context.Context, email, ip string, success bool) error
	CountFailedAttemptsByEmailSince(ctx context.Context, email string, since time.Time) (int, error)
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

func (r *authRepository) InsertLoginAttempt(ctx context.Context, email, ip string, success bool) error {
	var inet pqtype.Inet
	if parsed := net.ParseIP(ip); parsed != nil {
		inet = pqtype.Inet{IPNet: net.IPNet{IP: parsed, Mask: net.CIDRMask(32, 32)}}
	}
	return r.q.InsertLoginAttempt(ctx, db.InsertLoginAttemptParams{
		Email: sql.NullString{
			String: email,
			Valid:  email != "",
		},
		IpAddress: inet,
		Success:   success,
	})
}

func (r *authRepository) CountFailedAttemptsByEmailSince(ctx context.Context, email string, since time.Time) (int, error) {
	if email == "" {
		return 0, nil
	}
	n, err := r.q.CountFailedAttemptsByEmailSince(ctx, db.CountFailedAttemptsByEmailSinceParams{
		Email: sql.NullString{
			String: email,
			Valid:  true,
		},
		AttemptedAt: since,
	})
	return int(n), err
}

func (r *authRepository) CountFailedAttemptsByIPSince(ctx context.Context, ip string, since time.Time) (int, error) {
	var inet pqtype.Inet
	if parsed := net.ParseIP(ip); parsed != nil {
		inet = pqtype.Inet{IPNet: net.IPNet{IP: parsed, Mask: net.CIDRMask(32, 32)}}
	}
	n, err := r.q.CountFailedAttemptsByIPSince(ctx, db.CountFailedAttemptsByIPSinceParams{
		IpAddress:   inet,
		AttemptedAt: since,
	})
	return int(n), err
}

func (r *authRepository) GetBlockedIPUntil(ctx context.Context, ip string) (*time.Time, error) {
	var inet pqtype.Inet
	if parsed := net.ParseIP(ip); parsed != nil {
		inet = pqtype.Inet{IPNet: net.IPNet{IP: parsed, Mask: net.CIDRMask(32, 32)}}
	}
	row, err := r.q.GetBlockedIP(ctx, inet)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &row.BlockedUntil, nil
}

func (r *authRepository) BlockIPUntil(ctx context.Context, ip string, until time.Time) error {
	var inet pqtype.Inet
	if parsed := net.ParseIP(ip); parsed != nil {
		inet = pqtype.Inet{IPNet: net.IPNet{IP: parsed, Mask: net.CIDRMask(32, 32)}}
	}
	return r.q.UpsertBlockedIP(ctx, db.UpsertBlockedIPParams{
		IpAddress:    inet,
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

