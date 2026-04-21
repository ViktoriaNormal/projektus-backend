package repositories

import (
	"context"

	"projektus-backend/internal/catalog"
	"projektus-backend/internal/domain"
)

// ReferenceRepository — адаптер над пакетом `catalog` (in-memory справочники).
// Интерфейс сохраняется для совместимости с сервисами, которые получают его
// через DI. Сами данные живут в `internal/catalog`.
//
// Для прямого (не-DI) доступа предпочитайте обращаться к `catalog.*` напрямую —
// это дешевле и не требует context.
type ReferenceRepository interface {
	ListColumnSystemTypes(ctx context.Context) ([]domain.RefColumnSystemType, error)
	ListFieldTypes(ctx context.Context) ([]domain.FieldTypeDefinition, error)
	ListEstimationUnits(ctx context.Context) ([]domain.RefAvailable, error)
	ListPriorityTypes(ctx context.Context) ([]domain.RefPriorityType, error)
	ListPermissionAreas(ctx context.Context) ([]domain.RefPermissionArea, error)
	ListAccessLevels(ctx context.Context) ([]domain.RefKeyName, error)
}

type referenceRepository struct{}

func NewReferenceRepository() ReferenceRepository {
	return &referenceRepository{}
}

func (r *referenceRepository) ListColumnSystemTypes(_ context.Context) ([]domain.RefColumnSystemType, error) {
	return catalog.ColumnSystemTypes, nil
}

func (r *referenceRepository) ListFieldTypes(_ context.Context) ([]domain.FieldTypeDefinition, error) {
	return catalog.FieldTypes, nil
}

func (r *referenceRepository) ListEstimationUnits(_ context.Context) ([]domain.RefAvailable, error) {
	return catalog.EstimationUnits, nil
}

func (r *referenceRepository) ListPriorityTypes(_ context.Context) ([]domain.RefPriorityType, error) {
	return catalog.PriorityTypes, nil
}

func (r *referenceRepository) ListPermissionAreas(_ context.Context) ([]domain.RefPermissionArea, error) {
	return catalog.ProjectPermissionAreas, nil
}

func (r *referenceRepository) ListAccessLevels(_ context.Context) ([]domain.RefKeyName, error) {
	return catalog.AccessLevels, nil
}

// Алиасы для обратной совместимости с кодом, где ссылки ещё не переключены
// на catalog.*. Постепенно их переместить и эти алиасы удалить.
var (
	AllPermissions         = catalog.AllPermissions
	ProjectPermissionAreas = catalog.ProjectPermissionAreas
	AccessLevels           = catalog.AccessLevels
	ColumnSystemTypes      = catalog.ColumnSystemTypes
	FieldTypes             = catalog.FieldTypes
	ProjectStatuses        = catalog.ProjectStatuses
	EstimationUnits        = catalog.EstimationUnits
	PriorityTypes          = catalog.PriorityTypes
	DefaultColumns         = catalog.DefaultColumns
	DefaultBoardFields     = catalog.DefaultBoardFields
)

const (
	SystemPermissionManageRoles          = catalog.SystemPermissionManageRoles
	SystemPermissionManageUsers          = catalog.SystemPermissionManageUsers
	SystemPermissionManageProjects       = catalog.SystemPermissionManageProjects
	SystemPermissionManagePasswordPolicy = catalog.SystemPermissionManagePasswordPolicy
	SystemPermissionManageTemplates      = catalog.SystemPermissionManageTemplates
)
