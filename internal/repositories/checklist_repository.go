package repositories

import (
	"context"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
	"projektus-backend/pkg/errctx"
)

type ChecklistRepository interface {
	Create(ctx context.Context, taskID uuid.UUID, name string) (*domain.Checklist, error)
	UpdateName(ctx context.Context, checklistID uuid.UUID, name string) (*domain.Checklist, error)
	Delete(ctx context.Context, checklistID uuid.UUID) error
	ListByTask(ctx context.Context, taskID uuid.UUID) ([]domain.Checklist, error)
	CreateItem(ctx context.Context, checklistID uuid.UUID, content string, order int16) (*domain.ChecklistItem, error)
	ListItems(ctx context.Context, checklistID uuid.UUID) ([]domain.ChecklistItem, error)
	UpdateItemStatus(ctx context.Context, itemID uuid.UUID, isChecked bool) (*domain.ChecklistItem, error)
	UpdateItemContent(ctx context.Context, itemID uuid.UUID, content string) (*domain.ChecklistItem, error)
	DeleteItem(ctx context.Context, itemID uuid.UUID) error
}

type checklistRepository struct {
	q *db.Queries
}

func NewChecklistRepository(q *db.Queries) ChecklistRepository {
	return &checklistRepository{q: q}
}

func (r *checklistRepository) Create(ctx context.Context, taskID uuid.UUID, name string) (*domain.Checklist, error) {
	row, err := r.q.CreateChecklist(ctx, db.CreateChecklistParams{
		TaskID: taskID,
		Name:   name,
	})
	if err != nil {
		return nil, errctx.Wrap(err, "CreateChecklist", "taskID", taskID, "name", name)
	}
	return &domain.Checklist{
		ID:     row.ID,
		TaskID: row.TaskID,
		Name:   row.Name,
	}, nil
}

func (r *checklistRepository) ListByTask(ctx context.Context, taskID uuid.UUID) ([]domain.Checklist, error) {
	rows, err := r.q.ListChecklistsByTask(ctx, taskID)
	if err != nil {
		return nil, errctx.Wrap(err, "ListChecklistsByTask", "taskID", taskID)
	}
	result := make([]domain.Checklist, 0, len(rows))
	for _, row := range rows {
		result = append(result, domain.Checklist{
			ID:     row.ID,
			TaskID: row.TaskID,
			Name:   row.Name,
		})
	}
	return result, nil
}

func (r *checklistRepository) CreateItem(ctx context.Context, checklistID uuid.UUID, content string, order int16) (*domain.ChecklistItem, error) {
	row, err := r.q.CreateChecklistItem(ctx, db.CreateChecklistItemParams{
		ChecklistID: checklistID,
		Content:     content,
		IsChecked:   false,
		SortOrder:   order,
	})
	if err != nil {
		return nil, errctx.Wrap(err, "CreateChecklistItem", "checklistID", checklistID)
	}
	return &domain.ChecklistItem{
		ID:          row.ID,
		ChecklistID: row.ChecklistID,
		Content:     row.Content,
		IsChecked:   row.IsChecked,
		Order:       row.SortOrder,
	}, nil
}

func (r *checklistRepository) ListItems(ctx context.Context, checklistID uuid.UUID) ([]domain.ChecklistItem, error) {
	rows, err := r.q.ListChecklistItems(ctx, checklistID)
	if err != nil {
		return nil, errctx.Wrap(err, "ListChecklistItems", "checklistID", checklistID)
	}
	result := make([]domain.ChecklistItem, 0, len(rows))
	for _, row := range rows {
		result = append(result, domain.ChecklistItem{
			ID:          row.ID,
			ChecklistID: row.ChecklistID,
			Content:     row.Content,
			IsChecked:   row.IsChecked,
			Order:       row.SortOrder,
		})
	}
	return result, nil
}

func (r *checklistRepository) UpdateItemStatus(ctx context.Context, itemID uuid.UUID, isChecked bool) (*domain.ChecklistItem, error) {
	row, err := r.q.UpdateChecklistItemStatus(ctx, db.UpdateChecklistItemStatusParams{
		ID:        itemID,
		IsChecked: isChecked,
	})
	if err != nil {
		return nil, errctx.Wrap(err, "UpdateChecklistItemStatus", "itemID", itemID)
	}
	return &domain.ChecklistItem{
		ID:          row.ID,
		ChecklistID: row.ChecklistID,
		Content:     row.Content,
		IsChecked:   row.IsChecked,
		Order:       row.SortOrder,
	}, nil
}

func (r *checklistRepository) UpdateName(ctx context.Context, checklistID uuid.UUID, name string) (*domain.Checklist, error) {
	row, err := r.q.UpdateChecklistName(ctx, db.UpdateChecklistNameParams{ID: checklistID, Name: name})
	if err != nil {
		return nil, errctx.Wrap(err, "UpdateChecklistName", "checklistID", checklistID)
	}
	return &domain.Checklist{ID: row.ID, TaskID: row.TaskID, Name: row.Name}, nil
}

func (r *checklistRepository) Delete(ctx context.Context, checklistID uuid.UUID) error {
	return errctx.Wrap(r.q.DeleteChecklist(ctx, checklistID), "DeleteChecklist", "checklistID", checklistID)
}

func (r *checklistRepository) UpdateItemContent(ctx context.Context, itemID uuid.UUID, content string) (*domain.ChecklistItem, error) {
	row, err := r.q.UpdateChecklistItemContent(ctx, db.UpdateChecklistItemContentParams{ID: itemID, Content: content})
	if err != nil {
		return nil, errctx.Wrap(err, "UpdateChecklistItemContent", "itemID", itemID)
	}
	return &domain.ChecklistItem{
		ID: row.ID, ChecklistID: row.ChecklistID,
		Content: row.Content, IsChecked: row.IsChecked, Order: row.SortOrder,
	}, nil
}

func (r *checklistRepository) DeleteItem(ctx context.Context, itemID uuid.UUID) error {
	return errctx.Wrap(r.q.DeleteChecklistItem(ctx, itemID), "DeleteChecklistItem", "itemID", itemID)
}
