package dto

import (
	"time"

	"github.com/google/uuid"
)

type CommentResponse struct {
	ID        uuid.UUID  `json:"id"`
	TaskID    uuid.UUID  `json:"taskId"`
	AuthorID  uuid.UUID  `json:"authorId"`
	Content   string     `json:"content"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

type CreateCommentRequest struct {
	TaskID       uuid.UUID `json:"taskId" binding:"required"`
	AuthorMemberID uuid.UUID `json:"authorMemberId" binding:"required"`
	Content      string    `json:"content" binding:"required"`
}

