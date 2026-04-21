package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"

	"projektus-backend/config"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
	"projektus-backend/pkg/errctx"
	"projektus-backend/pkg/utils"
)

type AuthService interface {
	Register(ctx context.Context, username, email, password, fullName string) (*domain.User, error)
	Login(ctx context.Context, username, password, ip string) (accessToken, refreshToken string, user *domain.User, err error)
	Refresh(ctx context.Context, refreshToken string) (string, string, error)
	Logout(ctx context.Context, refreshToken string) error
	ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error
}

type authService struct {
	cfg             *config.Config
	users           repositories.UserRepository
	authRepo        repositories.AuthRepository
	passwords       PasswordService
	policySvc       *PasswordPolicyService
	rateLimit       RateLimitService
	roleSvc         *RoleService
	notificationSvc NotificationService
}

func NewAuthService(
	cfg *config.Config,
	users repositories.UserRepository,
	authRepo repositories.AuthRepository,
	passwords PasswordService,
	policySvc *PasswordPolicyService,
	rateLimit RateLimitService,
	roleSvc *RoleService,
	notificationSvc NotificationService,
) AuthService {
	return &authService{
		cfg:             cfg,
		users:           users,
		authRepo:        authRepo,
		passwords:       passwords,
		policySvc:       policySvc,
		rateLimit:       rateLimit,
		roleSvc:         roleSvc,
		notificationSvc: notificationSvc,
	}
}

func (s *authService) Register(ctx context.Context, username, email, password, fullName string) (*domain.User, error) {
	if err := s.policySvc.ValidatePassword(ctx, password); err != nil {
		return nil, errctx.Wrap(err, "Register", "username", username)
	}
	hash, err := s.passwords.HashPassword(password)
	if err != nil {
		return nil, errctx.Wrap(err, "Register", "username", username)
	}
	user, err := s.users.CreateUser(ctx, username, email, hash, fullName, nil)
	if err != nil {
		return nil, errctx.Wrap(err, "Register", "username", username)
	}
	_ = s.users.InsertPasswordHistory(ctx, user.ID, hash)
	if err := s.notificationSvc.InitializeDefaultSettings(ctx, user.ID.String()); err != nil {
		return nil, errctx.Wrap(err, "Register", "userID", user.ID)
	}

	// Каждому новому пользователю назначается базовая системная роль
	// «Обычный пользователь» — без неё пользователь остался бы с пустым
	// списком ролей, что запрещено (см. domain.ErrUserRequiresRole).
	if err := s.assignDefaultSystemRole(ctx, user.ID.String()); err != nil {
		return nil, errctx.Wrap(err, "Register", "userID", user.ID)
	}
	return user, nil
}

// assignDefaultSystemRole находит системную роль «Обычный пользователь»
// и привязывает её к только что созданному пользователю. Роль гарантированно
// присутствует в БД после bootstrap (ensureSystemRoles) или миграции 000030.
func (s *authService) assignDefaultSystemRole(ctx context.Context, userID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	roles, err := s.roleSvc.ListSystemRoles(ctx)
	if err != nil {
		return err
	}
	for _, r := range roles {
		if r.Name == SystemRoleNameRegularUser {
			return s.roleSvc.AssignSystemRolesToUser(ctx, uid, []uuid.UUID{r.ID})
		}
	}
	return domain.ErrUserRequiresRole
}

func (s *authService) Login(ctx context.Context, username, password, ip string) (string, string, *domain.User, error) {
	user, err := s.users.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			_ = s.rateLimit.CheckAndRecordLoginAttempt(ctx, "", username, ip, false)
			return "", "", nil, domain.ErrInvalidCredentials
		}
		return "", "", nil, errctx.Wrap(err, "Login", "username", username)
	}

	if !user.IsActive {
		return "", "", nil, domain.ErrUserBlocked
	}

	// Check if user is currently blocked due to previous failed attempts
	if blockedUntil, err := s.authRepo.GetBlockedUserUntil(ctx, user.ID.String()); err == nil && blockedUntil != nil && blockedUntil.After(time.Now()) {
		return "", "", nil, domain.ErrUserBlocked
	}

	if err := s.passwords.CheckPassword(user.PasswordHash, password); err != nil {
		_ = s.rateLimit.CheckAndRecordLoginAttempt(ctx, user.ID.String(), username, ip, false)
		return "", "", nil, domain.ErrInvalidCredentials
	}

	if err := s.rateLimit.CheckAndRecordLoginAttempt(ctx, user.ID.String(), username, ip, true); err != nil {
		if errors.Is(err, domain.ErrUserBlocked) || errors.Is(err, domain.ErrIPBlocked) {
			return "", "", nil, err
		}
		return "", "", nil, errctx.Wrap(err, "Login", "username", username, "userID", user.ID)
	}

	access, err := utils.GenerateAccessToken(
		s.cfg.JWTAccessSecret,
		s.cfg.AccessTokenTTL,
		user.ID.String(),
		user.Email,
		"",
	)
	if err != nil {
		return "", "", nil, errctx.Wrap(err, "Login", "username", username)
	}
	refresh, err := utils.GenerateRefreshToken(
		s.cfg.JWTRefreshSecret,
		s.cfg.RefreshTokenTTL,
		user.ID.String(),
	)
	if err != nil {
		return "", "", nil, errctx.Wrap(err, "Login", "username", username)
	}

	hash := sha256.Sum256([]byte(refresh))
	if err := s.authRepo.CreateRefreshToken(ctx, user.ID.String(), hex.EncodeToString(hash[:]), time.Now().Add(s.cfg.RefreshTokenTTL)); err != nil {
		return "", "", nil, errctx.Wrap(err, "Login", "username", username)
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
		return "", "", errctx.Wrap(err, "Refresh")
	}
	if !ok || userID != claims.UserID {
		return "", "", domain.ErrRefreshTokenRevoked
	}

	uid, err := uuid.Parse(claims.UserID)
	if err != nil {
		return "", "", domain.ErrInvalidToken
	}
	user, err := s.users.GetUserByID(ctx, uid)
	if err != nil {
		return "", "", errctx.Wrap(err, "Refresh", "userID", claims.UserID)
	}

	access, err := utils.GenerateAccessToken(
		s.cfg.JWTAccessSecret,
		s.cfg.AccessTokenTTL,
		user.ID.String(),
		user.Email,
		"",
	)
	if err != nil {
		return "", "", errctx.Wrap(err, "Refresh", "userID", user.ID)
	}

	newRefresh, err := utils.GenerateRefreshToken(
		s.cfg.JWTRefreshSecret,
		s.cfg.RefreshTokenTTL,
		user.ID.String(),
	)
	if err != nil {
		return "", "", errctx.Wrap(err, "Refresh", "userID", user.ID)
	}

	newHash := sha256.Sum256([]byte(newRefresh))
	if err := s.authRepo.CreateRefreshToken(ctx, user.ID.String(), hex.EncodeToString(newHash[:]), time.Now().Add(s.cfg.RefreshTokenTTL)); err != nil {
		return "", "", errctx.Wrap(err, "Refresh", "userID", user.ID)
	}

	// Revoke old token
	_ = s.authRepo.RevokeRefreshToken(ctx, hex.EncodeToString(hash[:]))

	return access, newRefresh, nil
}

func (s *authService) Logout(ctx context.Context, refreshToken string) error {
	hash := sha256.Sum256([]byte(refreshToken))
	err := s.authRepo.RevokeRefreshToken(ctx, hex.EncodeToString(hash[:]))
	return errctx.Wrap(err, "Logout")
}

func (s *authService) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return domain.ErrInvalidToken
	}
	user, err := s.users.GetUserByID(ctx, uid)
	if err != nil {
		return errctx.Wrap(err, "ChangePassword", "userID", userID)
	}

	if err := s.passwords.CheckPassword(user.PasswordHash, oldPassword); err != nil {
		return domain.ErrInvalidCredentials
	}

	if err := s.policySvc.ValidatePassword(ctx, newPassword); err != nil {
		return errctx.Wrap(err, "ChangePassword", "userID", userID)
	}

	lastHashes, err := s.users.GetLastPasswordHashes(ctx, uid, 3)
	if err != nil {
		return errctx.Wrap(err, "ChangePassword", "userID", userID)
	}
	if err := s.passwords.CheckNotRecentlyUsed(newPassword, lastHashes); err != nil {
		return err
	}

	newHash, err := s.passwords.HashPassword(newPassword)
	if err != nil {
		return errctx.Wrap(err, "ChangePassword", "userID", userID)
	}

	if err := s.users.UpdatePassword(ctx, uid, newHash); err != nil {
		return errctx.Wrap(err, "ChangePassword", "userID", userID)
	}
	if err := s.users.InsertPasswordHistory(ctx, uid, newHash); err != nil {
		return errctx.Wrap(err, "ChangePassword", "userID", userID)
	}

	if err := s.authRepo.RevokeAllUserTokens(ctx, userID); err != nil {
		return errctx.Wrap(err, "ChangePassword", "userID", userID)
	}

	return nil
}
