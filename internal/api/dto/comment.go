package dto

import (
	"time"

	"github.com/google/uuid"
)

type CommentResponse struct {
	ID        uuid.UUID  `json:"id"`
	TaskID    uuid.UUID  `json:"task_id"`
	AuthorID  uuid.UUID  `json:"author_id"`
	Content   string     `json:"content"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type CreateCommentRequest struct {
	TaskID       uuid.UUID `json:"task_id" binding:"required"`
	AuthorMemberID uuid.UUID `json:"author_member_id" binding:"required"`
	Content      string    `json:"content" binding:"required"`
}
