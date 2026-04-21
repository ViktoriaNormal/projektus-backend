package domain

import (
	"time"

	"github.com/google/uuid"
)

type Comment struct {
	ID              uuid.UUID        `json:"id"`
	TaskID          uuid.UUID        `json:"task_id"`
	AuthorID        uuid.UUID        `json:"author_id"`
	Content         string           `json:"content"`
	ParentCommentID *uuid.UUID       `json:"parent_comment_id,omitempty"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	Mentions        []CommentMention `json:"mentions,omitempty"`
}

type CommentMention struct {
	ID              uuid.UUID `json:"id"`
	CommentID       uuid.UUID `json:"comment_id"`
	ProjectMemberID uuid.UUID `json:"project_member_id"`
}
