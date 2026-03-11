package services

import (
	"context"
	"time"

	"projektus-backend/config"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
)

type RateLimitService interface {
	CheckAndRecordLoginAttempt(ctx context.Context, userID, email, ip string, success bool) error
}

type rateLimitService struct {
	cfg  *config.Config
	repo repositories.AuthRepository
}

func NewRateLimitService(cfg *config.Config, repo repositories.AuthRepository) RateLimitService {
	return &rateLimitService{
		cfg:  cfg,
		repo: repo,
	}
}

func (s *rateLimitService) CheckAndRecordLoginAttempt(ctx context.Context, userID, email, ip string, success bool) error {
	now := time.Now()

	// Cleanup expired blocks
	_ = s.repo.CleanupExpiredBlockedIPs(ctx)
	_ = s.repo.CleanupExpiredBlockedUsers(ctx)

	// Check user block if we know userID
	if userID != "" {
		userBlockedUntil, err := s.repo.GetBlockedUserUntil(ctx, userID)
		if err != nil {
			return err
		}
		if userBlockedUntil != nil && userBlockedUntil.After(now) {
			return domain.ErrUserBlocked
		}
	}

	// Check IP block
	blockedUntil, err := s.repo.GetBlockedIPUntil(ctx, ip)
	if err != nil {
		return err
	}
	if blockedUntil != nil && blockedUntil.After(now) {
		return domain.ErrIPBlocked
	}

	// If login failed, check thresholds
	if !success {
		emailSince := now.Add(-time.Duration(s.cfg.RateLimitEmailWindowMinutes) * time.Minute)
		ipSince := now.Add(-time.Duration(s.cfg.RateLimitIPWindowMinutes) * time.Minute)

		emailFails, err := s.repo.CountFailedAttemptsByEmailSince(ctx, email, emailSince)
		if err != nil {
			return err
		}
		ipFails, err := s.repo.CountFailedAttemptsByIPSince(ctx, ip, ipSince)
		if err != nil {
			return err
		}

		// Record current attempt
		if err := s.repo.InsertLoginAttempt(ctx, email, ip, false); err != nil {
			return err
		}

		if emailFails+1 >= s.cfg.RateLimitEmailMaxFailures && userID != "" {
			blockDuration := time.Duration(s.cfg.RateLimitEmailBlockMinutes) * time.Minute
			if err := s.repo.BlockUserUntil(ctx, userID, now.Add(blockDuration)); err != nil {
				return err
			}
			return domain.ErrUserBlocked
		}
		if ipFails+1 >= s.cfg.RateLimitIPMaxFailures {
			blockDuration := time.Duration(s.cfg.RateLimitIPWindowMinutes) * time.Minute
			if err := s.repo.BlockIPUntil(ctx, ip, now.Add(blockDuration)); err != nil {
				return err
			}
			return domain.ErrIPBlocked
		}

		return nil
	}

	// success
	if err := s.repo.InsertLoginAttempt(ctx, email, ip, true); err != nil {
		return err
	}
	return nil
}

