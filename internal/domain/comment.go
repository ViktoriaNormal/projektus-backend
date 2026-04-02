package domain

import "time"

type Comment struct {
	ID              string           `json:"id"`
	TaskID          string           `json:"task_id"`
	AuthorID        string           `json:"author_id"`
	Content         string           `json:"content"`
	ParentCommentID *string          `json:"parent_comment_id,omitempty"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	Mentions        []CommentMention `json:"mentions,omitempty"`
}

type CommentMention struct {
	ID              string `json:"id"`
	CommentID       string `json:"comment_id"`
	ProjectMemberID string `json:"project_member_id"`
}
