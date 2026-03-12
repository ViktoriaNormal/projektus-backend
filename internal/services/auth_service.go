package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"projektus-backend/config"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
	"projektus-backend/pkg/utils"
)

type AuthService interface {
	Register(ctx context.Context, username, email, password, fullName string) (*domain.User, error)
	Login(ctx context.Context, email, password, ip string) (accessToken, refreshToken string, user *domain.User, err error)
	Refresh(ctx context.Context, refreshToken string) (string, string, error)
	Logout(ctx context.Context, refreshToken string) error
	ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error
}

type authService struct {
	cfg       *config.Config
	users     repositories.UserRepository
	authRepo  repositories.AuthRepository
	passwords PasswordService
	rateLimit RateLimitService
	roleSvc   *RoleService
}

func NewAuthService(
	cfg *config.Config,
	users repositories.UserRepository,
	authRepo repositories.AuthRepository,
	passwords PasswordService,
	rateLimit RateLimitService,
	roleSvc *RoleService,
) AuthService {
	return &authService{
		cfg:       cfg,
		users:     users,
		authRepo:  authRepo,
		passwords: passwords,
		rateLimit: rateLimit,
		roleSvc:   roleSvc,
	}
}

func (s *authService) Register(ctx context.Context, username, email, password, fullName string) (*domain.User, error) {
	if err := s.passwords.ValidatePolicy(password); err != nil {
		return nil, err
	}
	hash, err := s.passwords.HashPassword(password)
	if err != nil {
		return nil, err
	}
	user, err := s.users.CreateUser(ctx, username, email, hash, fullName, nil)
	if err != nil {
		return nil, err
	}
	_ = s.users.InsertPasswordHistory(ctx, user.ID, hash)
	return user, nil
}

func (s *authService) Login(ctx context.Context, email, password, ip string) (string, string, *domain.User, error) {
	user, err := s.users.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			_ = s.rateLimit.CheckAndRecordLoginAttempt(ctx, "", email, ip, false)
			return "", "", nil, domain.ErrInvalidCredentials
		}
		return "", "", nil, err
	}

	if user.IsActive == false {
		return "", "", nil, domain.ErrUserBlocked
	}

	// Check if user is currently blocked due to previous failed attempts
	if blockedUntil, err := s.authRepo.GetBlockedUserUntil(ctx, user.ID); err == nil && blockedUntil != nil && blockedUntil.After(time.Now()) {
		return "", "", nil, domain.ErrUserBlocked
	}

	if err := s.passwords.CheckPassword(user.PasswordHash, password); err != nil {
		_ = s.rateLimit.CheckAndRecordLoginAttempt(ctx, user.ID, email, ip, false)
		return "", "", nil, domain.ErrInvalidCredentials
	}

	if err := s.rateLimit.CheckAndRecordLoginAttempt(ctx, user.ID, email, ip, true); err != nil {
		if errors.Is(err, domain.ErrUserBlocked) || errors.Is(err, domain.ErrIPBlocked) {
			return "", "", nil, err
		}
		return "", "", nil, err
	}

	access, err := utils.GenerateAccessToken(
		s.cfg.JWTAccessSecret,
		s.cfg.AccessTokenTTL,
		user.ID,
		user.Email,
		"",
	)
	if err != nil {
		return "", "", nil, err
	}
	refresh, err := utils.GenerateRefreshToken(
		s.cfg.JWTRefreshSecret,
		s.cfg.RefreshTokenTTL,
		user.ID,
	)
	if err != nil {
		return "", "", nil, err
	}

	hash := sha256.Sum256([]byte(refresh))
	if err := s.authRepo.CreateRefreshToken(ctx, user.ID, hex.EncodeToString(hash[:]), time.Now().Add(s.cfg.RefreshTokenTTL)); err != nil {
		return "", "", nil, err
	}

	return access, refresh, user, nil
}

func (s *authService) Refresh(ctx context.Context, refreshToken string) (string, string, error) {
	claims, err := utils.ParseRefreshToken(s.cfg.JWTRefreshSecret, refreshToken)
	if err != nil {
		return "", "", domain.ErrInvalidToken
	}

	hash := sha256.Sum256([]byte(refreshToken))
	userID, ok, err := s.authRepo.IsRefreshTokenValid(ctx, hex.EncodeToString(hash[:]))
	if err != nil {
		return "", "", err
	}
	if !ok || userID != claims.UserID {
		return "", "", domain.ErrRefreshTokenRevoked
	}

	user, err := s.users.GetUserByID(ctx, claims.UserID)
	if err != nil {
		return "", "", err
	}

	access, err := utils.GenerateAccessToken(
		s.cfg.JWTAccessSecret,
		s.cfg.AccessTokenTTL,
		user.ID,
		user.Email,
		"",
	)
	if err != nil {
		return "", "", err
	}

	newRefresh, err := utils.GenerateRefreshToken(
		s.cfg.JWTRefreshSecret,
		s.cfg.RefreshTokenTTL,
		user.ID,
	)
	if err != nil {
		return "", "", err
	}

	newHash := sha256.Sum256([]byte(newRefresh))
	if err := s.authRepo.CreateRefreshToken(ctx, user.ID, hex.EncodeToString(newHash[:]), time.Now().Add(s.cfg.RefreshTokenTTL)); err != nil {
		return "", "", err
	}

	// Revoke old token
	_ = s.authRepo.RevokeRefreshToken(ctx, hex.EncodeToString(hash[:]))

	return access, newRefresh, nil
}

func (s *authService) Logout(ctx context.Context, refreshToken string) error {
	hash := sha256.Sum256([]byte(refreshToken))
	return s.authRepo.RevokeRefreshToken(ctx, hex.EncodeToString(hash[:]))
}

func (s *authService) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	user, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	if err := s.passwords.CheckPassword(user.PasswordHash, oldPassword); err != nil {
		return domain.ErrInvalidCredentials
	}

	if err := s.passwords.ValidatePolicy(newPassword); err != nil {
		return err
	}

	lastHashes, err := s.users.GetLastPasswordHashes(ctx, userID, 3)
	if err != nil {
		return err
	}
	if err := s.passwords.CheckNotRecentlyUsed(newPassword, lastHashes); err != nil {
		return err
	}

	newHash, err := s.passwords.HashPassword(newPassword)
	if err != nil {
		return err
	}

	if err := s.users.UpdatePassword(ctx, userID, newHash); err != nil {
		return err
	}
	if err := s.users.InsertPasswordHistory(ctx, userID, newHash); err != nil {
		return err
	}

	// Revoke all existing refresh tokens after password change
	if err := s.authRepo.RevokeAllUserTokens(ctx, userID); err != nil {
		return err
	}

	return nil
}
