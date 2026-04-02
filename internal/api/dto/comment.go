package dto

import (
	"time"

	"github.com/google/uuid"
)

type CommentResponse struct {
	ID              uuid.UUID  `json:"id"`
	TaskID          uuid.UUID  `json:"task_id"`
	AuthorID        uuid.UUID  `json:"author_id"`
	Content         string     `json:"content"`
	ParentCommentID *uuid.UUID `json:"parent_comment_id,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type CreateCommentRequest struct {
	Content         string     `json:"content" binding:"required"`
	ParentCommentID *uuid.UUID `json:"parent_comment_id,omitempty"`
}
