package repositories

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
)

type PasswordPolicyRepository interface {
	GetCurrent(ctx context.Context) (*domain.PasswordPolicy, error)
	Insert(ctx context.Context, minLength int, requireDigits, requireLowercase, requireUppercase, requireSpecial bool, notes *string, updatedBy *uuid.UUID) (*domain.PasswordPolicy, error)
}

type passwordPolicyRepository struct {
	q *db.Queries
}

func NewPasswordPolicyRepository(q *db.Queries) PasswordPolicyRepository {
	return &passwordPolicyRepository{q: q}
}

func (r *passwordPolicyRepository) GetCurrent(ctx context.Context) (*domain.PasswordPolicy, error) {
	row, err := r.q.GetCurrentPasswordPolicy(ctx)
	if err != nil {
		return nil, err
	}
	return mapDBPolicyToDomain(row), nil
}

func (r *passwordPolicyRepository) Insert(ctx context.Context, minLength int, requireDigits, requireLowercase, requireUppercase, requireSpecial bool, notes *string, updatedBy *uuid.UUID) (*domain.PasswordPolicy, error) {
	arg := db.InsertPasswordPolicyParams{
		MinLength:        int32(minLength),
		RequireDigits:    requireDigits,
		RequireLowercase: requireLowercase,
		RequireUppercase: requireUppercase,
		RequireSpecial:   requireSpecial,
		UpdatedBy:        uuid.NullUUID{},
	}
	if notes != nil {
		arg.Notes = sql.NullString{String: *notes, Valid: true}
	} else {
		arg.Notes = sql.NullString{}
	}
	if updatedBy != nil {
		arg.UpdatedBy = uuid.NullUUID{UUID: *updatedBy, Valid: true}
	}
	row, err := r.q.InsertPasswordPolicy(ctx, arg)
	if err != nil {
		return nil, err
	}
	return mapDBPolicyToDomain(row), nil
}

func mapDBPolicyToDomain(p db.PasswordPolicy) *domain.PasswordPolicy {
	var notes *string
	if p.Notes.Valid {
		notes = &p.Notes.String
	}
	var updatedBy *uuid.UUID
	if p.UpdatedBy.Valid {
		updatedBy = &p.UpdatedBy.UUID
	}
	return &domain.PasswordPolicy{
		ID:               p.ID,
		MinLength:        int(p.MinLength),
		RequireDigits:    p.RequireDigits,
		RequireLowercase: p.RequireLowercase,
		RequireUppercase: p.RequireUppercase,
		RequireSpecial:   p.RequireSpecial,
		Notes:            notes,
		UpdatedAt:        p.UpdatedAt,
		UpdatedBy:        updatedBy,
	}
}
