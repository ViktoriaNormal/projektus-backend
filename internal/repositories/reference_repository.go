package repositories

import (
	"context"

	"projektus-backend/internal/db"
	"projektus-backend/pkg/errctx"
)

type ReferenceRepository interface {
	ListColumnSystemTypes(ctx context.Context) ([]db.RefColumnSystemType, error)
	ListTaskStatusTypes(ctx context.Context) ([]db.RefTaskStatusType, error)
	ListFieldTypes(ctx context.Context) ([]db.RefFieldType, error)
	ListEstimationUnits(ctx context.Context) ([]db.RefEstimationUnit, error)
	ListPriorityTypes(ctx context.Context) ([]db.RefPriorityType, error)
	ListSystemTaskFields(ctx context.Context) ([]db.RefSystemTaskField, error)
}

type referenceRepository struct {
	q *db.Queries
}

func NewReferenceRepository(q *db.Queries) ReferenceRepository {
	return &referenceRepository{q: q}
}

func (r *referenceRepository) ListColumnSystemTypes(ctx context.Context) ([]db.RefColumnSystemType, error) {
	rows, err := r.q.ListRefColumnSystemTypes(ctx)
	if err != nil {
		return nil, errctx.Wrap(err, "ListColumnSystemTypes")
	}
	return rows, nil
}

func (r *referenceRepository) ListTaskStatusTypes(ctx context.Context) ([]db.RefTaskStatusType, error) {
	rows, err := r.q.ListRefTaskStatusTypes(ctx)
	if err != nil {
		return nil, errctx.Wrap(err, "ListTaskStatusTypes")
	}
	return rows, nil
}

func (r *referenceRepository) ListFieldTypes(ctx context.Context) ([]db.RefFieldType, error) {
	rows, err := r.q.ListRefFieldTypes(ctx)
	if err != nil {
		return nil, errctx.Wrap(err, "ListFieldTypes")
	}
	return rows, nil
}

func (r *referenceRepository) ListEstimationUnits(ctx context.Context) ([]db.RefEstimationUnit, error) {
	rows, err := r.q.ListRefEstimationUnits(ctx)
	if err != nil {
		return nil, errctx.Wrap(err, "ListEstimationUnits")
	}
	return rows, nil
}

func (r *referenceRepository) ListPriorityTypes(ctx context.Context) ([]db.RefPriorityType, error) {
	rows, err := r.q.ListRefPriorityTypes(ctx)
	if err != nil {
		return nil, errctx.Wrap(err, "ListPriorityTypes")
	}
	return rows, nil
}

func (r *referenceRepository) ListSystemTaskFields(ctx context.Context) ([]db.RefSystemTaskField, error) {
	rows, err := r.q.ListRefSystemTaskFields(ctx)
	if err != nil {
		return nil, errctx.Wrap(err, "ListSystemTaskFields")
	}
	return rows, nil
}
