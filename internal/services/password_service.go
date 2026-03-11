package services

import (
	"regexp"

	"golang.org/x/crypto/bcrypt"

	"projektus-backend/internal/domain"
)

type PasswordService interface {
	HashPassword(plain string) (string, error)
	CheckPassword(hash, plain string) error
	ValidatePolicy(password string) error
	CheckNotRecentlyUsed(password string, recentHashes []string) error
}

type passwordService struct{}

func NewPasswordService() PasswordService {
	return &passwordService{}
}

func (s *passwordService) HashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (s *passwordService) CheckPassword(hash, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
}

func (s *passwordService) ValidatePolicy(password string) error {
	if len(password) < 8 {
		return domain.ErrPasswordPolicy
	}
	digit := regexp.MustCompile(`[0-9]`)
	upper := regexp.MustCompile(`[A-Z]`)
	special := regexp.MustCompile(`[^a-zA-Z0-9]`)

	if !digit.MatchString(password) || !upper.MatchString(password) || !special.MatchString(password) {
		return domain.ErrPasswordPolicy
	}
	return nil
}

func (s *passwordService) CheckNotRecentlyUsed(password string, recentHashes []string) error {
	for _, h := range recentHashes {
		if bcrypt.CompareHashAndPassword([]byte(h), []byte(password)) == nil {
			return domain.ErrPasswordReuse
		}
	}
	return nil
}

