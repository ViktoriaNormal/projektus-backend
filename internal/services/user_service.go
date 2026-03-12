package services

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
	"projektus-backend/pkg/utils"
)

type UserService interface {
	GetProfile(ctx context.Context, id string) (*domain.User, error)
	UpdateProfile(ctx context.Context, currentUserID, targetUserID, fullName, email string, isAdmin bool) (*domain.User, error)
	UpdateAvatar(ctx context.Context, currentUserID, targetUserID string, fileName string, data []byte, isAdmin bool) (*domain.User, error)
	SearchUsers(ctx context.Context, query string, limit, offset int32) ([]domain.User, error)
}

type userService struct {
	users repositories.UserRepository
}

func NewUserService(users repositories.UserRepository) UserService {
	return &userService{users: users}
}

func (s *userService) GetProfile(ctx context.Context, id string) (*domain.User, error) {
	return s.users.GetUserByID(ctx, id)
}

func (s *userService) UpdateProfile(ctx context.Context, currentUserID, targetUserID, fullName, email string, isAdmin bool) (*domain.User, error) {
	if currentUserID != targetUserID && !isAdmin {
		return nil, domain.ErrAccessDenied
	}
	if err := s.users.UpdateProfile(ctx, targetUserID, fullName, email); err != nil {
		return nil, err
	}
	return s.users.GetUserByID(ctx, targetUserID)
}

func (s *userService) UpdateAvatar(ctx context.Context, currentUserID, targetUserID string, fileName string, data []byte, isAdmin bool) (*domain.User, error) {
	if currentUserID != targetUserID && !isAdmin {
		return nil, domain.ErrAccessDenied
	}
	ext := filepath.Ext(fileName)
	newName := fmt.Sprintf("%s_%d%s", targetUserID, time.Now().Unix(), ext)
	relPath := filepath.Join("uploads", "avatars", newName)

	if err := utils.SaveFile(relPath, data); err != nil {
		return nil, err
	}
	if err := s.users.UpdateAvatar(ctx, targetUserID, "/"+relPath); err != nil {
		return nil, err
	}
	return s.users.GetUserByID(ctx, targetUserID)
}

func (s *userService) SearchUsers(ctx context.Context, query string, limit, offset int32) ([]domain.User, error) {
	return s.users.SearchUsers(ctx, query, limit, offset)
}

