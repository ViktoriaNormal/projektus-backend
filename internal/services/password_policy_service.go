package services

import (
	"context"
	"database/sql"
	"errors"
	"regexp"

	"github.com/google/uuid"

	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
	"projektus-backend/pkg/errctx"
)

// UpdatePasswordPolicyRequest — запрос на обновление парольной политики.
type UpdatePasswordPolicyRequest struct {
	MinLength        int
	RequireDigits    bool
	RequireLowercase bool
	RequireUppercase bool
	RequireSpecial   bool
	Notes            *string
}

// PasswordPolicyService — получение и обновление парольной политики, проверка пароля.
type PasswordPolicyService struct {
	repo repositories.PasswordPolicyRepository
}

func NewPasswordPolicyService(repo repositories.PasswordPolicyRepository) *PasswordPolicyService {
	return &PasswordPolicyService{repo: repo}
}

// GetCurrentPolicy возвращает текущую парольную политику.
func (s *PasswordPolicyService) GetCurrentPolicy(ctx context.Context) (*domain.PasswordPolicy, error) {
	p, err := s.repo.GetCurrent(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoPasswordPolicy
		}
		return nil, errctx.Wrap(err, "GetCurrentPolicy")
	}
	return p, nil
}

// UpdatePolicy сохраняет новую версию политики (актуальной считается последняя).
func (s *PasswordPolicyService) UpdatePolicy(ctx context.Context, req UpdatePasswordPolicyRequest, updatedBy uuid.UUID) (*domain.PasswordPolicy, error) {
	if req.MinLength < 1 || req.MinLength > 100 {
		return nil, domain.ErrInvalidInput
	}
	policy, err := s.repo.Insert(ctx,
		req.MinLength,
		req.RequireDigits,
		req.RequireLowercase,
		req.RequireUppercase,
		req.RequireSpecial,
		req.Notes,
		&updatedBy,
	)
	if err != nil {
		return nil, errctx.Wrap(err, "UpdatePolicy", "updatedBy", updatedBy)
	}
	return policy, nil
}

// ValidatePassword проверяет пароль по текущей политике из БД.
// Если политика ещё не настроена — валидация пропускается.
func (s *PasswordPolicyService) ValidatePassword(ctx context.Context, password string) error {
	policy, err := s.GetCurrentPolicy(ctx)
	if err != nil {
		if errors.Is(err, ErrNoPasswordPolicy) {
			return nil
		}
		return errctx.Wrap(err, "ValidatePassword")
	}
	return validatePasswordAgainstPolicy(password, policy)
}

func validatePasswordAgainstPolicy(password string, p *domain.PasswordPolicy) error {
	if len(password) < p.MinLength {
		return domain.ErrPasswordPolicy
	}
	hasDigit := regexp.MustCompile(`[0-9]`).MatchString
	hasLower := regexp.MustCompile(`[a-z]`).MatchString
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString
	hasSpecial := regexp.MustCompile(`[^a-zA-Z0-9]`).MatchString
	if p.RequireDigits && !hasDigit(password) {
		return domain.ErrPasswordPolicy
	}
	if p.RequireLowercase && !hasLower(password) {
		return domain.ErrPasswordPolicy
	}
	if p.RequireUppercase && !hasUpper(password) {
		return domain.ErrPasswordPolicy
	}
	if p.RequireSpecial && !hasSpecial(password) {
		return domain.ErrPasswordPolicy
	}
	return nil
}

// ErrNoPasswordPolicy возвращается, если в БД нет ни одной записи политики.
var ErrNoPasswordPolicy = errors.New("password policy not found")
