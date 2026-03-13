package services

import (
	"context"

	"github.com/google/uuid"

	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
)

type BoardService struct {
	repo repositories.BoardRepository
}

func NewBoardService(repo repositories.BoardRepository) *BoardService {
	return &BoardService{repo: repo}
}

func (s *BoardService) ListBoards(ctx context.Context, projectID uuid.UUID) ([]domain.Board, error) {
	return s.repo.ListProjectBoards(ctx, projectID.String())
}

func (s *BoardService) CreateBoard(ctx context.Context, projectID uuid.UUID, name, description string, order int16) (*domain.Board, error) {
	var desc *string
	if description != "" {
		desc = &description
	}
	pid := projectID.String()
	board := &domain.Board{
		ProjectID:   &pid,
		Name:        name,
		Description: desc,
		Order:       order,
	}
	return s.repo.CreateBoard(ctx, board)
}

func (s *BoardService) GetBoard(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
	return s.repo.GetBoardByID(ctx, id.String())
}

func (s *BoardService) UpdateBoard(ctx context.Context, b *domain.Board) (*domain.Board, error) {
	return s.repo.UpdateBoard(ctx, b)
}

func (s *BoardService) DeleteBoard(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteBoard(ctx, id.String())
}

func (s *BoardService) ListColumns(ctx context.Context, boardID uuid.UUID) ([]domain.Column, error) {
	return s.repo.ListColumns(ctx, boardID.String())
}

func (s *BoardService) CreateColumn(ctx context.Context, boardID uuid.UUID, name string, systemType *domain.SystemStatusType, wipLimit *int16, order int16) (*domain.Column, error) {
	if systemType == nil {
		return nil, domain.ErrInvalidInput
	}
	switch *systemType {
	case domain.StatusInitial, domain.StatusInProgress, domain.StatusPaused, domain.StatusCompleted, domain.StatusCancelled:
		// ok
	default:
		return nil, domain.ErrInvalidInput
	}

	col := &domain.Column{
		BoardID:    boardID.String(),
		Name:       name,
		SystemType: systemType,
		WipLimit:   wipLimit,
		Order:      order,
	}
	return s.repo.CreateColumn(ctx, col)
}

func (s *BoardService) UpdateColumn(ctx context.Context, c *domain.Column) (*domain.Column, error) {
	return s.repo.UpdateColumn(ctx, c)
}

func (s *BoardService) DeleteColumn(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteColumn(ctx, id.String())
}

func (s *BoardService) ListSwimlanes(ctx context.Context, boardID uuid.UUID) ([]domain.Swimlane, error) {
	return s.repo.ListSwimlanes(ctx, boardID.String())
}

func (s *BoardService) CreateSwimlane(ctx context.Context, boardID uuid.UUID, name string, wipLimit *int16, order int16) (*domain.Swimlane, error) {
	sw := &domain.Swimlane{
		BoardID:  boardID.String(),
		Name:     name,
		WipLimit: wipLimit,
		Order:    order,
	}
	return s.repo.CreateSwimlane(ctx, sw)
}

func (s *BoardService) UpdateSwimlane(ctx context.Context, sw *domain.Swimlane) (*domain.Swimlane, error) {
	return s.repo.UpdateSwimlane(ctx, sw)
}

func (s *BoardService) DeleteSwimlane(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteSwimlane(ctx, id.String())
}

func (s *BoardService) ListNotes(ctx context.Context, boardID uuid.UUID) ([]domain.Note, error) {
	return s.repo.ListNotes(ctx, boardID.String())
}

func (s *BoardService) CreateNoteForColumn(ctx context.Context, columnID uuid.UUID, content string) (*domain.Note, error) {
	cid := columnID.String()
	n := &domain.Note{
		ColumnID: &cid,
		Content:  content,
	}
	return s.repo.CreateNoteForColumn(ctx, n)
}

func (s *BoardService) CreateNoteForSwimlane(ctx context.Context, swimlaneID uuid.UUID, content string) (*domain.Note, error) {
	sid := swimlaneID.String()
	n := &domain.Note{
		SwimlaneID: &sid,
		Content:    content,
	}
	return s.repo.CreateNoteForSwimlane(ctx, n)
}

func (s *BoardService) UpdateNote(ctx context.Context, n *domain.Note) (*domain.Note, error) {
	return s.repo.UpdateNote(ctx, n)
}

func (s *BoardService) DeleteNote(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteNote(ctx, id.String())
}

