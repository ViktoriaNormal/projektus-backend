package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
	"projektus-backend/pkg/utils"
)

type UserService interface {
	GetProfile(ctx context.Context, id string) (*domain.User, error)
	UpdateProfile(ctx context.Context, currentUserID, targetUserID, fullName, email string, position *string, onVacation bool, isSick bool, altContactChannel *string, altContactInfo *string, isAdmin bool) (*domain.User, error)
	UpdateAvatar(ctx context.Context, currentUserID, targetUserID string, fileName string, data []byte, isAdmin bool) (*domain.User, error)
	// SearchUsers возвращает страницу пользователей по фильтру query и полное
	// число записей, подходящих под фильтр (для пагинатора на фронте).
	// Если limit == 0 — страница пустая, возвращаем только total.
	SearchUsers(ctx context.Context, query string, limit, offset int32) ([]domain.User, int64, error)
}

type userService struct {
	users repositories.UserRepository
}

func NewUserService(users repositories.UserRepository) UserService {
	return &userService{users: users}
}

func (s *userService) GetProfile(ctx context.Context, id string) (*domain.User, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, domain.ErrInvalidInput
	}
	return s.users.GetUserByID(ctx, uid)
}

func (s *userService) UpdateProfile(ctx context.Context, currentUserID, targetUserID, fullName, email string, position *string, onVacation bool, isSick bool, altContactChannel *string, altContactInfo *string, isAdmin bool) (*domain.User, error) {
	if currentUserID != targetUserID && !isAdmin {
		return nil, domain.ErrAccessDenied
	}
	tid, err := uuid.Parse(targetUserID)
	if err != nil {
		return nil, domain.ErrInvalidInput
	}
	if err := s.users.UpdateProfile(ctx, tid, fullName, email, position, onVacation, isSick, altContactChannel, altContactInfo); err != nil {
		return nil, err
	}
	return s.users.GetUserByID(ctx, tid)
}

var allowedAvatarExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".webp": true,
}

func (s *userService) UpdateAvatar(ctx context.Context, currentUserID, targetUserID string, fileName string, data []byte, isAdmin bool) (*domain.User, error) {
	if currentUserID != targetUserID && !isAdmin {
		return nil, domain.ErrAccessDenied
	}

	ext := strings.ToLower(filepath.Ext(fileName))
	if !allowedAvatarExtensions[ext] {
		return nil, domain.ErrInvalidInput
	}

	tid, err := uuid.Parse(targetUserID)
	if err != nil {
		return nil, domain.ErrInvalidInput
	}

	// Удалить старый аватар с диска
	user, err := s.users.GetUserByID(ctx, tid)
	if err != nil {
		return nil, err
	}
	if user.AvatarURL != nil && *user.AvatarURL != "" {
		oldPath := strings.TrimPrefix(*user.AvatarURL, "/")
		_ = os.Remove(oldPath)
	}

	newName := fmt.Sprintf("%s_%d%s", targetUserID, time.Now().Unix(), ext)
	relPath := filepath.Join("uploads", "avatars", newName)

	if err := utils.SaveFile(relPath, data); err != nil {
		return nil, err
	}
	if err := s.users.UpdateAvatar(ctx, tid, "/"+relPath); err != nil {
		return nil, err
	}
	return s.users.GetUserByID(ctx, tid)
}

func (s *userService) SearchUsers(ctx context.Context, query string, limit, offset int32) ([]domain.User, int64, error) {
	total, err := s.users.CountSearchUsers(ctx, query)
	if err != nil {
		return nil, 0, err
	}
	if limit == 0 {
		// Запрос только счётчика — страница не нужна (например, для пагинатора,
		// где отдельно загружается каждая видимая страница).
		return nil, total, nil
	}
	users, err := s.users.SearchUsers(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	return users, total, nil
}
